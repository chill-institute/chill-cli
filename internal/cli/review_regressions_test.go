package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
)

func TestLoginWithoutHealthEndpoint(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		t.Run(map[bool]string{false: "single custom host", true: "all probes fail"}[multiple], func(t *testing.T) {
			var probes atomic.Int32
			primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/healthz" {
					probes.Add(1)
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(`{"user_id":"test-user"}`))
			}))
			defer primary.Close()
			hosts := primary.URL
			if multiple {
				hosts += ",http://127.0.0.1:1"
			}
			app := failoverTestApp(t, hosts)
			app.stderr = &bytes.Buffer{}
			app.isInputTerminal = func(io.Reader) bool { return true }
			app.readSecret = func(io.Reader) (string, error) { return "new-token", nil }
			cmd := newAuthLoginCommand(app)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			cfg, err := app.loadConfig()
			if err != nil || cfg.AuthToken != "new-token" {
				t.Fatalf("login config=%+v err=%v", cfg, err)
			}
			if !multiple && probes.Load() != 0 {
				t.Fatal("single host was probed")
			}
			if multiple && !strings.Contains(app.stderr.(*bytes.Buffer).String(), "Warning: API health probes failed") {
				t.Fatalf("missing warning: %s", app.stderr)
			}
		})
	}
}

func TestRPCPreservesCallerDeadline(t *testing.T) {
	app := failoverTestApp(t, "https://api.chill.institute")
	app.rpcTimeout = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	callCtx, stop := app.rpcCallContext(ctx)
	defer stop()
	select {
	case <-callCtx.Done():
		t.Fatalf("caller budget shortened: %v", callCtx.Err())
	case <-time.After(20 * time.Millisecond):
	}
	cancelled, cancelNow := context.WithCancel(ctx)
	cancelNow()
	callCtx, stop = app.rpcCallContext(cancelled)
	defer stop()
	_, err := rpc.NewClient("https://api.example", nil).Call(callCtx, rpc.CallRequest{Procedure: "test/Read", AuthMode: rpc.AuthNone, ReadOnly: true})
	if err == nil {
		t.Fatal("ignored caller cancellation")
	}
}

func TestHealthProbeIsBoundedAndLeavesRequestBudget(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer fallback.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := rpc.NewClient(primary.URL, nil, rpc.WithFailoverBaseURLs(fallback.URL))
	start := time.Now()
	host, err := client.FirstReachableBaseURL(ctx)
	if err != nil || host != fallback.URL || time.Since(start) > 3*time.Second {
		t.Fatalf("probe host=%s err=%v elapsed=%v", host, err, time.Since(start))
	}
	if _, err := rpc.NewClient(host, nil).Call(ctx, rpc.CallRequest{Procedure: "test/Read", AuthMode: rpc.AuthNone, ReadOnly: true}); err != nil {
		t.Fatalf("request after probe: %v", err)
	}
}

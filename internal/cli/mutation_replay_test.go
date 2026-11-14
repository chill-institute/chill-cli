package cli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
)

func TestMutationDoesNotReplayAfterHeadersSent(t *testing.T) {
	var firstCalls, fallbackCalls atomic.Int32
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstCalls.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	}))
	defer first.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalls.Add(1)
	}))
	defer fallback.Close()
	app := failoverTestApp(t, first.URL+","+fallback.URL)
	command := newAddTransferCommand(app)
	command.SetArgs([]string{"--url", "https://example.test/file.torrent"})
	if err := command.Execute(); err == nil || firstCalls.Load() != 1 || fallbackCalls.Load() != 0 {
		t.Fatalf("error=%v first=%d fallback=%d", err, firstCalls.Load(), fallbackCalls.Load())
	}
}

func TestMutationDoesNotReplayAmbiguousTransportFailure(t *testing.T) {
	calls := 0
	client := rpc.NewClient("https://first.example", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("transport failed")
	})}, rpc.WithFailoverBaseURLs("https://second.example"))
	_, err := client.Call(context.Background(), rpc.CallRequest{Procedure: "test/Mutate", AuthMode: rpc.AuthNone})
	if err == nil || calls != 1 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

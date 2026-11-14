package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type countingHost struct {
	server *httptest.Server
	hits   atomic.Int32
}

func newCountingHost(t *testing.T, handler http.HandlerFunc) *countingHost {
	t.Helper()
	host := &countingHost{}
	host.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		host.hits.Add(1)
		handler(writer, request)
	}))
	t.Cleanup(host.server.Close)
	return host
}

func respondStatus(status int) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(`{"code":"x","message":"y"}`))
	}
}

func respondOK(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(`{"ok":true}`))
}

func closedPortURL(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return "http://" + address
}

func callWithFailover(t *testing.T, httpClient *http.Client, readOnly bool, baseURLs ...string) (CallResponse, error) {
	t.Helper()
	client := NewClient(baseURLs[0], httpClient, WithFailoverBaseURLs(baseURLs[1:]...))
	return client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/AddTransfer",
		AuthMode:  AuthNone,
		ReadOnly:  readOnly,
	})
}

func TestCallFailsOverWhenHostRefusesConnection(t *testing.T) {
	t.Parallel()

	healthy := newCountingHost(t, respondOK)
	for _, readOnly := range []bool{false, true} {
		response, err := callWithFailover(t, nil, readOnly, closedPortURL(t), healthy.server.URL)
		if err != nil {
			t.Fatalf("readOnly=%v: Call() error = %v", readOnly, err)
		}
		if response.BaseURL != healthy.server.URL {
			t.Fatalf("readOnly=%v: BaseURL = %q, want %q", readOnly, response.BaseURL, healthy.server.URL)
		}
	}
	if got := healthy.hits.Load(); got != 2 {
		t.Fatalf("healthy host hits = %d, want 2", got)
	}
}

func TestCallFailsOverWhenHostCannotBeResolved(t *testing.T) {
	t.Parallel()

	healthy := newCountingHost(t, respondOK)
	response, err := callWithFailover(t, nil, false, "https://api.chill-failover.invalid", healthy.server.URL)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if response.BaseURL != healthy.server.URL {
		t.Fatalf("BaseURL = %q, want %q", response.BaseURL, healthy.server.URL)
	}
}

func TestCallFailsOverReadOnlyRequestOnGatewayStatus(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		unavailable := newCountingHost(t, respondStatus(status))
		healthy := newCountingHost(t, respondOK)
		response, err := callWithFailover(t, nil, true, unavailable.server.URL, healthy.server.URL)
		if err != nil {
			t.Fatalf("status %d: Call() error = %v", status, err)
		}
		if response.BaseURL != healthy.server.URL || unavailable.hits.Load() != 1 {
			t.Fatalf("status %d: BaseURL = %q, unavailable hits = %d", status, response.BaseURL, unavailable.hits.Load())
		}
	}
}

func TestCallDoesNotReplayMutationOnGatewayStatus(t *testing.T) {
	t.Parallel()

	unavailable := newCountingHost(t, respondStatus(http.StatusServiceUnavailable))
	fallback := newCountingHost(t, respondOK)
	_, err := callWithFailover(t, nil, false, unavailable.server.URL, fallback.server.URL)

	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("Call() error = %v, want 503 APIError", err)
	}
	if got := fallback.hits.Load(); got != 0 {
		t.Fatalf("fallback host hits = %d, want 0", got)
	}
}

func TestCallReturnsApplicationErrorsWithoutFailover(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError} {
		failing := newCountingHost(t, respondStatus(status))
		fallback := newCountingHost(t, respondOK)
		_, err := callWithFailover(t, nil, true, failing.server.URL, fallback.server.URL)

		var apiErr APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
			t.Fatalf("status %d: Call() error = %v", status, err)
		}
		if got := fallback.hits.Load(); got != 0 {
			t.Fatalf("status %d: fallback host hits = %d, want 0", status, got)
		}
	}
}

func TestCallTimeoutNeverReplays(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	hanging := newCountingHost(t, func(http.ResponseWriter, *http.Request) { <-release })
	t.Cleanup(func() { close(release) })
	httpClient := &http.Client{Timeout: 200 * time.Millisecond}

	mutationFallback := newCountingHost(t, respondOK)
	if _, err := callWithFailover(t, httpClient, false, hanging.server.URL, mutationFallback.server.URL); err == nil {
		t.Fatal("mutation Call() error = nil, want timeout")
	}
	if got := mutationFallback.hits.Load(); got != 0 {
		t.Fatalf("mutation fallback hits = %d, want 0", got)
	}

	readFallback := newCountingHost(t, respondOK)
	_, err := callWithFailover(t, httpClient, true, hanging.server.URL, readFallback.server.URL)
	if err == nil || readFallback.hits.Load() != 0 {
		t.Fatalf("read-only error = %v, fallback hits = %d", err, readFallback.hits.Load())
	}
	if got := hanging.hits.Load(); got != 2 {
		t.Fatalf("hanging host hits = %d, want 2", got)
	}
}

func TestSingleHostClientDoesNotFailOver(t *testing.T) {
	t.Parallel()

	unavailable := newCountingHost(t, respondStatus(http.StatusServiceUnavailable))
	client := NewClient(unavailable.server.URL, nil)
	if got := client.BaseURLs(); len(got) != 1 || got[0] != unavailable.server.URL {
		t.Fatalf("BaseURLs() = %v", got)
	}
	_, err := client.Call(context.Background(), CallRequest{Procedure: "chill.v4.UserService/GetUserProfile", AuthMode: AuthNone, ReadOnly: true})
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("Call() error = %v, want 503 APIError", err)
	}
}

func TestFirstReachableBaseURLSkipsUnavailableHosts(t *testing.T) {
	t.Parallel()

	unavailable := newCountingHost(t, respondStatus(http.StatusBadGateway))
	healthy := newCountingHost(t, respondOK)
	client := NewClient(closedPortURL(t), nil, WithFailoverBaseURLs(unavailable.server.URL, healthy.server.URL))

	got, err := client.FirstReachableBaseURL(context.Background())
	if err != nil {
		t.Fatalf("FirstReachableBaseURL() error = %v", err)
	}
	if got != healthy.server.URL {
		t.Fatalf("FirstReachableBaseURL() = %q, want %q", got, healthy.server.URL)
	}
}

func TestSlowHealthyHostUsesRemainingDeadline(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		t.Run(fmt.Sprint(readOnly), func(t *testing.T) {
			first := newCountingHost(t, func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(300 * time.Millisecond)
				respondOK(w, r)
			})
			fallback := newCountingHost(t, respondOK)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			client := NewClient(first.server.URL, nil, WithFailoverBaseURLs(
				fallback.server.URL, closedPortURL(t), closedPortURL(t), closedPortURL(t),
			))
			response, err := client.Call(ctx, CallRequest{Procedure: "test/Call", AuthMode: AuthNone, ReadOnly: readOnly})
			if err != nil || response.BaseURL != first.server.URL || first.hits.Load() != 1 || fallback.hits.Load() != 0 {
				t.Fatalf("response=%+v err=%v first=%d fallback=%d", response, err, first.hits.Load(), fallback.hits.Load())
			}
		})
	}
}

func TestSuccessfulResponseBodyFailureDoesNotFailOver(t *testing.T) {
	for _, truncated := range []bool{false, true} {
		t.Run(fmt.Sprint(truncated), func(t *testing.T) {
			first := newCountingHost(t, func(w http.ResponseWriter, r *http.Request) {
				if truncated {
					w.Header().Set("Content-Length", "100")
				}
				_, _ = w.Write([]byte("oversized response"))
			})
			fallback := newCountingHost(t, respondOK)
			client := NewClient(first.server.URL, nil, WithFailoverBaseURLs(fallback.server.URL))
			if !truncated {
				client.responseBodyLimit = 4
			}
			_, err := client.Call(context.Background(), CallRequest{Procedure: "test/Read", AuthMode: AuthNone, ReadOnly: true})
			if err == nil || fallback.hits.Load() != 0 {
				t.Fatalf("error=%v fallback=%d", err, fallback.hits.Load())
			}
		})
	}
}

func TestReadOnlyFailsOverOnConnectionClosedBeforeResponse(t *testing.T) {
	first := newCountingHost(t, func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	})
	fallback := newCountingHost(t, respondOK)
	response, err := callWithFailover(t, nil, true, first.server.URL, fallback.server.URL)
	if err != nil || response.BaseURL != fallback.server.URL || first.hits.Load() != 1 || fallback.hits.Load() != 1 {
		t.Fatalf("response=%+v error=%v first=%d fallback=%d", response, err, first.hits.Load(), fallback.hits.Load())
	}
}

func TestMutationFailsOverOnTLSHandshakeFailure(t *testing.T) {
	first := httptest.NewTLSServer(http.HandlerFunc(respondOK))
	defer first.Close()
	fallback := newCountingHost(t, respondOK)
	response, err := callWithFailover(t, nil, false, first.URL, fallback.server.URL)
	if err != nil || response.BaseURL != fallback.server.URL || fallback.hits.Load() != 1 {
		t.Fatalf("response=%+v error=%v fallback=%d", response, err, fallback.hits.Load())
	}
}

func TestMutationFailsOverWhenTLSHandshakeTimesOut(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()
	fallback := newCountingHost(t, respondOK)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = 100 * time.Millisecond
	httpClient := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	response, err := callWithFailover(t, httpClient, false, "https://"+listener.Addr().String(), fallback.server.URL)
	if err != nil || response.BaseURL != fallback.server.URL || fallback.hits.Load() != 1 {
		t.Fatalf("response=%+v error=%v fallback=%d", response, err, fallback.hits.Load())
	}
}

func TestHealthProbesHaveIndependentDeadlines(t *testing.T) {
	first := newCountingHost(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	second := newCountingHost(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1200 * time.Millisecond)
		respondOK(w, r)
	})
	client := NewClient(first.server.URL, nil, WithFailoverBaseURLs(second.server.URL))
	host, err := client.FirstReachableBaseURL(context.Background())
	if err != nil || host != second.server.URL || first.hits.Load() != 1 || second.hits.Load() != 1 {
		t.Fatalf("host=%s error=%v first=%d second=%d", host, err, first.hits.Load(), second.hits.Load())
	}
}

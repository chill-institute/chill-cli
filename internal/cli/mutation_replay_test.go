package cli

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
)

type reviewRoundTripFunc func(*http.Request) (*http.Response, error)

func (f reviewRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMutationReplayUsesConnectionEvidence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		trace func(*httptrace.ClientTrace)
		retry bool
	}{
		{"failed then successful dial callback", func(tr *httptrace.ClientTrace) {
			tr.ConnectDone("tcp", "", errors.New("refused"))
			tr.ConnectDone("tcp", "", nil)
		}, true},
		{"successful then failed dial callback", func(tr *httptrace.ClientTrace) {
			tr.ConnectDone("tcp", "", nil)
			tr.ConnectDone("tcp", "", errors.New("refused"))
		}, true},
		{"failed TLS then successful callback", func(tr *httptrace.ClientTrace) {
			tr.TLSHandshakeDone(tls.ConnectionState{}, errors.New("TLS failure"))
			tr.TLSHandshakeDone(tls.ConnectionState{}, nil)
		}, true},
		{"waiting for connection", func(tr *httptrace.ClientTrace) { tr.GetConn("primary.example") }, true},
		{"headers written before dial failure", func(tr *httptrace.ClientTrace) {
			tr.WroteHeaders()
			tr.ConnectDone("tcp", "", errors.New("refused"))
		}, false},
		{"connection obtained", func(tr *httptrace.ClientTrace) {
			tr.ConnectDone("tcp", "", errors.New("refused"))
			tr.GotConn(httptrace.GotConnInfo{})
		}, false},
		{"untraced ambiguous error", func(*httptrace.ClientTrace) {}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			transport := reviewRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					tc.trace(httptrace.ContextClientTrace(r.Context()))
					return nil, errors.New("transport failed")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})
			_, err := rpc.NewClient("https://primary.example", &http.Client{Transport: transport}, rpc.WithFailoverBaseURLs("https://fallback.example")).Call(context.Background(), rpc.CallRequest{Procedure: "test/Mutate", AuthMode: rpc.AuthNone})
			if tc.retry && (err != nil || calls != 2) || !tc.retry && (err == nil || calls != 1) {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

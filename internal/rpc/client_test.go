package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (body *trackingReadCloser) Close() error {
	body.closed = true
	return nil
}

type failingReader struct {
	err error
}

func (reader failingReader) Read([]byte) (int, error) {
	return 0, reader.err
}

func TestNewClientUsesDefaultTimeoutWithoutCustomClient(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.chill.institute", nil)
	if client.httpClient == nil {
		t.Fatal("httpClient = nil")
	}
	if client.httpClient.Timeout != DefaultClientTimeout {
		t.Fatalf("Timeout = %v, want %v", client.httpClient.Timeout, DefaultClientTimeout)
	}
}

func TestNewClientCopiesCustomClientSettings(t *testing.T) {
	t.Parallel()

	customClient := &http.Client{Timeout: 5 * time.Second}
	client := NewClient("https://api.chill.institute", customClient)
	if client.httpClient == customClient {
		t.Fatal("NewClient() reused mutable custom client")
	}
	if client.httpClient.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %v, want 5s", client.httpClient.Timeout)
	}
	if customClient.CheckRedirect != nil {
		t.Fatal("NewClient() mutated custom CheckRedirect")
	}
}

func TestCallRejectsRedirectThatChangesAPIOrigin(t *testing.T) {
	t.Parallel()

	redirected := make(chan struct{}, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/plaintext" {
			redirected <- struct{}{}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		writer.Header().Set("Location", "http://"+request.Host+"/plaintext")
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthUser,
		AuthToken: "test-token",
	})
	if err == nil || !strings.Contains(err.Error(), "refusing redirect that changes API origin") {
		t.Fatalf("Call() error = %v, want unsafe redirect rejection", err)
	}
	select {
	case <-redirected:
		t.Fatal("unsafe redirect target received a request")
	default:
	}
}

func TestCallRejectsRedirectThatChangesAPIAuthority(t *testing.T) {
	t.Parallel()

	redirected := make(chan struct{}, 1)
	destination := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		redirected <- struct{}{}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer destination.Close()

	source := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", destination.URL+"/redirected")
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	client := NewClient(source.URL, source.Client())
	_, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthUser,
		AuthToken: "test-token",
	})
	if err == nil || !strings.Contains(err.Error(), "refusing redirect that changes API origin") {
		t.Fatalf("Call() error = %v, want unsafe redirect rejection", err)
	}
	select {
	case <-redirected:
		t.Fatal("cross-authority redirect target received a request")
	default:
	}
}

func TestCallAllowsSameOriginRedirect(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("redirected") == "true" {
			if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Fatalf("Authorization = %q", got)
			}
			_, _ = writer.Write([]byte(`{"status":"ok"}`))
			return
		}
		writer.Header().Set("Location", request.URL.Path+"?redirected=true")
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	response, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthUser,
		AuthToken: "test-token",
	})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if string(response.Body) != `{"status":"ok"}` {
		t.Fatalf("Body = %s", response.Body)
	}
}

func TestCallUserAuthAndRequestBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", request.Method, http.MethodPost)
		}
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserProfile" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if request.Header.Get("X-Request-Id") == "" {
			t.Fatal("expected X-Request-Id header")
		}

		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(payload) != 1 || payload["query"] != "movie" {
			t.Fatalf("payload = %#v", payload)
		}

		writer.Header().Set("X-Request-Id", "req-123")
		_, _ = writer.Write([]byte(`{"user_id":"user-1"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	response, err := client.Call(context.Background(), CallRequest{
		Procedure: "/chill.v4.UserService/GetUserProfile",
		Body:      map[string]any{"query": "movie"},
		AuthMode:  AuthUser,
		AuthToken: "test-token",
	})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if response.RequestID != "req-123" {
		t.Fatalf("RequestID = %q, want %q", response.RequestID, "req-123")
	}
	if strings.TrimSpace(string(response.Body)) != `{"user_id":"user-1"}` {
		t.Fatalf("Body = %s", string(response.Body))
	}
}

func TestCallNoAuthHeaderWhenModeNone(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		_, _ = writer.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthNone,
	})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
}

func TestCallReturnsStructuredAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"code":"invalid_auth_token","message":"invalid auth token","request_id":"req-500"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthUser,
		AuthToken: "bad-token",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	if apiErr.Code != "invalid_auth_token" {
		t.Fatalf("code = %q", apiErr.Code)
	}
	if apiErr.RequestID != "req-500" {
		t.Fatalf("request id = %q", apiErr.RequestID)
	}
}

func TestCallMissingUserToken(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.chill.institute", http.DefaultClient)
	_, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthUser,
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "missing auth token") {
		t.Fatalf("error = %v", err)
	}
}

func TestAPIErrorErrorIncludesBodyWhenEnvelopeMissing(t *testing.T) {
	t.Parallel()

	err := APIError{
		StatusCode: http.StatusBadGateway,
		Body:       "upstream failed",
	}

	if got := err.Error(); got != "api error (502): upstream failed" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestAPIErrorUnwrapsResponseReadFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("read failed")
	err := APIError{StatusCode: http.StatusBadGateway, Err: cause}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, cause) = false", err)
	}
	if got := err.Error(); got != "api error (502): read failed" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestCallRejectsDangerousProcedureNames(t *testing.T) {
	t.Parallel()

	client := NewClient("https://api.chill.institute", http.DefaultClient)
	invalid := []string{
		"",
		"../chill.v4.UserService/GetUserProfile",
		"chill.v4.UserService/GetUserProfile?x=1",
		"chill.v4.UserService/GetUserProfile#frag",
		"chill.v4.UserService/%2e%2e/GetUserProfile",
		"chill.v4.UserService\\GetUserProfile",
	}

	for _, procedure := range invalid {
		procedure := procedure
		t.Run(procedure, func(t *testing.T) {
			t.Parallel()

			_, err := client.Call(context.Background(), CallRequest{
				Procedure: procedure,
				AuthMode:  AuthNone,
			})
			if err == nil {
				t.Fatalf("Call(%q) error = nil, want rejection", procedure)
			}
		})
	}
}

func TestReadResponseBodyEnforcesLimit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		reader  io.Reader
		limit   int64
		want    string
		wantErr error
	}{
		{name: "empty", reader: strings.NewReader(""), limit: 3, want: ""},
		{name: "exact limit", reader: strings.NewReader("abc"), limit: 3, want: "abc"},
		{name: "overflow", reader: strings.NewReader("abcd"), limit: 3, wantErr: errResponseBodyTooLarge},
		{name: "reader failure", reader: failingReader{err: io.ErrUnexpectedEOF}, limit: 3, wantErr: io.ErrUnexpectedEOF},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			payload, err := readResponseBody(testCase.reader, testCase.limit)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("readResponseBody() error = %v, want %v", err, testCase.wantErr)
			}
			if string(payload) != testCase.want {
				t.Fatalf("payload = %q, want %q", payload, testCase.want)
			}
		})
	}
}

func TestCallAppliesStatusSpecificBodyLimits(t *testing.T) {
	t.Parallel()

	const payload = "between"
	testCases := []struct {
		name             string
		statusCode       int
		wantBody         string
		wantBodyTooLarge bool
	}{
		{name: "success uses response limit", statusCode: http.StatusOK, wantBody: payload},
		{name: "error uses smaller error limit", statusCode: http.StatusBadGateway, wantBodyTooLarge: true},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("X-Request-Id", "req-limit")
				writer.WriteHeader(testCase.statusCode)
				_, _ = writer.Write([]byte(payload))
			}))
			defer server.Close()

			client := NewClient(server.URL, server.Client())
			client.responseBodyLimit = 16
			client.errorBodyLimit = 4
			response, err := client.Call(context.Background(), CallRequest{
				Procedure: "chill.v4.UserService/GetUserProfile",
				AuthMode:  AuthNone,
			})
			if !testCase.wantBodyTooLarge {
				if err != nil {
					t.Fatalf("Call() error = %v", err)
				}
				if string(response.Body) != testCase.wantBody {
					t.Fatalf("Body = %q, want %q", response.Body, testCase.wantBody)
				}
				return
			}
			if !errors.Is(err, errResponseBodyTooLarge) {
				t.Fatalf("Call() error = %v, want body limit error", err)
			}
			var apiErr APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("Call() error = %T, want APIError", err)
			}
			if apiErr.StatusCode != testCase.statusCode || apiErr.RequestID != "req-limit" {
				t.Fatalf("APIError = %#v", apiErr)
			}
			if strings.Contains(err.Error(), payload) {
				t.Fatalf("Call() error exposed response body: %v", err)
			}
		})
	}
}

func TestCallPreservesErrorMetadataWhenBodyReadFails(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"X-Request-Id": []string{"req-read"}},
			Body:       io.NopCloser(failingReader{err: io.ErrUnexpectedEOF}),
		}, nil
	})}
	client := NewClient("https://api.chill.institute", httpClient)

	_, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthNone,
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Call() error = %v, want reader failure", err)
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Call() error = %T, want APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.RequestID != "req-read" {
		t.Fatalf("APIError = %#v", apiErr)
	}
}

func TestCallClosesOversizedResponseBody(t *testing.T) {
	t.Parallel()

	body := &trackingReadCloser{Reader: strings.NewReader("oversized")}
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})}
	client := NewClient("https://api.chill.institute", httpClient)
	client.responseBodyLimit = 4

	_, err := client.Call(context.Background(), CallRequest{
		Procedure: "chill.v4.UserService/GetUserProfile",
		AuthMode:  AuthNone,
	})
	if !errors.Is(err, errResponseBodyTooLarge) {
		t.Fatalf("Call() error = %v, want body limit error", err)
	}
	if !body.closed {
		t.Fatal("response body was not closed")
	}
}

// Package rpc is the HTTP transport for hosted chill.institute v4 procedures:
// same-origin redirects only, bounded response bodies, client identity headers,
// and the shared error envelope mapped to APIError.
package rpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
)

const (
	AuthNone              AuthMode = "none"
	AuthUser              AuthMode = "user"
	DefaultClientTimeout           = 20 * time.Second
	defaultConnectTimeout          = 5 * time.Second
	maxResponseBodyBytes           = 32 << 20
	maxErrorBodyBytes              = 64 << 10
	clientNameHeader               = "X-Chill-Client"
	clientVersionHeader            = "X-Chill-Client-Version"
	defaultClientName              = "cli"
	defaultClientVersion           = "dev"
)

var errResponseBodyTooLarge = errors.New("rpc response body exceeds limit")

type AuthMode string

type Client struct {
	baseURLs          []string
	httpClient        *http.Client
	responseBodyLimit int64
	errorBodyLimit    int64
	clientName        string
	clientVersion     string
}

// Option customizes a Client. Options apply in order.
type Option func(*Client)

func WithFailoverBaseURLs(baseURLs ...string) Option {
	return func(client *Client) {
		for _, baseURL := range baseURLs {
			trimmed := trimBaseURL(baseURL)
			if trimmed == "" || slices.Contains(client.baseURLs, trimmed) {
				continue
			}
			client.baseURLs = append(client.baseURLs, trimmed)
		}
	}
}

// WithClientName sets the X-Chill-Client header and request ID prefix.
// Empty or non-token values are ignored.
func WithClientName(name string) Option {
	return func(client *Client) {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" || strings.IndexFunc(trimmed, func(r rune) bool { return r > unicode.MaxASCII || unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
			return
		}
		client.clientName = trimmed
	}
}

// WithClientVersion sets the X-Chill-Client-Version header. Empty values are ignored.
func WithClientVersion(version string) Option {
	return func(client *Client) {
		trimmed := strings.TrimSpace(version)
		if trimmed == "" || strings.IndexFunc(trimmed, func(r rune) bool { return r > unicode.MaxASCII || unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
			return
		}
		client.clientVersion = trimmed
	}
}

type CallRequest struct {
	Procedure string
	Body      any
	AuthMode  AuthMode
	AuthToken string
	ReadOnly  bool
}

type CallResponse struct {
	BaseURL    string
	StatusCode int
	RequestID  string
	Body       json.RawMessage
}

type ErrorEnvelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	Body       string
	Err        error
}

func (err APIError) Error() string {
	if err.Code != "" || err.Message != "" {
		return fmt.Sprintf("api error (%d): %s: %s", err.StatusCode, err.Code, err.Message)
	}
	if err.Err != nil {
		return fmt.Sprintf("api error (%d): %v", err.StatusCode, err.Err)
	}
	return fmt.Sprintf("api error (%d): %s", err.StatusCode, err.Body)
}

func (err APIError) Unwrap() error {
	return err.Err
}

// NewClient returns a client for the hosted v4 API. An empty baseURL selects
// production. Without options the client identifies as "cli" version "dev".
func NewClient(baseURL string, httpClient *http.Client, opts ...Option) *Client {
	trimmedBaseURL := trimBaseURL(baseURL)
	if trimmedBaseURL == "" {
		trimmedBaseURL = "https://api.chill.institute"
	}

	client := &Client{
		baseURLs:          []string{trimmedBaseURL},
		httpClient:        prepareHTTPClient(httpClient),
		responseBodyLimit: maxResponseBodyBytes,
		errorBodyLimit:    maxErrorBodyBytes,
		clientName:        defaultClientName,
		clientVersion:     defaultClientVersion,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

func trimBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func (client Client) BaseURLs() []string {
	return append([]string(nil), client.baseURLs...)
}

func prepareHTTPClient(source *http.Client) *http.Client {
	if source == nil {
		source = &http.Client{Transport: defaultTransport()}
	}

	client := *source
	checkRedirect := source.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) == 0 || !sameOrigin(via[0].URL, request.URL) {
			return errors.New("refusing redirect that changes API origin")
		}
		if checkRedirect != nil {
			return checkRedirect(request, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &client
}

func defaultTransport() http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: defaultConnectTimeout, KeepAlive: 30 * time.Second}).DialContext
	transport.TLSHandshakeTimeout = defaultConnectTimeout
	return transport
}

func sameOrigin(initial *url.URL, destination *url.URL) bool {
	if initial == nil || destination == nil || destination.User != nil {
		return false
	}
	return strings.EqualFold(initial.Scheme, destination.Scheme) &&
		strings.EqualFold(initial.Hostname(), destination.Hostname()) &&
		effectivePort(initial) == effectivePort(destination)
}

func effectivePort(value *url.URL) string {
	if port := value.Port(); port != "" {
		return port
	}
	if strings.EqualFold(value.Scheme, "https") {
		return "443"
	}
	if strings.EqualFold(value.Scheme, "http") {
		return "80"
	}
	return ""
}

func (client Client) Call(ctx context.Context, req CallRequest) (CallResponse, error) {
	procedure, err := normalizeProcedure(req.Procedure)
	if err != nil {
		return CallResponse{}, err
	}

	payload, err := json.Marshal(defaultBody(req.Body))
	if err != nil {
		return CallResponse{}, fmt.Errorf("encode request body: %w", err)
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultClientTimeout)
		defer cancel()
	}
	var lastErr error
	for _, baseURL := range client.baseURLs {
		response, reach, err := client.callHost(ctx, baseURL, procedure, payload, req)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if ctx.Err() != nil || isTimeoutOrCancellation(err) || !reach.allowsRepeatOnAnotherHost(req.ReadOnly) {
			return CallResponse{}, err
		}
	}
	return CallResponse{}, lastErr
}

func (client Client) FirstReachableBaseURL(ctx context.Context) (string, error) {
	var lastErr error
	for _, baseURL := range client.baseURLs {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := client.checkHealth(probeCtx, baseURL)
		cancel()
		if err == nil {
			return baseURL, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	return "", fmt.Errorf("no reachable API host: %w", lastErr)
}

func isTimeoutOrCancellation(err error) bool {
	var netErr net.Error
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		(errors.As(err, &netErr) && netErr.Timeout())
}

func (client Client) checkHealth(ctx context.Context, baseURL string) error {
	endpoint, err := url.JoinPath(baseURL, "healthz")
	if err != nil {
		return fmt.Errorf("build health endpoint: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}
	request.Header.Set(clientNameHeader, client.clientName)
	request.Header.Set(clientVersionHeader, client.clientVersion)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("check %s: %w", baseURL, err)
	}
	_ = response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("check %s: status %d", baseURL, response.StatusCode)
	}
	return nil
}

type hostReach int

const (
	hostAnswered hostReach = iota
	requestNeverSent
	requestSentWithoutAnswer
	gatewayUnavailable
)

func (reach hostReach) allowsRepeatOnAnotherHost(readOnly bool) bool {
	switch reach {
	case requestNeverSent:
		return true
	case requestSentWithoutAnswer, gatewayUnavailable:
		return readOnly
	default:
		return false
	}
}

func isGatewayUnavailable(statusCode int) bool {
	return statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

func (client Client) callHost(ctx context.Context, baseURL string, procedure string, payload []byte, req CallRequest) (CallResponse, hostReach, error) {
	endpoint, err := url.JoinPath(baseURL, "v4", procedure)
	if err != nil {
		return CallResponse{}, hostAnswered, fmt.Errorf("build endpoint: %w", err)
	}

	var headersWritten atomic.Bool
	var connectionObtained atomic.Bool
	var connectionStarted atomic.Bool
	traceCtx := httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GetConn:      func(string) { connectionStarted.Store(true) },
		WroteHeaders: func() { headersWritten.Store(true) },
		GotConn:      func(httptrace.GotConnInfo) { connectionObtained.Store(true) },
	})

	httpRequest, err := http.NewRequestWithContext(traceCtx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return CallResponse{}, hostAnswered, fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("X-Request-Id", newRequestID(client.clientName))
	httpRequest.Header.Set(clientNameHeader, client.clientName)
	httpRequest.Header.Set(clientVersionHeader, client.clientVersion)

	if err := applyAuth(httpRequest, req.AuthMode, req.AuthToken); err != nil {
		return CallResponse{}, hostAnswered, err
	}

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		reach := hostAnswered
		var connectionErr *net.OpError
		if httpResponse == nil {
			if connectionStarted.Load() && !headersWritten.Load() && !connectionObtained.Load() {
				reach = requestNeverSent
			} else if errors.As(err, &connectionErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				reach = requestSentWithoutAnswer
			}
		}
		return CallResponse{}, reach, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	callResponse := CallResponse{
		BaseURL:    baseURL,
		StatusCode: httpResponse.StatusCode,
		RequestID:  strings.TrimSpace(httpResponse.Header.Get("X-Request-Id")),
	}
	reach := hostAnswered
	if isGatewayUnavailable(callResponse.StatusCode) {
		reach = gatewayUnavailable
	}

	responseBody, err := readResponseBody(httpResponse.Body, client.bodyLimit(callResponse.StatusCode))
	if err != nil {
		if callResponse.StatusCode < 200 || callResponse.StatusCode >= 300 {
			return CallResponse{}, reach, APIError{
				StatusCode: callResponse.StatusCode,
				RequestID:  callResponse.RequestID,
				Err:        fmt.Errorf("read error response: %w", err),
			}
		}
		return CallResponse{}, hostAnswered, fmt.Errorf("read response: %w", err)
	}
	callResponse.Body = responseBody

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return CallResponse{}, reach, parseAPIError(callResponse)
	}

	return callResponse, hostAnswered, nil
}

func (client Client) bodyLimit(statusCode int) int64 {
	if statusCode < 200 || statusCode >= 300 {
		if client.errorBodyLimit > 0 {
			return client.errorBodyLimit
		}
		return maxErrorBodyBytes
	}
	if client.responseBodyLimit > 0 {
		return client.responseBodyLimit
	}
	return maxResponseBodyBytes
}

func readResponseBody(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("response body limit must be positive")
	}
	payload, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > limit {
		return nil, fmt.Errorf("%w: maximum %d bytes", errResponseBodyTooLarge, limit)
	}
	return payload, nil
}

func normalizeProcedure(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("procedure is required")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if !strings.Contains(trimmed, "/") {
		return "", fmt.Errorf("invalid procedure %q", value)
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("invalid procedure %q", value)
	}
	if strings.Contains(trimmed, "..") || strings.Contains(trimmed, "%") || strings.ContainsAny(trimmed, `\?#`) {
		return "", fmt.Errorf("invalid procedure %q", value)
	}
	return trimmed, nil
}

func defaultBody(body any) any {
	if body == nil {
		return map[string]any{}
	}
	return body
}

func applyAuth(request *http.Request, mode AuthMode, authToken string) error {
	switch mode {
	case AuthNone:
		return nil
	case AuthUser:
		trimmedToken := strings.TrimSpace(authToken)
		if trimmedToken == "" {
			return errors.New("missing auth token")
		}
		request.Header.Set("Authorization", "Bearer "+trimmedToken)
		return nil
	default:
		return fmt.Errorf("unsupported auth mode %q", mode)
	}
}

func parseAPIError(response CallResponse) error {
	apiErr := APIError{StatusCode: response.StatusCode, RequestID: response.RequestID, Body: strings.TrimSpace(string(response.Body))}

	var envelope ErrorEnvelope
	if err := json.Unmarshal(response.Body, &envelope); err == nil {
		apiErr.Code = strings.TrimSpace(envelope.Code)
		apiErr.Message = strings.TrimSpace(envelope.Message)
		if strings.TrimSpace(envelope.RequestID) != "" {
			apiErr.RequestID = strings.TrimSpace(envelope.RequestID)
		}
	}

	return apiErr
}

func newRequestID(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(bytes)
}

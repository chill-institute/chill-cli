package rpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const (
	AuthNone             AuthMode = "none"
	AuthUser             AuthMode = "user"
	DefaultClientTimeout          = 20 * time.Second
	maxResponseBodyBytes          = 32 << 20
	maxErrorBodyBytes             = 64 << 10
)

var errResponseBodyTooLarge = errors.New("rpc response body exceeds limit")

type AuthMode string

type Client struct {
	baseURL           string
	httpClient        *http.Client
	responseBodyLimit int64
	errorBodyLimit    int64
}

type CallRequest struct {
	Procedure string
	Body      any
	AuthMode  AuthMode
	AuthToken string
}

type CallResponse struct {
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
}

func (err APIError) Error() string {
	if err.Code != "" || err.Message != "" {
		return fmt.Sprintf("api error (%d): %s: %s", err.StatusCode, err.Code, err.Message)
	}
	return fmt.Sprintf("api error (%d): %s", err.StatusCode, err.Body)
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBaseURL == "" {
		trimmedBaseURL = "https://api.chill.institute"
	}

	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: DefaultClientTimeout}
	}

	return &Client{
		baseURL:           trimmedBaseURL,
		httpClient:        client,
		responseBodyLimit: maxResponseBodyBytes,
		errorBodyLimit:    maxErrorBodyBytes,
	}
}

func (client Client) Call(ctx context.Context, req CallRequest) (CallResponse, error) {
	procedure, err := normalizeProcedure(req.Procedure)
	if err != nil {
		return CallResponse{}, err
	}

	endpoint, err := url.JoinPath(client.baseURL, "v4", procedure)
	if err != nil {
		return CallResponse{}, fmt.Errorf("build endpoint: %w", err)
	}

	payload, err := json.Marshal(defaultBody(req.Body))
	if err != nil {
		return CallResponse{}, fmt.Errorf("encode request body: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return CallResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("X-Request-Id", newRequestID())

	if err := applyAuth(httpRequest, req.AuthMode, req.AuthToken); err != nil {
		return CallResponse{}, err
	}

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return CallResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	responseBody, err := readResponseBody(httpResponse.Body, client.bodyLimit(httpResponse.StatusCode))
	if err != nil {
		return CallResponse{}, fmt.Errorf("read response: %w", err)
	}

	callResponse := CallResponse{
		StatusCode: httpResponse.StatusCode,
		RequestID:  strings.TrimSpace(httpResponse.Header.Get("X-Request-Id")),
		Body:       responseBody,
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return CallResponse{}, parseAPIError(callResponse)
	}

	return callResponse, nil
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

func newRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("cli-%d", time.Now().UnixNano())
	}
	return "cli-" + hex.EncodeToString(bytes)
}

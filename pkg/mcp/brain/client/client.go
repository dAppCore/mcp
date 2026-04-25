// SPDX-License-Identifier: EUPL-1.2

// Package client provides the shared OpenBrain HTTP client.
//
//	c := client.New(client.Options{URL: core.Env("CORE_BRAIN_URL"), Key: core.Env("CORE_BRAIN_KEY")})
//	_, err := c.Remember(ctx, client.RememberInput{
//		Org:     "core",
//		Project: "mcp",
//		Content: "Use one OpenBrain client for retry and circuit-breaker policy.",
//		Type:    "decision",
//	})
package client

import (
	"context"
	cryptorand "crypto/rand"
	"io"
	"io/fs"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

const (
	DefaultURL              = "https://api.lthn.sh"
	insecureBrainEnv        = "CORE_BRAIN_INSECURE"
	brainKeyFileMode        = fs.FileMode(0o600)
	defaultAgentID          = "cladius"
	defaultTimeout          = 30 * time.Second
	defaultMaxAttempts      = 3
	defaultBaseDelay        = 100 * time.Millisecond
	defaultFailureThreshold = 3
	defaultSuccessThreshold = 1
	defaultCircuitCooldown  = 30 * time.Second
	defaultMaxResponseBytes = int64(1 << 20)
	maxBackoffDelay         = 30 * time.Second
	maxRetryAfterDelay      = 60 * time.Second
	defaultRecallTopK       = 10
	defaultListLimit        = 50
)

// ErrCircuitOpen is returned when repeated upstream failures have opened the circuit.
var ErrCircuitOpen = core.NewError("brain client circuit open")

// Options configures the shared OpenBrain client.
type Options struct {
	URL              string
	Key              string
	Org              string
	AgentID          string
	HTTPClient       *http.Client
	MaxAttempts      int
	BaseDelay        time.Duration
	MaxResponseBytes int64
	CircuitBreaker   *CircuitBreaker
}

// Client calls the Laravel /v1/brain/* API with shared retry and circuit policy.
type Client struct {
	apiURL           string
	apiKey           string
	org              string
	agentID          string
	httpClient       *http.Client
	maxAttempts      int
	baseDelay        time.Duration
	maxResponseBytes int64
	circuitBreaker   *CircuitBreaker
	configErr        error
	sleepFunc        func(context.Context, time.Duration) error
}

// RememberInput is the request body for POST /v1/brain/remember.
type RememberInput struct {
	Content    string   `json:"content"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags,omitempty"`
	Org        string   `json:"org,omitempty"`
	Project    string   `json:"project,omitempty"`
	AgentID    string   `json:"agent_id,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	Supersedes string   `json:"supersedes,omitempty"`
	ExpiresIn  int      `json:"expires_in,omitempty"`
}

// RecallInput is the request body for POST /v1/brain/recall.
type RecallInput struct {
	Query         string  `json:"query"`
	TopK          int     `json:"top_k,omitempty"`
	Org           string  `json:"org,omitempty"`
	Project       string  `json:"project,omitempty"`
	Type          any     `json:"type,omitempty"`
	AgentID       string  `json:"agent_id,omitempty"`
	MinConfidence float64 `json:"min_confidence,omitempty"`
}

// ForgetInput selects the memory removed by DELETE /v1/brain/forget/{id}.
type ForgetInput struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// ListInput provides URL parameters for GET /v1/brain/list.
type ListInput struct {
	Org     string `json:"org,omitempty"`
	Project string `json:"project,omitempty"`
	Type    string `json:"type,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// CircuitState is the current breaker state.
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// CircuitBreakerOptions controls when the circuit opens and recovers.
type CircuitBreakerOptions struct {
	FailureThreshold int
	SuccessThreshold int
	Cooldown         time.Duration
}

// CircuitBreaker protects OpenBrain from repeated failed calls.
type CircuitBreaker struct {
	lock             sync.Mutex
	state            CircuitState
	failureThreshold int
	successThreshold int
	cooldown         time.Duration
	consecutiveFails int
	consecutiveWins  int
	openedAt         time.Time
	halfOpenInFlight bool
}

// New creates a shared OpenBrain client.
func New(options Options) *Client {
	apiURL := core.Trim(options.URL)
	if apiURL == "" {
		apiURL = DefaultURL
	}
	configErr := validateAPIURL(apiURL)
	agentID := core.Trim(options.AgentID)
	if agentID == "" {
		agentID = defaultAgentID
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	maxAttempts := options.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}
	baseDelay := options.BaseDelay
	if baseDelay <= 0 {
		baseDelay = defaultBaseDelay
	}
	maxResponseBytes := options.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultMaxResponseBytes
	}
	breaker := options.CircuitBreaker
	if breaker == nil {
		breaker = NewCircuitBreaker(CircuitBreakerOptions{})
	}

	return &Client{
		apiURL:           core.TrimSuffix(apiURL, "/"),
		apiKey:           core.Trim(options.Key),
		org:              core.Trim(options.Org),
		agentID:          agentID,
		httpClient:       httpClient,
		maxAttempts:      maxAttempts,
		baseDelay:        baseDelay,
		maxResponseBytes: maxResponseBytes,
		circuitBreaker:   breaker,
		configErr:        configErr,
		sleepFunc:        sleepDuration,
	}
}

// NewFromEnvironment reads CORE_BRAIN_* settings and ~/.claude/brain.key.
func NewFromEnvironment() *Client {
	apiKey, configErr := apiKeyFromEnvironment()
	client := New(Options{
		URL:     envOr("CORE_BRAIN_URL", DefaultURL),
		Key:     apiKey,
		Org:     core.Env("CORE_BRAIN_ORG"),
		AgentID: core.Env("CORE_BRAIN_AGENT_ID"),
	})
	if configErr != nil {
		client.configErr = configErr
	}
	return client
}

func validateAPIURL(apiURL string) error {
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return core.E("brain.client", "invalid API URL", err)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && core.Trim(core.Env(insecureBrainEnv)) == "true" {
		return nil
	}
	return core.E("brain.client", "API URL must use https unless CORE_BRAIN_INSECURE=true", nil)
}

// WriteBrainKey stores the OpenBrain API key at ~/.claude/brain.key with owner-only permissions.
func WriteBrainKey(apiKey string) error {
	home := core.Env("HOME")
	if home == "" {
		return core.E("brain.client", "HOME not set", nil)
	}
	return writeBrainKeyFile(brainKeyPath(home), apiKey)
}

// NewCircuitBreaker creates a circuit breaker with OpenBrain defaults.
func NewCircuitBreaker(options CircuitBreakerOptions) *CircuitBreaker {
	failureThreshold := options.FailureThreshold
	if failureThreshold <= 0 {
		failureThreshold = defaultFailureThreshold
	}
	successThreshold := options.SuccessThreshold
	if successThreshold <= 0 {
		successThreshold = defaultSuccessThreshold
	}
	cooldown := options.Cooldown
	if cooldown <= 0 {
		cooldown = defaultCircuitCooldown
	}
	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		cooldown:         cooldown,
	}
}

// State returns the current breaker state.
func (breaker *CircuitBreaker) State() CircuitState {
	if breaker == nil {
		return CircuitClosed
	}
	breaker.lock.Lock()
	defer breaker.lock.Unlock()
	return breaker.stateNow(time.Now())
}

// Remember stores a memory in OpenBrain.
func (c *Client) Remember(ctx context.Context, input RememberInput) (map[string]any, error) {
	input.Org = c.orgFor(input.Org)
	input.AgentID = c.agentFor(input.AgentID)
	return c.Call(ctx, http.MethodPost, "/v1/brain/remember", input)
}

// Recall searches memories in OpenBrain.
func (c *Client) Recall(ctx context.Context, input RecallInput) (map[string]any, error) {
	input.Org = c.orgFor(input.Org)
	input.AgentID = c.agentFor(input.AgentID)
	if input.TopK == 0 {
		input.TopK = defaultRecallTopK
	}
	return c.Call(ctx, http.MethodPost, "/v1/brain/recall", input)
}

// Forget removes one memory from OpenBrain.
func (c *Client) Forget(ctx context.Context, input ForgetInput) (map[string]any, error) {
	return c.Call(ctx, http.MethodDelete, core.Concat("/v1/brain/forget/", url.PathEscape(input.ID)), nil)
}

// List returns memories from OpenBrain using URL query filters.
func (c *Client) List(ctx context.Context, input ListInput) (map[string]any, error) {
	input.Org = c.orgFor(input.Org)
	if input.Limit == 0 {
		input.Limit = defaultListLimit
	}
	values := url.Values{}
	if input.Org != "" {
		values.Set("org", input.Org)
	}
	if input.Project != "" {
		values.Set("project", input.Project)
	}
	if input.Type != "" {
		values.Set("type", input.Type)
	}
	if input.AgentID != "" {
		values.Set("agent_id", input.AgentID)
	}
	values.Set("limit", core.Sprintf("%d", input.Limit))
	return c.Call(ctx, http.MethodGet, core.Concat("/v1/brain/list?", values.Encode()), nil)
}

// Call performs one OpenBrain API request through retry and circuit-breaker policy.
func (c *Client) Call(ctx context.Context, method, path string, body any) (map[string]any, error) {
	if c.configErr != nil {
		return nil, c.configErr
	}
	if c.apiKey == "" {
		return nil, core.E("brain.client", "no API key (set CORE_BRAIN_KEY or create ~/.claude/brain.key)", nil)
	}
	if err := c.circuitBreaker.beforeRequest(); err != nil {
		return nil, err
	}

	bodyString := ""
	if body != nil {
		bodyString = core.JSONMarshalString(body)
	}

	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		payload, retryable, retryAfter, hasRetryAfter, err := c.doOnce(ctx, method, path, bodyString, body != nil)
		if err == nil {
			c.circuitBreaker.recordSuccess()
			return payload, nil
		}

		lastErr = err
		if !retryable {
			c.circuitBreaker.recordIgnored()
			break
		}
		c.circuitBreaker.recordFailure()
		if c.circuitBreaker.State() == CircuitOpen || attempt == c.maxAttempts {
			break
		}
		var sleepErr error
		if hasRetryAfter {
			sleepErr = c.sleepFor(ctx, retryAfter)
		} else {
			sleepErr = c.sleep(ctx, attempt)
		}
		if sleepErr != nil {
			lastErr = sleepErr
			break
		}
	}

	return nil, lastErr
}

func (c *Client) doOnce(ctx context.Context, method, path, bodyString string, hasBody bool) (map[string]any, bool, time.Duration, bool, error) {
	var reader io.Reader
	if hasBody {
		reader = core.NewReader(bodyString)
	}
	requestURL, err := c.requestURL(path)
	if err != nil {
		return nil, false, 0, false, err
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, false, 0, false, core.E("brain.client", "create request", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", core.Concat("Bearer ", c.apiKey))
	if hasBody {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, 0, false, core.E("brain.client", "request cancelled", ctx.Err())
		}
		return nil, true, 0, false, core.E("brain.client", "request failed", err)
	}
	defer response.Body.Close()

	readResult := core.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if !readResult.OK {
		if readErr, ok := readResult.Value.(error); ok {
			return nil, false, 0, false, core.E("brain.client", "read response", readErr)
		}
		return nil, false, 0, false, core.E("brain.client", "read response", nil)
	}
	raw := readResult.Value.(string)
	if int64(len(raw)) > c.maxResponseBytes {
		return nil, false, 0, false, core.E("brain.client", "response too large", nil)
	}

	if response.StatusCode >= http.StatusBadRequest {
		retryAfter, hasRetryAfter := parseRetryAfter(response.Header.Get("Retry-After"), time.Now())
		return nil, retryableStatus(response.StatusCode), retryAfter, hasRetryAfter, core.E("brain.client", core.Concat("upstream returned ", response.Status, ": ", core.Trim(raw)), nil)
	}

	result := map[string]any{}
	if parseResult := core.JSONUnmarshalString(raw, &result); !parseResult.OK {
		if parseErr, ok := parseResult.Value.(error); ok {
			return nil, false, 0, false, core.E("brain.client", "parse response", parseErr)
		}
		return nil, false, 0, false, core.E("brain.client", "parse response", nil)
	}
	return result, false, 0, false, nil
}

func (c *Client) requestURL(path string) (string, error) {
	parsed, err := url.Parse(path)
	if err == nil && (parsed.IsAbs() || parsed.Host != "") {
		return "", core.E("brain.client", "absolute request URL rejected", nil)
	}
	if !core.HasPrefix(path, "/") {
		path = core.Concat("/", path)
	}
	return core.Concat(c.apiURL, path), nil
}

func (c *Client) sleep(ctx context.Context, attempt int) error {
	retryAttempt := attempt - 1
	delay := jitteredBackoffDelay(c.baseDelay, retryAttempt)
	return c.sleepFor(ctx, delay)
}

func (c *Client) sleepFor(ctx context.Context, delay time.Duration) error {
	if c.sleepFunc != nil {
		return c.sleepFunc(ctx, delay)
	}
	return sleepDuration(ctx, delay)
}

func sleepDuration(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return core.E("brain.client", "request cancelled", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func jitteredBackoffDelay(baseDelay time.Duration, attempt int) time.Duration {
	limit := backoffDelayLimit(baseDelay, attempt)
	if limit <= 0 {
		return 0
	}
	jitter, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(limit)))
	if err != nil {
		return limit
	}
	return time.Duration(jitter.Int64())
}

func backoffDelayLimit(baseDelay time.Duration, attempt int) time.Duration {
	if baseDelay <= 0 {
		return 0
	}
	if baseDelay >= maxBackoffDelay {
		return maxBackoffDelay
	}
	if attempt <= 0 {
		return baseDelay
	}

	delay := baseDelay
	for i := 0; i < attempt; i++ {
		if delay >= maxBackoffDelay/2 {
			return maxBackoffDelay
		}
		delay *= 2
	}
	if delay > maxBackoffDelay {
		return maxBackoffDelay
	}
	return delay
}

func (c *Client) orgFor(org string) string {
	org = core.Trim(org)
	if org != "" {
		return org
	}
	return c.org
}

func (c *Client) agentFor(agentID string) string {
	agentID = core.Trim(agentID)
	if agentID != "" {
		return agentID
	}
	return c.agentID
}

func (breaker *CircuitBreaker) beforeRequest() error {
	if breaker == nil {
		return nil
	}
	breaker.lock.Lock()
	defer breaker.lock.Unlock()

	state := breaker.stateNow(time.Now())
	if state == CircuitOpen {
		return ErrCircuitOpen
	}
	if state == CircuitHalfOpen {
		if breaker.halfOpenInFlight {
			return ErrCircuitOpen
		}
		breaker.halfOpenInFlight = true
	}
	return nil
}

func (breaker *CircuitBreaker) recordSuccess() {
	if breaker == nil {
		return
	}
	breaker.lock.Lock()
	defer breaker.lock.Unlock()

	breaker.halfOpenInFlight = false
	breaker.consecutiveFails = 0
	breaker.consecutiveWins++
	if breaker.state == CircuitHalfOpen && breaker.consecutiveWins >= breaker.successThreshold {
		breaker.state = CircuitClosed
		breaker.consecutiveWins = 0
	}
	if breaker.state == CircuitClosed {
		breaker.consecutiveWins = 0
	}
}

func (breaker *CircuitBreaker) recordFailure() {
	if breaker == nil {
		return
	}
	breaker.lock.Lock()
	defer breaker.lock.Unlock()

	breaker.halfOpenInFlight = false
	breaker.consecutiveWins = 0
	breaker.consecutiveFails++
	if breaker.state == CircuitHalfOpen || breaker.consecutiveFails >= breaker.failureThreshold {
		breaker.state = CircuitOpen
		breaker.openedAt = time.Now()
	}
}

func (breaker *CircuitBreaker) recordIgnored() {
	if breaker == nil {
		return
	}
	breaker.lock.Lock()
	defer breaker.lock.Unlock()
	breaker.halfOpenInFlight = false
}

func (breaker *CircuitBreaker) stateNow(now time.Time) CircuitState {
	if breaker.state == "" {
		breaker.state = CircuitClosed
	}
	if breaker.state == CircuitOpen && now.Sub(breaker.openedAt) >= breaker.cooldown {
		breaker.state = CircuitHalfOpen
		breaker.consecutiveFails = 0
		breaker.consecutiveWins = 0
		breaker.halfOpenInFlight = false
	}
	return breaker.state
}

func retryableStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = core.Trim(value)
	if value == "" {
		return 0, false
	}

	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0, true
		}
		maxSeconds := int64(maxRetryAfterDelay / time.Second)
		if seconds > maxSeconds {
			return maxRetryAfterDelay, true
		}
		return time.Duration(seconds) * time.Second, true
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	delay := retryAt.Sub(now)
	if delay <= 0 {
		return 0, true
	}
	if delay > maxRetryAfterDelay {
		return maxRetryAfterDelay, true
	}
	return delay, true
}

func envOr(key, fallback string) string {
	value := core.Env(key)
	if value != "" {
		return value
	}
	return fallback
}

func apiKeyFromEnvironment() (string, error) {
	if apiKey := core.Trim(core.Env("CORE_BRAIN_KEY")); apiKey != "" {
		return apiKey, nil
	}
	home := core.Env("HOME")
	if home == "" {
		return "", nil
	}
	apiKey, err := readBrainKeyFile(brainKeyPath(home))
	if err != nil {
		if core.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return apiKey, nil
}

func brainKeyPath(home string) string {
	return core.JoinPath(home, ".claude", "brain.key")
}

func readBrainKeyFile(path string) (string, error) {
	info, err := coreio.Local.Stat(path)
	if err != nil {
		return "", err
	}
	if brainKeyModeInsecure(info.Mode().Perm()) {
		return "", core.E("brain.client", "brain.key has insecure permissions, expected 0600", nil)
	}
	data, err := coreio.Local.Read(path)
	if err != nil {
		return "", err
	}
	return core.Trim(data), nil
}

func writeBrainKeyFile(path, apiKey string) error {
	if err := coreio.Local.WriteMode(path, core.Trim(apiKey)+"\n", brainKeyFileMode); err != nil {
		return err
	}
	if err := os.Chmod(path, brainKeyFileMode); err != nil {
		return core.E("brain.client", "chmod brain.key", err)
	}
	return nil
}

func brainKeyModeInsecure(mode fs.FileMode) bool {
	return mode.Perm()&^brainKeyFileMode != 0
}

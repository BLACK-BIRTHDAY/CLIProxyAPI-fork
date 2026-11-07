package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
)

type suspensionTestExecutor struct {
	mu    sync.Mutex
	calls map[string]int
}

func (e *suspensionTestExecutor) Identifier() string {
	return "antigravity"
}

func (e *suspensionTestExecutor) Execute(ctx context.Context, auth *Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.calls == nil {
		e.calls = make(map[string]int)
	}
	e.calls[auth.ID]++
	if auth.ID == "auth-1" {
		return cliproxyexecutor.Response{}, &Error{
			HTTPStatus: http.StatusForbidden,
			Code:       "TOS_VIOLATION",
			Message:    "This service has been disabled in this account due to Terms of Service violations.",
		}
	}
	return cliproxyexecutor.Response{Payload: []byte(`{"status":"ok"}`)}, nil
}

func (e *suspensionTestExecutor) ExecuteStream(context.Context, *Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, errors.New("stream not supported")
}

func (e *suspensionTestExecutor) Refresh(ctx context.Context, auth *Auth) (*Auth, error) {
	return auth, nil
}

func (e *suspensionTestExecutor) CountTokens(context.Context, *Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, errors.New("not supported")
}

func (e *suspensionTestExecutor) HttpRequest(context.Context, *Auth, *http.Request) (*http.Response, error) {
	return nil, errors.New("not supported")
}

func TestAntigravitySuspension_TosViolationCooldown(t *testing.T) {
	withQuotaCooldownEnabled(t)
	manager := NewManager(nil, nil, nil)
	auth := &Auth{
		ID:       "antigravity-auth-tos",
		Provider: "antigravity",
		Status:   StatusActive,
		ModelStates: map[string]*ModelState{
			"gemini-2.5-pro": {
				Status: StatusActive,
			},
			"gemini-2.5-flash": {
				Status: StatusActive,
			},
		},
	}
	if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("register: %v", errRegister)
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "antigravity", []*registry.ModelInfo{{ID: "gemini-2.5-pro"}, {ID: "gemini-2.5-flash"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })

	now := time.Now()
	manager.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: "antigravity",
		Model:    "gemini-2.5-pro",
		Success:  false,
		Error: &Error{
			HTTPStatus: http.StatusForbidden,
			Code:       "TOS_VIOLATION",
			Message:    "This service has been disabled in this account due to Terms of Service violations.",
		},
	})

	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatalf("auth not found")
	}
	if !updated.Unavailable {
		t.Errorf("expected auth to be unavailable")
	}
	if updated.StatusMessage != "tos_violation" {
		t.Errorf("expected statusMessage to be 'tos_violation', got %q", updated.StatusMessage)
	}
	expectedExpiry := now.Add(serviceScopedSuspensionCooldown)
	if updated.NextRetryAfter.Before(expectedExpiry.Add(-time.Minute)) || updated.NextRetryAfter.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("expected NextRetryAfter ~24h, got %v (relative to %v)", updated.NextRetryAfter, now)
	}
	flashState := updated.ModelStates["gemini-2.5-flash"]
	if flashState == nil || !flashState.Unavailable || flashState.StatusMessage != "tos_violation" {
		t.Errorf("expected sibling gemini-2.5-flash to be unavailable with tos_violation, got %+v", flashState)
	}
}

func TestAntigravitySuspension_ValidationRequiredCooldown(t *testing.T) {
	withQuotaCooldownEnabled(t)
	manager := NewManager(nil, nil, nil)
	auth := &Auth{
		ID:       "antigravity-auth-validation",
		Provider: "antigravity",
		Status:   StatusActive,
		ModelStates: map[string]*ModelState{
			"gemini-2.5-pro": {
				Status: StatusActive,
			},
			"gemini-2.5-flash": {
				Status: StatusActive,
			},
		},
	}
	if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("register: %v", errRegister)
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "antigravity", []*registry.ModelInfo{{ID: "gemini-2.5-pro"}, {ID: "gemini-2.5-flash"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })

	now := time.Now()
	manager.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: "antigravity",
		Model:    "gemini-2.5-pro",
		Success:  false,
		Error: &Error{
			HTTPStatus: http.StatusForbidden,
			Code:       "VALIDATION_REQUIRED",
			Message:    "Verify your account to continue.",
		},
	})

	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatalf("auth not found")
	}
	if !updated.Unavailable {
		t.Errorf("expected auth to be unavailable")
	}
	if updated.StatusMessage != "validation_required" {
		t.Errorf("expected statusMessage to be 'validation_required', got %q", updated.StatusMessage)
	}
	expectedExpiry := now.Add(serviceScopedSuspensionCooldown)
	if updated.NextRetryAfter.Before(expectedExpiry.Add(-time.Minute)) || updated.NextRetryAfter.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("expected NextRetryAfter ~24h, got %v (relative to %v)", updated.NextRetryAfter, now)
	}
	flashState := updated.ModelStates["gemini-2.5-flash"]
	if flashState == nil || !flashState.Unavailable || flashState.StatusMessage != "validation_required" {
		t.Errorf("expected sibling gemini-2.5-flash to be unavailable with validation_required, got %+v", flashState)
	}
}

func TestAntigravitySuspension_Ordinary403StandardCooldown(t *testing.T) {
	withQuotaCooldownEnabled(t)
	manager := NewManager(nil, nil, nil)
	auth := &Auth{
		ID:       "antigravity-auth-403",
		Provider: "antigravity",
		Status:   StatusActive,
		ModelStates: map[string]*ModelState{
			"gemini-2.5-pro": {
				Status: StatusActive,
			},
			"gemini-2.5-flash": {
				Status: StatusActive,
			},
		},
	}
	if _, errRegister := manager.Register(context.Background(), auth); errRegister != nil {
		t.Fatalf("register: %v", errRegister)
	}
	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth.ID, "antigravity", []*registry.ModelInfo{{ID: "gemini-2.5-pro"}, {ID: "gemini-2.5-flash"}})
	t.Cleanup(func() { reg.UnregisterClient(auth.ID) })

	now := time.Now()
	manager.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: "antigravity",
		Model:    "gemini-2.5-pro",
		Success:  false,
		Error: &Error{
			HTTPStatus: http.StatusForbidden,
			Message:    "Permission denied",
		},
	})

	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatalf("auth not found")
	}
	if updated.StatusMessage == "tos_violation" || updated.StatusMessage == "validation_required" {
		t.Errorf("did not expect suspension status message, got %q", updated.StatusMessage)
	}
	proState := updated.ModelStates["gemini-2.5-pro"]
	if proState == nil || !proState.Unavailable {
		t.Fatalf("expected gemini-2.5-pro to be unavailable")
	}
	expectedExpiry := now.Add(30 * time.Minute)
	if proState.NextRetryAfter.Before(expectedExpiry.Add(-time.Minute)) || proState.NextRetryAfter.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("expected NextRetryAfter ~30m, got %v (relative to %v)", proState.NextRetryAfter, now)
	}
	flashState := updated.ModelStates["gemini-2.5-flash"]
	if flashState != nil && flashState.Unavailable {
		t.Errorf("sibling gemini-2.5-flash should not be unavailable for ordinary 403, got %+v", flashState)
	}
}

func TestAntigravitySuspension_ApplyAuthFailureStateDirect(t *testing.T) {
	now := time.Now()

	// Case 1: TOS_VIOLATION sets tos_violation and 24h cooldown
	auth1 := &Auth{ID: "auth-tos", Status: StatusActive}
	errTos := &Error{
		HTTPStatus: http.StatusForbidden,
		Message:    "violation of terms of service detected",
	}
	applyAuthFailureState(auth1, errTos, nil, now, false)
	if !auth1.Unavailable || auth1.StatusMessage != "tos_violation" {
		t.Errorf("auth1 expected unavailable with tos_violation, got unavailable=%v, msg=%q", auth1.Unavailable, auth1.StatusMessage)
	}
	if diff := auth1.NextRetryAfter.Sub(now.Add(24 * time.Hour)); diff < -time.Second || diff > time.Second {
		t.Errorf("auth1 expected 24h cooldown, got %v diff from 24h", diff)
	}

	// Case 2: VALIDATION_REQUIRED sets validation_required and 24h cooldown
	auth2 := &Auth{ID: "auth-val", Status: StatusActive}
	errVal := &Error{
		HTTPStatus: http.StatusForbidden,
		Message:    "verify your account to continue",
	}
	applyAuthFailureState(auth2, errVal, nil, now, false)
	if !auth2.Unavailable || auth2.StatusMessage != "validation_required" {
		t.Errorf("auth2 expected unavailable with validation_required, got unavailable=%v, msg=%q", auth2.Unavailable, auth2.StatusMessage)
	}
	if diff := auth2.NextRetryAfter.Sub(now.Add(24 * time.Hour)); diff < -time.Second || diff > time.Second {
		t.Errorf("auth2 expected 24h cooldown, got %v diff from 24h", diff)
	}

	// Case 3: Ordinary 403 sets payment_required and 30m cooldown
	auth3 := &Auth{ID: "auth-403", Status: StatusActive}
	errOrd := &Error{
		HTTPStatus: http.StatusForbidden,
		Message:    "forbidden",
	}
	applyAuthFailureState(auth3, errOrd, nil, now, false)
	if !auth3.Unavailable || auth3.StatusMessage != "payment_required" {
		t.Errorf("auth3 expected unavailable with payment_required, got unavailable=%v, msg=%q", auth3.Unavailable, auth3.StatusMessage)
	}
	if diff := auth3.NextRetryAfter.Sub(now.Add(30 * time.Minute)); diff < -time.Second || diff > time.Second {
		t.Errorf("auth3 expected 30m cooldown, got %v diff from 30m", diff)
	}

	// Case 4: disableCooling suppresses NextRetryAfter
	auth4 := &Auth{ID: "auth-nocool", Status: StatusActive}
	applyAuthFailureState(auth4, errTos, nil, now, true)
	if auth4.Unavailable {
		t.Errorf("auth4 expected available when cooling is disabled")
	}
	if !auth4.NextRetryAfter.IsZero() {
		t.Errorf("auth4 expected zero NextRetryAfter, got %v", auth4.NextRetryAfter)
	}
}

func TestAntigravitySuspension_FailoverJitterHook(t *testing.T) {
	withQuotaCooldownEnabled(t)
	manager := NewManager(nil, nil, nil)
	executor := &suspensionTestExecutor{}
	manager.RegisterExecutor(executor)

	auth1 := &Auth{ID: "auth-1", Provider: "antigravity", Status: StatusActive}
	auth2 := &Auth{ID: "auth-2", Provider: "antigravity", Status: StatusActive}

	if _, err := manager.Register(context.Background(), auth1); err != nil {
		t.Fatalf("register auth1: %v", err)
	}
	if _, err := manager.Register(context.Background(), auth2); err != nil {
		t.Fatalf("register auth2: %v", err)
	}

	reg := registry.GetGlobalRegistry()
	reg.RegisterClient(auth1.ID, "antigravity", []*registry.ModelInfo{{ID: "gemini-2.5-pro"}})
	reg.RegisterClient(auth2.ID, "antigravity", []*registry.ModelInfo{{ID: "gemini-2.5-pro"}})
	t.Cleanup(func() {
		reg.UnregisterClient(auth1.ID)
		reg.UnregisterClient(auth2.ID)
	})

	var jitterCalls int
	prevHook := failoverContagionJitterFunc
	failoverContagionJitterFunc = func(ctx context.Context, provider string, err error) error {
		if provider == "antigravity" && statusCodeFromError(err) == http.StatusForbidden {
			jitterCalls++
		}
		return nil
	}
	t.Cleanup(func() { failoverContagionJitterFunc = prevHook })

	resp, err := manager.Execute(context.Background(), []string{"antigravity"}, cliproxyexecutor.Request{Model: "gemini-2.5-pro"}, cliproxyexecutor.Options{})
	if err != nil {
		t.Fatalf("expected execute to succeed via failover, got %v", err)
	}
	if string(resp.Payload) != `{"status":"ok"}` {
		t.Errorf("unexpected payload: %s", string(resp.Payload))
	}
	if jitterCalls != 1 {
		t.Errorf("expected exactly 1 failover jitter call, got %d", jitterCalls)
	}
}

func TestAntigravitySuspension_FailoverJitterContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err403 := &Error{HTTPStatus: http.StatusForbidden}
	err := defaultAntigravityFailoverJitterWait(ctx, "antigravity", err403)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestAntigravitySuspension_IsNonQuotaAuthCooling(t *testing.T) {
	now := time.Now()

	// 1. nil auth
	if IsNonQuotaAuthCooling(nil, now) {
		t.Error("expected false for nil auth")
	}

	// 2. Available auth
	authActive := &Auth{ID: "active", Status: StatusActive, Unavailable: false}
	if IsNonQuotaAuthCooling(authActive, now) {
		t.Error("expected false for available auth")
	}

	// 3. 429 Quota-cooled auth (Quota.Exceeded = true) -> should NOT be considered non-quota cooling
	authQuota := &Auth{
		ID:             "quota",
		Status:         StatusActive,
		Unavailable:    true,
		Quota:          QuotaState{Exceeded: true},
		NextRetryAfter: now.Add(10 * time.Minute),
	}
	if IsNonQuotaAuthCooling(authQuota, now) {
		t.Error("expected false for quota-cooled auth so it can still probe credits")
	}

	// 4. Non-quota cooling with past retry
	authPast := &Auth{
		ID:             "past",
		Status:         StatusActive,
		Unavailable:    true,
		NextRetryAfter: now.Add(-time.Minute),
	}
	if IsNonQuotaAuthCooling(authPast, now) {
		t.Error("expected false for expired cooldown")
	}

	// 5. Non-quota active cooling (e.g. 24h TOS violation or 403 payment_required)
	authSuspended := &Auth{
		ID:             "suspended",
		Status:         StatusActive,
		Unavailable:    true,
		StatusMessage:  "tos_violation",
		NextRetryAfter: now.Add(24 * time.Hour),
	}
	if !IsNonQuotaAuthCooling(authSuspended, now) {
		t.Error("expected true for actively suspended non-quota auth")
	}
}

package cliproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

func TestGeminiConfigModelsStaticMetadataInheritance(t *testing.T) {
	entry := &config.GeminiKey{
		Models: []config.GeminiModel{
			{
				Name:  "gemini-2.5-pro",
				Alias: "gemini-2.5-pro",
			},
		},
	}
	models := buildGeminiConfigModels(entry)
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	m := models[0]
	if m.InputTokenLimit <= 0 {
		t.Errorf("expected InputTokenLimit > 0, got %d", m.InputTokenLimit)
	}
	if m.OutputTokenLimit <= 0 {
		t.Errorf("expected OutputTokenLimit > 0, got %d", m.OutputTokenLimit)
	}
	if m.ContextLength <= 0 {
		t.Errorf("expected ContextLength > 0, got %d", m.ContextLength)
	}
	if m.MaxCompletionTokens <= 0 {
		t.Errorf("expected MaxCompletionTokens > 0, got %d", m.MaxCompletionTokens)
	}
	if len(m.SupportedInputModalities) == 0 {
		t.Errorf("expected non-empty SupportedInputModalities")
	}
	if len(m.SupportedOutputModalities) == 0 {
		t.Errorf("expected non-empty SupportedOutputModalities")
	}
	if m.Thinking == nil {
		t.Errorf("expected Thinking to be populated")
	}
	if m.DisplayName != "Gemini 2.5 Pro" {
		t.Errorf("expected DisplayName = %q, got %q", "Gemini 2.5 Pro", m.DisplayName)
	}
}

func TestGeminiConfigModelsUpstreamEnrichment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models" {
			http.NotFound(w, r)
			return
		}
		if apiKey := r.Header.Get("x-goog-api-key"); apiKey != "test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"models": [
				{
					"name": "models/gemini-4-argon",
					"version": "4.0",
					"displayName": "Gemini 4 Argon",
					"outputTokenLimit": 32768
				},
				{
					"name": "models/gemini-nano-banana-2.1",
					"version": "2.1",
					"displayName": "Nano Banana 2.1",
					"inputTokenLimit": 65536,
					"outputTokenLimit": 65536,
					"supportedInputModalities": ["text", "image"],
					"supportedOutputModalities": ["text", "image"],
					"thinking": true,
					"capabilities": {
						"google_search": true
					}
				}
			]
		}`))
	}))
	defer server.Close()

	entry := &config.GeminiKey{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
		Models: []config.GeminiModel{
			{
				Name:        "gemini-4-argon",
				Alias:       "gemini-4-argon",
				DisplayName: "Gemini 4 Argon",
			},
			{
				Name:  "gemini-nano-banana-2.1",
				Alias: "gemini-nano-banana-2.1",
			},
		},
	}

	initialModels := buildGeminiConfigModels(entry)
	if len(initialModels) != 2 {
		t.Fatalf("expected 2 models, got %d", len(initialModels))
	}

	geminiUpstreamModelsMu.Lock()
	geminiUpstreamModelsCache = make(map[string]geminiUpstreamCacheEntry)
	geminiUpstreamModelsMu.Unlock()

	svc := &Service{}
	enriched := svc.enrichGeminiConfigModelsFromUpstream(context.Background(), nil, nil, entry, initialModels)
	if len(enriched) != 2 {
		t.Fatalf("expected 2 models after enrichment, got %d", len(enriched))
	}

	var argon, banana *ModelInfo
	for _, m := range enriched {
		switch m.ID {
		case "gemini-4-argon":
			argon = m
		case "gemini-nano-banana-2.1":
			banana = m
		}
	}

	if argon == nil {
		t.Fatal("gemini-4-argon model not found in enriched models")
	}
	if argon.OutputTokenLimit != 32768 {
		t.Errorf("gemini-4-argon OutputTokenLimit = %d, want 32768", argon.OutputTokenLimit)
	}
	if argon.MaxCompletionTokens != 32768 {
		t.Errorf("gemini-4-argon MaxCompletionTokens = %d, want 32768", argon.MaxCompletionTokens)
	}

	if banana == nil {
		t.Fatal("gemini-nano-banana-2.1 model not found in enriched models")
	}
	if banana.InputTokenLimit != 65536 {
		t.Errorf("banana InputTokenLimit = %d, want 65536", banana.InputTokenLimit)
	}
	if banana.OutputTokenLimit != 65536 {
		t.Errorf("banana OutputTokenLimit = %d, want 65536", banana.OutputTokenLimit)
	}
	if banana.ContextLength != 65536 {
		t.Errorf("banana ContextLength = %d, want 65536", banana.ContextLength)
	}
	if banana.MaxCompletionTokens != 65536 {
		t.Errorf("banana MaxCompletionTokens = %d, want 65536", banana.MaxCompletionTokens)
	}
	if banana.DisplayName != "Nano Banana 2.1" {
		t.Errorf("banana DisplayName = %q, want %q", banana.DisplayName, "Nano Banana 2.1")
	}
	if len(banana.SupportedInputModalities) != 2 || banana.SupportedInputModalities[0] != "text" || banana.SupportedInputModalities[1] != "image" {
		t.Errorf("banana SupportedInputModalities = %v, want [text image]", banana.SupportedInputModalities)
	}
	if len(banana.SupportedOutputModalities) != 2 || banana.SupportedOutputModalities[0] != "text" || banana.SupportedOutputModalities[1] != "image" {
		t.Errorf("banana SupportedOutputModalities = %v, want [text image]", banana.SupportedOutputModalities)
	}
	if banana.Thinking == nil || !banana.Thinking.DynamicAllowed {
		t.Errorf("banana Thinking not populated or DynamicAllowed is false")
	}
	if banana.NativeCapabilities == nil || banana.NativeCapabilities.WebSearch == nil || !*banana.NativeCapabilities.WebSearch {
		t.Errorf("banana NativeCapabilities.WebSearch expected true")
	}
}

func TestApplyExcludedModelsWildcardAndExact(t *testing.T) {
	userDefined1 := &ModelInfo{
		ID:          "gemini-4-argon",
		UserDefined: true,
	}
	userDefined2 := &ModelInfo{
		ID:          "gemini-nano-banana-2.1",
		UserDefined: true,
	}
	catalogModel := &ModelInfo{
		ID:          "gemini-catalog-default",
		UserDefined: false,
	}

	// 1. Wildcard "*" should exclude non-user-defined models while preserving user-defined models.
	resWildcard := applyExcludedModels([]*ModelInfo{userDefined1, userDefined2, catalogModel}, []string{"*"})
	if len(resWildcard) != 2 {
		t.Fatalf("expected 2 user-defined models kept with wildcard *, got %d", len(resWildcard))
	}
	for _, m := range resWildcard {
		if !m.UserDefined {
			t.Errorf("expected only user-defined models, got %s", m.ID)
		}
	}

	// 2. Exact match should exclude matching user-defined model.
	resExact := applyExcludedModels([]*ModelInfo{userDefined1, userDefined2}, []string{"gemini-4-argon"})
	if len(resExact) != 1 {
		t.Fatalf("expected 1 model after exact exclusion, got %d", len(resExact))
	}
	if resExact[0].ID != "gemini-nano-banana-2.1" {
		t.Errorf("expected gemini-nano-banana-2.1 to survive, got %s", resExact[0].ID)
	}

	// 3. Pattern with wildcard should not exclude user-defined models.
	resPatternWildcard := applyExcludedModels([]*ModelInfo{userDefined1, catalogModel}, []string{"gemini-*"})
	if len(resPatternWildcard) != 1 || resPatternWildcard[0].ID != "gemini-4-argon" {
		t.Fatalf("expected gemini-4-argon to survive wildcard pattern, got %v", resPatternWildcard)
	}
}

func TestGeminiUpstreamModelsCachingAndClock(t *testing.T) {
	var hitCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-4-argon","outputTokenLimit":32768}]}`))
	}))
	defer server.Close()

	mockTime := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	oldNow := geminiUpstreamModelsNowFunc
	geminiUpstreamModelsNowFunc = func() time.Time {
		return mockTime
	}
	defer func() {
		geminiUpstreamModelsNowFunc = oldNow
	}()

	geminiUpstreamModelsMu.Lock()
	geminiUpstreamModelsCache = make(map[string]geminiUpstreamCacheEntry)
	geminiUpstreamModelsMu.Unlock()

	entry := &config.GeminiKey{
		BaseURL: server.URL,
		Models:  []config.GeminiModel{{Name: "gemini-4-argon"}},
	}
	models := []*ModelInfo{{ID: "gemini-4-argon", MetadataModelID: "gemini-4-argon"}}

	svc := &Service{}
	svc.enrichGeminiConfigModelsFromUpstream(context.Background(), nil, nil, entry, models)
	if hitCount != 1 {
		t.Fatalf("expected 1 server hit, got %d", hitCount)
	}

	// Advance time by 1 hour (within TTL)
	mockTime = mockTime.Add(1 * time.Hour)
	svc.enrichGeminiConfigModelsFromUpstream(context.Background(), nil, nil, entry, models)
	if hitCount != 1 {
		t.Fatalf("expected cached call to not hit server, got %d", hitCount)
	}

	// Advance time past TTL (4 hours total)
	mockTime = mockTime.Add(3 * time.Hour)
	svc.enrichGeminiConfigModelsFromUpstream(context.Background(), nil, nil, entry, models)
	if hitCount != 2 {
		t.Fatalf("expected expired cache call to hit server, got %d", hitCount)
	}
}

func TestBuildGeminiConfigModelsPromotesWhitespaceAliasToDisplayName(t *testing.T) {
	models := buildGeminiConfigModels(&config.GeminiKey{
		Models: []config.GeminiModel{
			{Name: "gemini-4-argon", Alias: "Gemini 4 Argon"},
			{Name: "gemini-nano-banana-2.1", Alias: "Nano Banana 2.1"},
			{Name: "gemini-3.8-flash", Alias: "g38f"},
		},
	})
	if len(models) != 3 {
		t.Fatalf("len(models) = %d, want 3", len(models))
	}
	if models[0].ID != "gemini-4-argon" || models[0].DisplayName != "Gemini 4 Argon" {
		t.Fatalf("models[0] = (%q, %q), want (gemini-4-argon, Gemini 4 Argon)", models[0].ID, models[0].DisplayName)
	}
	if models[1].ID != "gemini-nano-banana-2.1" || models[1].DisplayName != "Nano Banana 2.1" {
		t.Fatalf("models[1] = (%q, %q), want (gemini-nano-banana-2.1, Nano Banana 2.1)", models[1].ID, models[1].DisplayName)
	}
	if models[2].ID != "g38f" {
		t.Fatalf("models[2].ID = %q, want g38f", models[2].ID)
	}
}

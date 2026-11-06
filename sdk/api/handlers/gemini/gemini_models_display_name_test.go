package gemini

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
)

func TestGeminiModelsResponseUsesConfiguredDisplayName(t *testing.T) {
	const clientID = "gemini-display-name-catalog-test"
	const modelID = "gemini-display-name-catalog-test"
	registryRef := registry.GetGlobalRegistry()
	registryRef.RegisterClient(clientID, "gemini", []*registry.ModelInfo{{
		ID: modelID, Name: modelID, DisplayName: "Configured Gemini Name",
	}})
	t.Cleanup(func() {
		registryRef.UnregisterClient(clientID)
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	NewGeminiAPIHandler(&handlers.BaseAPIHandler{}).GeminiModels(ctx)

	var response struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"models"`
	}
	if errUnmarshal := json.Unmarshal(recorder.Body.Bytes(), &response); errUnmarshal != nil {
		t.Fatalf("decode response: %v", errUnmarshal)
	}
	for _, model := range response.Models {
		if model.Name == "models/"+modelID {
			if model.DisplayName != "Configured Gemini Name" {
				t.Fatalf("displayName = %q, want Configured Gemini Name", model.DisplayName)
			}
			return
		}
	}
	t.Fatalf("model %q not found in response", modelID)
}

func TestGeminiGetHandlerReturnsNormalizedModelMetadata(t *testing.T) {
	const clientID = "gemini-get-model-test-client"
	const modelID = "gemini-custom-model"

	registryRef := registry.GetGlobalRegistry()
	registryRef.RegisterClient(clientID, "gemini", []*registry.ModelInfo{
		{
			ID:                        modelID,
			Name:                      modelID, // raw name without models/ prefix
			DisplayName:               "",      // empty to test defaulting
			Description:               "",      // empty to test defaulting
			InputTokenLimit:           1048576,
			OutputTokenLimit:          65536,
			SupportedInputModalities:  []string{"TEXT", "IMAGE"},
			SupportedOutputModalities: []string{"TEXT"},
			Thinking:                  &registry.ThinkingSupport{DynamicAllowed: true},
		},
	})
	t.Cleanup(func() {
		registryRef.UnregisterClient(clientID)
	})

	handler := NewGeminiAPIHandler(&handlers.BaseAPIHandler{})

	// Test GET with action = "models/" + modelID
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "action", Value: "models/" + modelID}}
	handler.GeminiGetHandler(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response map[string]any
	if errUnmarshal := json.Unmarshal(recorder.Body.Bytes(), &response); errUnmarshal != nil {
		t.Fatalf("decode response: %v", errUnmarshal)
	}

	if got, _ := response["name"].(string); got != "models/"+modelID {
		t.Fatalf("name = %q, want %q", got, "models/"+modelID)
	}
	if got, _ := response["displayName"].(string); got != modelID {
		t.Fatalf("displayName = %q, want %q", got, modelID)
	}
	if got, _ := response["description"].(string); got != modelID {
		t.Fatalf("description = %q, want %q", got, modelID)
	}
	methods, ok := response["supportedGenerationMethods"].([]any)
	if !ok || len(methods) != 1 || methods[0] != "generateContent" {
		t.Fatalf("supportedGenerationMethods = %v, want [\"generateContent\"]", response["supportedGenerationMethods"])
	}
	if got, _ := response["inputTokenLimit"].(float64); got != 1048576 {
		t.Fatalf("inputTokenLimit = %v, want 1048576", response["inputTokenLimit"])
	}
	if got, _ := response["outputTokenLimit"].(float64); got != 65536 {
		t.Fatalf("outputTokenLimit = %v, want 65536", response["outputTokenLimit"])
	}
	inputMods, ok := response["supportedInputModalities"].([]any)
	if !ok || len(inputMods) != 2 || inputMods[0] != "TEXT" || inputMods[1] != "IMAGE" {
		t.Fatalf("supportedInputModalities = %v, want [\"TEXT\", \"IMAGE\"]", response["supportedInputModalities"])
	}
	outputMods, ok := response["supportedOutputModalities"].([]any)
	if !ok || len(outputMods) != 1 || outputMods[0] != "TEXT" {
		t.Fatalf("supportedOutputModalities = %v, want [\"TEXT\"]", response["supportedOutputModalities"])
	}
	if got, ok := response["thinking"].(bool); !ok || !got {
		t.Fatalf("thinking = %v, want true", response["thinking"])
	}

	// Also test action without "models/" prefix
	recorderRaw := httptest.NewRecorder()
	ctxRaw, _ := gin.CreateTestContext(recorderRaw)
	ctxRaw.Params = gin.Params{{Key: "action", Value: modelID}}
	handler.GeminiGetHandler(ctxRaw)

	if recorderRaw.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body: %s", recorderRaw.Code, http.StatusOK, recorderRaw.Body.String())
	}

	var responseRaw map[string]any
	if errUnmarshal := json.Unmarshal(recorderRaw.Body.Bytes(), &responseRaw); errUnmarshal != nil {
		t.Fatalf("decode raw response: %v", errUnmarshal)
	}
	if got, _ := responseRaw["name"].(string); got != "models/"+modelID {
		t.Fatalf("name = %q, want %q", got, "models/"+modelID)
	}
}

func TestGeminiGetHandlerOmitsThinkingWhenNil(t *testing.T) {
	const clientID = "gemini-get-model-no-thinking-client"
	const modelID = "gemini-non-thinking-model"

	registryRef := registry.GetGlobalRegistry()
	registryRef.RegisterClient(clientID, "gemini", []*registry.ModelInfo{
		{
			ID:          modelID,
			Name:        "models/" + modelID,
			DisplayName: "Non-Thinking Model",
		},
	})
	t.Cleanup(func() {
		registryRef.UnregisterClient(clientID)
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "action", Value: modelID}}

	handler := NewGeminiAPIHandler(&handlers.BaseAPIHandler{})
	handler.GeminiGetHandler(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response map[string]any
	if errUnmarshal := json.Unmarshal(recorder.Body.Bytes(), &response); errUnmarshal != nil {
		t.Fatalf("decode response: %v", errUnmarshal)
	}
	if _, exists := response["thinking"]; exists {
		t.Fatalf("expected thinking to be omitted when nil, got %v", response["thinking"])
	}
}

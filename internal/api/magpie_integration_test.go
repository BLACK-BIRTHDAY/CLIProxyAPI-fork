package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	proxyconfig "github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/magpie"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
)

func setupMockMagpieBackend(t *testing.T) (*httptest.Server, *string, *string) {
	t.Helper()
	var lastReceivedChatToolName string
	var lastReceivedGeminiToolName string

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":                 "gemini-4-argon",
						"displayName":        "gemini-4-argon",
						"input_token_limit":  1048576,
						"output_token_limit": 65536,
						"capabilities": map[string]any{
							"function_declarations": true,
							"thinking":              true,
						},
						"thinking_levels": []any{"minimal", "low", "medium", "high"},
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/chat/completions":
			bodyBytes, _ := io.ReadAll(r.Body)
			var bodyMap map[string]any
			_ = json.Unmarshal(bodyBytes, &bodyMap)

			// Capture received tool name
			if tools, ok := bodyMap["tools"].([]any); ok && len(tools) > 0 {
				if tool0, ok := tools[0].(map[string]any); ok {
					if fn, ok := tool0["function"].(map[string]any); ok {
						if fnName, ok := fn["name"].(string); ok {
							lastReceivedChatToolName = fnName
						}
					}
				}
			}

			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"id":      "chatcmpl-test-123",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gemini-4-argon",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_abc123",
									"type": "function",
									"function": map[string]any{
										"name":      lastReceivedChatToolName,
										"arguments": "{}",
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "streamGenerateContent"):
			bodyBytes, _ := io.ReadAll(r.Body)
			var bodyMap map[string]any
			_ = json.Unmarshal(bodyBytes, &bodyMap)

			// Capture received gemini tool name
			if tools, ok := bodyMap["tools"].([]any); ok && len(tools) > 0 {
				if tool0, ok := tools[0].(map[string]any); ok {
					if fns, ok := tool0["functionDeclarations"].([]any); ok && len(fns) > 0 {
						if fn0, ok := fns[0].(map[string]any); ok {
							if fnName, ok := fn0["name"].(string); ok {
								lastReceivedGeminiToolName = fnName
							}
						}
					}
				}
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			flusher, _ := w.(http.Flusher)

			eventData := map[string]any{
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"parts": []map[string]any{
								{
									"functionCall": map[string]any{
										"name": lastReceivedGeminiToolName,
										"args": map[string]any{},
									},
								},
							},
						},
						"finishReason": "STOP",
					},
				},
				"usageMetadata": map[string]any{
					"promptTokenCount":     15,
					"candidatesTokenCount": 25,
					"totalTokenCount":      40,
				},
			}
			eventBytes, _ := json.Marshal(eventData)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(eventBytes))
			if flusher != nil {
				flusher.Flush()
			}

		case r.Method == http.MethodGet && r.URL.Path == "/api/status":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"running": true,
				"ready":   true,
				"state":   "RUNNING",
				"version": "0.2.19-magpie.1",
				"accounts": map[string]any{
					"total":         1,
					"ready":         1,
					"busy":          0,
					"cooldown":      0,
					"auth_required": 0,
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/accounts":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"accounts": []map[string]any{
					{
						"id":      "test-account-1",
						"label":   "Test Account 1",
						"enabled": true,
					},
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/cooldowns":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})

		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true}`))
		}
	}))

	t.Cleanup(func() {
		mockBackend.Close()
	})

	magpie.SetTestBackendOverride(&magpie.TestBackendOverride{
		BaseURL: mockBackend.URL,
		APIKey:  "test-magpie-key",
		Config:  *magpie.NewDefaultConfig(),
	})
	t.Cleanup(func() {
		magpie.SetTestBackendOverride(nil)
	})

	return mockBackend, &lastReceivedChatToolName, &lastReceivedGeminiToolName
}

func TestMagpieModelCatalogAndDetail(t *testing.T) {
	setupMockMagpieBackend(t)

	cfg := &proxyconfig.Config{
		SDKConfig: sdkconfig.SDKConfig{
			APIKeys: []string{"test-client-key"},
		},
	}
	server := newTestServerWithConfig(t, cfg)

	// 1. /v1/models
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-client-key")
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /v1/models code = %d, body = %s", rr.Code, rr.Body.String())
	}
	var v1List struct {
		Object string           `json:"object"`
		Data   []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &v1List); err != nil {
		t.Fatalf("decode /v1/models: %v", err)
	}

	var foundMagpieModel map[string]any
	for _, m := range v1List.Data {
		if id, _ := m["id"].(string); id == "(MAGPIE)/gemini-4-argon" {
			foundMagpieModel = m
			break
		}
	}
	if foundMagpieModel == nil {
		t.Fatalf("GET /v1/models missing (MAGPIE)/gemini-4-argon: %#v", v1List.Data)
	}
	if ownedBy, _ := foundMagpieModel["owned_by"].(string); ownedBy != "magpie" {
		t.Fatalf("owned_by = %q, want 'magpie'", ownedBy)
	}
	if displayName, _ := foundMagpieModel["display_name"].(string); displayName != "(MAGPIE)/gemini-4-argon" {
		t.Fatalf("display_name = %q, want '(MAGPIE)/gemini-4-argon'", displayName)
	}

	// 2. /v1/models/(MAGPIE)/gemini-4-argon
	reqDetail := httptest.NewRequest(http.MethodGet, "/v1/models/(MAGPIE)/gemini-4-argon", nil)
	reqDetail.Header.Set("Authorization", "Bearer test-client-key")
	rrDetail := httptest.NewRecorder()
	server.engine.ServeHTTP(rrDetail, reqDetail)

	if rrDetail.Code != http.StatusOK {
		t.Fatalf("GET /v1/models/(MAGPIE)/gemini-4-argon code = %d, body = %s", rrDetail.Code, rrDetail.Body.String())
	}
	var detailModel map[string]any
	if err := json.Unmarshal(rrDetail.Body.Bytes(), &detailModel); err != nil {
		t.Fatalf("decode single model detail: %v", err)
	}
	if id, _ := detailModel["id"].(string); id != "(MAGPIE)/gemini-4-argon" {
		t.Fatalf("single model detail id = %q, want '(MAGPIE)/gemini-4-argon'", id)
	}

	// 3. /v1beta/models
	reqBeta := httptest.NewRequest(http.MethodGet, "/v1beta/models", nil)
	reqBeta.Header.Set("Authorization", "Bearer test-client-key")
	rrBeta := httptest.NewRecorder()
	server.engine.ServeHTTP(rrBeta, reqBeta)

	if rrBeta.Code != http.StatusOK {
		t.Fatalf("GET /v1beta/models code = %d, body = %s", rrBeta.Code, rrBeta.Body.String())
	}
	var betaList struct {
		Models []map[string]any `json:"models"`
	}
	if err := json.Unmarshal(rrBeta.Body.Bytes(), &betaList); err != nil {
		t.Fatalf("decode /v1beta/models: %v", err)
	}
	var foundBetaModel map[string]any
	for _, m := range betaList.Models {
		if name, _ := m["name"].(string); name == "models/(MAGPIE)/gemini-4-argon" {
			foundBetaModel = m
			break
		}
	}
	if foundBetaModel == nil {
		t.Fatalf("GET /v1beta/models missing models/(MAGPIE)/gemini-4-argon: %#v", betaList.Models)
	}
	if dn, _ := foundBetaModel["displayName"].(string); dn != "(MAGPIE)/gemini-4-argon" {
		t.Fatalf("displayName = %q, want '(MAGPIE)/gemini-4-argon'", dn)
	}

	// 4. GET /v1beta/models/(MAGPIE)/gemini-4-argon
	reqBetaDetail := httptest.NewRequest(http.MethodGet, "/v1beta/models/(MAGPIE)/gemini-4-argon", nil)
	reqBetaDetail.Header.Set("Authorization", "Bearer test-client-key")
	rrBetaDetail := httptest.NewRecorder()
	server.engine.ServeHTTP(rrBetaDetail, reqBetaDetail)

	if rrBetaDetail.Code != http.StatusOK {
		t.Fatalf("GET /v1beta/models/(MAGPIE)/gemini-4-argon code = %d, body = %s", rrBetaDetail.Code, rrBetaDetail.Body.String())
	}
	var betaSingle map[string]any
	if err := json.Unmarshal(rrBetaDetail.Body.Bytes(), &betaSingle); err != nil {
		t.Fatalf("decode /v1beta single model: %v", err)
	}
	if dn, _ := betaSingle["displayName"].(string); dn != "(MAGPIE)/gemini-4-argon" {
		t.Fatalf("GET /v1beta/models/(MAGPIE)/gemini-4-argon displayName = %q, want '(MAGPIE)/gemini-4-argon'", dn)
	}
}

func TestMagpieRequestForwardingWithToolAliasing(t *testing.T) {
	_, lastChatTool, lastGeminiTool := setupMockMagpieBackend(t)

	cfg := &proxyconfig.Config{
		SDKConfig: sdkconfig.SDKConfig{
			APIKeys: []string{"test-client-key"},
		},
	}
	server := newTestServerWithConfig(t, cfg)

	longToolName := "very_long_tool_identifier_name_that_exceeds_sixty_three_characters_long_123"
	if len(longToolName) < 64 {
		t.Fatalf("tool name must be >= 64 chars, got %d", len(longToolName))
	}

	// 1. POST /v1/chat/completions with long tool name
	chatReqBody := map[string]any{
		"model": "(MAGPIE)/gemini-4-argon",
		"messages": []map[string]any{
			{"role": "user", "content": "Hello!"},
		},
		"tools": []map[string]any{
			{
				"type": "function",
				"function": map[string]any{
					"name":        longToolName,
					"description": "A tool with an overly long identifier",
					"parameters": map[string]any{
						"type":       "object",
						"properties": map[string]any{},
					},
				},
			},
		},
	}
	chatJSON, _ := json.Marshal(chatReqBody)
	reqChat := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(chatJSON))
	reqChat.Header.Set("Authorization", "Bearer test-client-key")
	reqChat.Header.Set("Content-Type", "application/json")
	rrChat := httptest.NewRecorder()
	server.engine.ServeHTTP(rrChat, reqChat)

	if rrChat.Code != http.StatusOK {
		t.Fatalf("POST /v1/chat/completions code = %d, body = %s", rrChat.Code, rrChat.Body.String())
	}

	// Upstream received tool name must be aliased to <= 63 bytes
	if len(*lastChatTool) > 63 {
		t.Fatalf("upstream received tool name exceeded 63 bytes: %q (len=%d)", *lastChatTool, len(*lastChatTool))
	}
	if *lastChatTool == longToolName {
		t.Fatalf("upstream tool name was not aliased: %q", *lastChatTool)
	}

	// Downstream response must have the original long tool name restored
	var chatResp map[string]any
	if err := json.Unmarshal(rrChat.Body.Bytes(), &chatResp); err != nil {
		t.Fatalf("decode chat completions response: %v", err)
	}
	choices, _ := chatResp["choices"].([]any)
	if len(choices) == 0 {
		t.Fatalf("no choices in response: %#v", chatResp)
	}
	choice0, _ := choices[0].(map[string]any)
	msg, _ := choice0["message"].(map[string]any)
	toolCalls, _ := msg["tool_calls"].([]any)
	if len(toolCalls) == 0 {
		t.Fatalf("no tool_calls in message: %#v", msg)
	}
	call0, _ := toolCalls[0].(map[string]any)
	fn, _ := call0["function"].(map[string]any)
	restoredName, _ := fn["name"].(string)
	if restoredName != longToolName {
		t.Fatalf("restored tool name = %q, want original %q", restoredName, longToolName)
	}

	// 2. POST /v1beta/models/(MAGPIE)/gemini-4-argon:streamGenerateContent with long tool name
	geminiReqBody := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": "Call the long tool!"},
				},
			},
		},
		"tools": []map[string]any{
			{
				"functionDeclarations": []map[string]any{
					{
						"name":        longToolName,
						"description": "A long gemini tool",
						"parameters": map[string]any{
							"type":       "OBJECT",
							"properties": map[string]any{},
						},
					},
				},
			},
		},
	}
	geminiJSON, _ := json.Marshal(geminiReqBody)
	reqGemini := httptest.NewRequest(http.MethodPost, "/v1beta/models/(MAGPIE)/gemini-4-argon:streamGenerateContent", bytes.NewReader(geminiJSON))
	reqGemini.Header.Set("Authorization", "Bearer test-client-key")
	reqGemini.Header.Set("Content-Type", "application/json")
	rrGemini := httptest.NewRecorder()
	server.engine.ServeHTTP(rrGemini, reqGemini)

	if rrGemini.Code != http.StatusOK {
		t.Fatalf("POST /v1beta stream code = %d, body = %s", rrGemini.Code, rrGemini.Body.String())
	}

	// Upstream received tool name must be aliased to <= 63 bytes
	if len(*lastGeminiTool) > 63 {
		t.Fatalf("upstream received gemini tool name exceeded 63 bytes: %q (len=%d)", *lastGeminiTool, len(*lastGeminiTool))
	}
	if *lastGeminiTool == longToolName {
		t.Fatalf("upstream gemini tool name was not aliased: %q", *lastGeminiTool)
	}

	// Downstream SSE response must have the original long tool name restored
	scanner := bufio.NewScanner(rrGemini.Body)
	var foundRestoredToolInSSE bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			var sseVal map[string]any
			if err := json.Unmarshal([]byte(payload), &sseVal); err == nil {
				if candidates, ok := sseVal["candidates"].([]any); ok && len(candidates) > 0 {
					c0, _ := candidates[0].(map[string]any)
					content, _ := c0["content"].(map[string]any)
					parts, _ := content["parts"].([]any)
					if len(parts) > 0 {
						p0, _ := parts[0].(map[string]any)
						fc, _ := p0["functionCall"].(map[string]any)
						if name, _ := fc["name"].(string); name == longToolName {
							foundRestoredToolInSSE = true
							break
						}
					}
				}
			}
		}
	}
	if !foundRestoredToolInSSE {
		t.Fatalf("SSE response did not contain restored tool name %q; body: %s", longToolName, rrGemini.Body.String())
	}
}

func TestRootBrowserRedirect(t *testing.T) {
	cfg := &proxyconfig.Config{
		SDKConfig: sdkconfig.SDKConfig{
			APIKeys: []string{"test-client-key"},
		},
	}
	server := newTestServerWithConfig(t, cfg)

	// 1. Accept: text/html -> Redirect 302 to /management.html
	reqHTML := httptest.NewRequest(http.MethodGet, "/", nil)
	reqHTML.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	rrHTML := httptest.NewRecorder()
	server.engine.ServeHTTP(rrHTML, reqHTML)

	if rrHTML.Code != http.StatusFound {
		t.Fatalf("GET / with text/html code = %d, want %d", rrHTML.Code, http.StatusFound)
	}
	if loc := rrHTML.Header().Get("Location"); loc != "/management.html" {
		t.Fatalf("Location = %q, want '/management.html'", loc)
	}

	// 2. Accept: application/json -> 200 OK JSON
	reqJSON := httptest.NewRequest(http.MethodGet, "/", nil)
	reqJSON.Header.Set("Accept", "application/json")
	rrJSON := httptest.NewRecorder()
	server.engine.ServeHTTP(rrJSON, reqJSON)

	if rrJSON.Code != http.StatusOK {
		t.Fatalf("GET / with application/json code = %d, want %d", rrJSON.Code, http.StatusOK)
	}
	var res map[string]any
	if err := json.Unmarshal(rrJSON.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
	if msg, _ := res["message"].(string); msg != "CLI Proxy API Server" {
		t.Fatalf("message = %q, want 'CLI Proxy API Server'", msg)
	}
}

func TestMagpieManagementAndSessionCookie(t *testing.T) {
	setupMockMagpieBackend(t)
	t.Setenv("MAGPIE_BUILTIN_MENU", "1")
	t.Setenv("MANAGEMENT_PASSWORD", "test-mgmt-secret")

	cfg := &proxyconfig.Config{
		SDKConfig: sdkconfig.SDKConfig{
			APIKeys: []string{"test-client-key"},
		},
		RemoteManagement: proxyconfig.RemoteManagement{
			AllowRemote: true,
			SecretKey:   "test-mgmt-secret",
		},
	}
	server := newTestServerWithConfig(t, cfg)

	// 1. GET /v8/management/plugins with Bearer auth
	reqPlugins := httptest.NewRequest(http.MethodGet, "/v8/management/plugins", nil)
	reqPlugins.Header.Set("Authorization", "Bearer test-mgmt-secret")
	rrPlugins := httptest.NewRecorder()
	server.engine.ServeHTTP(rrPlugins, reqPlugins)

	if rrPlugins.Code != http.StatusOK {
		t.Fatalf("GET /v8/management/plugins code = %d, body = %s", rrPlugins.Code, rrPlugins.Body.String())
	}

	// Must set cpa_mgmt_session cookie
	setCookieHeader := rrPlugins.Header().Get("Set-Cookie")
	if !strings.Contains(setCookieHeader, "cpa_mgmt_session=") {
		t.Fatalf("Set-Cookie header missing cpa_mgmt_session: %q", setCookieHeader)
	}
	if !strings.Contains(setCookieHeader, "HttpOnly") {
		t.Fatalf("cpa_mgmt_session cookie missing HttpOnly: %q", setCookieHeader)
	}
	if !strings.Contains(setCookieHeader, "SameSite=Lax") {
		t.Fatalf("cpa_mgmt_session cookie missing SameSite=Lax: %q", setCookieHeader)
	}

	// Extract cookie value for subsequent request
	var cookieVal string
	for _, part := range strings.Split(setCookieHeader, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "cpa_mgmt_session=") {
			cookieVal = strings.TrimPrefix(part, "cpa_mgmt_session=")
			break
		}
	}
	if cookieVal == "" {
		t.Fatalf("failed to extract cpa_mgmt_session cookie from %q", setCookieHeader)
	}

	// Check that magpie plugin entry is listed with effective_enabled: true and 2 menus
	var pluginsResp struct {
		Plugins []struct {
			ID               string `json:"id"`
			EffectiveEnabled bool   `json:"effective_enabled"`
			Logo             string `json:"logo"`
			Menus            []struct {
				Path string `json:"path"`
				Menu string `json:"menu"`
			} `json:"menus"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(rrPlugins.Body.Bytes(), &pluginsResp); err != nil {
		t.Fatalf("decode plugins response: %v", err)
	}

	var foundMagpie bool
	for _, p := range pluginsResp.Plugins {
		if p.ID == "magpie" {
			foundMagpie = true
			if !p.EffectiveEnabled {
				t.Fatalf("magpie effective_enabled = false, want true")
			}
			if len(p.Menus) != 2 {
				t.Fatalf("magpie menus len = %d, want 2", len(p.Menus))
			}
			paths := []string{p.Menus[0].Path, p.Menus[1].Path}
			if !strings.Contains(strings.Join(paths, " "), "/v0/resource/plugins/magpie/quota") ||
				!strings.Contains(strings.Join(paths, " "), "/v0/resource/plugins/magpie/settings") {
				t.Fatalf("magpie menus missing expected paths: %#v", p.Menus)
			}
			break
		}
	}
	if !foundMagpie {
		t.Fatalf("GET /v8/management/plugins missing magpie entry: %#v", pluginsResp.Plugins)
	}

	// 2. Subsequent cookie-authenticated GET /v8/management/magpie/status WITHOUT Bearer token
	reqStatus := httptest.NewRequest(http.MethodGet, "/v8/management/magpie/status", nil)
	reqStatus.AddCookie(&http.Cookie{
		Name:  "cpa_mgmt_session",
		Value: cookieVal,
	})
	rrStatus := httptest.NewRecorder()
	server.engine.ServeHTTP(rrStatus, reqStatus)

	if rrStatus.Code != http.StatusOK {
		t.Fatalf("cookie-authenticated GET /v8/management/magpie/status code = %d, body = %s", rrStatus.Code, rrStatus.Body.String())
	}
	var statusBody map[string]any
	if err := json.Unmarshal(rrStatus.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("decode magpie status body: %v", err)
	}
	if proc, ok := statusBody["process"].(map[string]any); !ok || proc["running"] != true {
		t.Fatalf("magpie status process not running: %#v", statusBody)
	}

	// 3. GET /v0/resource/plugins/magpie/quota returns 200 HTML
	reqQuota := httptest.NewRequest(http.MethodGet, "/v0/resource/plugins/magpie/quota", nil)
	rrQuota := httptest.NewRecorder()
	server.engine.ServeHTTP(rrQuota, reqQuota)

	if rrQuota.Code != http.StatusOK {
		t.Fatalf("GET /v0/resource/plugins/magpie/quota code = %d, body = %s", rrQuota.Code, rrQuota.Body.String())
	}
	if ct := rrQuota.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rrQuota.Body.String(), "<title>MAGPIE") {
		t.Fatalf("quota HTML missing <title>MAGPIE: %s", rrQuota.Body.String()[:100])
	}

	// 4. GET /v0/resource/plugins/magpie/settings returns 200 HTML
	reqSettings := httptest.NewRequest(http.MethodGet, "/v0/resource/plugins/magpie/settings", nil)
	rrSettings := httptest.NewRecorder()
	server.engine.ServeHTTP(rrSettings, reqSettings)

	if rrSettings.Code != http.StatusOK {
		t.Fatalf("GET /v0/resource/plugins/magpie/settings code = %d, body = %s", rrSettings.Code, rrSettings.Body.String())
	}
	if ct := rrSettings.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rrSettings.Body.String(), "<title>MAGPIE") {
		t.Fatalf("settings HTML missing <title>MAGPIE: %s", rrSettings.Body.String()[:100])
	}
}

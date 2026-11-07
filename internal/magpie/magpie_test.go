package magpie

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	coreusage "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/usage"
)

func TestStripAndPrefixMagpieModel(t *testing.T) {
	cases := []struct {
		input       string
		expectedRaw string
		expectedOK  bool
	}{
		// ASCII slash prefix
		{"(MAGPIE)/gemini-3.1-pro-preview", "gemini-3.1-pro-preview", true},
		// Legacy ASCII prefix
		{"(MAGPIE)gemini-3.1-pro-preview", "gemini-3.1-pro-preview", true},
		// Full-width prefix (slash and legacy)
		{"（MAGPIE）/gemini-3.1-pro-preview", "gemini-3.1-pro-preview", true},
		{"（MAGPIE）gemini-3.1-pro-preview", "gemini-3.1-pro-preview", true},
		// Leading models/ with ASCII prefix
		{"models/(MAGPIE)/gemini-2.5-flash", "gemini-2.5-flash", true},
		{"models/(MAGPIE)gemini-2.5-flash", "gemini-2.5-flash", true},
		// ASCII prefix with inner models/
		{"(MAGPIE)/models/gemini-2.5-flash", "gemini-2.5-flash", true},
		// Leading models/ with full-width prefix
		{"models/（MAGPIE）gemini-2.5-flash", "gemini-2.5-flash", true},
		// Leading slash
		{"/(MAGPIE)/gemini-2.5-flash", "gemini-2.5-flash", true},
		{"/models/(MAGPIE)/gemini-2.5-flash", "gemini-2.5-flash", true},
		// Normal unprefixed model should return ("", false)
		{"gemini-3-flash", "", false},
		{"models/gemini-3-flash", "", false},
		{"/models/gemini-3-flash", "", false},
	}

	for _, tc := range cases {
		raw, ok := StripMagpiePrefix(tc.input)
		if ok != tc.expectedOK {
			t.Errorf("StripMagpiePrefix(%q) ok = %v, expected %v", tc.input, ok, tc.expectedOK)
		}
		if raw != tc.expectedRaw {
			t.Errorf("StripMagpiePrefix(%q) raw = %q, expected %q", tc.input, raw, tc.expectedRaw)
		}
		if IsMagpieModel(tc.input) != tc.expectedOK {
			t.Errorf("IsMagpieModel(%q) = %v, expected %v", tc.input, IsMagpieModel(tc.input), tc.expectedOK)
		}
	}

	// PrefixMagpieModel helper
	if got := PrefixMagpieModel("gemini-3.1-pro-preview"); got != "(MAGPIE)/gemini-3.1-pro-preview" {
		t.Errorf("PrefixMagpieModel = %q, expected %q", got, "(MAGPIE)/gemini-3.1-pro-preview")
	}
	if got := PrefixMagpieModel("models/gemini-3.1-pro-preview"); got != "(MAGPIE)/gemini-3.1-pro-preview" {
		t.Errorf("PrefixMagpieModel with models/ = %q, expected %q", got, "(MAGPIE)/gemini-3.1-pro-preview")
	}
	if got := PrefixMagpieModel("(MAGPIE)/gemini-3.1-pro-preview"); got != "(MAGPIE)/gemini-3.1-pro-preview" {
		t.Errorf("PrefixMagpieModel with prefix = %q, expected %q", got, "(MAGPIE)/gemini-3.1-pro-preview")
	}
}

func TestFilterStudioModels(t *testing.T) {
	rawCatalog := []map[string]any{
		{
			"id":                           "gemini-3.1-pro-preview",
			"name":                         "Gemini 3.1 Pro Preview",
			"supported_generation_methods": []any{"generateContent", "countTokens"},
			"input_token_limit":            float64(2097152),
			"output_token_limit":           float64(65536),
			"capabilities": map[string]any{
				"chat_model":            true,
				"thinking_level":        true,
				"function_declarations": true,
				"media_resolution":      true,
			},
		},
		{
			"id":                           "gemini-2.5-flash-image",
			"name":                         "Gemini 2.5 Flash Image",
			"supported_generation_methods": []any{"generateContent"},
			"capabilities": map[string]any{
				"chat_model":        true,
				"image_route":       true,
				"aspect_ratio":      true,
				"output_resolution": true,
			},
		},
		{
			"id":                           "gemini-coding-build",
			"name":                         "Gemini Coding Build",
			"supported_generation_methods": []any{"generateContent"},
			"capabilities": map[string]any{
				"chat_model": true,
			},
		},
		{
			"id":                           "gemini-4-argon-search",
			"name":                         "Gemini 4 Argon (Search)",
			"supported_generation_methods": []any{"generateContent"},
			"capabilities": map[string]any{
				"chat_model": true,
			},
		},
		{
			"id":                           "gemini-assistant",
			"name":                         "Gemini Assistant",
			"supported_generation_methods": []any{"generateContent"},
			"capabilities": map[string]any{
				"chat_model":      true,
				"assistant_route": true,
			},
		},
		{
			"id":                           "gemini-tts-model",
			"name":                         "Gemini TTS",
			"supported_generation_methods": []any{"generateContent"},
			"capabilities": map[string]any{
				"chat_model": true,
			},
		},
		{
			"id":                           "claude-sonnet-4-6",
			"name":                         "Claude Sonnet",
			"supported_generation_methods": []any{"generateContent"},
			"capabilities": map[string]any{
				"chat_model": true,
			},
		},
	}

	// 1. With image models allowed
	filteredWithImage := FilterStudioModels(rawCatalog, true)
	idMapWithImage := make(map[string]bool)
	for _, m := range filteredWithImage {
		idMapWithImage[m.ID] = true
	}
	if !idMapWithImage["gemini-3.1-pro-preview"] {
		t.Errorf("expected gemini-3.1-pro-preview to be included")
	}
	if !idMapWithImage["gemini-2.5-flash-image"] {
		t.Errorf("expected gemini-2.5-flash-image to be included when image models allowed")
	}
	if idMapWithImage["gemini-coding-build"] {
		t.Errorf("gemini-coding-build should be excluded (-build)")
	}
	if idMapWithImage["gemini-4-argon-search"] {
		t.Errorf("gemini-4-argon-search should be excluded (-search)")
	}
	if idMapWithImage["gemini-assistant"] {
		t.Errorf("gemini-assistant should be excluded (assistant_route)")
	}
	if idMapWithImage["gemini-tts-model"] {
		t.Errorf("gemini-tts-model should be excluded (tts)")
	}
	if idMapWithImage["claude-sonnet-4-6"] {
		t.Errorf("claude-sonnet-4-6 should be excluded (not gemini-)")
	}

	// 2. With image models disabled
	filteredNoImage := FilterStudioModels(rawCatalog, false)
	idMapNoImage := make(map[string]bool)
	for _, m := range filteredNoImage {
		idMapNoImage[m.ID] = true
	}
	if !idMapNoImage["gemini-3.1-pro-preview"] {
		t.Errorf("expected gemini-3.1-pro-preview to be included")
	}
	if idMapNoImage["gemini-2.5-flash-image"] {
		t.Errorf("gemini-2.5-flash-image should be excluded when image models disabled")
	}

	// Verify token limits and thinking levels
	for _, m := range filteredWithImage {
		if m.ID == "gemini-3.1-pro-preview" {
			if m.InputTokenLimit != 2097152 {
				t.Errorf("input token limit = %d, expected 2097152", m.InputTokenLimit)
			}
			if m.OutputTokenLimit != 65536 {
				t.Errorf("output token limit = %d, expected 65536", m.OutputTokenLimit)
			}
			if len(m.ThinkingLevels) != 4 || m.ThinkingLevels[0] != "minimal" {
				t.Errorf("thinking levels = %v, expected [minimal low medium high]", m.ThinkingLevels)
			}
			if !m.SupportsTools {
				t.Errorf("expected supports_tools = true")
			}
		}
	}
}

func TestConfigAndEnvSyncAndHydration(t *testing.T) {
	dataDir := t.TempDir()

	cfg := NewDefaultConfig()
	cfg.InternalPort = 2055
	cfg.Proxy = "" // empty, will inherit
	inheritedProxy := "http://127.0.0.1:7890"

	// 1. Ensure studio config created
	studioCfg, err := EnsureStudioConfig(dataDir, cfg, inheritedProxy)
	if err != nil {
		t.Fatalf("EnsureStudioConfig failed: %v", err)
	}
	if studioCfg.Port != 2055 {
		t.Errorf("studioCfg.Port = %d, expected 2055", studioCfg.Port)
	}
	if !strings.HasPrefix(studioCfg.APIKey, "as_") {
		t.Errorf("APIKey = %q, expected prefix 'as_'", studioCfg.APIKey)
	}
	if studioCfg.AdminUsername != "magpie" {
		t.Errorf("AdminUsername = %q, expected 'magpie'", studioCfg.AdminUsername)
	}
	if studioCfg.Proxy != inheritedProxy {
		t.Errorf("studioCfg.Proxy = %q, expected inherited %q", studioCfg.Proxy, inheritedProxy)
	}

	// 2. Check .env content
	envPath := filepath.Join(dataDir, ".env")
	envBytes, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	envText := string(envBytes)
	if !strings.Contains(envText, "LISTEN_ADDR=127.0.0.1:2055") {
		t.Errorf(".env missing LISTEN_ADDR=127.0.0.1:2055, got: %s", envText)
	}
	if !strings.Contains(envText, "PREVIEW_LISTEN_ADDR=127.0.0.1:2057") {
		t.Errorf(".env missing PREVIEW_LISTEN_ADDR=127.0.0.1:2057, got: %s", envText)
	}
	if !strings.Contains(envText, "PROXY=http://127.0.0.1:7890") {
		t.Errorf(".env missing PROXY=http://127.0.0.1:7890, got: %s", envText)
	}
	if !strings.Contains(envText, "WAA_BACKEND=spidermonkey") {
		t.Errorf(".env missing WAA_BACKEND=spidermonkey, got: %s", envText)
	}
	if !strings.Contains(envText, "AUTO_START=true") {
		t.Errorf(".env missing AUTO_START=true, got: %s", envText)
	}

	// 3. Explicit proxy on cfg overrides inheritedProxy
	cfg.Proxy = "socks5://127.0.0.1:1080"
	resolved := ResolveMagpieProxy(cfg, inheritedProxy)
	if resolved != "socks5://127.0.0.1:1080" {
		t.Errorf("resolved proxy = %q, expected 'socks5://127.0.0.1:1080'", resolved)
	}

	if err := SyncStudioEnv(dataDir, studioCfg, cfg, inheritedProxy); err != nil {
		t.Fatalf("SyncStudioEnv failed: %v", err)
	}
	envBytes2, _ := os.ReadFile(envPath)
	if !strings.Contains(string(envBytes2), "PROXY=socks5://127.0.0.1:1080") {
		t.Errorf(".env missing updated PROXY, got: %s", string(envBytes2))
	}

	// 4. Test LoadGatewayConfig hydration from existing .env when magpie-gateway.json is absent
	testDir2 := t.TempDir()
	// Pre-create .env with existing Cloud Run values
	customEnv := "PER_ACCOUNT_CONCURRENCY=3\nWARM_WORKER_LIMIT=6\nMAX_ACTIVE_WORKERS=6\nWARM_STARTUP_CONCURRENCY=3\nROUTING_STRATEGY=round-robin\n"
	_ = os.WriteFile(filepath.Join(testDir2, ".env"), []byte(customEnv), 0644)

	loadedCfg, err := LoadGatewayConfig(testDir2, "")
	if err != nil {
		t.Fatalf("LoadGatewayConfig failed: %v", err)
	}
	if loadedCfg.PerAccountConcurrency != 3 {
		t.Errorf("loadedCfg.PerAccountConcurrency = %d, expected 3", loadedCfg.PerAccountConcurrency)
	}
	if loadedCfg.WarmWorkerLimit != 6 {
		t.Errorf("loadedCfg.WarmWorkerLimit = %d, expected 6", loadedCfg.WarmWorkerLimit)
	}
	if loadedCfg.MaxActiveWorkers != 6 {
		t.Errorf("loadedCfg.MaxActiveWorkers = %d, expected 6", loadedCfg.MaxActiveWorkers)
	}
	if loadedCfg.WarmStartupConcurrency != 3 {
		t.Errorf("loadedCfg.WarmStartupConcurrency = %d, expected 3", loadedCfg.WarmStartupConcurrency)
	}
	if loadedCfg.RoutingStrategy != "round-robin" {
		t.Errorf("loadedCfg.RoutingStrategy = %q, expected 'round-robin'", loadedCfg.RoutingStrategy)
	}
}

func TestAccountBundleExportImportAndTraversalRejection(t *testing.T) {
	dataDir := t.TempDir()

	// 1. Path traversal rejection tests
	traversalCases := []string{
		"../escape",
		"foo/bar",
		"foo\\bar",
		".hidden",
		"acc\x00zero",
		"acc\nnewline",
	}
	for _, badID := range traversalCases {
		badBundle := &MagpieAccountBundle{
			Version:    1,
			ExportedAt: "2026-11-07T00:00:00Z",
			Accounts: []MagpieAccountExportItem{
				{
					AccountID:    badID,
					StorageState: map[string]any{"cookies": []any{}},
				},
			},
		}
		if _, err := ImportAccountsBundle(context.Background(), dataDir, badBundle, nil, ""); err == nil {
			t.Errorf("ImportAccountsBundle should have rejected account_id %q", badID)
		}
	}

	// 2. Reject non-object storage_state
	invalidStateBundle := &MagpieAccountBundle{
		Version:    1,
		ExportedAt: "2026-11-07T00:00:00Z",
		Accounts: []MagpieAccountExportItem{
			{
				AccountID:    "valid-id",
				StorageState: "not-an-object",
			},
		},
	}
	if _, err := ImportAccountsBundle(context.Background(), dataDir, invalidStateBundle, nil, ""); err == nil {
		t.Errorf("ImportAccountsBundle should have rejected non-object storage_state")
	}

	// 3. Valid import
	validBundle := &MagpieAccountBundle{
		Version:    1,
		ExportedAt: "2026-11-07T00:00:00Z",
		Accounts: []MagpieAccountExportItem{
			{
				AccountID:    "google-user-1",
				StorageState: map[string]any{"cookies": []any{"test-cookie-1"}},
				AccountMeta: map[string]any{
					"label":    "My Primary Account",
					"enabled":  true,
					"proxy":    "",
					"locale":   "zh-CN",
					"timezone": "Asia/Tokyo",
				},
				Catalog: map[string]any{"models": []any{"gemini-3.1-pro-preview"}},
			},
			{
				AccountID:    "google-user-2",
				StorageState: map[string]any{"cookies": []any{"test-cookie-2"}},
				// account_meta omitted to verify default synthesis
			},
		},
		WAAInterpreters: map[string]string{
			"interpreter.js": "console.log('interpreter');",
		},
	}

	importRes, err := ImportAccountsBundle(context.Background(), dataDir, validBundle, nil, "")
	if err != nil {
		t.Fatalf("ImportAccountsBundle failed: %v", err)
	}
	if importRes.ImportedCount != 2 {
		t.Errorf("ImportedCount = %d, expected 2", importRes.ImportedCount)
	}

	// Verify files written on disk
	acc1StatePath := filepath.Join(dataDir, "auth", "google-user-1", "storage-state.json")
	if fi, err := os.Stat(acc1StatePath); err != nil || fi.Size() == 0 {
		t.Errorf("storage-state.json for google-user-1 not written properly")
	}

	acc2MetaPath := filepath.Join(dataDir, "auth", "google-user-2", "account.json")
	acc2MetaBytes, err := os.ReadFile(acc2MetaPath)
	if err != nil {
		t.Fatalf("account.json for google-user-2 was not synthesized: %v", err)
	}
	var acc2Meta map[string]any
	if err := json.Unmarshal(acc2MetaBytes, &acc2Meta); err != nil || acc2Meta["label"] != "google-user-2" {
		t.Errorf("synthesized account.json invalid: %s", string(acc2MetaBytes))
	}

	waaPath := filepath.Join(dataDir, "auth", ".waa-interpreters", "interpreter.js")
	if fi, err := os.Stat(waaPath); err != nil || fi.Size() == 0 {
		t.Errorf("interpreter.js not written")
	}

	// 4. Test ExportAccountsBundle
	exported, err := ExportAccountsBundle(dataDir)
	if err != nil {
		t.Fatalf("ExportAccountsBundle failed: %v", err)
	}
	if exported.Version != 1 {
		t.Errorf("Exported bundle version = %d, expected 1", exported.Version)
	}
	if len(exported.Accounts) != 2 {
		t.Fatalf("Exported accounts count = %d, expected 2", len(exported.Accounts))
	}
	if exported.Accounts[0].AccountID != "google-user-1" || exported.Accounts[1].AccountID != "google-user-2" {
		t.Errorf("Exported accounts ordering incorrect: %+v", exported.Accounts)
	}
	if exported.WAAInterpreters["interpreter.js"] != "console.log('interpreter');" {
		t.Errorf("WAAInterpreters not preserved in export")
	}

	// 5. Test DeleteAccount
	if err := DeleteAccount(context.Background(), dataDir, "google-user-1"); err != nil {
		t.Errorf("DeleteAccount failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "auth", "google-user-1")); !os.IsNotExist(err) {
		t.Errorf("google-user-1 dir should have been removed")
	}
}

func TestToolAliasingAndSchemaNormalization(t *testing.T) {
	// 1. ToolAliasOf length constraints
	shortName := "calculate_tax"
	if alias := ToolAliasOf(shortName); alias != shortName {
		t.Errorf("ToolAliasOf(%q) = %q, expected unchanged", shortName, alias)
	}

	longName := "this_is_an_extremely_long_tool_identifier_that_definitely_exceeds_sixty_three_characters_limit_in_gemini"
	alias := ToolAliasOf(longName)
	if len(alias) > 63 {
		t.Errorf("ToolAliasOf(%q) length = %d > 63", longName, len(alias))
	}
	if !strings.HasPrefix(alias, "this_is_an_extremely_long_tool_identifier_that_definit") {
		t.Errorf("alias prefix mismatch: %s", alias)
	}
	// Check determinism
	if alias2 := ToolAliasOf(longName); alias2 != alias {
		t.Errorf("ToolAliasOf is not deterministic: %q vs %q", alias, alias2)
	}

	// UTF-8 boundary safety with multi-byte characters
	unicodeLongName := "这是一个非常非常长的工具函数名称用来测试多字节字符在边界切分时是否安全_tool_function_test_run"
	uAlias := ToolAliasOf(unicodeLongName)
	if len(uAlias) > 63 {
		t.Errorf("unicode alias length = %d > 63", len(uAlias))
	}
	if !utf8.ValidString(uAlias) {
		t.Errorf("unicode alias produced invalid UTF-8 string: %s", uAlias)
	}

	// 2. AliasGeminiTools & RestoreToolAliases
	geminiReq := map[string]any{
		"contents": []any{
			map[string]any{
				"role": "model",
				"parts": []any{
					map[string]any{
						"functionCall": map[string]any{
							"name": longName,
							"args": map[string]any{},
						},
					},
				},
			},
		},
		"tools": []any{
			map[string]any{
				"functionDeclarations": []any{
					map[string]any{
						"name": longName,
						"parameters": map[string]any{
							"type": "object",
						},
					},
				},
			},
		},
	}

	reverseGemini := AliasGeminiTools(geminiReq)
	if len(reverseGemini) != 1 {
		t.Fatalf("expected 1 reverse alias entry, got %d", len(reverseGemini))
	}

	// Verify declaration was aliased and parameter schema normalized to type: "OBJECT" and properties: {}
	decl := geminiReq["tools"].([]any)[0].(map[string]any)["functionDeclarations"].([]any)[0].(map[string]any)
	if decl["name"] != alias {
		t.Errorf("declaration name = %v, expected %s", decl["name"], alias)
	}
	params := decl["parameters"].(map[string]any)
	if params["type"] != "OBJECT" {
		t.Errorf("params type = %v, expected 'OBJECT'", params["type"])
	}
	if params["properties"] == nil {
		t.Errorf("params properties missing")
	}

	// Verify RestoreToolAliases restores original tool name
	geminiResp := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{
							"functionCall": map[string]any{
								"name": alias,
							},
						},
					},
				},
			},
		},
	}
	restored := RestoreToolAliases(geminiResp, reverseGemini)
	if !restored {
		t.Errorf("RestoreToolAliases returned false")
	}
	fcName := geminiResp["candidates"].([]any)[0].(map[string]any)["content"].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionCall"].(map[string]any)["name"]
	if fcName != longName {
		t.Errorf("restored functionCall name = %v, expected %s", fcName, longName)
	}

	// 3. AliasOpenAITools & RestoreToolAliases
	openaiReq := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"function": map[string]any{
							"name": longName,
						},
					},
				},
			},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": longName,
					"parameters": map[string]any{
						"type": "object",
					},
				},
			},
		},
	}

	reverseOpenAI := AliasOpenAITools(openaiReq)
	if len(reverseOpenAI) != 1 {
		t.Fatalf("expected 1 reverse alias entry for OpenAI, got %d", len(reverseOpenAI))
	}

	openaiResp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"function": map[string]any{
								"name": alias,
							},
						},
					},
				},
			},
		},
	}
	restoredOAI := RestoreToolAliases(openaiResp, reverseOpenAI)
	if !restoredOAI {
		t.Errorf("RestoreToolAliases for OpenAI returned false")
	}
	oaiName := openaiResp["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)["name"]
	if oaiName != longName {
		t.Errorf("restored OpenAI function name = %v, expected %s", oaiName, longName)
	}

	// 4. Gemini omitted and nil parameters schema normalization
	geminiReqOmitted := map[string]any{
		"tools": []any{
			map[string]any{
				"functionDeclarations": []any{
					map[string]any{
						"name": "omitted_params_func",
					},
				},
			},
		},
	}
	_ = AliasGeminiTools(geminiReqOmitted)
	fdOmitted := geminiReqOmitted["tools"].([]any)[0].(map[string]any)["functionDeclarations"].([]any)[0].(map[string]any)
	if p, ok := fdOmitted["parameters"].(map[string]any); !ok || p["type"] != "OBJECT" || p["properties"] == nil {
		t.Errorf("omitted Gemini parameters not normalized to OBJECT and empty properties: %v", fdOmitted["parameters"])
	}

	geminiReqNil := map[string]any{
		"tools": []any{
			map[string]any{
				"functionDeclarations": []any{
					map[string]any{
						"name":       "nil_params_func",
						"parameters": nil,
					},
				},
			},
		},
	}
	_ = AliasGeminiTools(geminiReqNil)
	fdNil := geminiReqNil["tools"].([]any)[0].(map[string]any)["functionDeclarations"].([]any)[0].(map[string]any)
	if p, ok := fdNil["parameters"].(map[string]any); !ok || p["type"] != "OBJECT" || p["properties"] == nil {
		t.Errorf("nil Gemini parameters not normalized to OBJECT and empty properties: %v", fdNil["parameters"])
	}

	// 5. OpenAI omitted and nil parameters schema normalization
	openaiReqOmitted := map[string]any{
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": "omitted_params_func",
				},
			},
		},
	}
	_ = AliasOpenAITools(openaiReqOmitted)
	fnOmitted := openaiReqOmitted["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if _, exists := fnOmitted["parameters"]; exists {
		t.Errorf("omitted OpenAI parameters should remain omitted, but found: %v", fnOmitted)
	}

	openaiReqNil := map[string]any{
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":       "nil_params_func",
					"parameters": nil,
				},
			},
		},
	}
	_ = AliasOpenAITools(openaiReqNil)
	fnNil := openaiReqNil["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if p, ok := fnNil["parameters"].(map[string]any); !ok || p["type"] != "object" || p["properties"] == nil {
		t.Errorf("nil OpenAI parameters not normalized to object and empty properties: %v", fnNil["parameters"])
	}
}

type mockUsageSink struct {
	mu      sync.Mutex
	records []coreusage.Record
}

func (m *mockUsageSink) HandleUsage(ctx context.Context, record coreusage.Record) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, record)
}

func TestForwardGeminiRequests(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer as_test_secret" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		path := r.URL.Path
		if path == "/v1beta/models/gemini-2.5-flash:generateContent" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{"text": "Hello world from Gemini"},
							},
						},
					},
				},
				"modelVersion": "gemini-2.5-flash",
				"usageMetadata": map[string]any{
					"promptTokenCount":        float64(10),
					"candidatesTokenCount":    float64(20),
					"totalTokenCount":         float64(30),
					"cachedContentTokenCount": float64(5),
				},
			})
			return
		}

		if path == "/v1beta/models/gemini-2.5-flash:streamGenerateContent" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"stream chunk 1\"}]}}],\"modelVersion\":\"gemini-2.5-flash\"}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		if path == "/v1beta/models/gemini-retry:generateContent" {
			attemptCount++
			if attemptCount == 1 {
				// Malformed response on first attempt
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"candidates": []any{
						map[string]any{
							"finishReason": "MALFORMED_RESPONSE",
							"content": map[string]any{
								"parts": []any{},
							},
						},
					},
				})
				return
			}
			// Succeeded on retry
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"candidates": []any{
					map[string]any{
						"finishReason": "STOP",
						"content": map[string]any{
							"parts": []any{
								map[string]any{"text": "Recovered output after retry"},
							},
						},
					},
				},
				"modelVersion": "gemini-retry",
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	usageSink := &mockUsageSink{}
	coreusage.RegisterNamedPlugin("test-gemini-sink", usageSink)

	testCfg := NewDefaultConfig()
	testCfg.Enabled = true
	SetTestBackendOverride(&TestBackendOverride{
		BaseURL: server.URL,
		APIKey:  "as_test_secret",
		Config:  *testCfg,
	})
	defer SetTestBackendOverride(nil)

	// 1. Non-streaming JSON test
	req := httptest.NewRequest("POST", "/v1beta/models/(MAGPIE)/gemini-2.5-flash:generateContent", strings.NewReader(`{"contents":[]}`))
	w := httptest.NewRecorder()
	ForwardGeminiRequest(w, req, "gemini-2.5-flash", "generateContent", []byte(`{"contents":[]}`), "", "user_key_1")

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ForwardGeminiRequest JSON returned status %d", resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	var respMap map[string]any
	if err := json.Unmarshal(bodyBytes, &respMap); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if respMap["modelVersion"] != "(MAGPIE)/gemini-2.5-flash" {
		t.Errorf("modelVersion = %v, expected '(MAGPIE)/gemini-2.5-flash'", respMap["modelVersion"])
	}

	// Verify cached tokens were extracted into usage
	time.Sleep(50 * time.Millisecond)
	usageSink.mu.Lock()
	if len(usageSink.records) > 0 {
		last := usageSink.records[len(usageSink.records)-1]
		if last.Detail.CachedTokens != 5 {
			t.Errorf("expected cached tokens 5, got %d", last.Detail.CachedTokens)
		}
	} else {
		t.Errorf("no usage record captured for Gemini request")
	}
	usageSink.mu.Unlock()

	// 2. Streaming SSE test
	reqStream := httptest.NewRequest("POST", "/v1beta/models/(MAGPIE)/gemini-2.5-flash:streamGenerateContent", strings.NewReader(`{"contents":[]}`))
	wStream := httptest.NewRecorder()
	ForwardGeminiRequest(wStream, reqStream, "gemini-2.5-flash", "streamGenerateContent", []byte(`{"contents":[]}`), "", "user_key_1")

	respStream := wStream.Result()
	if respStream.StatusCode != http.StatusOK {
		t.Fatalf("ForwardGeminiRequest stream returned status %d", respStream.StatusCode)
	}
	streamText := wStream.Body.String()
	if !strings.Contains(streamText, "stream chunk 1") {
		t.Errorf("stream response missing chunk: %s", streamText)
	}
	if !strings.Contains(streamText, "(MAGPIE)/gemini-2.5-flash") {
		t.Errorf("stream response missing rewritten model: %s", streamText)
	}

	// 3. Silent retry on MALFORMED_RESPONSE test
	attemptCount = 0
	reqRetry := httptest.NewRequest("POST", "/v1beta/models/(MAGPIE)/gemini-retry:generateContent", strings.NewReader(`{"contents":[]}`))
	wRetry := httptest.NewRecorder()
	ForwardGeminiRequest(wRetry, reqRetry, "gemini-retry", "generateContent", []byte(`{"contents":[]}`), "", "user_key_1")

	respRetry := wRetry.Result()
	if respRetry.StatusCode != http.StatusOK {
		t.Fatalf("ForwardGeminiRequest retry returned status %d", respRetry.StatusCode)
	}
	retryBytes, _ := io.ReadAll(respRetry.Body)
	if !strings.Contains(string(retryBytes), "Recovered output after retry") {
		t.Errorf("retry did not recover output: %s", string(retryBytes))
	}
	if attemptCount != 2 {
		t.Errorf("expected 2 attempts, got %d", attemptCount)
	}
}

func TestForwardOpenAIChatRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer as_test_secret" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(bodyBytes, &body)

		// Assert model received by upstream has MAGPIE prefix stripped
		reqModel, _ := body["model"].(string)
		if reqModel != "gemini-3.1-pro-preview" {
			t.Errorf("upstream received model %q, expected 'gemini-3.1-pro-preview'", reqModel)
		}

		isStream, _ := body["stream"].(bool)
		if isStream {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"model\":\"gemini-3.1-pro-preview\",\"choices\":[{\"delta\":{\"content\":\"Hello streaming\"}}]}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "chatcmpl-1",
			"object": "chat.completion",
			"model":  "gemini-3.1-pro-preview",
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello OpenAI",
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(15),
				"completion_tokens": float64(25),
				"total_tokens":      float64(40),
				"prompt_tokens_details": map[string]any{
					"cached_tokens": float64(8),
				},
			},
		})
	}))
	defer server.Close()

	usageSink := &mockUsageSink{}
	coreusage.RegisterNamedPlugin("test-openai-sink", usageSink)

	testCfg := NewDefaultConfig()
	testCfg.Enabled = true
	SetTestBackendOverride(&TestBackendOverride{
		BaseURL: server.URL,
		APIKey:  "as_test_secret",
		Config:  *testCfg,
	})
	defer SetTestBackendOverride(nil)

	// 1. Non-streaming JSON test
	reqJSON := []byte(`{"model":"(MAGPIE)/gemini-3.1-pro-preview","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(string(reqJSON)))
	w := httptest.NewRecorder()
	ForwardOpenAIChatRequest(w, req, "gemini-3.1-pro-preview", reqJSON, "", "test_key")

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ForwardOpenAIChatRequest returned status %d", resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	var respMap map[string]any
	_ = json.Unmarshal(bodyBytes, &respMap)
	if respMap["model"] != "(MAGPIE)/gemini-3.1-pro-preview" {
		t.Errorf("OpenAI response model = %v, expected '(MAGPIE)/gemini-3.1-pro-preview'", respMap["model"])
	}

	// Verify cached tokens were extracted into usage
	time.Sleep(50 * time.Millisecond)
	usageSink.mu.Lock()
	if len(usageSink.records) > 0 {
		last := usageSink.records[len(usageSink.records)-1]
		if last.Detail.CachedTokens != 8 {
			t.Errorf("expected cached tokens 8, got %d", last.Detail.CachedTokens)
		}
	} else {
		t.Errorf("no usage record captured for OpenAI request")
	}
	usageSink.mu.Unlock()

	// 2. Streaming SSE test
	reqStreamJSON := []byte(`{"model":"(MAGPIE)/gemini-3.1-pro-preview","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	reqStream := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(string(reqStreamJSON)))
	wStream := httptest.NewRecorder()
	ForwardOpenAIChatRequest(wStream, reqStream, "gemini-3.1-pro-preview", reqStreamJSON, "", "test_key")

	respStream := wStream.Result()
	if respStream.StatusCode != http.StatusOK {
		t.Fatalf("ForwardOpenAIChatRequest stream returned status %d", respStream.StatusCode)
	}
	streamText := wStream.Body.String()
	if !strings.Contains(streamText, "Hello streaming") {
		t.Errorf("stream response missing content: %s", streamText)
	}
	if !strings.Contains(streamText, "(MAGPIE)/gemini-3.1-pro-preview") {
		t.Errorf("stream response missing rewritten model: %s", streamText)
	}
}

func TestForwardingBypassesOutboundProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "generateContent") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modelVersion": "gemini-2.5-flash",
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{"text": "mocked gemini response"},
							},
						},
					},
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/v1/chat/completions") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "chatcmpl-test",
				"model": "gemini-3.1-pro-preview",
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"role":    "assistant",
							"content": "mocked openai response",
						},
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	testCfg := NewDefaultConfig()
	testCfg.Enabled = true
	// Set an unreachable proxy address. If loopback forwarding routes through this proxy, it will fail!
	testCfg.Proxy = "http://127.0.0.1:1"
	SetTestBackendOverride(&TestBackendOverride{
		BaseURL: server.URL,
		APIKey:  "as_test_secret",
		Config:  *testCfg,
	})
	defer SetTestBackendOverride(nil)

	// 1. ForwardGeminiRequest must succeed even with dead proxy configured
	reqGemini := httptest.NewRequest("POST", "/v1beta/models/(MAGPIE)/gemini-2.5-flash:generateContent", strings.NewReader(`{"contents":[]}`))
	wGemini := httptest.NewRecorder()
	ForwardGeminiRequest(wGemini, reqGemini, "gemini-2.5-flash", "generateContent", []byte(`{"contents":[]}`), "", "user_key_1")
	if wGemini.Code != http.StatusOK {
		t.Fatalf("ForwardGeminiRequest failed with status %d (loopback must bypass proxy): %s", wGemini.Code, wGemini.Body.String())
	}

	// 2. ForwardOpenAIChatRequest must succeed even with dead proxy configured
	reqOpenAI := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"(MAGPIE)/gemini-3.1-pro-preview","messages":[{"role":"user","content":"hi"}]}`))
	wOpenAI := httptest.NewRecorder()
	ForwardOpenAIChatRequest(wOpenAI, reqOpenAI, "gemini-3.1-pro-preview", []byte(`{"model":"(MAGPIE)/gemini-3.1-pro-preview","messages":[{"role":"user","content":"hi"}]}`), "", "user_key_1")
	if wOpenAI.Code != http.StatusOK {
		t.Fatalf("ForwardOpenAIChatRequest failed with status %d (loopback must bypass proxy): %s", wOpenAI.Code, wOpenAI.Body.String())
	}
}

func TestFetchMagpieModelsInMemoryCacheAndInvalidation(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_STUDIO_DATA_DIR", dataDir)
	modelsCachePath := filepath.Join(dataDir, "models-cache.json")
	initialModels := []MagpieModelInfo{
		{
			ID:          "gemini-2.5-flash",
			PrefixedID:  "(MAGPIE)/gemini-2.5-flash",
			DisplayName: "Gemini 2.5 Flash",
		},
	}
	cacheBytes, _ := json.Marshal(initialModels)
	_ = os.WriteFile(modelsCachePath, cacheBytes, 0644)

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	_ = SaveGatewayConfig(dataDir, cfg)

	// Invalidate to start fresh
	InvalidateModelsCache()

	// 1. Fetch models when daemon config is not present should fall back to models-cache.json
	models := FetchMagpieModels(context.Background(), "")
	if len(models) != 1 || models[0].ID != "gemini-2.5-flash" {
		t.Fatalf("expected fallback model from cache, got: %v", models)
	}

	// 2. Modify disk file; in-memory cache should still serve cached results within TTL
	_ = os.WriteFile(modelsCachePath, []byte("[]"), 0644)
	modelsCached := FetchMagpieModels(context.Background(), "")
	if len(modelsCached) != 1 || modelsCached[0].ID != "gemini-2.5-flash" {
		t.Fatalf("expected in-memory cached model, got: %v", modelsCached)
	}

	// 3. Invalidate cache; now it should reload from disk
	InvalidateModelsCache()
	modelsReloaded := FetchMagpieModels(context.Background(), "")
	if len(modelsReloaded) != 0 {
		t.Fatalf("expected empty models after cache invalidation, got: %v", modelsReloaded)
	}
}

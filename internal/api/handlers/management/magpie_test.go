package management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/magpie"
)

func TestMagpieListPlugins(t *testing.T) {
	t.Setenv("MAGPIE_BUILTIN_MENU", "1")
	t.Setenv("AI_STUDIO_DATA_DIR", t.TempDir())

	gin.SetMode(gin.TestMode)
	h := &Handler{
		cfg: &config.Config{
			Plugins: config.PluginsConfig{
				Enabled: false,
			},
		},
		configFilePath: writeTestConfigFile(t),
	}

	router := gin.New()
	router.GET("/v8/management/plugins", h.ListPlugins)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v8/management/plugins", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /v8/management/plugins status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp pluginListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal pluginListResponse failed: %v", err)
	}

	if !resp.PluginsEnabled {
		t.Fatalf("expected resp.PluginsEnabled to be true when magpie is present, got false")
	}

	var magpieEntry *pluginListEntry
	for i := range resp.Plugins {
		if resp.Plugins[i].ID == "magpie" {
			magpieEntry = &resp.Plugins[i]
			break
		}
	}

	if magpieEntry == nil {
		t.Fatalf("magpie plugin entry not found in response: %#v", resp.Plugins)
	}

	if !magpieEntry.Enabled {
		t.Errorf("magpie.Enabled = false, want true")
	}
	if !magpieEntry.EffectiveEnabled {
		t.Errorf("magpie.EffectiveEnabled = false, want true")
	}
	if len(magpieEntry.ConfigFields) == 0 {
		t.Errorf("magpie.ConfigFields is empty, want non-empty schema")
	}
	if magpieEntry.Metadata == nil || len(magpieEntry.Metadata.ConfigFields) == 0 {
		t.Errorf("magpie.Metadata.ConfigFields is empty, want non-empty schema")
	}

	expectedFields := []string{
		"auto_start",
		"routing_strategy",
		"per_account_concurrency",
		"warm_worker_limit",
		"max_active_workers",
		"warm_startup_concurrency",
		"request_timeout_ms",
		"first_event_hedge",
		"first_event_hedge_models",
		"allow_images",
		"allow_tools",
		"image_models",
		"proxy",
	}
	fieldNames := make(map[string]bool)
	for _, f := range magpieEntry.ConfigFields {
		fieldNames[f.Name] = true
	}
	for _, expected := range expectedFields {
		if !fieldNames[expected] {
			t.Errorf("expected config field %q not found in magpie.ConfigFields", expected)
		}
	}
}

func TestMagpieConfigV8Get(t *testing.T) {
	t.Setenv("MAGPIE_BUILTIN_MENU", "1")
	t.Setenv("AI_STUDIO_DATA_DIR", t.TempDir())

	gin.SetMode(gin.TestMode)
	h := &Handler{
		cfg:            &config.Config{},
		configFilePath: writeTestConfigFile(t),
	}

	router := gin.New()
	router.GET("/v8/management/config/*path", h.ConfigV8)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v8/management/config/plugins/configs/magpie", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /v8/management/config/plugins/configs/magpie status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("unmarshal config map failed: %v", err)
	}

	if priority, ok := m["priority"].(float64); !ok || priority != 0 {
		t.Errorf("m[priority] = %v, want 0", m["priority"])
	}
	if enabled, ok := m["enabled"].(bool); !ok || !enabled {
		t.Errorf("m[enabled] = %v, want true", m["enabled"])
	}
	if autoStart, ok := m["auto_start"].(bool); !ok || !autoStart {
		t.Errorf("m[auto_start] = %v, want true", m["auto_start"])
	}
	if strategy, ok := m["routing_strategy"].(string); !ok || strategy != "fill-first" {
		t.Errorf("m[routing_strategy] = %v, want 'fill-first'", m["routing_strategy"])
	}
	if _, ok := m["per_account_concurrency"]; !ok {
		t.Errorf("m[per_account_concurrency] missing")
	}
	if _, ok := m["warm_worker_limit"]; !ok {
		t.Errorf("m[warm_worker_limit] missing")
	}
	if _, ok := m["max_active_workers"]; !ok {
		t.Errorf("m[max_active_workers] missing")
	}
	if _, ok := m["request_timeout_ms"]; !ok {
		t.Errorf("m[request_timeout_ms] missing")
	}
	if _, ok := m["first_event_hedge"]; !ok {
		t.Errorf("m[first_event_hedge] missing")
	}
}

func TestMagpieToggleEnabledAndReflectInPlugins(t *testing.T) {
	t.Setenv("MAGPIE_BUILTIN_MENU", "1")
	t.Setenv("AI_STUDIO_DATA_DIR", t.TempDir())

	gin.SetMode(gin.TestMode)
	h := &Handler{
		cfg: &config.Config{
			Plugins: config.PluginsConfig{
				Enabled: false,
			},
		},
		configFilePath: writeTestConfigFile(t),
	}

	router := gin.New()
	router.GET("/v8/management/plugins", h.ListPlugins)
	router.GET("/v8/management/config/*path", h.ConfigV8)
	router.PUT("/v8/management/config/*path", h.ConfigV8)
	router.PATCH("/v8/management/config/*path", h.ConfigV8)

	// 1. Initial check - enabled should be true
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v8/management/config/plugins/configs/magpie/enabled", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET enabled status = %d; body = %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "true" {
		t.Fatalf("expected enabled == true, got %s", rec.Body.String())
	}

	// Verify plugin listing shows enabled: true, effective_enabled: true
	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/v8/management/plugins", nil)
	router.ServeHTTP(recList, reqList)
	var listResp pluginListResponse
	if err := json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	var magpiePlugin *pluginListEntry
	for i := range listResp.Plugins {
		if listResp.Plugins[i].ID == "magpie" {
			magpiePlugin = &listResp.Plugins[i]
			break
		}
	}
	if magpiePlugin == nil || !magpiePlugin.Enabled || !magpiePlugin.EffectiveEnabled {
		t.Fatalf("initial magpie plugin state wrong: %#v", magpiePlugin)
	}

	// 2. PUT /v8/management/config/plugins/configs/magpie/enabled with false
	recPut := httptest.NewRecorder()
	reqPut := httptest.NewRequest(http.MethodPut, "/v8/management/config/plugins/configs/magpie/enabled", bytes.NewBufferString("false"))
	reqPut.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusOK {
		t.Fatalf("PUT enabled status = %d; body = %s", recPut.Code, recPut.Body.String())
	}

	// 3. Verify GET enabled returns false
	recGet2 := httptest.NewRecorder()
	reqGet2 := httptest.NewRequest(http.MethodGet, "/v8/management/config/plugins/configs/magpie/enabled", nil)
	router.ServeHTTP(recGet2, reqGet2)
	if recGet2.Code != http.StatusOK {
		t.Fatalf("GET enabled after disable status = %d", recGet2.Code)
	}
	if strings.TrimSpace(recGet2.Body.String()) != "false" {
		t.Fatalf("expected enabled == false, got %s", recGet2.Body.String())
	}

	// 4. Verify GET /v8/management/plugins reflects enabled: false, effective_enabled: false
	recList2 := httptest.NewRecorder()
	reqList2 := httptest.NewRequest(http.MethodGet, "/v8/management/plugins", nil)
	router.ServeHTTP(recList2, reqList2)
	var listResp2 pluginListResponse
	if err := json.Unmarshal(recList2.Body.Bytes(), &listResp2); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	var magpiePlugin2 *pluginListEntry
	for i := range listResp2.Plugins {
		if listResp2.Plugins[i].ID == "magpie" {
			magpiePlugin2 = &listResp2.Plugins[i]
			break
		}
	}
	if magpiePlugin2 == nil {
		t.Fatalf("magpie plugin missing after disable")
	}
	if magpiePlugin2.Enabled {
		t.Errorf("magpie.Enabled after disable = true, want false")
	}
	if magpiePlugin2.EffectiveEnabled {
		t.Errorf("magpie.EffectiveEnabled after disable = true, want false")
	}

	// 5. PATCH /v8/management/config/plugins/configs/magpie/enabled with {"enabled": true}
	recPatch := httptest.NewRecorder()
	reqPatch := httptest.NewRequest(http.MethodPatch, "/v8/management/config/plugins/configs/magpie/enabled", bytes.NewBufferString(`{"enabled": true}`))
	reqPatch.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recPatch, reqPatch)
	if recPatch.Code != http.StatusOK {
		t.Fatalf("PATCH enabled status = %d; body = %s", recPatch.Code, recPatch.Body.String())
	}

	// 6. Verify GET /v8/management/plugins reflects enabled: true, effective_enabled: true again
	recList3 := httptest.NewRecorder()
	reqList3 := httptest.NewRequest(http.MethodGet, "/v8/management/plugins", nil)
	router.ServeHTTP(recList3, reqList3)
	var listResp3 pluginListResponse
	if err := json.Unmarshal(recList3.Body.Bytes(), &listResp3); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	var magpiePlugin3 *pluginListEntry
	for i := range listResp3.Plugins {
		if listResp3.Plugins[i].ID == "magpie" {
			magpiePlugin3 = &listResp3.Plugins[i]
			break
		}
	}
	if magpiePlugin3 == nil || !magpiePlugin3.Enabled || !magpiePlugin3.EffectiveEnabled {
		t.Fatalf("magpie plugin not re-enabled: %#v", magpiePlugin3)
	}
}

func TestMagpieConfigV8PutAndPatchFullConfig(t *testing.T) {
	t.Setenv("MAGPIE_BUILTIN_MENU", "1")
	t.Setenv("AI_STUDIO_DATA_DIR", t.TempDir())

	gin.SetMode(gin.TestMode)
	h := &Handler{
		cfg:            &config.Config{},
		configFilePath: writeTestConfigFile(t),
	}

	router := gin.New()
	router.GET("/v8/management/config/*path", h.ConfigV8)
	router.PUT("/v8/management/config/*path", h.ConfigV8)

	putBody := `{"routing_strategy": "round-robin", "per_account_concurrency": 4, "proxy": "http://127.0.0.1:8888"}`
	recPut := httptest.NewRecorder()
	reqPut := httptest.NewRequest(http.MethodPut, "/v8/management/config/plugins/configs/magpie", bytes.NewBufferString(putBody))
	reqPut.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusOK {
		t.Fatalf("PUT config status = %d; body = %s", recPut.Code, recPut.Body.String())
	}

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/v8/management/config/plugins/configs/magpie", nil)
	router.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("GET config status = %d; body = %s", recGet.Code, recGet.Body.String())
	}

	var m map[string]any
	if err := json.Unmarshal(recGet.Body.Bytes(), &m); err != nil {
		t.Fatalf("unmarshal config failed: %v", err)
	}
	if m["routing_strategy"] != "round-robin" {
		t.Errorf("routing_strategy = %v, want 'round-robin'", m["routing_strategy"])
	}
	if conc, ok := m["per_account_concurrency"].(float64); !ok || conc != 4 {
		t.Errorf("per_account_concurrency = %v, want 4", m["per_account_concurrency"])
	}
	if m["proxy"] != "http://127.0.0.1:8888" {
		t.Errorf("proxy = %v, want 'http://127.0.0.1:8888'", m["proxy"])
	}
	if priority, ok := m["priority"].(float64); !ok || priority != 0 {
		t.Errorf("priority = %v, want 0", m["priority"])
	}
}

func TestMagpieDirectPluginHandlers(t *testing.T) {
	t.Setenv("MAGPIE_BUILTIN_MENU", "1")
	t.Setenv("AI_STUDIO_DATA_DIR", t.TempDir())

	gin.SetMode(gin.TestMode)
	h := &Handler{
		cfg:            &config.Config{},
		configFilePath: writeTestConfigFile(t),
	}

	router := gin.New()
	router.GET("/v0/management/plugins/:id/config", h.GetPluginConfig)
	router.PATCH("/v0/management/plugins/:id/enabled", h.PatchPluginEnabled)
	router.PUT("/v0/management/plugins/:id/config", h.PutPluginConfig)
	router.PATCH("/v0/management/plugins/:id/config", h.PatchPluginConfig)
	router.DELETE("/v0/management/plugins/:id", h.DeletePlugin)

	// DELETE builtin magpie should fail with 400
	recDel := httptest.NewRecorder()
	reqDel := httptest.NewRequest(http.MethodDelete, "/v0/management/plugins/magpie", nil)
	router.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusBadRequest {
		t.Errorf("DELETE /v0/management/plugins/magpie status = %d, want %d", recDel.Code, http.StatusBadRequest)
	}

	// GET config
	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/v0/management/plugins/magpie/config", nil)
	router.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Errorf("GET /v0/management/plugins/magpie/config status = %d, want %d", recGet.Code, http.StatusOK)
	}

	// PATCH enabled to false
	recPatch := httptest.NewRecorder()
	reqPatch := httptest.NewRequest(http.MethodPatch, "/v0/management/plugins/magpie/enabled", bytes.NewBufferString(`{"enabled": false}`))
	reqPatch.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recPatch, reqPatch)
	if recPatch.Code != http.StatusOK {
		t.Errorf("PATCH /v0/management/plugins/magpie/enabled status = %d, want %d", recPatch.Code, http.StatusOK)
	}

	// Verify gateway config was written as enabled = false
	dataDir := magpie.ResolveDataDir("")
	loadedCfg, errLoad := magpie.LoadGatewayConfig(dataDir, "")
	if errLoad != nil {
		t.Fatalf("LoadGatewayConfig failed: %v", errLoad)
	}
	if loadedCfg.Enabled {
		t.Errorf("loadedCfg.Enabled = true, want false")
	}
}

func TestMagpieLoginAndReloginHandlers(t *testing.T) {
	tmpDataDir := t.TempDir()
	t.Setenv("AI_STUDIO_DATA_DIR", tmpDataDir)
	t.Setenv("MAGPIE_BUILTIN_MENU", "1")

	// Set up mock AIStudio2API daemon server
	mockDaemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/auth/login" && r.Method == http.MethodPost:
			http.SetCookie(w, &http.Cookie{Name: "aistudio_admin", Value: "mock-session-cookie"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.URL.Path == "/api/status" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"running":true,"ready":true,"state":"RUNNING","version":"0.2.19-magpie.1"}`))
		case r.URL.Path == "/api/accounts" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true,"action":"login_started"}`))
		case r.URL.Path == "/api/accounts/acc-test/login" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true,"account_id":"acc-test"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockDaemon.Close()

	parts := strings.Split(mockDaemon.Listener.Addr().String(), ":")
	port, errPort := strconv.Atoi(parts[len(parts)-1])
	if errPort != nil {
		t.Fatalf("failed to parse mock server port: %v", errPort)
	}

	// Write config.json pointing to the mock server port
	cfgJSON := fmt.Sprintf(`{
		"version": 1,
		"port": %d,
		"apiKey": "as_testkey123",
		"adminUsername": "magpie",
		"adminPassword": "testpassword"
	}`, port)
	if errWrite := os.WriteFile(filepath.Join(tmpDataDir, "config.json"), []byte(cfgJSON), 0644); errWrite != nil {
		t.Fatalf("failed to write config.json: %v", errWrite)
	}

	gin.SetMode(gin.TestMode)
	h := &Handler{
		cfg:            &config.Config{},
		configFilePath: writeTestConfigFile(t),
	}

	router := gin.New()
	router.POST("/v8/management/magpie/accounts/login", h.LoginMagpieAccount)
	router.POST("/v8/management/magpie/accounts/:accountId/relogin", h.ReloginMagpieAccount)

	// 1. Test POST /v8/management/magpie/accounts/login
	t.Run("login_new_account", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v8/management/magpie/accounts/login", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("POST /v8/management/magpie/accounts/login status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if action, ok := resp["action"].(string); !ok || action != "login_started" {
			t.Errorf("resp[action] = %v, want 'login_started'", resp["action"])
		}
	})

	// 2. Test POST /v8/management/magpie/accounts/acc-test/relogin
	t.Run("relogin_existing_account", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v8/management/magpie/accounts/acc-test/relogin", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("POST /v8/management/magpie/accounts/acc-test/relogin status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if accID, ok := resp["account_id"].(string); !ok || accID != "acc-test" {
			t.Errorf("resp[account_id] = %v, want 'acc-test'", resp["account_id"])
		}
	})

	// 3. Test POST /v8/management/magpie/accounts/%20/relogin with whitespace accountId
	t.Run("relogin_empty_account_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v8/management/magpie/accounts/%20/relogin", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("POST /v8/management/magpie/accounts/%%20/relogin status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

package magpie

import (
	"strings"
	"testing"
)

func TestLogoDataURI(t *testing.T) {
	if len(LogoDataURI) == 0 {
		t.Fatal("LogoDataURI must not be empty")
	}
	expectedPrefix := "data:image/svg+xml;base64,"
	if !strings.HasPrefix(LogoDataURI, expectedPrefix) {
		t.Fatalf("LogoDataURI must start with %q, got: %s", expectedPrefix, LogoDataURI[:30])
	}
	if len(LogoDataURI) < 100 {
		t.Fatalf("LogoDataURI too short (%d bytes)", len(LogoDataURI))
	}
}

func TestQuotaPageHTML(t *testing.T) {
	htmlBytes := QuotaPageHTML()
	if len(htmlBytes) == 0 {
		t.Fatal("QuotaPageHTML returned empty byte slice")
	}
	html := string(htmlBytes)

	// Validate standard HTML5 structure
	requiredTags := []string{
		"<!DOCTYPE html>",
		"<html",
		"</html>",
		"<head>",
		"</head>",
		"<body>",
		"</body>",
		"<script>",
		"</script>",
		"<style>",
		"</style>",
	}
	for _, tag := range requiredTags {
		if !strings.Contains(html, tag) {
			t.Errorf("QuotaPageHTML missing required tag %q", tag)
		}
	}

	// Validate theme integration tokens and observer
	themeTokens := []string{
		"data-theme",
		"MutationObserver",
		"cli-proxy-theme",
		"prefers-color-scheme",
		"--bg",
		"--card",
		"--border",
		"--text",
	}
	for _, token := range themeTokens {
		if !strings.Contains(html, token) {
			t.Errorf("QuotaPageHTML missing theme token %q", token)
		}
	}

	// Validate expected management v8 API endpoints
	requiredEndpoints := []string{
		"/v8/management/magpie/quota",
		"/v8/management/magpie/status",
		"/v8/management/magpie/start",
		"/v8/management/magpie/restart",
		"/v8/management/magpie/accounts/export",
		"/v8/management/magpie/accounts/import",
		"/v8/management/magpie/accounts/",
		"/toggle",
	}
	for _, ep := range requiredEndpoints {
		if !strings.Contains(html, ep) {
			t.Errorf("QuotaPageHTML missing required endpoint %q", ep)
		}
	}

	// Validate Cloud Run headless features and local export script guidance
	headlessTokens := []string{
		"scripts/export-magpie-bundle.mjs",
		"magpie-accounts-bundle",
		"cooldown-windows-list",
		"data-countdown-until",
		"/v0/resource/plugins/magpie/settings",
	}
	for _, tok := range headlessTokens {
		if !strings.Contains(html, tok) {
			t.Errorf("QuotaPageHTML missing headless feature token %q", tok)
		}
	}

	// Validate management auth deobfuscation and fallback tokens
	authTokens := []string{
		"resolveStoredManagementKey",
		"deobfuscateCpaStorage",
		"cli-proxy-api-webui::secure-storage",
		"magpie-mgmt-key",
	}
	for _, tok := range authTokens {
		if !strings.Contains(html, tok) {
			t.Errorf("QuotaPageHTML missing auth token %q", tok)
		}
	}

	// Ensure zero dead desktop-only endpoints exist
	forbiddenTokens := []string{
		"login_magpie_account",
		"relogin_magpie_account",
		"install_magpie_backend",
		"/v0/management/",
	}
	for _, tok := range forbiddenTokens {
		if strings.Contains(html, tok) {
			t.Errorf("QuotaPageHTML must not contain forbidden desktop token %q", tok)
		}
	}
}

func TestSettingsPageHTML(t *testing.T) {
	htmlBytes := SettingsPageHTML()
	if len(htmlBytes) == 0 {
		t.Fatal("SettingsPageHTML returned empty byte slice")
	}
	html := string(htmlBytes)

	// Validate standard HTML5 structure
	requiredTags := []string{
		"<!DOCTYPE html>",
		"<html",
		"</html>",
		"<head>",
		"</head>",
		"<body>",
		"</body>",
		"<script>",
		"</script>",
		"<style>",
		"</style>",
	}
	for _, tag := range requiredTags {
		if !strings.Contains(html, tag) {
			t.Errorf("SettingsPageHTML missing required tag %q", tag)
		}
	}

	// Validate theme integration tokens and observer
	themeTokens := []string{
		"data-theme",
		"MutationObserver",
		"cli-proxy-theme",
		"prefers-color-scheme",
		"--bg",
		"--card",
		"--border",
		"--text",
	}
	for _, token := range themeTokens {
		if !strings.Contains(html, token) {
			t.Errorf("SettingsPageHTML missing theme token %q", token)
		}
	}

	// Validate expected management v8 API endpoints
	requiredEndpoints := []string{
		"/v8/management/magpie/status",
		"/v8/management/magpie/config",
		"/v8/management/magpie/start",
		"/v8/management/magpie/stop",
		"/v8/management/magpie/restart",
		"/v0/resource/plugins/magpie/quota",
	}
	for _, ep := range requiredEndpoints {
		if !strings.Contains(html, ep) {
			t.Errorf("SettingsPageHTML missing required endpoint %q", ep)
		}
	}

	// Validate all configuration form fields
	requiredFormFields := []string{
		`name="enabled"`,
		`name="auto_start"`,
		`name="allow_tools"`,
		`name="allow_images"`,
		`name="image_models"`,
		`name="routing_strategy"`,
		`name="per_account_concurrency"`,
		`name="warm_worker_limit"`,
		`name="max_active_workers"`,
		`name="warm_startup_concurrency"`,
		`name="first_event_hedge"`,
		`name="first_event_hedge_models"`,
		`name="request_timeout_ms"`,
		`name="proxy"`,
		`name="internal_port"`,
		`name="data_dir"`,
	}
	for _, field := range requiredFormFields {
		if !strings.Contains(html, field) {
			t.Errorf("SettingsPageHTML missing required form field %q", field)
		}
	}

	// Validate unified gateway guidance and pre-installed binary notice
	unifiedTokens := []string{
		"(MAGPIE)/",
		"/app/magpie/aistudio2api",
		"v0.2.19-magpie.1",
	}
	for _, tok := range unifiedTokens {
		if !strings.Contains(html, tok) {
			t.Errorf("SettingsPageHTML missing unified gateway token %q", tok)
		}
	}

	// Validate management auth deobfuscation and fallback tokens
	authTokens := []string{
		"resolveStoredManagementKey",
		"deobfuscateCpaStorage",
		"cli-proxy-api-webui::secure-storage",
		"magpie-mgmt-key",
	}
	for _, tok := range authTokens {
		if !strings.Contains(html, tok) {
			t.Errorf("SettingsPageHTML missing auth token %q", tok)
		}
	}

	// Ensure zero dead desktop-only endpoints exist
	forbiddenTokens := []string{
		"login_magpie_account",
		"relogin_magpie_account",
		"install_magpie_backend",
		"/v0/management/",
	}
	for _, tok := range forbiddenTokens {
		if strings.Contains(html, tok) {
			t.Errorf("SettingsPageHTML must not contain forbidden desktop token %q", tok)
		}
	}
}

func TestNoExternalCDNDependencies(t *testing.T) {
	pages := map[string][]byte{
		"QuotaPage":    QuotaPageHTML(),
		"SettingsPage": SettingsPageHTML(),
	}

	externalCDNs := []string{
		"cdnjs.cloudflare.com",
		"cdn.jsdelivr.net",
		"unpkg.com",
		"fonts.googleapis.com",
		"fonts.gstatic.com",
	}

	for pageName, bytes := range pages {
		html := string(bytes)
		for _, cdn := range externalCDNs {
			if strings.Contains(html, cdn) {
				t.Errorf("%s contains external CDN dependency %q (violating offline/strict CSP rule)", pageName, cdn)
			}
		}
	}
}

func TestAuthDeobfuscationTokens(t *testing.T) {
	pages := map[string][]byte{
		"QuotaPage":    QuotaPageHTML(),
		"SettingsPage": SettingsPageHTML(),
	}

	requiredTokens := []string{
		"resolveStoredManagementKey",
		"deobfuscateCpaStorage",
		"cli-proxy-api-webui::secure-storage",
		"magpie-mgmt-key",
	}

	for pageName, bytes := range pages {
		html := string(bytes)
		for _, tok := range requiredTokens {
			if !strings.Contains(html, tok) {
				t.Errorf("%s missing required auth deobfuscation token %q", pageName, tok)
			}
		}
	}
}

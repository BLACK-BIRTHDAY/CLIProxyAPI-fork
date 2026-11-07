package magpie

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MagpieConfig holds configuration for the managed MAGPIE (AIStudio2API) backend.
// JSON tags match the snake_case format used in antigravity tools.
type MagpieConfig struct {
	Enabled                bool   `json:"enabled"`
	AutoStart              bool   `json:"auto_start"`
	InternalPort           int    `json:"internal_port"`
	DataDir                string `json:"data_dir"`
	Proxy                  string `json:"proxy"`
	AllowImages            bool   `json:"allow_images"`
	AllowTools             bool   `json:"allow_tools"`
	ImageModels            bool   `json:"image_models"`
	RoutingStrategy        string `json:"routing_strategy"`
	PerAccountConcurrency  int    `json:"per_account_concurrency"`
	WarmWorkerLimit        int    `json:"warm_worker_limit"`
	MaxActiveWorkers       int    `json:"max_active_workers"`
	WarmStartupConcurrency int    `json:"warm_startup_concurrency"`
	RequestTimeoutMs       int64  `json:"request_timeout_ms"`
	FirstEventHedge        string `json:"first_event_hedge"`
	FirstEventHedgeModels  string `json:"first_event_hedge_models"`
}

// NewDefaultConfig returns a MagpieConfig populated with recommended defaults.
func NewDefaultConfig() *MagpieConfig {
	return &MagpieConfig{
		Enabled:                true,
		AutoStart:              true,
		InternalPort:           2048,
		AllowImages:            true,
		AllowTools:             true,
		ImageModels:            true,
		RoutingStrategy:        "fill-first",
		PerAccountConcurrency:  1,
		WarmWorkerLimit:        1,
		MaxActiveWorkers:       1,
		WarmStartupConcurrency: 1,
		RequestTimeoutMs:       1800000, // 30 minutes
		FirstEventHedge:        "off",
		FirstEventHedgeModels:  "gemini-4-argon",
	}
}

// NormalizeWorkerLimits enforces boundary constraints on worker concurrency limits.
// It clamps per_account_concurrency >= 1, warm_worker_limit >= 1,
// max_active_workers >= warm_worker_limit, and 1 <= warm_startup_concurrency <= max_active_workers.
func (c *MagpieConfig) NormalizeWorkerLimits() (warmLimit, maxActive, warmStartup int) {
	if c.PerAccountConcurrency < 1 {
		c.PerAccountConcurrency = 1
	}
	if c.WarmWorkerLimit < 1 {
		c.WarmWorkerLimit = 1
	}
	if c.MaxActiveWorkers < c.WarmWorkerLimit {
		c.MaxActiveWorkers = c.WarmWorkerLimit
	}
	if c.WarmStartupConcurrency < 1 {
		c.WarmStartupConcurrency = 1
	}
	if c.WarmStartupConcurrency > c.MaxActiveWorkers {
		c.WarmStartupConcurrency = c.MaxActiveWorkers
	}
	return c.WarmWorkerLimit, c.MaxActiveWorkers, c.WarmStartupConcurrency
}

// ShouldExposeBuiltinMenu reports whether the built-in MAGPIE plugin entry should be listed in ListPlugins.
// It is active when pre-installed in the container (/app/magpie/aistudio2api) or explicitly enabled via MAGPIE_BUILTIN_MENU=1.
func ShouldExposeBuiltinMenu() bool {
	if PreinstalledBinaryPath() != "" {
		return true
	}
	if os.Getenv("MAGPIE_BUILTIN_MENU") == "1" {
		return true
	}
	return false
}

// IsInstalledOrConfigured reports whether the managed MAGPIE backend is pre-installed,
// running, configured on disk, or requested via environment variables.
func IsInstalledOrConfigured() bool {
	if PreinstalledBinaryPath() != "" {
		return true
	}
	if os.Getenv("MAGPIE_BUILTIN_MENU") == "1" || os.Getenv("K_SERVICE") != "" {
		return true
	}
	dataDir := ResolveDataDir("")
	if ResolveExecutablePath(dataDir) != "" {
		return true
	}
	if HasImportedAccounts(dataDir) {
		return true
	}
	if _, err := os.Stat(filepath.Join(dataDir, "magpie-gateway.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dataDir, "config.json")); err == nil {
		return true
	}
	return false
}

// ResolveDataDir resolves the root data directory for the managed MAGPIE backend.
// It checks explicitDir, AI_STUDIO_DATA_DIR, CPA_DATA_DIR, WRITABLE_PATH,
// /data, /root/.antigravity_tools, and falls back to ~/.local/share/magpie-ai-studio.
func ResolveDataDir(explicitDir string) string {
	if strings.TrimSpace(explicitDir) != "" {
		return strings.TrimSpace(explicitDir)
	}
	if v := os.Getenv("AI_STUDIO_DATA_DIR"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if v := os.Getenv("CPA_DATA_DIR"); strings.TrimSpace(v) != "" {
		return filepath.Join(strings.TrimSpace(v), "magpie")
	}
	if v := os.Getenv("WRITABLE_PATH"); strings.TrimSpace(v) != "" {
		return filepath.Join(strings.TrimSpace(v), "magpie")
	}
	if fi, err := os.Stat("/data"); err == nil && fi.IsDir() {
		return "/data/magpie"
	}
	if fi, err := os.Stat("/root/.antigravity_tools"); err == nil && fi.IsDir() {
		return "/root/.antigravity_tools/magpie"
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".local", "share", "magpie-ai-studio")
	}
	return filepath.Join(".", ".local", "share", "magpie-ai-studio")
}

// ResolveRuntimeTempDir returns the runtime directory used for logs and PID files.
// It uses /tmp/magpie on Unix or filepath.Join(os.TempDir(), "magpie") on other platforms
// so that FUSE mounts are never touched by process stdout logs or cross-instance stale PIDs.
func ResolveRuntimeTempDir() string {
	var dir string
	if fi, err := os.Stat("/tmp"); err == nil && fi.IsDir() {
		dir = "/tmp/magpie"
	} else {
		dir = filepath.Join(os.TempDir(), "magpie")
	}
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// ResolveMagpieProxy returns the effective upstream proxy URL for MAGPIE,
// falling back to inheritedProxy or AI_STUDIO_PROXY environment variable.
func ResolveMagpieProxy(cfg *MagpieConfig, inheritedProxy string) string {
	if cfg != nil && strings.TrimSpace(cfg.Proxy) != "" {
		return strings.TrimSpace(cfg.Proxy)
	}
	if strings.TrimSpace(inheritedProxy) != "" {
		return strings.TrimSpace(inheritedProxy)
	}
	if v := os.Getenv("AI_STUDIO_PROXY"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return ""
}

// StudioConfigFile represents the internal configuration stored in <data_dir>/config.json.
type StudioConfigFile struct {
	Version       int    `json:"version"`
	Port          int    `json:"port"`
	APIKey        string `json:"apiKey"`
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword"`
	Proxy         string `json:"proxy,omitempty"`
}

// EnsureStudioConfig ensures <data_dir>/config.json exists, synchronizes port/proxy,
// and invokes SyncStudioEnv to keep <data_dir>/.env in sync.
func EnsureStudioConfig(dataDir string, cfg *MagpieConfig, inheritedProxy string) (*StudioConfigFile, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}
	authDir := filepath.Join(dataDir, "auth")
	if err := os.MkdirAll(authDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create auth dir: %w", err)
	}

	configPath := filepath.Join(dataDir, "config.json")
	var studioCfg StudioConfigFile
	exists := false
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &studioCfg); err == nil && studioCfg.Port > 0 && studioCfg.APIKey != "" {
			exists = true
		}
	}

	effectiveProxy := ResolveMagpieProxy(cfg, inheritedProxy)
	if !exists {
		port := 2048
		if cfg != nil && cfg.InternalPort > 0 {
			port = cfg.InternalPort
		}
		var apiBytes [32]byte
		var pwdBytes [32]byte
		_, _ = rand.Read(apiBytes[:])
		_, _ = rand.Read(pwdBytes[:])

		studioCfg = StudioConfigFile{
			Version:       1,
			Port:          port,
			APIKey:        "as_" + hex.EncodeToString(apiBytes[:]),
			AdminUsername: "magpie",
			AdminPassword: hex.EncodeToString(pwdBytes[:]),
			Proxy:         effectiveProxy,
		}
		data, err := json.MarshalIndent(studioCfg, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to serialize config.json: %w", err)
		}
		tmpPath := filepath.Join(dataDir, fmt.Sprintf(".config.json.%x.tmp", apiBytes[:4]))
		if err := os.WriteFile(tmpPath, append(data, '\n'), 0600); err != nil {
			return nil, fmt.Errorf("failed to write config.json: %w", err)
		}
		if err := os.Rename(tmpPath, configPath); err != nil {
			return nil, fmt.Errorf("failed to rename config.json: %w", err)
		}
	} else {
		dirty := false
		if cfg != nil && cfg.InternalPort > 0 && studioCfg.Port != cfg.InternalPort {
			studioCfg.Port = cfg.InternalPort
			dirty = true
		}
		if studioCfg.Proxy != effectiveProxy {
			studioCfg.Proxy = effectiveProxy
			dirty = true
		}
		if dirty {
			if data, err := json.MarshalIndent(studioCfg, "", "  "); err == nil {
				_ = os.WriteFile(configPath, append(data, '\n'), 0600)
			}
		}
	}

	if err := SyncStudioEnv(dataDir, &studioCfg, cfg, inheritedProxy); err != nil {
		return nil, err
	}
	return &studioCfg, nil
}

// EnvKeys defines the exact settings AIStudio2API reads from .env in canonical order.
var EnvKeys = []string{
	"AISTUDIO_AUTH_STATES",
	"LISTEN_ADDR",
	"PREVIEW_LISTEN_ADDR",
	"PROXY_API_KEY",
	"API_KEY_ACCOUNTS",
	"PROXY",
	"INIT_TIMEOUT",
	"REQUEST_TIMEOUT",
	"WARM_WORKER_LIMIT",
	"MAX_ACTIVE_WORKERS",
	"WARM_STARTUP_CONCURRENCY",
	"PER_ACCOUNT_CONCURRENCY",
	"ROUTING_STRATEGY",
	"UPSTREAM_CHANNELS",
	"MODEL_MAPPING",
	"MODEL_CHANNELS",
	"TEMPORARY_CHAT",
	"WAA_BACKEND",
	"MEDIA_OUTPUT",
	"MEDIA_BASE_URL",
	"AUTO_START",
	"REQUEST_BODY_LOG",
	"LOG_FILE",
	"LOG_MAX_TOTAL_MB",
	"ADMIN_AUTH_ENABLED",
	"ADMIN_USERNAME",
	"ADMIN_PASSWORD",
	"BUILD_NATIVE_NONSTREAM",
	"NONSTREAM_KEEPALIVE",
	"FAKE_STREAM",
	"YOUTUBE_LINKS",
	"SEARCH_MODEL_ALIASES",
	"GOOGLE_SEARCH",
	"FIRST_EVENT_HEDGE",
	"FIRST_EVENT_HEDGE_MODELS",
	"CONTINUATION_REWRITE",
	"CONTINUATION_PROMPT",
}

// SettingDefaults defines the baseline defaults for keys in .env when missing.
var SettingDefaults = map[string]string{
	"REQUEST_BODY_LOG":         "false",
	"LOG_FILE":                 "false",
	"TEMPORARY_CHAT":           "true",
	"PER_ACCOUNT_CONCURRENCY":  "1",
	"WARM_WORKER_LIMIT":        "1",
	"MAX_ACTIVE_WORKERS":       "1",
	"WARM_STARTUP_CONCURRENCY": "1",
	"ROUTING_STRATEGY":         "fill-first",
	"REQUEST_TIMEOUT":          "30m",
	"INIT_TIMEOUT":             "2m",
	"NONSTREAM_KEEPALIVE":      "false",
	"FIRST_EVENT_HEDGE":        "off",
	"SEARCH_MODEL_ALIASES":     "false",
}

// FormatEnvValue formats a string value for safe writing to .env.
func FormatEnvValue(value string) string {
	if strings.ContainsAny(value, " \t\r\n#\"'") {
		b, err := json.Marshal(value)
		if err == nil {
			return string(b)
		}
	}
	return value
}

// ParseEnvValue parses a raw value string read from .env.
func ParseEnvValue(raw string, whereLoc string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if strings.HasPrefix(raw, "'") {
		if len(raw) < 2 || !strings.HasSuffix(raw, "'") {
			return "", fmt.Errorf("%s: unterminated single quote", whereLoc)
		}
		return raw[1 : len(raw)-1], nil
	}
	if strings.HasPrefix(raw, "\"") {
		var s string
		if err := json.Unmarshal([]byte(raw), &s); err != nil {
			return "", fmt.Errorf("%s: invalid double-quoted value", whereLoc)
		}
		return s, nil
	}
	if idx := strings.Index(raw, " #"); idx != -1 {
		return strings.TrimSpace(raw[:idx]), nil
	}
	return strings.TrimSpace(raw), nil
}

// ReadStudioEnv reads .env key-value pairs matching EnvKeys from dataDir.
func ReadStudioEnv(dataDir string) map[string]string {
	envPath := filepath.Join(dataDir, ".env")
	content, err := os.ReadFile(envPath)
	if err != nil {
		return make(map[string]string)
	}
	lines := strings.Split(string(content), "\n")
	values := make(map[string]string)
	envKeySet := make(map[string]bool, len(EnvKeys))
	for _, k := range EnvKeys {
		envKeySet[k] = true
	}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq == -1 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		if envKeySet[k] {
			whereLoc := fmt.Sprintf("%s:%d", envPath, i+1)
			rawVal := strings.TrimSpace(line[eq+1:])
			val, err := ParseEnvValue(rawVal, whereLoc)
			if err != nil {
				val = rawVal
			}
			values[k] = val
		}
	}
	return values
}

// SyncStudioEnv synchronizes <data_dir>/.env with canonical settings and MagpieConfig.
func SyncStudioEnv(dataDir string, studioCfg *StudioConfigFile, cfg *MagpieConfig, inheritedProxy string) error {
	if cfg == nil {
		cfg = NewDefaultConfig()
	}
	values := ReadStudioEnv(dataDir)

	// Apply setting defaults only for missing keys
	for k, v := range SettingDefaults {
		if _, ok := values[k]; !ok {
			values[k] = v
		}
	}

	// Plugin-owned settings
	authPath := filepath.Join(dataDir, "auth")
	values["LISTEN_ADDR"] = fmt.Sprintf("127.0.0.1:%d", studioCfg.Port)
	values["AISTUDIO_AUTH_STATES"] = authPath
	values["PROXY_API_KEY"] = studioCfg.APIKey
	values["ADMIN_AUTH_ENABLED"] = "true"
	values["ADMIN_USERNAME"] = studioCfg.AdminUsername
	values["ADMIN_PASSWORD"] = studioCfg.AdminPassword
	values["AUTO_START"] = "true"
	values["WAA_BACKEND"] = "spidermonkey"
	values["UPSTREAM_CHANNELS"] = "playground"
	values["BUILD_NATIVE_NONSTREAM"] = "false"
	values["SEARCH_MODEL_ALIASES"] = "false"

	defaultPreviewPort := studioCfg.Port + 2
	if studioCfg.Port == 2050 {
		defaultPreviewPort = 2051
	}
	currentPreviewPort := 0
	if addr, ok := values["PREVIEW_LISTEN_ADDR"]; ok {
		trimmed := strings.TrimSpace(addr)
		if idx := strings.LastIndex(trimmed, ":"); idx != -1 {
			if p, err := strconv.Atoi(trimmed[idx+1:]); err == nil {
				currentPreviewPort = p
			}
		}
	}
	if currentPreviewPort == 0 || currentPreviewPort == studioCfg.Port {
		values["PREVIEW_LISTEN_ADDR"] = fmt.Sprintf("127.0.0.1:%d", defaultPreviewPort)
	}

	// Concurrency and worker limits
	warmLimit, maxActive, warmStartup := cfg.NormalizeWorkerLimits()
	values["PER_ACCOUNT_CONCURRENCY"] = strconv.Itoa(cfg.PerAccountConcurrency)
	values["WARM_WORKER_LIMIT"] = strconv.Itoa(warmLimit)
	values["MAX_ACTIVE_WORKERS"] = strconv.Itoa(maxActive)
	values["WARM_STARTUP_CONCURRENCY"] = strconv.Itoa(warmStartup)

	routingStrategy := "fill-first"
	if cfg.RoutingStrategy != "" {
		routingStrategy = cfg.RoutingStrategy
	}
	values["ROUTING_STRATEGY"] = routingStrategy

	firstEventHedge := "off"
	if cfg.FirstEventHedge != "" {
		firstEventHedge = cfg.FirstEventHedge
	}
	values["FIRST_EVENT_HEDGE"] = firstEventHedge
	values["FIRST_EVENT_HEDGE_MODELS"] = cfg.FirstEventHedgeModels

	resolvedProxy := ResolveMagpieProxy(cfg, inheritedProxy)
	values["PROXY"] = resolvedProxy

	var sb strings.Builder
	for _, key := range EnvKeys {
		if val, ok := values[key]; ok {
			sb.WriteString(key)
			sb.WriteByte('=')
			sb.WriteString(FormatEnvValue(val))
			sb.WriteByte('\n')
		}
	}
	newContent := sb.String()

	envPath := filepath.Join(dataDir, ".env")
	if existing, err := os.ReadFile(envPath); err == nil && string(existing) == newContent {
		return nil
	}

	var tmpBytes [4]byte
	_, _ = rand.Read(tmpBytes[:])
	tmpPath := filepath.Join(dataDir, fmt.Sprintf(".env.%x.tmp", tmpBytes[:]))
	if err := os.WriteFile(tmpPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}
	if err := os.Rename(tmpPath, envPath); err != nil {
		return fmt.Errorf("failed to rename .env file: %w", err)
	}
	return nil
}

// ServiceEnv builds the runtime environment variable pairs for AIStudio2API.
func ServiceEnv(dataDir string, studioCfg *StudioConfigFile, effectiveProxy string) [][2]string {
	authPath := filepath.Join(dataDir, "auth")
	env := [][2]string{
		{"LISTEN_ADDR", fmt.Sprintf("127.0.0.1:%d", studioCfg.Port)},
		{"AISTUDIO_AUTH_STATES", authPath},
		{"PROXY_API_KEY", studioCfg.APIKey},
		{"ADMIN_AUTH_ENABLED", "true"},
		{"ADMIN_USERNAME", studioCfg.AdminUsername},
		{"ADMIN_PASSWORD", studioCfg.AdminPassword},
		{"AUTO_START", "true"},
		{"WAA_BACKEND", "spidermonkey"},
		{"UPSTREAM_CHANNELS", "playground"},
		{"BUILD_NATIVE_NONSTREAM", "false"},
		{"SEARCH_MODEL_ALIASES", "false"},
	}
	if strings.TrimSpace(effectiveProxy) != "" {
		env = append(env, [2]string{"PROXY", strings.TrimSpace(effectiveProxy)})
	}
	return env
}

// LoadGatewayConfig loads MagpieConfig from <data_dir>/magpie-gateway.json.
// If the file does not exist, it initializes default settings and hydrates them
// from existing <data_dir>/config.json and <data_dir>/.env so that existing Cloud Run
// environment variables (e.g. PER_ACCOUNT_CONCURRENCY=3, WARM_WORKER_LIMIT=6) are seamlessly preserved.
func LoadGatewayConfig(dataDir string, inheritedProxy string) (*MagpieConfig, error) {
	gatewayPath := filepath.Join(dataDir, "magpie-gateway.json")
	if data, err := os.ReadFile(gatewayPath); err == nil {
		var cfg MagpieConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return &cfg, nil
		}
	}

	cfg := NewDefaultConfig()
	// Hydrate from existing config.json and .env
	HydrateConfig(context.Background(), dataDir, cfg, inheritedProxy)
	_ = SaveGatewayConfig(dataDir, cfg)
	return cfg, nil
}

// SaveGatewayConfig writes MagpieConfig to <data_dir>/magpie-gateway.json atomically.
func SaveGatewayConfig(dataDir string, cfg *MagpieConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	gatewayPath := filepath.Join(dataDir, "magpie-gateway.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	var tmpBytes [4]byte
	_, _ = rand.Read(tmpBytes[:])
	tmpPath := filepath.Join(dataDir, fmt.Sprintf(".magpie-gateway.json.%x.tmp", tmpBytes[:]))
	if err := os.WriteFile(tmpPath, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, gatewayPath)
}

// HydrateConfig updates cfg with values found in <data_dir>/config.json and <data_dir>/.env,
// and optionally from live GET /api/config if the daemon is currently running.
// Returns true if any field was modified.
func HydrateConfig(ctx context.Context, dataDir string, cfg *MagpieConfig, inheritedProxy string) bool {
	if cfg == nil {
		return false
	}
	initial := *cfg
	wasInheritingProxy := strings.TrimSpace(cfg.Proxy) == ""
	inheritedTrimmed := strings.TrimSpace(inheritedProxy)

	// 1. Sync internal_port from config.json
	configPath := filepath.Join(dataDir, "config.json")
	if content, err := os.ReadFile(configPath); err == nil {
		var studioCfg StudioConfigFile
		if err := json.Unmarshal(content, &studioCfg); err == nil && studioCfg.Port > 0 {
			cfg.InternalPort = studioCfg.Port
		}
	}

	// 2. Sync from .env
	envValues := ReadStudioEnv(dataDir)
	if v, ok := envValues["PER_ACCOUNT_CONCURRENCY"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 {
			cfg.PerAccountConcurrency = n
		}
	}
	if v, ok := envValues["WARM_WORKER_LIMIT"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 {
			cfg.WarmWorkerLimit = n
		}
	}
	if v, ok := envValues["MAX_ACTIVE_WORKERS"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 {
			cfg.MaxActiveWorkers = n
		}
	}
	if v, ok := envValues["WARM_STARTUP_CONCURRENCY"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 {
			cfg.WarmStartupConcurrency = n
		}
	}
	cfg.NormalizeWorkerLimits()

	if v, ok := envValues["ROUTING_STRATEGY"]; ok && strings.TrimSpace(v) != "" {
		cfg.RoutingStrategy = strings.TrimSpace(v)
	}
	if v, ok := envValues["FIRST_EVENT_HEDGE"]; ok && strings.TrimSpace(v) != "" {
		cfg.FirstEventHedge = strings.TrimSpace(v)
	}
	if v, ok := envValues["FIRST_EVENT_HEDGE_MODELS"]; ok {
		cfg.FirstEventHedgeModels = strings.TrimSpace(v)
	}
	if v, ok := envValues["PROXY"]; ok {
		p := strings.TrimSpace(v)
		if !(wasInheritingProxy && p == inheritedTrimmed) {
			cfg.Proxy = p
		}
	}

	// 3. Sync from live GET /api/config if daemon is responsive
	if liveCfgBytes, err := StudioAdminRequest(ctx, dataDir, "GET", "/api/config", nil, 2*time.Second); err == nil {
		var liveCfg map[string]any
		if err := json.Unmarshal(liveCfgBytes, &liveCfg); err == nil {
			parseInt := func(val any) (int, bool) {
				switch v := val.(type) {
				case float64:
					return int(v), true
				case string:
					n, err := strconv.Atoi(strings.TrimSpace(v))
					return n, err == nil
				default:
					return 0, false
				}
			}
			if n, ok := parseInt(liveCfg["per_account_concurrency"]); ok && n >= 1 {
				cfg.PerAccountConcurrency = n
			}
			if n, ok := parseInt(liveCfg["warm_worker_limit"]); ok && n >= 1 {
				cfg.WarmWorkerLimit = n
			}
			if n, ok := parseInt(liveCfg["max_active_workers"]); ok && n >= 1 {
				cfg.MaxActiveWorkers = n
			}
			if n, ok := parseInt(liveCfg["warm_startup_concurrency"]); ok && n >= 1 {
				cfg.WarmStartupConcurrency = n
			}
			cfg.NormalizeWorkerLimits()

			if s, ok := liveCfg["routing_strategy"].(string); ok && strings.TrimSpace(s) != "" {
				cfg.RoutingStrategy = strings.TrimSpace(s)
			}
			if s, ok := liveCfg["first_event_hedge"].(string); ok && strings.TrimSpace(s) != "" {
				cfg.FirstEventHedge = strings.TrimSpace(s)
			}
			if s, ok := liveCfg["first_event_hedge_models"].(string); ok {
				cfg.FirstEventHedgeModels = strings.TrimSpace(s)
			}
			if s, ok := liveCfg["proxy"].(string); ok {
				p := strings.TrimSpace(s)
				if !(wasInheritingProxy && p == inheritedTrimmed) {
					cfg.Proxy = p
				}
			}
		}
	}

	return initial != *cfg
}

// SaveMagpieConfig persists cfg to <data_dir>/magpie-gateway.json and invokes ApplyMagpieConfig
// to push the updated configuration to the active backend.
func SaveMagpieConfig(ctx context.Context, dataDir string, cfg *MagpieConfig, inheritedProxy string) error {
	if err := SaveGatewayConfig(dataDir, cfg); err != nil {
		return fmt.Errorf("failed to save gateway config: %w", err)
	}
	return ApplyMagpieConfig(ctx, dataDir, cfg, inheritedProxy)
}

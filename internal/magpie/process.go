package magpie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// MagpieStudioVersion is the pinned upstream AIStudio2API release version.
	MagpieStudioVersion = "0.2.19-magpie.1"
	// PreinstalledMagpieBin is the container-bundled binary path for Cloud Run.
	PreinstalledMagpieBin = "/app/magpie/aistudio2api"
)

// PreinstalledBinaryPath returns the path to the container pre-installed binary,
// or empty string if it does not exist.
func PreinstalledBinaryPath() string {
	candidate := os.Getenv("MAGPIE_PREINSTALLED_BIN")
	if strings.TrimSpace(candidate) == "" {
		candidate = PreinstalledMagpieBin
	}
	if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
		return candidate
	}
	return ""
}

// ResolveExecutablePath locates the binary to run AIStudio2API.
// It prioritizes the container pre-installed binary because GCS FUSE mount directories
// are mounted with noexec permissions (leading to Permission denied / os error 13).
func ResolveExecutablePath(dataDir string) string {
	if pre := PreinstalledBinaryPath(); pre != "" {
		return pre
	}
	binName := "aistudio2api"
	if runtime.GOOS == "windows" {
		binName = "aistudio2api.exe"
	}
	pkgBin := filepath.Join(dataDir, "package", binName)
	if fi, err := os.Stat(pkgBin); err == nil && !fi.IsDir() {
		if runtime.GOOS != "windows" {
			_ = os.Chmod(pkgBin, 0755)
		}
		return pkgBin
	}
	return ""
}

// EnsurePreinstalledBinarySeeded writes install-manifest.json into dataDir
// when running against a pre-installed binary.
func EnsurePreinstalledBinarySeeded(dataDir string) {
	pre := PreinstalledBinaryPath()
	if pre == "" {
		return
	}
	_ = os.MkdirAll(dataDir, 0755)
	manifestPath := filepath.Join(dataDir, "install-manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return
	}
	platform := runtime.GOOS
	if platform == "windows" {
		platform = "win32"
	}
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	manifest := map[string]any{
		"version":      MagpieStudioVersion,
		"platform":     platform,
		"arch":         arch,
		"preinstalled": true,
		"binaryPath":   pre,
	}
	if data, err := json.MarshalIndent(manifest, "", "  "); err == nil {
		_ = os.WriteFile(manifestPath, append(data, '\n'), 0644)
	}
}

// MagpieInstallStatus captures the local installation state of AIStudio2API and Camoufox.
type MagpieInstallStatus struct {
	Installed     bool   `json:"installed"`
	Version       string `json:"version,omitempty"`
	BinaryPath    string `json:"binary_path,omitempty"`
	CamoufoxReady bool   `json:"camoufox_ready"`
	DataDir       string `json:"data_dir"`
	ManifestValid bool   `json:"manifest_valid"`
}

// GetInstallStatus returns the installation status for AIStudio2API and Camoufox.
func GetInstallStatus(dataDir string) MagpieInstallStatus {
	EnsurePreinstalledBinarySeeded(dataDir)
	resolvedBin := ResolveExecutablePath(dataDir)
	installed := resolvedBin != ""

	manifestPath := filepath.Join(dataDir, "install-manifest.json")
	manifestValid := false
	var version string

	if PreinstalledBinaryPath() != "" {
		version = MagpieStudioVersion
		manifestValid = true
	} else if installed {
		if content, errRead := os.ReadFile(manifestPath); errRead == nil {
			var m map[string]any
			if errUnmarshal := json.Unmarshal(content, &m); errUnmarshal == nil {
				if v, ok := m["version"].(string); ok {
					version = v
					manifestValid = (v == MagpieStudioVersion)
				}
			}
		}
	}

	return MagpieInstallStatus{
		Installed:     installed,
		Version:       version,
		BinaryPath:    resolvedBin,
		CamoufoxReady: false,
		DataDir:       dataDir,
		ManifestValid: manifestValid,
	}
}

// StudioAdminRequest performs an authenticated admin request to the local AIStudio2API daemon.
func StudioAdminRequest(ctx context.Context, dataDir string, method, path string, body any, timeout time.Duration) ([]byte, error) {
	configPath := filepath.Join(dataDir, "config.json")
	content, errRead := os.ReadFile(configPath)
	if errRead != nil {
		return nil, fmt.Errorf("MAGPIE configuration config.json not found: %w", errRead)
	}
	var studioCfg StudioConfigFile
	if errUnmarshal := json.Unmarshal(content, &studioCfg); errUnmarshal != nil {
		return nil, fmt.Errorf("failed to parse config.json: %w", errUnmarshal)
	}

	baseOrigin := fmt.Sprintf("http://127.0.0.1:%d", studioCfg.Port)
	client := &http.Client{
		Timeout: timeout,
	}

	// 1. Admin login to obtain session cookie
	loginURL := fmt.Sprintf("%s/api/auth/login", baseOrigin)
	loginPayload, _ := json.Marshal(map[string]string{
		"username": studioCfg.AdminUsername,
		"password": studioCfg.AdminPassword,
	})
	loginReq, errLoginReq := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewReader(loginPayload))
	if errLoginReq != nil {
		return nil, fmt.Errorf("failed to create login request: %w", errLoginReq)
	}
	loginReq.Header.Set("Content-Type", "application/json")

	loginRes, errLoginDo := client.Do(loginReq)
	if errLoginDo != nil {
		return nil, fmt.Errorf("admin login connection failed: %w", errLoginDo)
	}
	defer func() {
		if errClose := loginRes.Body.Close(); errClose != nil {
			log.Debugf("failed to close admin login response body: %v", errClose)
		}
	}()

	if loginRes.StatusCode < 200 || loginRes.StatusCode >= 300 {
		return nil, fmt.Errorf("admin login failed: HTTP %d", loginRes.StatusCode)
	}

	var adminCookie string
	for _, cookie := range loginRes.Cookies() {
		if cookie.Name == "aistudio_admin" {
			adminCookie = cookie.String()
			break
		}
	}
	if adminCookie == "" {
		for _, raw := range loginRes.Header.Values("Set-Cookie") {
			if strings.Contains(raw, "aistudio_admin=") {
				parts := strings.Split(raw, ";")
				adminCookie = strings.TrimSpace(parts[0])
				break
			}
		}
	}

	// 2. Execute target request
	targetURL := fmt.Sprintf("%s%s", baseOrigin, path)
	var bodyReader io.Reader
	if body != nil {
		switch b := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(b)
		case string:
			bodyReader = strings.NewReader(b)
		default:
			jsonData, errMarshal := json.Marshal(b)
			if errMarshal != nil {
				return nil, fmt.Errorf("failed to serialize body: %w", errMarshal)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req, errReq := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
	if errReq != nil {
		return nil, fmt.Errorf("failed to create admin request: %w", errReq)
	}
	if adminCookie != "" {
		req.Header.Set("Cookie", adminCookie)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, errDo := client.Do(req)
	if errDo != nil {
		return nil, fmt.Errorf("admin request failed: %w", errDo)
	}
	defer func() {
		if errClose := res.Body.Close(); errClose != nil {
			log.Debugf("failed to close admin response body: %v", errClose)
		}
	}()

	if res.StatusCode == http.StatusNoContent {
		return []byte(`{"success":true}`), nil
	}

	resBytes, errReadBody := io.ReadAll(res.Body)
	if errReadBody != nil {
		return nil, fmt.Errorf("failed to read response body: %w", errReadBody)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("admin request failed (HTTP %d): %s", res.StatusCode, string(resBytes))
	}

	if len(bytes.TrimSpace(resBytes)) == 0 {
		return []byte(`{"success":true}`), nil
	}
	return resBytes, nil
}

// MagpieProcessStatus reports process and account metrics from /api/status.
type MagpieProcessStatus struct {
	Running              bool   `json:"running"`
	Ready                bool   `json:"ready"`
	State                string `json:"state"`
	Port                 int    `json:"port"`
	Version              string `json:"version,omitempty"`
	AccountsTotal        int    `json:"accounts_total"`
	AccountsReady        int    `json:"accounts_ready"`
	AccountsBusy         int    `json:"accounts_busy"`
	AccountsCooldown     int    `json:"accounts_cooldown"`
	AccountsAuthRequired int    `json:"accounts_auth_required"`
}

// CheckServiceStatus queries /api/status and parses daemon readiness and account states.
func CheckServiceStatus(ctx context.Context, dataDir string) (*MagpieProcessStatus, error) {
	resBytes, errAdmin := StudioAdminRequest(ctx, dataDir, "GET", "/api/status", nil, 2500*time.Millisecond)
	if errAdmin != nil {
		return nil, errAdmin
	}
	var raw map[string]any
	if errUnmarshal := json.Unmarshal(resBytes, &raw); errUnmarshal != nil {
		return nil, fmt.Errorf("invalid json in status: %w", errUnmarshal)
	}

	running, _ := raw["running"].(bool)
	ready, _ := raw["ready"].(bool)
	state, _ := raw["state"].(string)
	if state == "" {
		state = "UNKNOWN"
	}
	var version string
	if v, ok := raw["version"].(string); ok {
		version = v
	}

	var total, readyAcc, busy, cooldown, authReq int
	if accounts, ok := raw["accounts"].(map[string]any); ok {
		getInt := func(val any) int {
			if f, ok := val.(float64); ok {
				return int(f)
			}
			return 0
		}
		total = getInt(accounts["total"])
		readyAcc = getInt(accounts["ready"])
		busy = getInt(accounts["busy"])
		cooldown = getInt(accounts["cooldown"])
		authReq = getInt(accounts["auth_required"])
	}

	port := 2048
	configPath := filepath.Join(dataDir, "config.json")
	if c, errReadConfig := os.ReadFile(configPath); errReadConfig == nil {
		var studioCfg StudioConfigFile
		if errUnmarshal := json.Unmarshal(c, &studioCfg); errUnmarshal == nil && studioCfg.Port > 0 {
			port = studioCfg.Port
		}
	}

	return &MagpieProcessStatus{
		Running:              running,
		Ready:                ready,
		State:                state,
		Port:                 port,
		Version:              version,
		AccountsTotal:        total,
		AccountsReady:        readyAcc,
		AccountsBusy:         busy,
		AccountsCooldown:     cooldown,
		AccountsAuthRequired: authReq,
	}, nil
}

var startBackendMu sync.Mutex

// StartBackend launches the managed AIStudio2API process and waits for it to become responsive.
func StartBackend(ctx context.Context, dataDir string, cfg *MagpieConfig, inheritedProxy string) (*MagpieProcessStatus, error) {
	EnsurePreinstalledBinarySeeded(dataDir)
	startBackendMu.Lock()
	defer startBackendMu.Unlock()

	// Check if already running or responsive
	if st, errStatus := CheckServiceStatus(ctx, dataDir); errStatus == nil {
		if st.Running {
			return st, nil
		}
		if st.State == "STOPPED" {
			_, _ = StudioAdminRequest(ctx, dataDir, "POST", "/api/control/start", nil, 60*time.Second)
			time.Sleep(500 * time.Millisecond)
			if resSt, errRetry := CheckServiceStatus(ctx, dataDir); errRetry == nil {
				return resSt, nil
			}
		}
		return st, nil
	}

	binPath := ResolveExecutablePath(dataDir)
	if binPath == "" {
		return nil, fmt.Errorf("AIStudio2API binary not installed. Please install it first")
	}

	studioCfg, errEnsure := EnsureStudioConfig(dataDir, cfg, inheritedProxy)
	if errEnsure != nil {
		return nil, errEnsure
	}

	// In container environments, remove any stale copy of the binary on the noexec FUSE mount
	if PreinstalledBinaryPath() != "" {
		stalePkgBin := filepath.Join(dataDir, "package", "aistudio2api")
		_ = os.Remove(stalePkgBin)
	}

	tmpDir := ResolveRuntimeTempDir()
	logPath := filepath.Join(tmpDir, "backend.log")
	logFile, errOpen := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if errOpen != nil {
		logPath = filepath.Join(os.TempDir(), "magpie-backend.log")
		logFile, errOpen = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if errOpen != nil {
			return nil, fmt.Errorf("failed to open backend.log: %w", errOpen)
		}
	}

	cmd := exec.Command(binPath, "-open-ui=false")
	cmd.Dir = dataDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Clear conflicting inherited env vars starting with AISTUDIO_ or WAA_
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "AISTUDIO_") && !strings.HasPrefix(kv, "WAA_") {
			env = append(env, kv)
		}
	}

	effectiveProxy := ResolveMagpieProxy(cfg, inheritedProxy)
	for _, pair := range ServiceEnv(dataDir, studioCfg, effectiveProxy) {
		env = append(env, fmt.Sprintf("%s=%s", pair[0], pair[1]))
	}
	cmd.Env = env

	if errStart := cmd.Start(); errStart != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("failed to spawn AIStudio2API (bin=%s, cwd=%s): %w", binPath, dataDir, errStart)
	}

	pidPath := filepath.Join(tmpDir, "backend.pid")
	_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0644)

	// Poll status until responsive (up to 30 seconds)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(400 * time.Millisecond)
		if st, errPoll := CheckServiceStatus(ctx, dataDir); errPoll == nil {
			if st.State == "STOPPED" && !st.Running {
				_, _ = StudioAdminRequest(ctx, dataDir, "POST", "/api/control/start", nil, 60*time.Second)
			}
			if st.Running || st.Ready {
				return st, nil
			}
		}
	}

	return CheckServiceStatus(ctx, dataDir)
}

// StopBackend stops the managed AIStudio2API process gracefully via /api/control/stop
// and terminates its PID if still active.
func StopBackend(ctx context.Context, dataDir string) error {
	// 1. Try graceful /api/control/stop
	_, _ = StudioAdminRequest(ctx, dataDir, "POST", "/api/control/stop", nil, 5*time.Second)

	// 2. Terminate PID if recorded in runtime temp dir or dataDir
	pidPaths := []string{
		filepath.Join(ResolveRuntimeTempDir(), "backend.pid"),
		filepath.Join(dataDir, "backend.pid"),
	}
	for _, pidPath := range pidPaths {
		if content, errRead := os.ReadFile(pidPath); errRead == nil {
			if pid, errParse := strconv.Atoi(strings.TrimSpace(string(content))); errParse == nil && pid > 0 {
				if runtime.GOOS == "windows" {
					_ = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid)).Run()
				} else {
					if p, errFind := os.FindProcess(pid); errFind == nil {
						_ = p.Kill()
					}
				}
			}
			_ = os.Remove(pidPath)
		}
	}
	return nil
}

// RestartBackend stops and restarts the AIStudio2API backend.
func RestartBackend(ctx context.Context, dataDir string, cfg *MagpieConfig, inheritedProxy string) (*MagpieProcessStatus, error) {
	_ = StopBackend(ctx, dataDir)
	time.Sleep(500 * time.Millisecond)
	return StartBackend(ctx, dataDir, cfg, inheritedProxy)
}

// EnsureRunning starts the backend if it is not currently running.
func EnsureRunning(ctx context.Context, dataDir string, cfg *MagpieConfig, inheritedProxy string) error {
	if st, errStatus := CheckServiceStatus(ctx, dataDir); errStatus == nil && st.Running {
		return nil
	}
	_, errStart := StartBackend(ctx, dataDir, cfg, inheritedProxy)
	return errStart
}

// ApplyMagpieConfig pushes the configuration in cfg to <data_dir>/config.json,
// <data_dir>/.env, and the active AIStudio2API daemon.
func ApplyMagpieConfig(ctx context.Context, dataDir string, cfg *MagpieConfig, inheritedProxy string) error {
	configPath := filepath.Join(dataDir, "config.json")
	var oldPort int
	if content, errRead := os.ReadFile(configPath); errRead == nil {
		var oldCfg StudioConfigFile
		if errUnmarshal := json.Unmarshal(content, &oldCfg); errUnmarshal == nil {
			oldPort = oldCfg.Port
		}
	}
	portChanged := oldPort > 0 && cfg.InternalPort > 0 && oldPort != cfg.InternalPort

	statusBefore, _ := CheckServiceStatus(ctx, dataDir)
	studioCfg, errEnsure := EnsureStudioConfig(dataDir, cfg, inheritedProxy)
	if errEnsure != nil {
		return errEnsure
	}

	if statusBefore != nil {
		if portChanged {
			if statusBefore.Running {
				_, errRestart := RestartBackend(ctx, dataDir, cfg, inheritedProxy)
				return errRestart
			}
			_ = StopBackend(ctx, dataDir)
			return nil
		}

		// Whenever the admin endpoint is reachable, push updated configuration
		if curCfgBytes, errCurCfg := StudioAdminRequest(ctx, dataDir, "GET", "/api/config", nil, 5*time.Second); errCurCfg == nil {
			var merged map[string]any
			if errUnmarshal := json.Unmarshal(curCfgBytes, &merged); errUnmarshal == nil {
				warmLimit, maxActive, warmStartup := cfg.NormalizeWorkerLimits()
				routingStrategy := "fill-first"
				if cfg.RoutingStrategy != "" {
					routingStrategy = cfg.RoutingStrategy
				}
				firstEventHedge := "off"
				if cfg.FirstEventHedge != "" {
					firstEventHedge = cfg.FirstEventHedge
				}
				resolvedProxy := ResolveMagpieProxy(cfg, inheritedProxy)

				merged["per_account_concurrency"] = cfg.PerAccountConcurrency
				merged["warm_worker_limit"] = warmLimit
				merged["max_active_workers"] = maxActive
				merged["warm_startup_concurrency"] = warmStartup
				merged["routing_strategy"] = routingStrategy
				merged["first_event_hedge"] = firstEventHedge
				merged["first_event_hedge_models"] = cfg.FirstEventHedgeModels
				merged["proxy"] = resolvedProxy

				if savedBytes, errPutCfg := StudioAdminRequest(ctx, dataDir, "PUT", "/api/config", merged, 10*time.Second); errPutCfg == nil {
					_ = SyncStudioEnv(dataDir, studioCfg, cfg, inheritedProxy)
					var savedRes map[string]any
					if errUnmarshalRes := json.Unmarshal(savedBytes, &savedRes); errUnmarshalRes == nil {
						if req, ok := savedRes["management_restart_required"].(bool); ok && req {
							_, _ = RestartBackend(ctx, dataDir, cfg, inheritedProxy)
						} else if req, ok := savedRes["service_restart_required"].(bool); ok && req && statusBefore.Running {
							_, _ = StudioAdminRequest(ctx, dataDir, "POST", "/api/control/stop", nil, 10*time.Second)
							time.Sleep(500 * time.Millisecond)
							_, _ = StudioAdminRequest(ctx, dataDir, "POST", "/api/control/start", nil, 60*time.Second)
						}
					}
				}
			}
		}
	}
	return nil
}

// HasImportedAccounts checks if <data_dir>/auth has at least one account directory
// containing a non-empty storage-state.json file.
func HasImportedAccounts(dataDir string) bool {
	authDir := filepath.Join(dataDir, "auth")
	entries, errReadDir := os.ReadDir(authDir)
	if errReadDir != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			statePath := filepath.Join(authDir, entry.Name(), "storage-state.json")
			if fi, errStat := os.Stat(statePath); errStat == nil && fi.Size() > 0 {
				return true
			}
		}
	}
	return false
}

// AutoStartBackground starts the managed backend in a background goroutine if enabled,
// auto_start is true, and at least one imported account exists.
func AutoStartBackground(ctx context.Context, inheritedProxy string) {
	dataDir := ResolveDataDir("")
	cfg, errLoad := LoadGatewayConfig(dataDir, inheritedProxy)
	if errLoad == nil && cfg.Enabled && cfg.AutoStart && HasImportedAccounts(dataDir) {
		go func() {
			startCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			log.Infof("[MAGPIE] Auto-starting managed backend in background (dataDir=%s)", dataDir)
			if _, errStart := StartBackend(startCtx, dataDir, cfg, inheritedProxy); errStart != nil {
				log.Warnf("[MAGPIE] Background auto-start failed: %v", errStart)
			}
		}()
	}
}

// StopManagedBackend terminates the managed AIStudio2API process in the resolved data directory.
func StopManagedBackend(ctx context.Context) {
	dataDir := ResolveDataDir("")
	_ = StopBackend(ctx, dataDir)
}

// GetFullStatus aggregates process status, install status, config, and quota for management endpoints.
func GetFullStatus(ctx context.Context, dataDir string, inheritedProxy string) (map[string]any, error) {
	cfg, _ := LoadGatewayConfig(dataDir, inheritedProxy)
	install := GetInstallStatus(dataDir)
	procStatus, _ := CheckServiceStatus(ctx, dataDir)
	quota, _ := GetQuotaSummary(ctx, dataDir)

	result := map[string]any{
		"config":  cfg,
		"install": install,
	}
	if procStatus != nil {
		result["process"] = procStatus
	}
	if quota != nil {
		result["quota"] = quota
	}
	return result, nil
}

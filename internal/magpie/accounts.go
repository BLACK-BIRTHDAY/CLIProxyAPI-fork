package magpie

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"
)

// MagpieServiceInfo provides connection and administrative metadata for MAGPIE.
type MagpieServiceInfo struct {
	ProxyDisplay  string `json:"proxy_display,omitempty"`
	AdminURL      string `json:"admin_url"`
	AdminUsername string `json:"admin_username"`
	ConfigPath    string `json:"config_path"`
}

// MagpieQuotaResponse aggregates accounts, cooldowns, and runtime service info.
type MagpieQuotaResponse struct {
	Status      any                `json:"status"`
	Accounts    []any              `json:"accounts"`
	Cooldowns   any                `json:"cooldowns"`
	ServiceInfo *MagpieServiceInfo `json:"service_info,omitempty"`
}

// GetQuotaSummary aggregates /api/status, /api/accounts, /api/cooldowns and local service metadata.
func GetQuotaSummary(ctx context.Context, dataDir string) (*MagpieQuotaResponse, error) {
	statusBytes, err := StudioAdminRequest(ctx, dataDir, "GET", "/api/status", nil, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch status: %w", err)
	}
	var statusVal any
	_ = json.Unmarshal(statusBytes, &statusVal)

	var accounts []any
	if accountsBytes, err := StudioAdminRequest(ctx, dataDir, "GET", "/api/accounts", nil, 5*time.Second); err == nil {
		var accWrapper map[string]any
		if err := json.Unmarshal(accountsBytes, &accWrapper); err == nil {
			if list, ok := accWrapper["accounts"].([]any); ok {
				accounts = list
			}
		} else {
			var rawList []any
			if err := json.Unmarshal(accountsBytes, &rawList); err == nil {
				accounts = rawList
			}
		}
	}

	var cooldowns any
	if cdBytes, err := StudioAdminRequest(ctx, dataDir, "GET", "/api/cooldowns", nil, 5*time.Second); err == nil {
		_ = json.Unmarshal(cdBytes, &cooldowns)
	}

	configFilePath := filepath.Join(dataDir, "config.json")
	port := 2048
	adminUsername := "magpie"
	var proxyDisplay string

	if content, err := os.ReadFile(configFilePath); err == nil {
		var cfg StudioConfigFile
		if err := json.Unmarshal(content, &cfg); err == nil {
			if cfg.Port > 0 {
				port = cfg.Port
			}
			if cfg.AdminUsername != "" {
				adminUsername = cfg.AdminUsername
			}
			if strings.TrimSpace(cfg.Proxy) != "" {
				proxyDisplay = strings.TrimSpace(cfg.Proxy)
			}
		}
	}

	if proxyDisplay == "" {
		envValues := ReadStudioEnv(dataDir)
		if p, ok := envValues["PROXY"]; ok && strings.TrimSpace(p) != "" {
			proxyDisplay = strings.TrimSpace(p)
		}
	}

	serviceInfo := &MagpieServiceInfo{
		ProxyDisplay:  proxyDisplay,
		AdminURL:      fmt.Sprintf("http://127.0.0.1:%d", port),
		AdminUsername: adminUsername,
		ConfigPath:    configFilePath,
	}

	return &MagpieQuotaResponse{
		Status:      statusVal,
		Accounts:    accounts,
		Cooldowns:   cooldowns,
		ServiceInfo: serviceInfo,
	}, nil
}

// ToggleAccount enables or disables a MAGPIE account, preserving label, proxy, locale and timezone.
func ToggleAccount(ctx context.Context, dataDir string, cfg *MagpieConfig, inheritedProxy string, accountID string, enabled bool) (any, error) {
	if err := EnsureRunning(ctx, dataDir, cfg, inheritedProxy); err != nil {
		return nil, fmt.Errorf("failed to ensure backend running: %w", err)
	}

	label := accountID
	proxy := ""
	locale := "zh-CN"
	timezone := "Asia/Tokyo"

	if accBytes, err := StudioAdminRequest(ctx, dataDir, "GET", "/api/accounts", nil, 5*time.Second); err == nil {
		var accMap map[string]any
		var list []any
		if err := json.Unmarshal(accBytes, &accMap); err == nil {
			if l, ok := accMap["accounts"].([]any); ok {
				list = l
			}
		} else {
			_ = json.Unmarshal(accBytes, &list)
		}
		for _, rawItem := range list {
			if item, ok := rawItem.(map[string]any); ok {
				match := false
				if id, ok := item["id"].(string); ok && id == accountID {
					match = true
				}
				if id, ok := item["account_id"].(string); ok && id == accountID {
					match = true
				}
				if match {
					if l, ok := item["label"].(string); ok && l != "" {
						label = l
					}
					if p, ok := item["proxy"].(string); ok {
						proxy = p
					}
					if loc, ok := item["locale"].(string); ok && loc != "" {
						locale = loc
					}
					if tz, ok := item["timezone"].(string); ok && tz != "" {
						timezone = tz
					}
					break
				}
			}
		}
	}

	encodedID := url.PathEscape(accountID)
	reqPath := fmt.Sprintf("/api/accounts/%s", encodedID)
	body := map[string]any{
		"label":    label,
		"enabled":  enabled,
		"proxy":    proxy,
		"locale":   locale,
		"timezone": timezone,
	}

	resBytes, err := StudioAdminRequest(ctx, dataDir, "PUT", reqPath, body, 10*time.Second)
	if err != nil {
		return nil, err
	}
	var res any
	_ = json.Unmarshal(resBytes, &res)
	return res, nil
}

// DeleteAccount removes an account from the running backend and deletes its storage state files.
func DeleteAccount(ctx context.Context, dataDir string, accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return fmt.Errorf("account_id must not be empty")
	}
	if strings.HasPrefix(accountID, ".") {
		return fmt.Errorf("account_id %q must not start with '.'", accountID)
	}
	if strings.Contains(accountID, "/") || strings.Contains(accountID, "\\") || strings.Contains(accountID, "..") {
		return fmt.Errorf("account_id %q contains invalid path traversal characters", accountID)
	}
	for _, r := range accountID {
		if unicode.IsControl(r) {
			return fmt.Errorf("account_id %q contains control characters", accountID)
		}
	}

	authDir := filepath.Join(dataDir, "auth")
	accountDir := filepath.Join(authDir, accountID)
	legacyAuthFile := filepath.Join(authDir, fmt.Sprintf("%s.json", accountID))

	isRunning := false
	if st, err := CheckServiceStatus(ctx, dataDir); err == nil && st.Running {
		isRunning = true
	}

	if isRunning {
		encodedID := url.PathEscape(accountID)
		reqPath := fmt.Sprintf("/api/accounts/%s", encodedID)
		if _, err := StudioAdminRequest(ctx, dataDir, "DELETE", reqPath, nil, 10*time.Second); err != nil {
			return fmt.Errorf("delete account API failed: %w", err)
		}
		_ = os.RemoveAll(accountDir)
		_ = os.Remove(legacyAuthFile)
		return nil
	}

	dirExists := false
	if fi, err := os.Stat(accountDir); err == nil && fi.IsDir() {
		dirExists = true
	}
	legacyExists := false
	if fi, err := os.Stat(legacyAuthFile); err == nil && !fi.IsDir() {
		legacyExists = true
	}

	if dirExists || legacyExists {
		_ = os.RemoveAll(accountDir)
		_ = os.Remove(legacyAuthFile)
		return nil
	}

	return fmt.Errorf("account %s not found and backend is not running", accountID)
}

// MagpieAccountExportItem represents a single account's complete state within an account bundle.
type MagpieAccountExportItem struct {
	AccountID           string `json:"account_id"`
	StorageState        any    `json:"storage_state"`
	AccountMeta         any    `json:"account_meta,omitempty"`
	CamoufoxFingerprint any    `json:"camoufox_fingerprint,omitempty"`
	Catalog             any    `json:"catalog,omitempty"`
	RuntimeState        any    `json:"runtime_state,omitempty"`
}

// MagpieAccountBundle is the wire format for importing and exporting MAGPIE accounts,
// 100% compatible with antigravity tools's MagpieAccountBundle.
type MagpieAccountBundle struct {
	Version         int                       `json:"version"`
	ExportedAt      string                    `json:"exported_at"`
	Accounts        []MagpieAccountExportItem `json:"accounts"`
	WAAInterpreters map[string]string         `json:"waa_interpreters,omitempty"`
}

// MagpieImportResult reports the outcome of importing an account bundle.
type MagpieImportResult struct {
	ImportedCount int      `json:"imported_count"`
	AccountIDs    []string `json:"account_ids"`
}

// ExportAccountsBundle exports all MAGPIE accounts from <data_dir>/auth/ into a portable bundle.
func ExportAccountsBundle(dataDir string) (*MagpieAccountBundle, error) {
	authDir := filepath.Join(dataDir, "auth")
	accounts := make([]MagpieAccountExportItem, 0)
	waaInterpreters := make(map[string]string)

	if fi, err := os.Stat(authDir); err == nil && fi.IsDir() {
		waaDir := filepath.Join(authDir, ".waa-interpreters")
		if wEntries, err := os.ReadDir(waaDir); err == nil {
			for _, entry := range wEntries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
					p := filepath.Join(waaDir, entry.Name())
					if jsBytes, err := os.ReadFile(p); err == nil {
						waaInterpreters[entry.Name()] = string(jsBytes)
					}
				}
			}
		}

		entries, err := os.ReadDir(authDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read auth dir: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			accountID := entry.Name()
			if strings.HasPrefix(accountID, ".") {
				continue
			}
			accountPath := filepath.Join(authDir, accountID)
			storageStatePath := filepath.Join(accountPath, "storage-state.json")
			content, err := os.ReadFile(storageStatePath)
			if err != nil {
				continue
			}
			var storageState any
			if err := json.Unmarshal(content, &storageState); err != nil {
				continue
			}
			if _, ok := storageState.(map[string]any); !ok {
				continue
			}

			readOptJSON := func(fname string) any {
				p := filepath.Join(accountPath, fname)
				if b, err := os.ReadFile(p); err == nil {
					var v any
					if err := json.Unmarshal(b, &v); err == nil {
						return v
					}
				}
				return nil
			}

			accounts = append(accounts, MagpieAccountExportItem{
				AccountID:           accountID,
				StorageState:        storageState,
				AccountMeta:         readOptJSON("account.json"),
				CamoufoxFingerprint: readOptJSON("camoufox-fingerprint.json"),
				Catalog:             readOptJSON("catalog.json"),
				RuntimeState:        readOptJSON("runtime-state.json"),
			})
		}
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].AccountID < accounts[j].AccountID
	})

	return &MagpieAccountBundle{
		Version:         1,
		ExportedAt:      time.Now().UTC().Format(time.RFC3339),
		Accounts:        accounts,
		WAAInterpreters: waaInterpreters,
	}, nil
}

// ImportAccountsBundle imports an account bundle into <data_dir>/auth/, validating
// path traversal, synthesizing default account.json if absent, and hot-reloading the backend.
func ImportAccountsBundle(ctx context.Context, dataDir string, bundle *MagpieAccountBundle, cfg *MagpieConfig, inheritedProxy string) (*MagpieImportResult, error) {
	if bundle == nil {
		return nil, fmt.Errorf("bundle is nil")
	}

	// 1. Validate all accounts in bundle
	for _, item := range bundle.Accounts {
		accountID := item.AccountID
		if strings.TrimSpace(accountID) == "" {
			return nil, fmt.Errorf("account_id must not be empty")
		}
		if strings.HasPrefix(accountID, ".") {
			return nil, fmt.Errorf("account_id %q must not start with '.'", accountID)
		}
		if strings.Contains(accountID, "/") || strings.Contains(accountID, "\\") || strings.Contains(accountID, "..") {
			return nil, fmt.Errorf("account_id %q contains invalid or path traversal characters", accountID)
		}
		for _, r := range accountID {
			if unicode.IsControl(r) {
				return nil, fmt.Errorf("account_id %q contains control characters", accountID)
			}
		}
		if _, ok := item.StorageState.(map[string]any); !ok {
			return nil, fmt.Errorf("account %q storage_state must be a JSON object", accountID)
		}
	}

	// 2. Write accounts into <dataDir>/auth/<accountID>/
	authDir := filepath.Join(dataDir, "auth")
	if err := os.MkdirAll(authDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create auth directory: %w", err)
	}

	if len(bundle.WAAInterpreters) > 0 {
		waaDir := filepath.Join(authDir, ".waa-interpreters")
		_ = os.MkdirAll(waaDir, 0700)
		for fname, jsContent := range bundle.WAAInterpreters {
			if fname != "" && !strings.HasPrefix(fname, ".") && !strings.Contains(fname, "/") &&
				!strings.Contains(fname, "\\") && !strings.Contains(fname, "..") && strings.HasSuffix(fname, ".js") {
				_ = os.WriteFile(filepath.Join(waaDir, fname), []byte(jsContent), 0644)
			}
		}
	}

	importedIDs := make([]string, 0, len(bundle.Accounts))
	for _, item := range bundle.Accounts {
		accountDir := filepath.Join(authDir, item.AccountID)
		if err := os.MkdirAll(accountDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create directory for account %s: %w", item.AccountID, err)
		}

		stateFile := filepath.Join(accountDir, "storage-state.json")
		stateBytes, err := json.MarshalIndent(item.StorageState, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to serialize storage-state for %s: %w", item.AccountID, err)
		}
		if err := os.WriteFile(stateFile, stateBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to write storage-state for %s: %w", item.AccountID, err)
		}

		// Write account.json (synthesizing default if absent so AIStudio2API never errors with missing account.json)
		accountMetaVal := item.AccountMeta
		if accountMetaVal == nil {
			accountMetaVal = map[string]any{
				"label":    item.AccountID,
				"enabled":  true,
				"proxy":    "",
				"locale":   "zh-CN",
				"timezone": "Asia/Tokyo",
			}
		}
		if metaBytes, err := json.MarshalIndent(accountMetaVal, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(accountDir, "account.json"), metaBytes, 0644)
		}

		optFiles := []struct {
			name string
			val  any
		}{
			{"camoufox-fingerprint.json", item.CamoufoxFingerprint},
			{"catalog.json", item.Catalog},
			{"runtime-state.json", item.RuntimeState},
		}
		for _, opt := range optFiles {
			if opt.val != nil {
				if b, err := json.MarshalIndent(opt.val, "", "  "); err == nil {
					_ = os.WriteFile(filepath.Join(accountDir, opt.name), b, 0644)
				}
			}
		}

		importedIDs = append(importedIDs, item.AccountID)
	}

	// 3. Hot-reload backend if currently running, or start if auto_start is enabled
	InvalidateModelsCache()
	if st, err := CheckServiceStatus(ctx, dataDir); err == nil && st.Running {
		if _, err := RestartBackend(ctx, dataDir, cfg, inheritedProxy); err != nil {
			log.Warnf("[MAGPIE] Failed to restart backend after account import: %v", err)
		}
	} else if cfg != nil && cfg.AutoStart && GetInstallStatus(dataDir).Installed {
		if _, err := StartBackend(ctx, dataDir, cfg, inheritedProxy); err != nil {
			log.Warnf("[MAGPIE] Failed to start backend after account import: %v", err)
		}
	}

	return &MagpieImportResult{
		ImportedCount: len(importedIDs),
		AccountIDs:    importedIDs,
	}, nil
}

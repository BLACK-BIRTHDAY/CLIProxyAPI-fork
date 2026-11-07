package magpie

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldExposeBuiltinMenu(t *testing.T) {
	// 1. Explicit MAGPIE_BUILTIN_MENU=1
	t.Run("explicit_env_menu", func(t *testing.T) {
		t.Setenv("MAGPIE_BUILTIN_MENU", "1")
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		if !ShouldExposeBuiltinMenu() {
			t.Errorf("ShouldExposeBuiltinMenu() = false, want true when MAGPIE_BUILTIN_MENU=1")
		}
	})

	// 2. Container pre-installed binary
	t.Run("preinstalled_binary", func(t *testing.T) {
		tmpDir := t.TempDir()
		binPath := filepath.Join(tmpDir, "mock-aistudio2api")
		if err := os.WriteFile(binPath, []byte("fake binary"), 0755); err != nil {
			t.Fatalf("failed to write fake binary: %v", err)
		}
		t.Setenv("MAGPIE_BUILTIN_MENU", "")
		t.Setenv("MAGPIE_PREINSTALLED_BIN", binPath)

		if !ShouldExposeBuiltinMenu() {
			t.Errorf("ShouldExposeBuiltinMenu() = false, want true when PreinstalledBinaryPath() != \"\"")
		}
	})

	// 3. Dynamic activation via ActivateBuiltinMenu()
	t.Run("activate_builtin_menu", func(t *testing.T) {
		t.Setenv("MAGPIE_BUILTIN_MENU", "")
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))
		t.Cleanup(func() {
			builtinMenuActivated.Store(false)
		})

		builtinMenuActivated.Store(false)
		if ShouldExposeBuiltinMenu() {
			t.Errorf("ShouldExposeBuiltinMenu() = true, want false before ActivateBuiltinMenu()")
		}

		ActivateBuiltinMenu()
		if !ShouldExposeBuiltinMenu() {
			t.Errorf("ShouldExposeBuiltinMenu() = false, want true after ActivateBuiltinMenu()")
		}
	})

	// 4. Default unactivated state
	t.Run("unactivated_default", func(t *testing.T) {
		t.Setenv("MAGPIE_BUILTIN_MENU", "")
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))
		t.Cleanup(func() {
			builtinMenuActivated.Store(false)
		})
		builtinMenuActivated.Store(false)

		if ShouldExposeBuiltinMenu() {
			t.Errorf("ShouldExposeBuiltinMenu() = true, want false when unactivated")
		}
	})
}

func TestIsInstalledOrConfigured(t *testing.T) {
	// 1. Cloud Run K_SERVICE environment
	t.Run("cloud_run_k_service", func(t *testing.T) {
		t.Setenv("MAGPIE_BUILTIN_MENU", "")
		t.Setenv("K_SERVICE", "cliproxy-service")
		t.Setenv("AI_STUDIO_DATA_DIR", t.TempDir())
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		if !IsInstalledOrConfigured() {
			t.Errorf("IsInstalledOrConfigured() = false, want true when K_SERVICE is set")
		}
	})

	// 2. On-disk config.json
	t.Run("configured_on_disk_config_json", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("MAGPIE_BUILTIN_MENU", "")
		t.Setenv("K_SERVICE", "")
		t.Setenv("AI_STUDIO_DATA_DIR", tmpDir)
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		cfgPath := filepath.Join(tmpDir, "config.json")
		if err := os.WriteFile(cfgPath, []byte(`{"port":2048}`), 0644); err != nil {
			t.Fatalf("failed to write config.json: %v", err)
		}

		if !IsInstalledOrConfigured() {
			t.Errorf("IsInstalledOrConfigured() = false, want true when config.json exists")
		}
	})

	// 3. On-disk magpie-gateway.json
	t.Run("configured_on_disk_gateway_json", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("MAGPIE_BUILTIN_MENU", "")
		t.Setenv("K_SERVICE", "")
		t.Setenv("AI_STUDIO_DATA_DIR", tmpDir)
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		gwPath := filepath.Join(tmpDir, "magpie-gateway.json")
		if err := os.WriteFile(gwPath, []byte(`{"enabled":true}`), 0644); err != nil {
			t.Fatalf("failed to write magpie-gateway.json: %v", err)
		}

		if !IsInstalledOrConfigured() {
			t.Errorf("IsInstalledOrConfigured() = false, want true when magpie-gateway.json exists")
		}
	})

	// 4. Clean empty dir
	t.Run("unconfigured_empty_dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("MAGPIE_BUILTIN_MENU", "")
		t.Setenv("K_SERVICE", "")
		t.Setenv("AI_STUDIO_DATA_DIR", tmpDir)
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		if IsInstalledOrConfigured() {
			t.Errorf("IsInstalledOrConfigured() = true, want false in clean unconfigured directory")
		}
	})
}

func TestAutoStartBackground_ActivatesMenu(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AI_STUDIO_DATA_DIR", tmpDir)
	t.Setenv("MAGPIE_BUILTIN_MENU", "")
	t.Setenv("K_SERVICE", "")
	t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))
	t.Cleanup(func() {
		builtinMenuActivated.Store(false)
	})
	builtinMenuActivated.Store(false)

	// Write config.json so IsInstalledOrConfigured() is true
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"port":2048}`), 0644); err != nil {
		t.Fatalf("failed to write config.json: %v", err)
	}

	AutoStartBackground(context.Background(), "")
	if !ShouldExposeBuiltinMenu() {
		t.Errorf("ShouldExposeBuiltinMenu() = false, want true after AutoStartBackground() with configured backend")
	}
}

func TestGetInstallStatus_Camoufox(t *testing.T) {
	// 1. Local environment with camoufox directory present
	t.Run("camoufox_detected_locally", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("K_SERVICE", "")
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		camoufoxDir := filepath.Join(tmpDir, "runtime", "camoufox")
		if err := os.MkdirAll(camoufoxDir, 0755); err != nil {
			t.Fatalf("failed to create camoufox dir: %v", err)
		}

		st := GetInstallStatus(tmpDir)
		if !st.CamoufoxReady {
			t.Errorf("GetInstallStatus().CamoufoxReady = false, want true when runtime/camoufox exists")
		}
	})

	// 2. Local environment without camoufox directory
	t.Run("camoufox_missing_locally", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("K_SERVICE", "")
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		st := GetInstallStatus(tmpDir)
		if st.CamoufoxReady {
			t.Errorf("GetInstallStatus().CamoufoxReady = true, want false when runtime/camoufox is missing")
		}
	})

	// 3. Cloud Run container environment (K_SERVICE set) should never report camoufox ready
	t.Run("cloud_run_camoufox_disabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("K_SERVICE", "cliproxy-service")
		t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

		camoufoxDir := filepath.Join(tmpDir, "runtime", "camoufox")
		if err := os.MkdirAll(camoufoxDir, 0755); err != nil {
			t.Fatalf("failed to create camoufox dir: %v", err)
		}

		st := GetInstallStatus(tmpDir)
		if st.CamoufoxReady {
			t.Errorf("GetInstallStatus().CamoufoxReady = true, want false on Cloud Run (K_SERVICE)")
		}
	})
}

func TestAccountValidation_Relogin(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := NewDefaultConfig()
	ctx := context.Background()

	invalidIDs := []string{
		"",
		"   ",
		".hidden",
		"..",
		"foo/bar",
		"foo\\bar",
		"../escape",
		"acc\x00null",
		"acc\nnewline",
	}

	for _, id := range invalidIDs {
		_, err := ReloginAccount(ctx, tmpDir, cfg, "", id)
		if err == nil {
			t.Errorf("ReloginAccount with invalid ID %q expected error, got nil", id)
		}
	}
}

func TestNormalizeWorkerLimits(t *testing.T) {
	cfg := &MagpieConfig{
		PerAccountConcurrency:  0,
		WarmWorkerLimit:        0,
		MaxActiveWorkers:       0,
		WarmStartupConcurrency: 0,
	}
	warm, max, startup := cfg.NormalizeWorkerLimits()
	if warm < 1 {
		t.Errorf("warm = %d, want >= 1", warm)
	}
	if max < warm {
		t.Errorf("max = %d, want >= warm (%d)", max, warm)
	}
	if startup < 1 || startup > max {
		t.Errorf("startup = %d, want between 1 and max (%d)", startup, max)
	}
}

func TestGetInstallStatus_ManifestVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("K_SERVICE", "")
	t.Setenv("MAGPIE_PREINSTALLED_BIN", filepath.Join(t.TempDir(), "nonexistent"))

	// Create fake binary so installed is true
	binName := "aistudio2api"
	if os.PathSeparator == '\\' {
		binName = "aistudio2api.exe"
	}
	pkgDir := filepath.Join(tmpDir, "package")
	_ = os.MkdirAll(pkgDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgDir, binName), []byte("fake binary"), 0755)

	// 1. Current pinned version (0.3.0-magpie.1)
	manifestPath := filepath.Join(tmpDir, "install-manifest.json")
	_ = os.WriteFile(manifestPath, []byte(`{"version":"0.3.0-magpie.1"}`), 0644)
	stValid := GetInstallStatus(tmpDir)
	if !stValid.ManifestValid {
		t.Errorf("GetInstallStatus().ManifestValid = false, want true for version 0.3.0-magpie.1")
	}
	if stValid.Version != "0.3.0-magpie.1" {
		t.Errorf("GetInstallStatus().Version = %q, want '0.3.0-magpie.1'", stValid.Version)
	}

	// 2. Old version (0.2.19-magpie.1) should now be invalid
	_ = os.WriteFile(manifestPath, []byte(`{"version":"0.2.19-magpie.1"}`), 0644)
	stOld := GetInstallStatus(tmpDir)
	if stOld.ManifestValid {
		t.Errorf("GetInstallStatus().ManifestValid = true, want false for old version 0.2.19-magpie.1")
	}
}

func TestEnvKeys_StripThoughtSignaturesAndNoTemporaryChat(t *testing.T) {
	// 1. Check TEMPORARY_CHAT is completely absent from EnvKeys and SettingDefaults
	for _, key := range EnvKeys {
		if key == "TEMPORARY_CHAT" {
			t.Errorf("EnvKeys still contains obsolete TEMPORARY_CHAT")
		}
	}
	if _, ok := SettingDefaults["TEMPORARY_CHAT"]; ok {
		t.Errorf("SettingDefaults still contains obsolete TEMPORARY_CHAT")
	}

	// 2. Check STRIP_THOUGHT_SIGNATURES is in EnvKeys right after YOUTUBE_LINKS
	foundStrip := false
	for i, key := range EnvKeys {
		if key == "STRIP_THOUGHT_SIGNATURES" {
			foundStrip = true
			if i == 0 || EnvKeys[i-1] != "YOUTUBE_LINKS" {
				t.Errorf("STRIP_THOUGHT_SIGNATURES should be right after YOUTUBE_LINKS, but previous is %q", EnvKeys[i-1])
			}
			break
		}
	}
	if !foundStrip {
		t.Errorf("EnvKeys missing STRIP_THOUGHT_SIGNATURES")
	}

	// 3. Test SyncStudioEnv writes STRIP_THOUGHT_SIGNATURES and does not emit TEMPORARY_CHAT
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	_ = os.WriteFile(envPath, []byte("STRIP_THOUGHT_SIGNATURES=true\n"), 0600)
	studioCfg := &StudioConfigFile{
		Port:          2048,
		APIKey:        "as_test",
		AdminUsername: "magpie",
		AdminPassword: "pwd",
	}
	cfg := NewDefaultConfig()
	if err := SyncStudioEnv(tmpDir, studioCfg, cfg, ""); err != nil {
		t.Fatalf("SyncStudioEnv failed: %v", err)
	}
	writtenBytes, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	written := string(writtenBytes)
	if !strings.Contains(written, "STRIP_THOUGHT_SIGNATURES=true") {
		t.Errorf(".env missing STRIP_THOUGHT_SIGNATURES=true: %s", written)
	}
	if strings.Contains(written, "TEMPORARY_CHAT") {
		t.Errorf(".env should not contain TEMPORARY_CHAT: %s", written)
	}
}

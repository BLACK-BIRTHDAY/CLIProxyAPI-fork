package magpie

import (
	"context"
	"os"
	"path/filepath"
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

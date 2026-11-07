package helps

import (
	"regexp"
	"testing"

	"github.com/google/uuid"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
)

func TestDerivedSessionProviderMappings(t *testing.T) {
	t.Parallel()

	metadata := map[string]any{cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:test-root"}
	codexID := DerivedSessionUUID("codex", metadata)
	xaiID := DerivedSessionUUID("xai", metadata)
	if _, errParse := uuid.Parse(codexID); errParse != nil {
		t.Fatalf("Codex mapping %q is not a UUID: %v", codexID, errParse)
	}
	if _, errParse := uuid.Parse(xaiID); errParse != nil {
		t.Fatalf("xAI mapping %q is not a UUID: %v", xaiID, errParse)
	}
	if codexID == xaiID {
		t.Fatalf("provider namespaces produced the same UUID: %q", codexID)
	}
	if repeated := DerivedSessionUUID("codex", metadata); repeated != codexID {
		t.Fatalf("Codex mapping is not stable: first=%q repeated=%q", codexID, repeated)
	}

	antigravityID := DerivedAntigravitySessionID(metadata)
	if matched := regexp.MustCompile(`^-[0-9]+$`).MatchString(antigravityID); !matched {
		t.Fatalf("Antigravity mapping = %q, want negative decimal", antigravityID)
	}
	if repeated := DerivedAntigravitySessionID(metadata); repeated != antigravityID {
		t.Fatalf("Antigravity mapping is not stable: first=%q repeated=%q", antigravityID, repeated)
	}
}

func TestProviderSessionUUIDPrefersExecutionSession(t *testing.T) {
	t.Parallel()

	first := map[string]any{
		cliproxyexecutor.ExecutionSessionMetadataKey: "connection-1",
		cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:first-root",
	}
	second := map[string]any{
		cliproxyexecutor.ExecutionSessionMetadataKey: "connection-1",
		cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:second-root",
	}
	firstID := ProviderSessionUUID("codex", first)
	secondID := ProviderSessionUUID("codex", second)
	if firstID == "" || firstID != secondID {
		t.Fatalf("execution session did not stabilize provider UUID: first=%q second=%q", firstID, secondID)
	}
	if firstID == DerivedSessionUUID("codex", first) {
		t.Fatalf("provider UUID did not prefer execution session: %q", firstID)
	}
}

func TestDerivedSessionProviderMappingsRequireIdentity(t *testing.T) {
	t.Parallel()

	if got := DerivedSessionUUID("codex", nil); got != "" {
		t.Fatalf("DerivedSessionUUID() = %q, want empty", got)
	}
	if got := DerivedAntigravitySessionID(nil); got != "" {
		t.Fatalf("DerivedAntigravitySessionID() = %q, want empty", got)
	}
}

func TestInstallationSaltEnvOverride(t *testing.T) {
	const defaultSalt = "agy-win64:877f8cb6-6be4-41df-ac52-9bfb796f16fc:c8ae5f08-31c2-5519-b3e1-48dcc1ac8bcd"
	t.Setenv("ANTIGRAVITY_INSTALLATION_SALT", "")
	if got := InstallationSalt(); got != defaultSalt {
		t.Errorf("InstallationSalt() = %q, want default %q", got, defaultSalt)
	}

	metadata := map[string]any{cliproxyexecutor.DerivedSessionIDMetadataKey: "ctx:v1:test-root"}
	defaultAgyDec := DerivedAntigravitySessionID(metadata)
	defaultAgyUUID := DerivedSessionUUID("antigravity", metadata)
	defaultCodexUUID := DerivedSessionUUID("codex", metadata)

	t.Setenv("ANTIGRAVITY_INSTALLATION_SALT", "  custom-salt-123  ")
	if got := InstallationSalt(); got != "custom-salt-123" {
		t.Errorf("InstallationSalt() = %q, want %q", got, "custom-salt-123")
	}

	customAgyDec := DerivedAntigravitySessionID(metadata)
	customAgyUUID := DerivedSessionUUID("antigravity", metadata)
	customCodexUUID := DerivedSessionUUID("codex", metadata)

	if customAgyDec == defaultAgyDec {
		t.Errorf("DerivedAntigravitySessionID() should change with ANTIGRAVITY_INSTALLATION_SALT: %q", customAgyDec)
	}
	if customAgyUUID == defaultAgyUUID {
		t.Errorf("DerivedSessionUUID(antigravity) should change with ANTIGRAVITY_INSTALLATION_SALT: %q", customAgyUUID)
	}
	if customCodexUUID != defaultCodexUUID {
		t.Errorf("DerivedSessionUUID(codex) should NOT change with ANTIGRAVITY_INSTALLATION_SALT: got %q, want %q", customCodexUUID, defaultCodexUUID)
	}
}

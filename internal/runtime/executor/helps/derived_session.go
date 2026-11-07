package helps

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	cliproxysession "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/session"
)

// DerivedSessionID returns the first context-derived session identity in metadata order.
func DerivedSessionID(metadataSets ...map[string]any) string {
	for _, metadata := range metadataSets {
		if derivedID := cliproxysession.DerivedID(metadata); derivedID != "" {
			return derivedID
		}
	}
	return ""
}

// DerivedSessionUUID maps a derived session identity to a provider-scoped stable UUID.
func DerivedSessionUUID(provider string, metadataSets ...map[string]any) string {
	return stableProviderSessionUUID(provider, "derived-session", DerivedSessionID(metadataSets...))
}

// ProviderSessionUUID prefers a long-lived execution session and falls back to the derived identity.
func ProviderSessionUUID(provider string, metadataSets ...map[string]any) string {
	for _, metadata := range metadataSets {
		if executionID := metadataString(metadata, cliproxyexecutor.ExecutionSessionMetadataKey); executionID != "" {
			return stableProviderSessionUUID(provider, "execution-session", executionID)
		}
	}
	return DerivedSessionUUID(provider, metadataSets...)
}

const (
	// Private per-installation salt derived from this host's real Antigravity installation UUID
	// (server.go:1786 = 877f8cb6-6be4-41df-ac52-9bfb796f16fc / .updaterId = c8ae5f08-31c2-5519-b3e1-48dcc1ac8bcd)
	// so session IDs never collide with public CLIProxyAPI / ccextra / Antigravity-Manager cohorts.
	privateInstallationSalt = "agy-win64:877f8cb6-6be4-41df-ac52-9bfb796f16fc:c8ae5f08-31c2-5519-b3e1-48dcc1ac8bcd"
)

// InstallationSalt returns the per-installation salt, using ANTIGRAVITY_INSTALLATION_SALT env var if set.
func InstallationSalt() string {
	if salt := strings.TrimSpace(os.Getenv("ANTIGRAVITY_INSTALLATION_SALT")); salt != "" {
		return salt
	}
	return privateInstallationSalt
}

func stableProviderSessionUUID(provider string, kind string, identityValue string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	identityValue = strings.TrimSpace(identityValue)
	if provider == "" || identityValue == "" {
		return ""
	}
	namespace := "cli-proxy-api"
	if provider == "antigravity" {
		namespace = InstallationSalt()
	}
	identity := strings.Join([]string{namespace, provider, kind, identityValue}, "\x00")
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(identity)).String()
}

// DerivedAntigravitySessionID maps a derived session identity to Antigravity's negative decimal format.
func DerivedAntigravitySessionID(metadataSets ...map[string]any) string {
	derivedID := DerivedSessionID(metadataSets...)
	if derivedID == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(InstallationSalt() + ":antigravity:session\x00" + derivedID))
	value := int64(binary.BigEndian.Uint64(sum[:8])) & 0x7FFFFFFFFFFFFFFF
	return "-" + strconv.FormatInt(value, 10)
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

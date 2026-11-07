package misc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func overrideAntigravityVersionURLsForTest(t *testing.T, hubManifestURL string) func() {
	t.Helper()

	oldHubManifest := antigravityHubLatestManifestURL
	antigravityHubLatestManifestURL = hubManifestURL

	return func() {
		antigravityHubLatestManifestURL = oldHubManifest
	}
}

func overrideAntigravityVersionCacheForTest(t *testing.T, version string, expiry time.Time) func() {
	t.Helper()

	antigravityVersionMu.Lock()
	oldVersion := cachedAntigravityVersion
	oldExpiry := antigravityVersionExpiry
	cachedAntigravityVersion = version
	antigravityVersionExpiry = expiry
	antigravityVersionMu.Unlock()

	return func() {
		antigravityVersionMu.Lock()
		cachedAntigravityVersion = oldVersion
		antigravityVersionExpiry = oldExpiry
		antigravityVersionMu.Unlock()
	}
}

func TestAntigravityLatestVersionUsesCurrentHubFallback(t *testing.T) {
	restore := overrideAntigravityVersionCacheForTest(t, "", time.Time{})
	defer restore()

	version := AntigravityLatestVersion()
	if version != antigravityFallbackVersion {
		t.Fatalf("AntigravityLatestVersion() = %q, want %q", version, antigravityFallbackVersion)
	}
}

// Cloud Code resolves newer models only for clients reporting at least 2.23.0;
// older versions get 404 Requested entity was not found.
func TestAntigravityFallbackVersionMeetsBackendFloor(t *testing.T) {
	const floorMajor, floorMinor = 2, 23

	var major, minor, patch int
	if _, err := fmt.Sscanf(antigravityFallbackVersion, "%d.%d.%d", &major, &minor, &patch); err != nil {
		t.Fatalf("antigravityFallbackVersion = %q is not a dotted version: %v", antigravityFallbackVersion, err)
	}
	if major < floorMajor || (major == floorMajor && minor < floorMinor) {
		t.Fatalf("antigravityFallbackVersion = %q, want at least %d.%d.0", antigravityFallbackVersion, floorMajor, floorMinor)
	}
}

func TestAntigravityUserAgentUsesHubFamily(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    string
	}{
		{"2.23.0", "antigravity/hub/2.23.0 (aidev_client; os_type=windows; arch=amd64; cl=998057262)"},
		{"2.24.0", "antigravity/hub/2.24.0 (aidev_client; os_type=windows; arch=amd64; cl=1000335682)"},
		{"2.28.0", "antigravity/hub/2.28.0 (aidev_client; os_type=windows; arch=amd64; cl=1008300195)"},
		{"2.29.0", "antigravity/hub/2.29.0 (aidev_client; os_type=windows; arch=amd64; cl=1013060670)"},
		{"2.30.0", "antigravity/hub/2.30.0 (aidev_client; os_type=windows; arch=amd64)"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			restore := overrideAntigravityVersionCacheForTest(t, tc.version, time.Now().Add(time.Hour))
			defer restore()
			if got := AntigravityUserAgent(); got != tc.want {
				t.Fatalf("AntigravityUserAgent() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAntigravityFallbackVersionHasMatchingChangelist(t *testing.T) {
	if antigravityHubChangelists[antigravityFallbackVersion] == "" {
		t.Fatal("fallback version must be updated together with its verified changelist")
	}
}

func TestAntigravityVersionFromUserAgentParsesHubFamily(t *testing.T) {
	for _, userAgent := range []string{
		"antigravity/hub/2.28.0 windows/amd64",
		"antigravity/hub/2.28.0 (aidev_client; os_type=windows; arch=amd64; cl=1008300195)",
		"antigravity/hub/2.28.0 (aidev_client; os_type=windows; arch=amd64; cl=1008300195) google-api-nodejs-client/10.3.0",
	} {
		if got := AntigravityVersionFromUserAgent(userAgent); got != "2.28.0" {
			t.Fatalf("AntigravityVersionFromUserAgent(%q) = %q, want %q", userAgent, got, "2.28.0")
		}
	}
}

func TestAntigravityVersionFromUserAgentParsesLegacyFamily(t *testing.T) {
	if got := AntigravityVersionFromUserAgent("antigravity/1.23.2 windows/amd64"); got != "1.23.2" {
		t.Fatalf("AntigravityVersionFromUserAgent() = %q, want %q", got, "1.23.2")
	}
}

func TestAntigravityLoadCodeAssistUserAgentUsesShortUA(t *testing.T) {
	restore := overrideAntigravityVersionCacheForTest(t, "2.28.0", time.Now().Add(time.Hour))
	defer restore()

	want := "antigravity/hub/2.28.0 (aidev_client; os_type=windows; arch=amd64; cl=1008300195)"
	if got := AntigravityLoadCodeAssistUserAgent(""); got != want {
		t.Fatalf("AntigravityLoadCodeAssistUserAgent() = %q, want %q", got, want)
	}
	if got := AntigravityLoadCodeAssistUserAgent(want); got != want {
		t.Fatalf("AntigravityLoadCodeAssistUserAgent(configured) = %q, want %q", got, want)
	}
}

func TestAntigravityOnboardUserUserAgentUsesLongUA(t *testing.T) {
	restore := overrideAntigravityVersionCacheForTest(t, "2.28.0", time.Now().Add(time.Hour))
	defer restore()

	want := "antigravity/hub/2.28.0 (aidev_client; os_type=windows; arch=amd64; cl=1008300195) google-api-nodejs-client/10.3.0"
	if got := AntigravityOnboardUserUserAgent(""); got != want {
		t.Fatalf("AntigravityOnboardUserUserAgent() = %q, want %q", got, want)
	}
}

func TestFetchAntigravityLatestVersionUsesHubManifest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hub/latest-arm64-mac.yml":
			if got := r.Header.Get("User-Agent"); got != "electron-builder" {
				t.Errorf("hub manifest User-Agent = %q, want %q", got, "electron-builder")
			}
			if got := r.Header.Get("Cache-Control"); got != "no-cache" {
				t.Errorf("hub manifest Cache-Control = %q, want %q", got, "no-cache")
			}
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write([]byte("version: 2.2.1\npath: Antigravity-arm64-mac.zip\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	restore := overrideAntigravityVersionURLsForTest(t, server.URL+"/hub/latest-arm64-mac.yml")
	defer restore()

	version, errFetch := fetchAntigravityLatestVersion(context.Background())
	if errFetch != nil {
		t.Fatalf("fetchAntigravityLatestVersion() error = %v", errFetch)
	}
	if version != "2.2.1" {
		t.Fatalf("fetchAntigravityLatestVersion() = %q, want %q", version, "2.2.1")
	}
}

func TestFetchAntigravityLatestVersionReturnsHubManifestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary outage", http.StatusInternalServerError)
	}))
	defer server.Close()

	restore := overrideAntigravityVersionURLsForTest(t, server.URL+"/hub/latest-arm64-mac.yml")
	defer restore()

	_, errFetch := fetchAntigravityLatestVersion(context.Background())
	if errFetch == nil {
		t.Fatal("fetchAntigravityLatestVersion() error = nil, want error")
	}
}

type dummyRoundTripper struct{}

func (d *dummyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestCloneTransportWithHTTP11(t *testing.T) {
	// 1. With standard transport
	cloned := CloneTransportWithHTTP11(nil)
	if cloned == nil {
		t.Fatal("CloneTransportWithHTTP11(nil) returned nil with default transport")
	}
	if cloned.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be false")
	}
	if cloned.TLSNextProto == nil || len(cloned.TLSNextProto) != 0 {
		t.Error("TLSNextProto should be non-nil empty map")
	}
	if cloned.TLSClientConfig == nil || cloned.TLSClientConfig.NextProtos != nil {
		t.Error("TLSClientConfig.NextProtos should be nil")
	}

	// 2. When DefaultTransport is replaced by non-*http.Transport
	oldDef := http.DefaultTransport
	defer func() { http.DefaultTransport = oldDef }()
	http.DefaultTransport = &dummyRoundTripper{}
	if custom := CloneTransportWithHTTP11(nil); custom != nil {
		t.Errorf("expected nil when http.DefaultTransport is not *http.Transport, got %v", custom)
	}
}

func TestAntigravityBackgroundProbesDisabled(t *testing.T) {
	tests := []struct {
		envVal string
		want   bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"random", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{" True ", true},
	}
	for _, tc := range tests {
		t.Run("env="+tc.envVal, func(t *testing.T) {
			t.Setenv("ANTIGRAVITY_DISABLE_BACKGROUND_PROBES", tc.envVal)
			if got := AntigravityBackgroundProbesDisabled(); got != tc.want {
				t.Errorf("AntigravityBackgroundProbesDisabled() for %q = %v, want %v", tc.envVal, got, tc.want)
			}
		})
	}
}

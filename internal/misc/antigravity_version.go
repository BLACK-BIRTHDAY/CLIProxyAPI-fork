// Package misc provides miscellaneous utility functions for the CLI Proxy API server.
package misc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	// antigravityFallbackVersion is the client version reported when the hub
	// manifest has not been fetched yet or cannot be reached. Cloud Code rejects
	// newer models for clients below 2.23.0, so this floor must stay at or above
	// that version.
	// Matched pair captured from local Windows x64 Antigravity client:
	// version = "2.28.0", cl = "1008300195", os_type=windows; arch=amd64
	antigravityFallbackVersion = "2.28.0"
	antigravityHubPlatform     = "os_type=windows; arch=amd64"
	antigravityVersionCacheTTL = 6 * time.Hour
	antigravityFetchTimeout    = 10 * time.Second
	AntigravityNodeAPIClientUA = "google-api-nodejs-client/10.3.0"
	AntigravityGoogAPIClientUA = "gl-node/22.21.1"
)

var (
	antigravityHubLatestManifestURL = "https://antigravity-hub-auto-updater-974169037036.us-central1.run.app/manifest/latest.yml"
)

// Changelists belong to specific Hub releases, not to the latest manifest in
// general. Update the fallback release and its entry together.
// Only versions with a verified matching changelist may be used by the background updater
// so that an unverified remote version never emits a bare or mismatched UA.
var antigravityHubChangelists = map[string]string{
	"2.23.0": "998057262",
	"2.24.0": "1000335682",
	"2.28.0": "1008300195",
	"2.29.0": "1013060670",
}

type antigravityHubUpdaterManifest struct {
	Version string `yaml:"version"`
}

var (
	cachedAntigravityVersion = antigravityFallbackVersion
	antigravityVersionMu     sync.RWMutex
	antigravityVersionExpiry time.Time
	antigravityUpdaterOnce   sync.Once
)

// StartAntigravityVersionUpdater starts a background goroutine that periodically refreshes the cached antigravity version.
// This is intentionally decoupled from request execution to avoid blocking executors on version lookups.
func StartAntigravityVersionUpdater(ctx context.Context) {
	antigravityUpdaterOnce.Do(func() {
		go runAntigravityVersionUpdater(ctx)
	})
}

func runAntigravityVersionUpdater(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	ticker := time.NewTicker(antigravityVersionCacheTTL / 2)
	defer ticker.Stop()

	log.Infof("periodic antigravity version refresh started (interval=%s)", antigravityVersionCacheTTL/2)

	refreshAntigravityVersion(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshAntigravityVersion(ctx)
		}
	}
}

func refreshAntigravityVersion(ctx context.Context) {
	version, errFetch := fetchAntigravityLatestVersion(ctx)

	antigravityVersionMu.Lock()
	defer antigravityVersionMu.Unlock()

	now := time.Now()

	if errFetch == nil && antigravityHubChangelists[version] != "" {
		cachedAntigravityVersion = version
		antigravityVersionExpiry = now.Add(antigravityVersionCacheTTL)
		log.WithField("version", version).Info("fetched latest antigravity version")
		return
	}

	if cachedAntigravityVersion == "" || now.After(antigravityVersionExpiry) {
		cachedAntigravityVersion = antigravityFallbackVersion
		antigravityVersionExpiry = now.Add(antigravityVersionCacheTTL)
		log.WithError(errFetch).Warn("failed to refresh antigravity version, using fallback version")
		return
	}

	log.WithError(errFetch).Debug("failed to refresh antigravity version, keeping cached value")
}

// AntigravityLatestVersion returns the cached antigravity version refreshed by StartAntigravityVersionUpdater.
// It falls back to antigravityFallbackVersion if the cache is empty or stale.
func AntigravityLatestVersion() string {
	antigravityVersionMu.RLock()
	if cachedAntigravityVersion != "" && time.Now().Before(antigravityVersionExpiry) {
		v := cachedAntigravityVersion
		antigravityVersionMu.RUnlock()
		return v
	}
	antigravityVersionMu.RUnlock()

	return antigravityFallbackVersion
}

// AntigravityUserAgent returns the User-Agent string used by the Antigravity Hub family.
func AntigravityUserAgent() string {
	version := AntigravityLatestVersion()
	metadata := "aidev_client; " + antigravityHubPlatform
	if changelist := antigravityHubChangelists[version]; changelist != "" {
		metadata += "; cl=" + changelist
	}
	return fmt.Sprintf("antigravity/hub/%s (%s)", version, metadata)
}

func isAntigravityFamilyUserAgent(lower string) bool {
	return strings.HasPrefix(lower, "antigravity/hub/") || strings.HasPrefix(lower, "antigravity/")
}

func antigravityBaseUserAgent(userAgent string) string {
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		return AntigravityUserAgent()
	}
	lower := strings.ToLower(userAgent)
	if isAntigravityFamilyUserAgent(lower) {
		if idx := strings.Index(lower, " google-api-nodejs-client/"); idx >= 0 {
			trimmed := strings.TrimSpace(userAgent[:idx])
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return userAgent
}

// AntigravityRequestUserAgent returns the Antigravity runtime UA used by
// generate/stream/model-list requests.
func AntigravityRequestUserAgent(userAgent string) string {
	return antigravityBaseUserAgent(userAgent)
}

// AntigravityLoadCodeAssistUserAgent returns the Antigravity runtime UA used by
// loadCodeAssist requests.
func AntigravityLoadCodeAssistUserAgent(userAgent string) string {
	return AntigravityRequestUserAgent(userAgent)
}

// AntigravityOnboardUserUserAgent returns the long Antigravity control-plane UA
// used by onboardUser requests.
func AntigravityOnboardUserUserAgent(userAgent string) string {
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		return AntigravityUserAgent() + " " + AntigravityNodeAPIClientUA
	}
	lower := strings.ToLower(userAgent)
	if !isAntigravityFamilyUserAgent(lower) {
		return userAgent
	}
	if strings.Contains(lower, "google-api-nodejs-client/") {
		return userAgent
	}
	return antigravityBaseUserAgent(userAgent) + " " + AntigravityNodeAPIClientUA
}

// AntigravityVersionFromUserAgent extracts the Antigravity version prefix from
// either the short or long Antigravity UA forms.
func AntigravityVersionFromUserAgent(userAgent string) string {
	base := antigravityBaseUserAgent(userAgent)
	lower := strings.ToLower(base)
	if strings.HasPrefix(lower, "antigravity/hub/") {
		rest := base[len("antigravity/hub/"):]
		if idx := strings.IndexAny(rest, " 	"); idx >= 0 {
			rest = rest[:idx]
		}
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return AntigravityLatestVersion()
		}
		return rest
	}
	const legacyPrefix = "antigravity/"
	if !strings.HasPrefix(lower, legacyPrefix) {
		return AntigravityLatestVersion()
	}
	rest := base[len(legacyPrefix):]
	if idx := strings.IndexAny(rest, " 	"); idx >= 0 {
		rest = rest[:idx]
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return AntigravityLatestVersion()
	}
	return rest
}

func fetchAntigravityLatestVersion(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	client := &http.Client{Timeout: antigravityFetchTimeout}
	return fetchAntigravityHubLatestManifestVersion(ctx, client)
}

func fetchAntigravityHubLatestManifestVersion(ctx context.Context, client *http.Client) (string, error) {
	httpReq, errReq := http.NewRequestWithContext(ctx, http.MethodGet, antigravityHubLatestManifestURL, nil)
	if errReq != nil {
		return "", fmt.Errorf("build antigravity Hub updater manifest request: %w", errReq)
	}
	httpReq.Header.Set("User-Agent", "electron-builder")
	httpReq.Header.Set("Cache-Control", "no-cache")

	resp, errDo := client.Do(httpReq)
	if errDo != nil {
		return "", fmt.Errorf("fetch antigravity Hub updater manifest: %w", errDo)
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			log.WithError(errClose).Warn("antigravity Hub updater manifest response body close error")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("antigravity Hub updater manifest returned status %d", resp.StatusCode)
	}

	raw, errRead := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if errRead != nil {
		return "", fmt.Errorf("read antigravity Hub updater manifest: %w", errRead)
	}

	var manifest antigravityHubUpdaterManifest
	if errDecode := yaml.Unmarshal(raw, &manifest); errDecode != nil {
		return "", fmt.Errorf("decode antigravity Hub updater manifest: %w", errDecode)
	}

	version := strings.TrimSpace(manifest.Version)
	if version == "" {
		return "", errors.New("antigravity Hub updater manifest returned empty version")
	}
	if !isValidAntigravitySemVersion(version) {
		return "", fmt.Errorf("antigravity Hub updater manifest returned invalid version %q", version)
	}
	return version, nil
}

func isValidAntigravitySemVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}

	return true
}

// CloneTransportWithHTTP11 clones the base HTTP transport (or http.DefaultTransport
// if base is nil) and configures it for strict HTTP/1.1 usage without ALPN or HTTP/2 upgrades.
// If base is nil and http.DefaultTransport is not a *http.Transport (e.g. replaced by custom test RoundTripper),
// it returns nil so callers preserve their custom transport.
func CloneTransportWithHTTP11(base *http.Transport) *http.Transport {
	if base == nil {
		defTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok || defTransport == nil {
			return nil
		}
		base = defTransport
	}
	clone := base.Clone()
	clone.ForceAttemptHTTP2 = false
	// Wipe TLSNextProto to prevent implicit HTTP/2 upgrade.
	clone.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	if clone.TLSClientConfig == nil {
		clone.TLSClientConfig = &tls.Config{}
	} else {
		clone.TLSClientConfig = clone.TLSClientConfig.Clone()
	}
	// Native Antigravity sends no ALPN extension. With HTTP/2 disabled above,
	// an empty NextProtos keeps the wire shape aligned while using HTTP/1.1.
	clone.TLSClientConfig.NextProtos = nil
	return clone
}

// AntigravityBackgroundProbesDisabled reports whether background probes such as
// loadCodeAssist (credits refresh) and fetchAvailableModels (model discovery)
// are disabled via the ANTIGRAVITY_DISABLE_BACKGROUND_PROBES environment variable.
func AntigravityBackgroundProbesDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ANTIGRAVITY_DISABLE_BACKGROUND_PROBES")))
	return v == "1" || v == "true"
}

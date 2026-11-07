package magpie

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// MagpiePrefix is the standard canonical prefix for MAGPIE models.
	MagpiePrefix = "(MAGPIE)/"
	// MagpieLegacyPrefix is the legacy prefix without slash.
	MagpieLegacyPrefix = "(MAGPIE)"
	// MagpieFullwidthPrefix is the full-width parenthesis prefix with slash.
	MagpieFullwidthPrefix = "（MAGPIE）/"
	// MagpieFullwidthLegacyPrefix is the full-width parenthesis prefix without slash.
	MagpieFullwidthLegacyPrefix = "（MAGPIE）"
)

// StripMagpiePrefix strips any variant of (MAGPIE)/, (MAGPIE), （MAGPIE）/, or （MAGPIE）
// as well as optional leading or inner "models/" prefixes and leading slashes.
// Returns the raw model ID and true if a MAGPIE prefix was present, or ("", false) if not.
func StripMagpiePrefix(model string) (string, bool) {
	s := strings.TrimLeft(model, "/")
	s = strings.TrimPrefix(s, "models/")

	var stripped string
	var matched bool

	if strings.HasPrefix(s, MagpiePrefix) {
		stripped = strings.TrimPrefix(s, MagpiePrefix)
		matched = true
	} else if strings.HasPrefix(s, MagpieLegacyPrefix) {
		stripped = strings.TrimPrefix(s, MagpieLegacyPrefix)
		matched = true
	} else if strings.HasPrefix(s, MagpieFullwidthPrefix) {
		stripped = strings.TrimPrefix(s, MagpieFullwidthPrefix)
		matched = true
	} else if strings.HasPrefix(s, MagpieFullwidthLegacyPrefix) {
		stripped = strings.TrimPrefix(s, MagpieFullwidthLegacyPrefix)
		matched = true
	}

	if !matched {
		return "", false
	}

	stripped = strings.TrimLeft(stripped, "/")
	stripped = strings.TrimPrefix(stripped, "models/")
	return stripped, true
}

// PrefixMagpieModel normalizes a raw model ID into the canonical "(MAGPIE)/<model>" format,
// stripping any existing (MAGPIE) or "models/" prefixes first.
func PrefixMagpieModel(rawModel string) string {
	clean := strings.TrimLeft(rawModel, "/")
	clean = strings.TrimPrefix(clean, "models/")
	if stripped, ok := StripMagpiePrefix(clean); ok {
		clean = stripped
	}
	return MagpiePrefix + clean
}

// IsMagpieModel reports whether model carries any MAGPIE namespace prefix.
func IsMagpieModel(model string) bool {
	_, ok := StripMagpiePrefix(model)
	return ok
}

// MagpieModelInfo describes a filtered model supported by the MAGPIE backend.
type MagpieModelInfo struct {
	ID                         string   `json:"id"`
	PrefixedID                 string   `json:"prefixed_id"`
	DisplayName                string   `json:"display_name"`
	Description                string   `json:"description,omitempty"`
	InputTokenLimit            int64    `json:"input_token_limit"`
	OutputTokenLimit           int64    `json:"output_token_limit"`
	ThinkingLevels             []string `json:"thinking_levels,omitempty"`
	SupportsTools              bool     `json:"supports_tools"`
	IsImage                    bool     `json:"is_image"`
	SupportedGenerationMethods []string `json:"supported_generation_methods,omitempty"`
}

var reExclude = regexp.MustCompile(`(?i)(?:image|nano-banana|tts|audio|live|transcribe|robotics|research|video|music|canvas)`)

var nonTextCaps = []string{
	"image_route",
	"speech_route",
	"live_route",
	"aspect_ratio",
	"output_resolution",
	"interactions_api",
	"transcription_output",
}

var imageCaps = map[string]bool{
	"image_route":       true,
	"aspect_ratio":      true,
	"output_resolution": true,
}

// FilterStudioModels filters raw models returned by GET /v1/models from AIStudio2API.
// It is an exact port of filter_studio_models in antigravity tools.
func FilterStudioModels(rawModels []map[string]any, allowImageModels bool) []MagpieModelInfo {
	result := make([]MagpieModelInfo, 0)
	seen := make(map[string]bool)

	listsChannels := false
	for _, item := range rawModels {
		if c, ok := item["channels"].([]any); ok && len(c) > 0 {
			listsChannels = true
			break
		}
	}

	for _, item := range rawModels {
		id, ok := item["id"].(string)
		if !ok || id == "" || !strings.HasPrefix(id, "gemini-") {
			continue
		}
		lowerID := strings.ToLower(id)
		if strings.HasSuffix(lowerID, "-build") || strings.HasSuffix(lowerID, "-search") {
			continue
		}

		// supported_generation_methods
		var methods []string
		if mArr, ok := item["supported_generation_methods"].([]any); ok {
			for _, m := range mArr {
				if s, ok := m.(string); ok {
					methods = append(methods, s)
				}
			}
		}
		if len(methods) > 0 {
			hasGenContent := false
			for _, m := range methods {
				if m == "generateContent" {
					hasGenContent = true
					break
				}
			}
			if !hasGenContent {
				continue
			}
		}

		// capabilities
		caps, _ := item["capabilities"].(map[string]any)
		chatModel := true
		if caps != nil {
			if v, ok := caps["chat_model"].(bool); ok {
				chatModel = v
			}
		}
		assistantRoute := false
		if caps != nil {
			if v, ok := caps["assistant_route"].(bool); ok {
				assistantRoute = v
			}
		}
		if !chatModel || assistantRoute {
			continue
		}

		isImage := false
		if caps != nil {
			if v, ok := caps["image_route"].(bool); ok {
				isImage = v
			}
		}
		if isImage {
			if !allowImageModels {
				continue
			}
		} else if reExclude.MatchString(id) {
			continue
		}

		// Check non-text capabilities
		if caps != nil {
			hasDisallowedNonText := false
			for _, capName := range nonTextCaps {
				if isSet, ok := caps[capName].(bool); ok && isSet {
					if !(isImage && imageCaps[capName]) {
						hasDisallowedNonText = true
						break
					}
				}
			}
			if hasDisallowedNonText {
				continue
			}
		}

		// Channels check
		if listsChannels {
			cArr, ok := item["channels"].([]any)
			if !ok || len(cArr) == 0 {
				continue
			}
		}

		if seen[id] {
			continue
		}
		seen[id] = true

		parseTokenLimit := func(key string) int64 {
			if val, ok := item[key]; ok {
				switch v := val.(type) {
				case float64:
					return int64(v)
				case int64:
					return v
				case int:
					return int64(v)
				}
			}
			return 0
		}

		upstreamInput := parseTokenLimit("input_token_limit")
		var inputTokenLimit int64
		if upstreamInput > 0 {
			inputTokenLimit = upstreamInput
		} else if isImage {
			inputTokenLimit = 32768
		} else {
			inputTokenLimit = 1048576
		}

		upstreamOutput := parseTokenLimit("output_token_limit")
		var outputTokenLimit int64
		if upstreamOutput > 0 {
			outputTokenLimit = upstreamOutput
		} else if isImage {
			outputTokenLimit = 8192
		} else {
			outputTokenLimit = 65536
		}

		displayName := id
		if name, ok := item["name"].(string); ok && name != "" {
			displayName = name
		}
		var description string
		if desc, ok := item["description"].(string); ok {
			description = desc
		}

		supportsTools := false
		if caps != nil {
			if st, ok := caps["function_declarations"].(bool); ok {
				supportsTools = st
			}
		}

		var thinkingLevels []string
		if tArr, ok := item["thinking_levels"].([]any); ok {
			for _, t := range tArr {
				if s, ok := t.(string); ok {
					thinkingLevels = append(thinkingLevels, s)
				}
			}
		} else if caps != nil {
			tVal, _ := caps["thinking"].(bool)
			tlVal, _ := caps["thinking_level"].(bool)
			if tVal || tlVal {
				thinkingLevels = []string{"minimal", "low", "medium", "high"}
			}
		}

		prefixedID := PrefixMagpieModel(id)
		result = append(result, MagpieModelInfo{
			ID:                         id,
			PrefixedID:                 prefixedID,
			DisplayName:                displayName,
			Description:                description,
			InputTokenLimit:            inputTokenLimit,
			OutputTokenLimit:           outputTokenLimit,
			ThinkingLevels:             thinkingLevels,
			SupportsTools:              supportsTools,
			IsImage:                    isImage,
			SupportedGenerationMethods: methods,
		})
	}

	return result
}

type modelsCacheEntry struct {
	models           []MagpieModelInfo
	dataDir          string
	allowImageModels bool
	expiresAt        time.Time
}

var (
	modelsCacheMu sync.RWMutex
	modelsCache   *modelsCacheEntry
)

// InvalidateModelsCache flushes the in-memory models cache.
func InvalidateModelsCache() {
	modelsCacheMu.Lock()
	defer modelsCacheMu.Unlock()
	modelsCache = nil
}

// FetchMagpieModels queries the active MAGPIE backend for models, caches them to
// <data_dir>/models-cache.json on success, and falls back to cached models if the backend
// is warming up or temporarily unreachable.
func FetchMagpieModels(ctx context.Context, inheritedProxy string) []MagpieModelInfo {
	if override := GetTestBackendOverride(); override != nil {
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: nil,
				DialContext: (&net.Dialer{
					Timeout:   1500 * time.Millisecond,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
			Timeout: 3 * time.Second,
		}
		req, errReq := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/models", override.BaseURL), nil)
		if errReq == nil {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", override.APIKey))
			if res, errDo := client.Do(req); errDo == nil {
				defer func() {
					if errClose := res.Body.Close(); errClose != nil {
						log.Debugf("failed to close test override models response body: %v", errClose)
					}
				}()
				if res.StatusCode == http.StatusOK {
					var body map[string]any
					if errDecode := json.NewDecoder(res.Body).Decode(&body); errDecode == nil {
						if dataArr, ok := body["data"].([]any); ok {
							var raw []map[string]any
							for _, item := range dataArr {
								if m, ok := item.(map[string]any); ok {
									raw = append(raw, m)
								}
							}
							return FilterStudioModels(raw, override.Config.ImageModels)
						}
					}
				}
			}
		}
		return nil
	}

	dataDir := ResolveDataDir("")
	cfg, errLoad := LoadGatewayConfig(dataDir, inheritedProxy)
	if errLoad != nil || !cfg.Enabled {
		return nil
	}

	// 1. Fast in-memory cache check
	modelsCacheMu.RLock()
	if modelsCache != nil && modelsCache.dataDir == dataDir && modelsCache.allowImageModels == cfg.ImageModels && time.Now().Before(modelsCache.expiresAt) {
		res := make([]MagpieModelInfo, len(modelsCache.models))
		copy(res, modelsCache.models)
		modelsCacheMu.RUnlock()
		return res
	}
	modelsCacheMu.RUnlock()

	cachePath := filepath.Join(dataDir, "models-cache.json")
	loadCache := func() []MagpieModelInfo {
		modelsCacheMu.RLock()
		if modelsCache != nil && len(modelsCache.models) > 0 {
			res := make([]MagpieModelInfo, len(modelsCache.models))
			copy(res, modelsCache.models)
			modelsCacheMu.RUnlock()
			return res
		}
		modelsCacheMu.RUnlock()

		if content, errRead := os.ReadFile(cachePath); errRead == nil {
			var cached []MagpieModelInfo
			if errUnmarshal := json.Unmarshal(content, &cached); errUnmarshal == nil && len(cached) > 0 {
				modelsCacheMu.Lock()
				modelsCache = &modelsCacheEntry{
					models:           cached,
					dataDir:          dataDir,
					allowImageModels: cfg.ImageModels,
					expiresAt:        time.Now().Add(15 * time.Second),
				}
				modelsCacheMu.Unlock()
				return cached
			}
		}
		return nil
	}

	hasAnyCache := false
	modelsCacheMu.RLock()
	if modelsCache != nil && len(modelsCache.models) > 0 {
		hasAnyCache = true
	}
	modelsCacheMu.RUnlock()
	if !hasAnyCache {
		if fi, errStat := os.Stat(cachePath); errStat == nil && fi.Size() > 0 {
			hasAnyCache = true
		}
	}

	configPath := filepath.Join(dataDir, "config.json")
	content, errReadConfig := os.ReadFile(configPath)
	if errReadConfig != nil {
		return loadCache()
	}
	var studioCfg StudioConfigFile
	if errUnmarshal := json.Unmarshal(content, &studioCfg); errUnmarshal != nil || studioCfg.Port <= 0 {
		return loadCache()
	}

	dialTimeout := 5 * time.Second
	if hasAnyCache {
		dialTimeout = 1500 * time.Millisecond
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/v1/models", studioCfg.Port)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: nil, // Loopback connection must bypass outbound proxy
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
		Timeout: dialTimeout,
	}
	req, errReq := http.NewRequestWithContext(ctx, "GET", url, nil)
	if errReq != nil {
		return loadCache()
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", studioCfg.APIKey))

	res, errDo := client.Do(req)
	if errDo != nil || res.StatusCode != http.StatusOK {
		if res != nil && res.Body != nil {
			if errClose := res.Body.Close(); errClose != nil {
				log.Debugf("failed to close models response body: %v", errClose)
			}
		}
		// If unreachable and auto_start enabled, trigger background warm-up
		if cfg.AutoStart && HasImportedAccounts(dataDir) {
			go func() {
				_ = EnsureRunning(context.Background(), dataDir, cfg, inheritedProxy)
			}()
		}
		return loadCache()
	}
	defer func() {
		if errClose := res.Body.Close(); errClose != nil {
			log.Debugf("failed to close models response body: %v", errClose)
		}
	}()

	bodyBytes, errReadBody := io.ReadAll(res.Body)
	if errReadBody != nil {
		return loadCache()
	}
	var body map[string]any
	if errUnmarshal := json.Unmarshal(bodyBytes, &body); errUnmarshal != nil {
		return loadCache()
	}

	dataArr, ok := body["data"].([]any)
	if !ok {
		return loadCache()
	}
	var rawModels []map[string]any
	for _, item := range dataArr {
		if m, ok := item.(map[string]any); ok {
			rawModels = append(rawModels, m)
		}
	}

	filtered := FilterStudioModels(rawModels, cfg.ImageModels)
	if len(filtered) > 0 {
		modelsCacheMu.Lock()
		modelsCache = &modelsCacheEntry{
			models:           filtered,
			dataDir:          dataDir,
			allowImageModels: cfg.ImageModels,
			expiresAt:        time.Now().Add(15 * time.Second),
		}
		modelsCacheMu.Unlock()

		if cacheBytes, errMarshal := json.MarshalIndent(filtered, "", "  "); errMarshal == nil {
			_ = os.WriteFile(cachePath, append(cacheBytes, '\n'), 0644)
		}
	}
	return filtered
}

// SynthesizeGeminiModelMap returns a synthesized Gemini /v1beta/models response entry
// for a model when not explicitly found in the active catalog.
func SynthesizeGeminiModelMap(rawModel string) map[string]any {
	return ToGeminiModelMap(MagpieModelInfo{
		ID:                         rawModel,
		PrefixedID:                 PrefixMagpieModel(rawModel),
		DisplayName:                PrefixMagpieModel(rawModel),
		Description:                fmt.Sprintf("Google AI Studio model: %s", rawModel),
		InputTokenLimit:            1048576,
		OutputTokenLimit:           65536,
		SupportedGenerationMethods: []string{"generateContent", "streamGenerateContent", "countTokens"},
	})
}

// ToOpenAIModelMap converts a MagpieModelInfo into an OpenAI /v1/models response entry,
// setting owned_by="magpie" and id=prefixed_id.
func ToOpenAIModelMap(m MagpieModelInfo) map[string]any {
	return map[string]any{
		"id":                 m.PrefixedID,
		"object":             "model",
		"created":            1700000000,
		"owned_by":           "magpie",
		"display_name":       m.PrefixedID,
		"input_token_limit":  m.InputTokenLimit,
		"output_token_limit": m.OutputTokenLimit,
	}
}

// ToGeminiModelMap converts a MagpieModelInfo into a Gemini /v1beta/models response entry,
// setting name="models/(MAGPIE)/..." and displayName="(MAGPIE)/...".
func ToGeminiModelMap(m MagpieModelInfo) map[string]any {
	return map[string]any{
		"name":                       "models/" + m.PrefixedID,
		"displayName":                m.PrefixedID,
		"description":                m.Description,
		"inputTokenLimit":            m.InputTokenLimit,
		"outputTokenLimit":           m.OutputTokenLimit,
		"supportedGenerationMethods": m.SupportedGenerationMethods,
	}
}

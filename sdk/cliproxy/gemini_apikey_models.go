package cliproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MoeForge/moego/ojson"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/modelconfig"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/thinking"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/proxyutil"
	log "github.com/sirupsen/logrus"
)

type geminiUpstreamModelMetadata struct {
	ID                         string
	Version                    string
	DisplayName                string
	Description                string
	InputTokenLimit            int
	OutputTokenLimit           int
	SupportedGenerationMethods []string
	SupportedInputModalities   []string
	SupportedOutputModalities  []string
	Thinking                   bool
	ThinkingLevels             []string
	Capabilities               geminiCapabilities
}

type geminiCapabilities struct {
	MediaResolution bool
	AgenticVideo    bool
	ImageRoute      bool
	ImageSearch     bool
	SpeechRoute     bool
	GoogleSearch    bool
	Thinking        bool
}

type geminiUpstreamCacheEntry struct {
	expiresAt time.Time
	models    []geminiUpstreamModelMetadata
}

var (
	geminiUpstreamModelsNowFunc = time.Now
	geminiUpstreamModelsCache   = make(map[string]geminiUpstreamCacheEntry)
	geminiUpstreamModelsMu      sync.RWMutex
)

// enrichGeminiConfigModelsFromUpstream queries the upstream /v1beta/models endpoint
// for the given Gemini API key entry and enriches matching configured models with
// upstream metadata (token limits, modalities, thinking, capabilities, etc.).
func (s *Service) enrichGeminiConfigModelsFromUpstream(ctx context.Context, cfg *config.Config, auth *coreauth.Auth, entry *config.GeminiKey, models []*ModelInfo) []*ModelInfo {
	if entry == nil || strings.TrimSpace(entry.BaseURL) == "" || len(models) == 0 {
		return models
	}

	var proxyURL string
	if strings.TrimSpace(entry.ProxyURL) != "" {
		proxyURL = strings.TrimSpace(entry.ProxyURL)
	} else if auth != nil && strings.TrimSpace(auth.ProxyURL) != "" {
		proxyURL = strings.TrimSpace(auth.ProxyURL)
	} else if cfg != nil && strings.TrimSpace(cfg.ProxyURL) != "" {
		proxyURL = strings.TrimSpace(cfg.ProxyURL)
	}

	cacheKey := fmt.Sprintf("%s|%s|%s", strings.TrimRight(strings.TrimSpace(entry.BaseURL), "/"), strings.TrimSpace(entry.APIKey), proxyURL)

	now := geminiUpstreamModelsNowFunc()
	geminiUpstreamModelsMu.RLock()
	cached, hasCache := geminiUpstreamModelsCache[cacheKey]
	geminiUpstreamModelsMu.RUnlock()

	var upstreamModels []geminiUpstreamModelMetadata
	if hasCache && now.Before(cached.expiresAt) {
		upstreamModels = cached.models
	} else {
		fetched, err := fetchGeminiUpstreamModels(ctx, entry, proxyURL)
		if err != nil {
			log.Debugf("gemini upstream models fetch failed: %v", err)
			if hasCache {
				upstreamModels = cached.models
			}
		} else {
			upstreamModels = fetched
			geminiUpstreamModelsMu.Lock()
			geminiUpstreamModelsCache[cacheKey] = geminiUpstreamCacheEntry{
				expiresAt: now.Add(registry.ModelsRefreshInterval),
				models:    fetched,
			}
			geminiUpstreamModelsMu.Unlock()
		}
	}

	if len(upstreamModels) == 0 {
		return models
	}

	for _, info := range models {
		if info == nil {
			continue
		}
		for _, upstream := range upstreamModels {
			if !matchGeminiModel(info, upstream.ID) {
				continue
			}
			applyGeminiUpstreamMetadata(info, upstream)
			break
		}
	}

	return models
}

func fetchGeminiUpstreamModels(ctx context.Context, entry *config.GeminiKey, proxyURL string) ([]geminiUpstreamModelMetadata, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(entry.BaseURL), "/")
	var fetchURL string
	if strings.HasSuffix(baseURL, "/v1beta/models") {
		fetchURL = baseURL
	} else if strings.HasSuffix(baseURL, "/v1beta") {
		fetchURL = baseURL + "/models"
	} else {
		fetchURL = baseURL + "/v1beta/models"
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if errReq != nil {
		return nil, errReq
	}

	if apiKey := strings.TrimSpace(entry.APIKey); apiKey != "" {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	for k, v := range entry.Headers {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}

	client := &http.Client{}
	if proxyURL != "" {
		transport, _, errBuild := proxyutil.BuildHTTPTransport(proxyURL)
		if errBuild != nil {
			return nil, errBuild
		}
		if transport != nil {
			client.Transport = transport
		}
	}

	resp, errDo := client.Do(req)
	if errDo != nil {
		return nil, errDo
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini upstream models endpoint returned status %d", resp.StatusCode)
	}

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return nil, errRead
	}

	return parseGeminiUpstreamModels(body)
}

func parseGeminiUpstreamModels(body []byte) ([]geminiUpstreamModelMetadata, error) {
	if !ojson.ValidBytes(body) {
		return nil, fmt.Errorf("invalid json response from upstream models endpoint")
	}
	root := ojson.ParseBytes(body)
	var modelsNode ojson.Result
	if mn := root.Get("models"); mn.IsArray() {
		modelsNode = mn
	} else if root.IsArray() {
		modelsNode = root
	} else {
		return nil, nil
	}

	items := modelsNode.Array()
	out := make([]geminiUpstreamModelMetadata, 0, len(items))
	for _, item := range items {
		rawName := item.Get("name").String()
		if rawName == "" {
			rawName = item.Get("id").String()
		}
		modelID := strings.TrimPrefix(strings.TrimSpace(rawName), "models/")
		if modelID == "" {
			continue
		}

		meta := geminiUpstreamModelMetadata{
			ID:          modelID,
			Version:     strings.TrimSpace(item.Get("version").String()),
			DisplayName: strings.TrimSpace(item.Get("displayName").String()),
			Description: strings.TrimSpace(item.Get("description").String()),
		}
		if meta.DisplayName == "" {
			meta.DisplayName = strings.TrimSpace(item.Get("display_name").String())
		}

		inputLimit := int(item.Get("inputTokenLimit").Int())
		if inputLimit <= 0 {
			inputLimit = int(item.Get("input_token_limit").Int())
		}
		meta.InputTokenLimit = inputLimit

		outputLimit := int(item.Get("outputTokenLimit").Int())
		if outputLimit <= 0 {
			outputLimit = int(item.Get("output_token_limit").Int())
		}
		meta.OutputTokenLimit = outputLimit

		if sgm := item.Get("supportedGenerationMethods"); sgm.IsArray() {
			for _, m := range sgm.Array() {
				if s := strings.TrimSpace(m.String()); s != "" {
					meta.SupportedGenerationMethods = append(meta.SupportedGenerationMethods, s)
				}
			}
		} else if sgm := item.Get("supported_generation_methods"); sgm.IsArray() {
			for _, m := range sgm.Array() {
				if s := strings.TrimSpace(m.String()); s != "" {
					meta.SupportedGenerationMethods = append(meta.SupportedGenerationMethods, s)
				}
			}
		}

		if sim := item.Get("supportedInputModalities"); sim.IsArray() {
			for _, m := range sim.Array() {
				if s := strings.ToLower(strings.TrimSpace(m.String())); s != "" {
					meta.SupportedInputModalities = append(meta.SupportedInputModalities, s)
				}
			}
		} else if sim := item.Get("supported_input_modalities"); sim.IsArray() {
			for _, m := range sim.Array() {
				if s := strings.ToLower(strings.TrimSpace(m.String())); s != "" {
					meta.SupportedInputModalities = append(meta.SupportedInputModalities, s)
				}
			}
		}

		if som := item.Get("supportedOutputModalities"); som.IsArray() {
			for _, m := range som.Array() {
				if s := strings.ToLower(strings.TrimSpace(m.String())); s != "" {
					meta.SupportedOutputModalities = append(meta.SupportedOutputModalities, s)
				}
			}
		} else if som := item.Get("supported_output_modalities"); som.IsArray() {
			for _, m := range som.Array() {
				if s := strings.ToLower(strings.TrimSpace(m.String())); s != "" {
					meta.SupportedOutputModalities = append(meta.SupportedOutputModalities, s)
				}
			}
		}

		meta.Thinking = item.Get("thinking").Bool()

		if tl := item.Get("thinkingLevels"); tl.IsArray() {
			for _, l := range tl.Array() {
				if s := strings.TrimSpace(l.String()); s != "" {
					meta.ThinkingLevels = append(meta.ThinkingLevels, s)
				}
			}
		} else if tl := item.Get("thinking_levels"); tl.IsArray() {
			for _, l := range tl.Array() {
				if s := strings.TrimSpace(l.String()); s != "" {
					meta.ThinkingLevels = append(meta.ThinkingLevels, s)
				}
			}
		}

		capsObj := item.Get("capabilities")
		if capsObj.IsObject() {
			meta.Capabilities.MediaResolution = capsObj.Get("media_resolution").Bool()
			meta.Capabilities.AgenticVideo = capsObj.Get("agentic_video").Bool()
			meta.Capabilities.ImageRoute = capsObj.Get("image_route").Bool()
			meta.Capabilities.ImageSearch = capsObj.Get("image_search").Bool()
			meta.Capabilities.SpeechRoute = capsObj.Get("speech_route").Bool()
			meta.Capabilities.GoogleSearch = capsObj.Get("google_search").Bool()
			meta.Capabilities.Thinking = capsObj.Get("thinking").Bool()
		} else if capsObj.IsArray() {
			for _, c := range capsObj.Array() {
				flag := strings.ToLower(strings.TrimSpace(c.String()))
				switch flag {
				case "media_resolution":
					meta.Capabilities.MediaResolution = true
				case "agentic_video":
					meta.Capabilities.AgenticVideo = true
				case "image_route":
					meta.Capabilities.ImageRoute = true
				case "image_search":
					meta.Capabilities.ImageSearch = true
				case "speech_route":
					meta.Capabilities.SpeechRoute = true
				case "google_search":
					meta.Capabilities.GoogleSearch = true
				case "thinking":
					meta.Capabilities.Thinking = true
				}
			}
		}

		out = append(out, meta)
	}

	return out, nil
}

func matchGeminiModel(info *ModelInfo, upstreamID string) bool {
	normUpstream := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(upstreamID), "models/"))
	if normUpstream == "" {
		return false
	}
	candidates := []string{info.MetadataModelID, info.ID}
	for _, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			continue
		}
		candNorm := strings.ToLower(strings.TrimPrefix(cand, "models/"))
		if candNorm == normUpstream {
			return true
		}
		baseName := strings.TrimSpace(thinking.ParseSuffix(cand).ModelName)
		baseNorm := strings.ToLower(strings.TrimPrefix(baseName, "models/"))
		if baseNorm == normUpstream {
			return true
		}
	}
	return false
}

func applyGeminiUpstreamMetadata(info *ModelInfo, upstream geminiUpstreamModelMetadata) {
	if upstream.Version != "" {
		info.Version = upstream.Version
	}
	if upstream.Description != "" {
		info.Description = upstream.Description
	}

	if (info.DisplayName == "" ||
		strings.EqualFold(info.DisplayName, info.MetadataModelID) ||
		strings.EqualFold(info.DisplayName, info.ID) ||
		strings.EqualFold(info.DisplayName, strings.TrimPrefix(info.MetadataModelID, "models/")) ||
		strings.EqualFold(info.DisplayName, strings.TrimPrefix(info.ID, "models/"))) &&
		upstream.DisplayName != "" {
		info.DisplayName = upstream.DisplayName
	}

	if info.MaxContextLength <= 0 && upstream.InputTokenLimit > 0 {
		info.InputTokenLimit = upstream.InputTokenLimit
		info.ContextLength = upstream.InputTokenLimit
	}
	if upstream.OutputTokenLimit > 0 {
		info.OutputTokenLimit = upstream.OutputTokenLimit
		info.MaxCompletionTokens = upstream.OutputTokenLimit
	}

	if len(upstream.SupportedGenerationMethods) > 0 {
		info.SupportedGenerationMethods = append([]string(nil), upstream.SupportedGenerationMethods...)
	}

	if len(upstream.SupportedInputModalities) > 0 {
		info.SupportedInputModalities = append([]string(nil), upstream.SupportedInputModalities...)
	} else if len(info.SupportedInputModalities) == 0 {
		if upstream.Capabilities.MediaResolution || upstream.Capabilities.AgenticVideo {
			info.SupportedInputModalities = []string{"text", "image", "audio", "video"}
		} else if upstream.Capabilities.ImageRoute || upstream.Capabilities.ImageSearch {
			info.SupportedInputModalities = []string{"text", "image"}
		}
	}

	if len(upstream.SupportedOutputModalities) > 0 {
		info.SupportedOutputModalities = append([]string(nil), upstream.SupportedOutputModalities...)
	} else if len(info.SupportedOutputModalities) == 0 {
		if upstream.Capabilities.ImageRoute {
			info.SupportedOutputModalities = []string{"text", "image"}
		} else if upstream.Capabilities.SpeechRoute {
			info.SupportedOutputModalities = []string{"audio"}
		} else if len(info.SupportedInputModalities) > 0 {
			info.SupportedOutputModalities = []string{"text"}
		}
	}

	if !info.ExplicitThinking && info.Thinking == nil && (upstream.Thinking || upstream.Capabilities.Thinking || len(upstream.ThinkingLevels) > 0) {
		info.Thinking = modelconfig.NormalizeThinkingSupport(&registry.ThinkingSupport{
			DynamicAllowed: true,
			Levels:         upstream.ThinkingLevels,
		})
	}

	if upstream.Capabilities.GoogleSearch {
		if info.NativeCapabilities == nil {
			info.NativeCapabilities = &registry.NativeCapabilities{}
		}
		webSearch := true
		info.NativeCapabilities.WebSearch = &webSearch
	}
}

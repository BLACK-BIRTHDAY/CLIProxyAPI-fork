package magpie

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	coreusage "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/usage"
	log "github.com/sirupsen/logrus"
)

// MaxStudioToolNameLen is the maximum tool identifier length supported by Google AI Studio.
const MaxStudioToolNameLen = 63

// MaxMalformedRetries is the maximum number of silent temperature-perturbed retry attempts
// executed upon encountering a Malformed Empty Upstream Reply.
const MaxMalformedRetries = 2

// TestBackendOverride provides test hooks for mock upstream servers in unit and integration tests.
type TestBackendOverride struct {
	BaseURL string
	APIKey  string
	Config  MagpieConfig
}

var (
	testBackendOverrideMu sync.RWMutex
	testBackendOverride   *TestBackendOverride
)

// SetTestBackendOverride configures or clears the global test backend override.
func SetTestBackendOverride(override *TestBackendOverride) {
	testBackendOverrideMu.Lock()
	defer testBackendOverrideMu.Unlock()
	testBackendOverride = override
}

// GetTestBackendOverride returns the active test backend override, if set.
func GetTestBackendOverride() *TestBackendOverride {
	testBackendOverrideMu.RLock()
	defer testBackendOverrideMu.RUnlock()
	return testBackendOverride
}

// InternalBaseURLAndKey resolves the internal base URL, API key, and configuration for forwarding.
func InternalBaseURLAndKey(ctx context.Context, inheritedProxy string) (string, string, *MagpieConfig, error) {
	if override := GetTestBackendOverride(); override != nil {
		cfgCopy := override.Config
		return override.BaseURL, override.APIKey, &cfgCopy, nil
	}
	if mockURL := os.Getenv("MAGPIE_INTERNAL_BASE_URL"); mockURL != "" {
		mockKey := os.Getenv("MAGPIE_INTERNAL_API_KEY")
		if mockKey == "" {
			mockKey = "as_test_key"
		}
		return mockURL, mockKey, NewDefaultConfig(), nil
	}

	dataDir := ResolveDataDir("")
	cfg, err := LoadGatewayConfig(dataDir, inheritedProxy)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load gateway config: %w", err)
	}
	if cfg.AutoStart {
		_ = EnsureRunning(ctx, dataDir, cfg, inheritedProxy)
	}
	studioCfg, err := EnsureStudioConfig(dataDir, cfg, inheritedProxy)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to ensure studio config: %w", err)
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", studioCfg.Port)
	return baseURL, studioCfg.APIKey, cfg, nil
}

// ToolAliasOf computes a deterministic shortened identifier for tool names exceeding 63 bytes.
// It keeps up to 54 valid UTF-8 prefix bytes and appends a '_' plus the first 8 hex characters of the SHA-256 hash.
func ToolAliasOf(name string) string {
	if len(name) <= MaxStudioToolNameLen {
		return name
	}
	prefixEnd := 54
	if prefixEnd > len(name) {
		prefixEnd = len(name)
	}
	for prefixEnd > 0 && !utf8.RuneStart(name[prefixEnd]) {
		prefixEnd--
	}
	sum := sha256.Sum256([]byte(name))
	digest := hex.EncodeToString(sum[:])
	return fmt.Sprintf("%s_%s", name[:prefixEnd], digest[:8])
}

// AliasGeminiTools rewrites function declaration and call names exceeding 63 bytes in a Gemini request
// and normalizes empty or object parameters to type: "OBJECT" and properties: {}.
// Returns a reverse alias map for restoring original tool names on response.
func AliasGeminiTools(val map[string]any) map[string]string {
	reverse := make(map[string]string)
	target := val
	if req, ok := val["request"].(map[string]any); ok {
		target = req
	}

	mapName := func(m map[string]any, key string) {
		if s, ok := m[key].(string); ok && len(s) > MaxStudioToolNameLen {
			alias := ToolAliasOf(s)
			reverse[alias] = s
			m[key] = alias
		}
	}

	if tools, ok := target["tools"].([]any); ok {
		for _, rawTool := range tools {
			tool, ok := rawTool.(map[string]any)
			if !ok {
				continue
			}
			for _, declKey := range []string{"functionDeclarations", "function_declarations"} {
				if decls, ok := tool[declKey].([]any); ok {
					for _, rawFD := range decls {
						fd, ok := rawFD.(map[string]any)
						if !ok {
							continue
						}
						mapName(fd, "name")
						if params, ok := fd["parameters"].(map[string]any); ok {
							typeVal, hasType := params["type"].(string)
							isEmptyOrObject := !hasType || strings.EqualFold(typeVal, "object")
							if isEmptyOrObject {
								params["type"] = "OBJECT"
								if _, ok := params["properties"]; !ok || params["properties"] == nil {
									params["properties"] = map[string]any{}
								}
							}
						}
					}
				}
			}
		}
	}

	for _, tcKey := range []string{"toolConfig", "tool_config"} {
		if tc, ok := target[tcKey].(map[string]any); ok {
			for _, fccKey := range []string{"functionCallingConfig", "function_calling_config"} {
				if fcc, ok := tc[fccKey].(map[string]any); ok {
					for _, afnKey := range []string{"allowedFunctionNames", "allowed_function_names"} {
						if allowed, ok := fcc[afnKey].([]any); ok {
							for i, item := range allowed {
								if s, ok := item.(string); ok && len(s) > MaxStudioToolNameLen {
									alias := ToolAliasOf(s)
									reverse[alias] = s
									allowed[i] = alias
								}
							}
						}
					}
				}
			}
		}
	}

	if contents, ok := target["contents"].([]any); ok {
		for _, rawContent := range contents {
			content, ok := rawContent.(map[string]any)
			if !ok {
				continue
			}
			if parts, ok := content["parts"].([]any); ok {
				for _, rawPart := range parts {
					part, ok := rawPart.(map[string]any)
					if !ok {
						continue
					}
					for _, key := range []string{"functionCall", "function_call", "functionResponse", "function_response"} {
						if fc, ok := part[key].(map[string]any); ok {
							mapName(fc, "name")
						}
					}
				}
			}
		}
	}

	return reverse
}

// AliasOpenAITools rewrites tool, function, and call names exceeding 63 bytes in an OpenAI request
// and normalizes empty or object parameters to type: "object" and properties: {}.
// Returns a reverse alias map for restoring original tool names on response.
func AliasOpenAITools(val map[string]any) map[string]string {
	reverse := make(map[string]string)
	mapName := func(m map[string]any, key string) {
		if s, ok := m[key].(string); ok && len(s) > MaxStudioToolNameLen {
			alias := ToolAliasOf(s)
			reverse[alias] = s
			m[key] = alias
		}
	}

	if tools, ok := val["tools"].([]any); ok {
		for _, rawTool := range tools {
			tool, ok := rawTool.(map[string]any)
			if !ok {
				continue
			}
			if fn, ok := tool["function"].(map[string]any); ok {
				mapName(fn, "name")
				if params, ok := fn["parameters"].(map[string]any); ok {
					typeVal, hasType := params["type"].(string)
					isEmptyOrObject := !hasType || strings.EqualFold(typeVal, "object")
					if isEmptyOrObject {
						if !hasType {
							params["type"] = "object"
						}
						if _, ok := params["properties"]; !ok || params["properties"] == nil {
							params["properties"] = map[string]any{}
						}
					}
				}
			}
			mapName(tool, "name")
		}
	}

	if functions, ok := val["functions"].([]any); ok {
		for _, rawFunc := range functions {
			if fn, ok := rawFunc.(map[string]any); ok {
				mapName(fn, "name")
			}
		}
	}

	if tc, ok := val["tool_choice"].(map[string]any); ok {
		if fn, ok := tc["function"].(map[string]any); ok {
			mapName(fn, "name")
		}
	}

	if fc, ok := val["function_call"].(map[string]any); ok {
		mapName(fc, "name")
	}

	if messages, ok := val["messages"].([]any); ok {
		for _, rawMsg := range messages {
			msg, ok := rawMsg.(map[string]any)
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role == "tool" || role == "function" {
				mapName(msg, "name")
			}
			if fc, ok := msg["function_call"].(map[string]any); ok {
				mapName(fc, "name")
			}
			if toolCalls, ok := msg["tool_calls"].([]any); ok {
				for _, rawCall := range toolCalls {
					if call, ok := rawCall.(map[string]any); ok {
						if fn, ok := call["function"].(map[string]any); ok {
							mapName(fn, "name")
						}
					}
				}
			}
		}
	}

	return reverse
}

// RestoreToolAliases reverses aliased tool names back to their original identifiers
// across OpenAI message/delta/function_call structures and Gemini candidate parts.
func RestoreToolAliases(val map[string]any, reverse map[string]string) bool {
	if len(reverse) == 0 || val == nil {
		return false
	}
	restored := false

	// OpenAI format: choices[].message or choices[].delta
	if choices, ok := val["choices"].([]any); ok {
		for _, rawChoice := range choices {
			choice, ok := rawChoice.(map[string]any)
			if !ok {
				continue
			}
			for _, containerKey := range []string{"message", "delta"} {
				if container, ok := choice[containerKey].(map[string]any); ok {
					if fc, ok := container["function_call"].(map[string]any); ok {
						if nameStr, ok := fc["name"].(string); ok {
							if orig, ok := reverse[nameStr]; ok {
								fc["name"] = orig
								restored = true
							}
						}
					}
					if toolCalls, ok := container["tool_calls"].([]any); ok {
						for _, rawCall := range toolCalls {
							if call, ok := rawCall.(map[string]any); ok {
								if fn, ok := call["function"].(map[string]any); ok {
									if nameStr, ok := fn["name"].(string); ok {
										if orig, ok := reverse[nameStr]; ok {
											fn["name"] = orig
											restored = true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Gemini format: candidates[].content.parts[].functionCall
	geminiRoot := val
	if resp, ok := val["response"].(map[string]any); ok {
		geminiRoot = resp
	}
	if candidates, ok := geminiRoot["candidates"].([]any); ok {
		for _, rawCand := range candidates {
			cand, ok := rawCand.(map[string]any)
			if !ok {
				continue
			}
			if content, ok := cand["content"].(map[string]any); ok {
				if parts, ok := content["parts"].([]any); ok {
					for _, rawPart := range parts {
						part, ok := rawPart.(map[string]any)
						if !ok {
							continue
						}
						for _, key := range []string{"functionCall", "function_call"} {
							if fc, ok := part[key].(map[string]any); ok {
								if nameStr, ok := fc["name"].(string); ok {
									if orig, ok := reverse[nameStr]; ok {
										fc["name"] = orig
										restored = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return restored
}

// StripGeminiTools removes tools and toolConfig from a Gemini request.
func StripGeminiTools(val map[string]any) {
	target := val
	if req, ok := val["request"].(map[string]any); ok {
		target = req
	}
	delete(target, "tools")
	delete(target, "toolConfig")
	delete(target, "tool_config")
}

// StripGeminiImages removes inlineData and fileData parts from a Gemini request.
func StripGeminiImages(val map[string]any) {
	target := val
	if req, ok := val["request"].(map[string]any); ok {
		target = req
	}
	if contents, ok := target["contents"].([]any); ok {
		for _, rawContent := range contents {
			content, ok := rawContent.(map[string]any)
			if !ok {
				continue
			}
			if parts, ok := content["parts"].([]any); ok {
				filtered := make([]any, 0, len(parts))
				for _, rawPart := range parts {
					part, ok := rawPart.(map[string]any)
					if !ok {
						filtered = append(filtered, rawPart)
						continue
					}
					hasInline := part["inlineData"] != nil || part["inline_data"] != nil
					hasFile := part["fileData"] != nil || part["file_data"] != nil
					if !hasInline && !hasFile {
						filtered = append(filtered, rawPart)
					}
				}
				if len(filtered) == 0 {
					filtered = append(filtered, map[string]any{
						"text": "[Image removed: allow_images is disabled]",
					})
				}
				content["parts"] = filtered
			}
		}
	}
}

// StripOpenAITools removes tools and tool_choice from an OpenAI chat request.
func StripOpenAITools(val map[string]any) {
	delete(val, "tools")
	delete(val, "tool_choice")
}

// StripOpenAIImages removes image_url content items from messages in an OpenAI chat request.
func StripOpenAIImages(val map[string]any) {
	if messages, ok := val["messages"].([]any); ok {
		for _, rawMsg := range messages {
			msg, ok := rawMsg.(map[string]any)
			if !ok {
				continue
			}
			if parts, ok := msg["content"].([]any); ok {
				filtered := make([]any, 0, len(parts))
				for _, rawPart := range parts {
					if p, ok := rawPart.(map[string]any); ok {
						if t, ok := p["type"].(string); ok && t == "image_url" {
							continue
						}
					}
					filtered = append(filtered, rawPart)
				}
				if len(filtered) == 0 {
					msg["content"] = "[Image removed: allow_images is disabled]"
				} else {
					msg["content"] = filtered
				}
			}
		}
	}
}

// IsMalformedFinishReason checks if a finish reason signifies a malformed response or function call.
func IsMalformedFinishReason(reason string) bool {
	return strings.EqualFold(reason, "MALFORMED_RESPONSE") ||
		strings.EqualFold(reason, "MALFORMED_FUNCTION_CALL")
}

// GeminiValueHasValidOutput returns true if any candidate part carries non-thought text or tool output.
func GeminiValueHasValidOutput(val map[string]any) bool {
	root := val
	if resp, ok := val["response"].(map[string]any); ok {
		root = resp
	}
	if candidates, ok := root["candidates"].([]any); ok {
		for _, rawCand := range candidates {
			cand, ok := rawCand.(map[string]any)
			if !ok {
				continue
			}
			if content, ok := cand["content"].(map[string]any); ok {
				if parts, ok := content["parts"].([]any); ok {
					for _, rawPart := range parts {
						part, ok := rawPart.(map[string]any)
						if !ok {
							continue
						}
						if part["functionCall"] != nil || part["function_call"] != nil ||
							part["inlineData"] != nil || part["inline_data"] != nil ||
							part["executableCode"] != nil || part["codeExecutionResult"] != nil {
							return true
						}
						thought, _ := part["thought"].(bool)
						if !thought {
							if text, ok := part["text"].(string); ok && len(text) > 0 {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func isGeminiMalformedValue(val map[string]any) bool {
	root := val
	if resp, ok := val["response"].(map[string]any); ok {
		root = resp
	}
	if candidates, ok := root["candidates"].([]any); ok {
		for _, rawCand := range candidates {
			cand, ok := rawCand.(map[string]any)
			if !ok {
				continue
			}
			reason, _ := cand["finishReason"].(string)
			if reason == "" {
				reason, _ = cand["finish_reason"].(string)
			}
			if IsMalformedFinishReason(reason) && !GeminiValueHasValidOutput(val) {
				return true
			}
		}
	}
	return false
}

// OpenAIValueHasValidOutput returns true if message or delta contains non-empty content or tool calls.
func OpenAIValueHasValidOutput(val map[string]any) bool {
	if choices, ok := val["choices"].([]any); ok {
		for _, rawChoice := range choices {
			choice, ok := rawChoice.(map[string]any)
			if !ok {
				continue
			}
			for _, key := range []string{"message", "delta"} {
				if container, ok := choice[key].(map[string]any); ok {
					if container["function_call"] != nil {
						return true
					}
					if calls, ok := container["tool_calls"].([]any); ok && len(calls) > 0 {
						return true
					}
					if content, ok := container["content"].(string); ok && len(content) > 0 {
						return true
					}
				}
			}
		}
	}
	return false
}

func isOpenAIMalformedValue(val map[string]any) bool {
	if choices, ok := val["choices"].([]any); ok {
		for _, rawChoice := range choices {
			choice, ok := rawChoice.(map[string]any)
			if !ok {
				continue
			}
			reason, _ := choice["finish_reason"].(string)
			if reason == "" {
				reason, _ = choice["finishReason"].(string)
			}
			if IsMalformedFinishReason(reason) && !OpenAIValueHasValidOutput(val) {
				return true
			}
		}
	}
	return false
}

// BuildGeminiRetryBody creates a copy of the request payload with perturbed temperature.
func BuildGeminiRetryBody(base map[string]any, attempt int) map[string]any {
	if attempt == 0 {
		return base
	}
	cloned := cloneMap(base)
	target := cloned
	if req, ok := cloned["request"].(map[string]any); ok {
		target = req
	}
	genCfg, ok := target["generationConfig"].(map[string]any)
	if !ok || genCfg == nil {
		genCfg = make(map[string]any)
		target["generationConfig"] = genCfg
	}
	baseTemp := 1.0
	if t, ok := genCfg["temperature"].(float64); ok {
		baseTemp = t
	}
	perturbed := baseTemp + float64(attempt)*0.01
	if perturbed < 0.01 {
		perturbed = 0.01
	} else if perturbed > 2.0 {
		perturbed = 2.0
	}
	genCfg["temperature"] = perturbed
	return cloned
}

// BuildOpenAIRetryBody creates a copy of the OpenAI chat request with perturbed temperature.
func BuildOpenAIRetryBody(base map[string]any, attempt int) map[string]any {
	if attempt == 0 {
		return base
	}
	cloned := cloneMap(base)
	baseTemp := 1.0
	if t, ok := cloned["temperature"].(float64); ok {
		baseTemp = t
	}
	perturbed := baseTemp + float64(attempt)*0.01
	if perturbed < 0.01 {
		perturbed = 0.01
	} else if perturbed > 2.0 {
		perturbed = 2.0
	}
	cloned["temperature"] = perturbed
	return cloned
}

func cloneMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var cloned map[string]any
	_ = json.Unmarshal(b, &cloned)
	return cloned
}

func getForwardHTTPClient(proxyURL string) *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if strings.TrimSpace(proxyURL) != "" {
		if parsed, err := url.Parse(strings.TrimSpace(proxyURL)); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		}
	}
	return &http.Client{
		Transport: transport,
		// Timeout: 0 - post-connection timeouts are forbidden per AGENTS.md
	}
}

// ForwardGeminiRequest handles forwarding of Gemini requests to the local MAGPIE daemon.
func ForwardGeminiRequest(w http.ResponseWriter, r *http.Request, rawModel, method string, rawBody []byte, inheritedProxy string, apiKeyPrincipal string) {
	baseURL, apiKey, magpieCfg, err := InternalBaseURLAndKey(r.Context(), inheritedProxy)
	if err != nil {
		http.Error(w, fmt.Sprintf("MAGPIE backend error: %v", err), http.StatusBadGateway)
		return
	}

	if !magpieCfg.Enabled {
		http.Error(w, "MAGPIE integration is disabled", http.StatusServiceUnavailable)
		return
	}

	cleanRawModel := strings.TrimPrefix(rawModel, "models/")
	if stripped, ok := StripMagpiePrefix(cleanRawModel); ok {
		cleanRawModel = stripped
	}

	var bodyMap map[string]any
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &bodyMap)
	}
	if bodyMap == nil {
		bodyMap = make(map[string]any)
	}

	// Apply feature toggles
	if !magpieCfg.AllowTools {
		StripGeminiTools(bodyMap)
	}
	if !magpieCfg.AllowImages {
		StripGeminiImages(bodyMap)
	}
	reverseToolMap := AliasGeminiTools(bodyMap)

	basePath := fmt.Sprintf("%s/v1beta/models/%s:%s", baseURL, cleanRawModel, method)
	targetURL := basePath
	if r.URL.RawQuery != "" {
		var filtered []string
		for _, part := range strings.Split(r.URL.RawQuery, "&") {
			if part != "" && part != "key" && !strings.HasPrefix(part, "key=") {
				filtered = append(filtered, part)
			}
		}
		if len(filtered) > 0 {
			targetURL = fmt.Sprintf("%s?%s", basePath, strings.Join(filtered, "&"))
		}
	}

	client := getForwardHTTPClient(ResolveMagpieProxy(magpieCfg, inheritedProxy))
	isStreaming := (method == "streamGenerateContent")
	startTime := time.Now()

	prefixedModel := PrefixMagpieModel(cleanRawModel)
	reqID := coreusage.ExecutionRequestIDFromContext(r.Context())
	if reqID == "" {
		reqID = uuid.New().String()
	}
	traceID := coreusage.TraceIDFromContext(r.Context())

	if isStreaming {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, _ := w.(http.Flusher)

		var inputTokens, outputTokens, reasoningTokens, totalTokens int64

		for attempt := 0; attempt <= MaxMalformedRetries; attempt++ {
			reqBody := BuildGeminiRetryBody(bodyMap, attempt)
			bodyBytes, _ := json.Marshal(reqBody)

			upstreamReq, err := http.NewRequestWithContext(r.Context(), "POST", targetURL, bytes.NewReader(bodyBytes))
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to construct upstream request: %v", err), http.StatusInternalServerError)
				return
			}
			upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			upstreamReq.Header.Set("x-goog-api-key", apiKey)
			upstreamReq.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(upstreamReq)
			if err != nil {
				if attempt == 0 {
					errJSON, _ := json.Marshal(map[string]any{
						"error": map[string]any{
							"code":    502,
							"message": fmt.Sprintf("Failed to reach MAGPIE backend: %v", err),
							"status":  "UNAVAILABLE",
						},
					})
					_, _ = fmt.Fprintf(w, "data: %s\n\n", string(errJSON))
					if flusher != nil {
						flusher.Flush()
					}
				}
				return
			}

			cType := resp.Header.Get("Content-Type")
			if resp.StatusCode != http.StatusOK && !strings.Contains(cType, "text/event-stream") {
				w.WriteHeader(resp.StatusCode)
				_, _ = io.Copy(w, resp.Body)
				_ = resp.Body.Close()
				return
			}

			reader := bufio.NewReader(resp.Body)
			var preYieldFrames [][]byte
			committed := false
			sawMalformed := false

			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					outLine := line
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "data:") {
						payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
						if payload != "" && payload != "[DONE]" {
							var val map[string]any
							if err := json.Unmarshal([]byte(payload), &val); err == nil {
								RestoreToolAliases(val, reverseToolMap)
								// Rewrite model
								root := val
								if r, ok := val["response"].(map[string]any); ok {
									root = r
								}
								if mv, ok := root["modelVersion"].(string); ok && mv != "" {
									root["modelVersion"] = PrefixMagpieModel(mv)
								}
								if m, ok := root["model"].(string); ok && m != "" {
									root["model"] = PrefixMagpieModel(m)
								}

								// Token accounting
								if meta, ok := root["usageMetadata"].(map[string]any); ok {
									getInt := func(k string) int64 {
										if f, ok := meta[k].(float64); ok {
											return int64(f)
										}
										return 0
									}
									if p := getInt("promptTokenCount"); p > 0 {
										inputTokens = p
									}
									if c := getInt("candidatesTokenCount"); c > 0 {
										outputTokens = c
									}
									if t := getInt("totalTokenCount"); t > 0 {
										totalTokens = t
									}
									if th := getInt("thoughtsTokenCount"); th > 0 {
										reasoningTokens = th
									}
								}

								if !committed {
									if GeminiValueHasValidOutput(val) {
										committed = true
										for _, f := range preYieldFrames {
											_, _ = w.Write(f)
										}
										preYieldFrames = nil
									} else if isGeminiMalformedValue(val) {
										sawMalformed = true
									}
								}

								if serialized, err := json.Marshal(val); err == nil {
									outLine = fmt.Sprintf("data: %s\n", string(serialized))
								}
							}
						}
					}

					if committed {
						_, _ = w.Write([]byte(outLine))
						if flusher != nil {
							flusher.Flush()
						}
					} else {
						preYieldFrames = append(preYieldFrames, []byte(outLine))
					}
				}
				if err != nil {
					break
				}
			}
			_ = resp.Body.Close()

			if !committed && sawMalformed && attempt < MaxMalformedRetries {
				log.Warnf("[MAGPIE] Gemini SSE stream returned MALFORMED_RESPONSE with no valid output; retrying silently (%d/%d)", attempt+1, MaxMalformedRetries)
				continue
			}

			// If not committed, flush buffered pre-yield frames
			for _, f := range preYieldFrames {
				_, _ = w.Write(f)
			}
			if flusher != nil {
				flusher.Flush()
			}

			// Record usage
			coreusage.PublishRecord(r.Context(), coreusage.Record{
				RequestID:   reqID,
				TraceID:     traceID,
				Provider:    "magpie",
				Model:       prefixedModel,
				APIKey:      apiKeyPrincipal,
				Stream:      true,
				RequestedAt: startTime,
				Latency:     time.Since(startTime),
				Detail: coreusage.Detail{
					InputTokens:     inputTokens,
					OutputTokens:    outputTokens,
					ReasoningTokens: reasoningTokens,
					TotalTokens:     totalTokens,
				},
			})
			return
		}
		return
	}

	// Non-streaming JSON request
	for attempt := 0; attempt <= MaxMalformedRetries; attempt++ {
		reqBody := BuildGeminiRetryBody(bodyMap, attempt)
		bodyBytes, _ := json.Marshal(reqBody)

		upstreamReq, err := http.NewRequestWithContext(r.Context(), "POST", targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to construct upstream request: %v", err), http.StatusInternalServerError)
			return
		}
		upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
		upstreamReq.Header.Set("x-goog-api-key", apiKey)
		upstreamReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(upstreamReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to reach MAGPIE backend: %v", err), http.StatusBadGateway)
			return
		}

		respBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusBadGateway)
			return
		}

		var jsonVal map[string]any
		if err := json.Unmarshal(respBytes, &jsonVal); err == nil && resp.StatusCode == http.StatusOK {
			if isGeminiMalformedValue(jsonVal) && attempt < MaxMalformedRetries {
				log.Warnf("[MAGPIE] Gemini JSON response returned MALFORMED_RESPONSE with no valid output; retrying silently (%d/%d)", attempt+1, MaxMalformedRetries)
				continue
			}

			RestoreToolAliases(jsonVal, reverseToolMap)
			root := jsonVal
			if resObj, ok := jsonVal["response"].(map[string]any); ok {
				root = resObj
			}
			if mv, ok := root["modelVersion"].(string); ok && mv != "" {
				root["modelVersion"] = PrefixMagpieModel(mv)
			}
			if m, ok := root["model"].(string); ok && m != "" {
				root["model"] = PrefixMagpieModel(m)
			}

			var inputTokens, outputTokens, reasoningTokens, totalTokens int64
			if meta, ok := root["usageMetadata"].(map[string]any); ok {
				getInt := func(k string) int64 {
					if f, ok := meta[k].(float64); ok {
						return int64(f)
					}
					return 0
				}
				inputTokens = getInt("promptTokenCount")
				outputTokens = getInt("candidatesTokenCount")
				totalTokens = getInt("totalTokenCount")
				reasoningTokens = getInt("thoughtsTokenCount")
			}

			coreusage.PublishRecord(r.Context(), coreusage.Record{
				RequestID:   reqID,
				TraceID:     traceID,
				Provider:    "magpie",
				Model:       prefixedModel,
				APIKey:      apiKeyPrincipal,
				Stream:      false,
				RequestedAt: startTime,
				Latency:     time.Since(startTime),
				Detail: coreusage.Detail{
					InputTokens:     inputTokens,
					OutputTokens:    outputTokens,
					ReasoningTokens: reasoningTokens,
					TotalTokens:     totalTokens,
				},
			})

			outBytes, _ := json.Marshal(jsonVal)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(outBytes)
			return
		}

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBytes)
		return
	}
}

// ForwardOpenAIChatRequest handles forwarding of OpenAI chat completion requests to the local MAGPIE daemon.
func ForwardOpenAIChatRequest(w http.ResponseWriter, r *http.Request, rawModel string, rawBody []byte, inheritedProxy string, apiKeyPrincipal string) {
	baseURL, apiKey, magpieCfg, err := InternalBaseURLAndKey(r.Context(), inheritedProxy)
	if err != nil {
		http.Error(w, fmt.Sprintf("MAGPIE backend error: %v", err), http.StatusBadGateway)
		return
	}

	if !magpieCfg.Enabled {
		http.Error(w, "MAGPIE integration is disabled", http.StatusServiceUnavailable)
		return
	}

	cleanRawModel := strings.TrimPrefix(rawModel, "models/")
	if stripped, ok := StripMagpiePrefix(cleanRawModel); ok {
		cleanRawModel = stripped
	}

	var bodyMap map[string]any
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &bodyMap)
	}
	if bodyMap == nil {
		bodyMap = make(map[string]any)
	}

	// Downstream request model sent to internal daemon must have MAGPIE prefix stripped
	bodyMap["model"] = cleanRawModel

	// Apply feature toggles
	if !magpieCfg.AllowTools {
		StripOpenAITools(bodyMap)
	}
	if !magpieCfg.AllowImages {
		StripOpenAIImages(bodyMap)
	}
	reverseToolMap := AliasOpenAITools(bodyMap)

	targetURL := fmt.Sprintf("%s/v1/chat/completions", baseURL)
	client := getForwardHTTPClient(ResolveMagpieProxy(magpieCfg, inheritedProxy))
	isStreamReq, _ := bodyMap["stream"].(bool)
	startTime := time.Now()

	prefixedModel := PrefixMagpieModel(cleanRawModel)
	reqID := coreusage.ExecutionRequestIDFromContext(r.Context())
	if reqID == "" {
		reqID = uuid.New().String()
	}
	traceID := coreusage.TraceIDFromContext(r.Context())

	if isStreamReq {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher, _ := w.(http.Flusher)

		var inputTokens, outputTokens, reasoningTokens, totalTokens int64

		for attempt := 0; attempt <= MaxMalformedRetries; attempt++ {
			reqBody := BuildOpenAIRetryBody(bodyMap, attempt)
			bodyBytes, _ := json.Marshal(reqBody)

			upstreamReq, err := http.NewRequestWithContext(r.Context(), "POST", targetURL, bytes.NewReader(bodyBytes))
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to construct upstream request: %v", err), http.StatusInternalServerError)
				return
			}
			upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			upstreamReq.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(upstreamReq)
			if err != nil {
				if attempt == 0 {
					errJSON, _ := json.Marshal(map[string]any{
						"error": map[string]any{
							"message": fmt.Sprintf("Failed to reach MAGPIE backend: %v", err),
							"type":    "api_error",
							"code":    "bad_gateway",
						},
					})
					_, _ = fmt.Fprintf(w, "data: %s\n\n", string(errJSON))
					if flusher != nil {
						flusher.Flush()
					}
				}
				return
			}

			cType := resp.Header.Get("Content-Type")
			if resp.StatusCode != http.StatusOK && !strings.Contains(cType, "text/event-stream") {
				w.WriteHeader(resp.StatusCode)
				_, _ = io.Copy(w, resp.Body)
				_ = resp.Body.Close()
				return
			}

			reader := bufio.NewReader(resp.Body)
			var preYieldFrames [][]byte
			committed := false
			sawMalformed := false

			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					outLine := line
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "data:") {
						payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
						if payload == "[DONE]" {
							outLine = "data: [DONE]\n\n"
						} else if payload != "" {
							var val map[string]any
							if err := json.Unmarshal([]byte(payload), &val); err == nil {
								RestoreToolAliases(val, reverseToolMap)
								if _, ok := val["model"]; ok {
									val["model"] = prefixedModel
								}

								// Extract usage if available
								if u, ok := val["usage"].(map[string]any); ok {
									getInt := func(k string) int64 {
										if f, ok := u[k].(float64); ok {
											return int64(f)
										}
										return 0
									}
									if p := getInt("prompt_tokens"); p > 0 {
										inputTokens = p
									}
									if c := getInt("completion_tokens"); c > 0 {
										outputTokens = c
									}
									if t := getInt("total_tokens"); t > 0 {
										totalTokens = t
									}
									if details, ok := u["completion_tokens_details"].(map[string]any); ok {
										if r, ok := details["reasoning_tokens"].(float64); ok {
											reasoningTokens = int64(r)
										}
									}
								}

								if !committed {
									if OpenAIValueHasValidOutput(val) {
										committed = true
										for _, f := range preYieldFrames {
											_, _ = w.Write(f)
										}
										preYieldFrames = nil
									} else if isOpenAIMalformedValue(val) {
										sawMalformed = true
									}
								}

								if serialized, err := json.Marshal(val); err == nil {
									outLine = fmt.Sprintf("data: %s\n\n", string(serialized))
								}
							}
						}
					}

					if committed {
						_, _ = w.Write([]byte(outLine))
						if flusher != nil {
							flusher.Flush()
						}
					} else {
						preYieldFrames = append(preYieldFrames, []byte(outLine))
					}
				}
				if err != nil {
					break
				}
			}
			_ = resp.Body.Close()

			if !committed && sawMalformed && attempt < MaxMalformedRetries {
				log.Warnf("[MAGPIE] OpenAI SSE stream returned malformed finish_reason with no valid output; retrying silently (%d/%d)", attempt+1, MaxMalformedRetries)
				continue
			}

			// If not committed, flush whatever was buffered
			for _, f := range preYieldFrames {
				_, _ = w.Write(f)
			}
			if flusher != nil {
				flusher.Flush()
			}

			coreusage.PublishRecord(r.Context(), coreusage.Record{
				RequestID:   reqID,
				TraceID:     traceID,
				Provider:    "magpie",
				Model:       prefixedModel,
				APIKey:      apiKeyPrincipal,
				Stream:      true,
				RequestedAt: startTime,
				Latency:     time.Since(startTime),
				Detail: coreusage.Detail{
					InputTokens:     inputTokens,
					OutputTokens:    outputTokens,
					ReasoningTokens: reasoningTokens,
					TotalTokens:     totalTokens,
				},
			})
			return
		}
		return
	}

	// Non-streaming JSON request
	for attempt := 0; attempt <= MaxMalformedRetries; attempt++ {
		reqBody := BuildOpenAIRetryBody(bodyMap, attempt)
		bodyBytes, _ := json.Marshal(reqBody)

		upstreamReq, err := http.NewRequestWithContext(r.Context(), "POST", targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to construct upstream request: %v", err), http.StatusInternalServerError)
			return
		}
		upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
		upstreamReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(upstreamReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to reach MAGPIE backend: %v", err), http.StatusBadGateway)
			return
		}

		respBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusBadGateway)
			return
		}

		var jsonVal map[string]any
		if err := json.Unmarshal(respBytes, &jsonVal); err == nil && resp.StatusCode == http.StatusOK {
			if isOpenAIMalformedValue(jsonVal) && attempt < MaxMalformedRetries {
				log.Warnf("[MAGPIE] OpenAI JSON response returned malformed finish_reason with no valid output; retrying silently (%d/%d)", attempt+1, MaxMalformedRetries)
				continue
			}

			RestoreToolAliases(jsonVal, reverseToolMap)
			if _, ok := jsonVal["model"]; ok {
				jsonVal["model"] = prefixedModel
			}

			var inputTokens, outputTokens, reasoningTokens, totalTokens int64
			if u, ok := jsonVal["usage"].(map[string]any); ok {
				getInt := func(k string) int64 {
					if f, ok := u[k].(float64); ok {
						return int64(f)
					}
					return 0
				}
				inputTokens = getInt("prompt_tokens")
				outputTokens = getInt("completion_tokens")
				totalTokens = getInt("total_tokens")
				if details, ok := u["completion_tokens_details"].(map[string]any); ok {
					if r, ok := details["reasoning_tokens"].(float64); ok {
						reasoningTokens = int64(r)
					}
				}
			}

			coreusage.PublishRecord(r.Context(), coreusage.Record{
				RequestID:   reqID,
				TraceID:     traceID,
				Provider:    "magpie",
				Model:       prefixedModel,
				APIKey:      apiKeyPrincipal,
				Stream:      false,
				RequestedAt: startTime,
				Latency:     time.Since(startTime),
				Detail: coreusage.Detail{
					InputTokens:     inputTokens,
					OutputTokens:    outputTokens,
					ReasoningTokens: reasoningTokens,
					TotalTokens:     totalTokens,
				},
			})

			outBytes, _ := json.Marshal(jsonVal)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(outBytes)
			return
		}

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBytes)
		return
	}
}

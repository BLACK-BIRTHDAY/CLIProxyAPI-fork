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
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/MoeForge/moego/ojson"
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
	InvalidateModelsCache()
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
	cfg, errLoad := LoadGatewayConfig(dataDir, inheritedProxy)
	if errLoad != nil {
		return "", "", nil, fmt.Errorf("failed to load gateway config: %w", errLoad)
	}
	if cfg.AutoStart {
		_ = EnsureRunning(ctx, dataDir, cfg, inheritedProxy)
	}
	studioCfg, errStudio := EnsureStudioConfig(dataDir, cfg, inheritedProxy)
	if errStudio != nil {
		return "", "", nil, fmt.Errorf("failed to ensure studio config: %w", errStudio)
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
						paramsRaw, hasParams := fd["parameters"]
						if !hasParams || paramsRaw == nil {
							fd["parameters"] = map[string]any{
								"type":       "OBJECT",
								"properties": map[string]any{},
							}
						} else if params, ok := paramsRaw.(map[string]any); ok {
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
				if paramsRaw, hasParams := fn["parameters"]; hasParams {
					if paramsRaw == nil {
						fn["parameters"] = map[string]any{
							"type":       "object",
							"properties": map[string]any{},
						}
					} else if params, ok := paramsRaw.(map[string]any); ok {
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

// RestoreToolAliasesDoc reverses aliased tool names back to their original identifiers
// across OpenAI message/delta/function_call structures and Gemini candidate parts using *ojson.Doc.
func RestoreToolAliasesDoc(doc *ojson.Doc, reverse map[string]string) bool {
	if len(reverse) == 0 || doc == nil {
		return false
	}
	restored := false

	// OpenAI format: choices[].message or choices[].delta
	if choices := doc.Get("choices"); choices.IsArray() {
		choices.ForEach(func(choiceIdx, choice ojson.Result) bool {
			choicePath := fmt.Sprintf("choices.%d", choiceIdx.Int())
			for _, containerKey := range []string{"message", "delta"} {
				container := choice.Get(containerKey)
				if !container.Exists() {
					continue
				}
				containerPath := choicePath + "." + containerKey

				// function_call.name
				if fcName := container.Get("function_call.name"); fcName.Type == ojson.String {
					if orig, ok := reverse[fcName.String()]; ok {
						_ = doc.Set(containerPath+".function_call.name", orig)
						restored = true
					}
				}

				// tool_calls[].function.name
				if toolCalls := container.Get("tool_calls"); toolCalls.IsArray() {
					toolCalls.ForEach(func(callIdx, call ojson.Result) bool {
						callPath := fmt.Sprintf("%s.tool_calls.%d", containerPath, callIdx.Int())
						if fnName := call.Get("function.name"); fnName.Type == ojson.String {
							if orig, ok := reverse[fnName.String()]; ok {
								_ = doc.Set(callPath+".function.name", orig)
								restored = true
							}
						}
						return true
					})
				}
			}
			return true
		})
	}

	// Gemini format: candidates[].content.parts[].functionCall
	candidatesPrefix := "candidates"
	candidates := doc.Get("candidates")
	if !candidates.Exists() && doc.Get("response.candidates").Exists() {
		candidatesPrefix = "response.candidates"
		candidates = doc.Get("response.candidates")
	}
	if candidates.IsArray() {
		candidates.ForEach(func(candIdx, cand ojson.Result) bool {
			candPath := fmt.Sprintf("%s.%d", candidatesPrefix, candIdx.Int())
			parts := cand.Get("content.parts")
			if parts.IsArray() {
				parts.ForEach(func(partIdx, part ojson.Result) bool {
					partPath := fmt.Sprintf("%s.content.parts.%d", candPath, partIdx.Int())
					for _, key := range []string{"functionCall", "function_call"} {
						if fcName := part.Get(key + ".name"); fcName.Type == ojson.String {
							if orig, ok := reverse[fcName.String()]; ok {
								_ = doc.Set(partPath+"."+key+".name", orig)
								restored = true
							}
						}
					}
					return true
				})
			}
			return true
		})
	}

	return restored
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

// GeminiDocHasValidOutput returns true if any candidate part carries non-thought text, tools, or media output.
func GeminiDocHasValidOutput(doc *ojson.Doc) bool {
	candidates := doc.Get("candidates")
	if !candidates.Exists() && doc.Get("response.candidates").Exists() {
		candidates = doc.Get("response.candidates")
	}
	if !candidates.IsArray() {
		return false
	}
	hasValid := false
	candidates.ForEach(func(_, cand ojson.Result) bool {
		parts := cand.Get("content.parts")
		if parts.IsArray() {
			parts.ForEach(func(_, part ojson.Result) bool {
				if part.Get("functionCall").Exists() || part.Get("function_call").Exists() ||
					part.Get("inlineData").Exists() || part.Get("inline_data").Exists() ||
					part.Get("executableCode").Exists() || part.Get("codeExecutionResult").Exists() {
					hasValid = true
					return false
				}
				if !part.Get("thought").Bool() {
					if text := part.Get("text").String(); len(text) > 0 {
						hasValid = true
						return false
					}
				}
				return true
			})
		}
		return !hasValid
	})
	return hasValid
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

func isGeminiMalformedDoc(doc *ojson.Doc) bool {
	candidates := doc.Get("candidates")
	if !candidates.Exists() && doc.Get("response.candidates").Exists() {
		candidates = doc.Get("response.candidates")
	}
	if !candidates.IsArray() {
		return false
	}
	sawMalformed := false
	candidates.ForEach(func(_, cand ojson.Result) bool {
		reason := cand.Get("finishReason").String()
		if reason == "" {
			reason = cand.Get("finish_reason").String()
		}
		if IsMalformedFinishReason(reason) && !GeminiDocHasValidOutput(doc) {
			sawMalformed = true
			return false
		}
		return true
	})
	return sawMalformed
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

// OpenAIDocHasValidOutput returns true if message or delta contains non-empty content or tool calls.
func OpenAIDocHasValidOutput(doc *ojson.Doc) bool {
	choices := doc.Get("choices")
	if !choices.IsArray() {
		return false
	}
	hasValid := false
	choices.ForEach(func(_, choice ojson.Result) bool {
		for _, key := range []string{"message", "delta"} {
			container := choice.Get(key)
			if !container.Exists() {
				continue
			}
			if container.Get("function_call").Exists() {
				hasValid = true
				return false
			}
			if toolCalls := container.Get("tool_calls"); toolCalls.IsArray() && len(toolCalls.Array()) > 0 {
				hasValid = true
				return false
			}
			if content := container.Get("content").String(); len(content) > 0 {
				hasValid = true
				return false
			}
		}
		return true
	})
	return hasValid
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

func isOpenAIMalformedDoc(doc *ojson.Doc) bool {
	choices := doc.Get("choices")
	if !choices.IsArray() {
		return false
	}
	sawMalformed := false
	choices.ForEach(func(_, choice ojson.Result) bool {
		reason := choice.Get("finish_reason").String()
		if reason == "" {
			reason = choice.Get("finishReason").String()
		}
		if IsMalformedFinishReason(reason) && !OpenAIDocHasValidOutput(doc) {
			sawMalformed = true
			return false
		}
		return true
	})
	return sawMalformed
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

// BuildGeminiRetryBodyBytes creates a copy of the request payload with perturbed temperature using *ojson.Doc.
func BuildGeminiRetryBodyBytes(base []byte, attempt int) []byte {
	if attempt == 0 || len(base) == 0 {
		return base
	}
	doc := ojson.NewDoc(base)
	path := "generationConfig.temperature"
	if doc.Get("request").Exists() {
		path = "request.generationConfig.temperature"
	}
	baseTemp := 1.0
	if t := doc.Get(path); t.Exists() && t.Type == ojson.Number {
		baseTemp = t.Float()
	}
	perturbed := baseTemp + float64(attempt)*0.01
	if perturbed < 0.01 {
		perturbed = 0.01
	} else if perturbed > 2.0 {
		perturbed = 2.0
	}
	_ = doc.Set(path, perturbed)
	return doc.Bytes()
}

// BuildGeminiRetryBody creates a copy of the request payload with perturbed temperature.
func BuildGeminiRetryBody(base map[string]any, attempt int) map[string]any {
	if attempt == 0 || base == nil {
		return base
	}
	b, errMarshal := json.Marshal(base)
	if errMarshal != nil {
		return base
	}
	ret := BuildGeminiRetryBodyBytes(b, attempt)
	var out map[string]any
	_ = json.Unmarshal(ret, &out)
	return out
}

// BuildOpenAIRetryBodyBytes creates a copy of the OpenAI chat request with perturbed temperature using *ojson.Doc.
func BuildOpenAIRetryBodyBytes(base []byte, attempt int) []byte {
	if attempt == 0 || len(base) == 0 {
		return base
	}
	doc := ojson.NewDoc(base)
	baseTemp := 1.0
	if t := doc.Get("temperature"); t.Exists() && t.Type == ojson.Number {
		baseTemp = t.Float()
	}
	perturbed := baseTemp + float64(attempt)*0.01
	if perturbed < 0.01 {
		perturbed = 0.01
	} else if perturbed > 2.0 {
		perturbed = 2.0
	}
	_ = doc.Set("temperature", perturbed)
	return doc.Bytes()
}

// BuildOpenAIRetryBody creates a copy of the OpenAI chat request with perturbed temperature.
func BuildOpenAIRetryBody(base map[string]any, attempt int) map[string]any {
	if attempt == 0 || base == nil {
		return base
	}
	b, errMarshal := json.Marshal(base)
	if errMarshal != nil {
		return base
	}
	ret := BuildOpenAIRetryBodyBytes(b, attempt)
	var out map[string]any
	_ = json.Unmarshal(ret, &out)
	return out
}

func getForwardHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: nil, // Loopback forwarding must never route through outbound proxy
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
	return &http.Client{
		Transport: transport,
	}
}

type tokenAccounting struct {
	inputTokens     int64
	outputTokens    int64
	reasoningTokens int64
	totalTokens     int64
	cachedTokens    int64
}

func extractGeminiDocUsage(doc *ojson.Doc, acc *tokenAccounting) {
	meta := doc.Get("usageMetadata")
	if !meta.Exists() && doc.Get("response.usageMetadata").Exists() {
		meta = doc.Get("response.usageMetadata")
	}
	if !meta.Exists() {
		return
	}
	if p := meta.Get("promptTokenCount").Int(); p > 0 {
		acc.inputTokens = p
	}
	if c := meta.Get("candidatesTokenCount").Int(); c > 0 {
		acc.outputTokens = c
	}
	if t := meta.Get("totalTokenCount").Int(); t > 0 {
		acc.totalTokens = t
	}
	if th := meta.Get("thoughtsTokenCount").Int(); th > 0 {
		acc.reasoningTokens = th
	}
	if ca := meta.Get("cachedContentTokenCount").Int(); ca > 0 {
		acc.cachedTokens = ca
	}
}

func extractOpenAIDocUsage(doc *ojson.Doc, acc *tokenAccounting) {
	u := doc.Get("usage")
	if !u.Exists() {
		return
	}
	if p := u.Get("prompt_tokens").Int(); p > 0 {
		acc.inputTokens = p
	}
	if c := u.Get("completion_tokens").Int(); c > 0 {
		acc.outputTokens = c
	}
	if t := u.Get("total_tokens").Int(); t > 0 {
		acc.totalTokens = t
	}
	if th := u.Get("completion_tokens_details.reasoning_tokens").Int(); th > 0 {
		acc.reasoningTokens = th
	}
	if ca := u.Get("prompt_tokens_details.cached_tokens").Int(); ca > 0 {
		acc.cachedTokens = ca
	}
}

func rewriteGeminiDocModel(doc *ojson.Doc) {
	if mv := doc.Get("modelVersion"); mv.Exists() && mv.String() != "" {
		_ = doc.Set("modelVersion", PrefixMagpieModel(mv.String()))
	}
	if mv := doc.Get("response.modelVersion"); mv.Exists() && mv.String() != "" {
		_ = doc.Set("response.modelVersion", PrefixMagpieModel(mv.String()))
	}
	if m := doc.Get("model"); m.Exists() && m.String() != "" {
		_ = doc.Set("model", PrefixMagpieModel(m.String()))
	}
	if m := doc.Get("response.model"); m.Exists() && m.String() != "" {
		_ = doc.Set("response.model", PrefixMagpieModel(m.String()))
	}
}

func rewriteOpenAIDocModel(doc *ojson.Doc, prefixedModel string) {
	if doc.Get("model").Exists() {
		_ = doc.Set("model", prefixedModel)
	}
}

type sseForwardParams struct {
	w               http.ResponseWriter
	r               *http.Request
	client          *http.Client
	targetURL       string
	authHeaders     map[string]string
	baseBodyBytes   []byte
	perturbBody     func(base []byte, attempt int) []byte
	transformDoc    func(doc *ojson.Doc, isCommitCandidate *bool, isMalformed *bool, acc *tokenAccounting)
	logName         string
	provider        string
	prefixedModel   string
	apiKeyPrincipal string
	reqID           string
	traceID         string
	startTime       time.Time
}

func forwardSSEStream(params sseForwardParams) {
	params.w.Header().Set("Content-Type", "text/event-stream")
	params.w.Header().Set("Cache-Control", "no-cache")
	params.w.Header().Set("Connection", "keep-alive")
	params.w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := params.w.(http.Flusher)

	var acc tokenAccounting

	for attempt := 0; attempt <= MaxMalformedRetries; attempt++ {
		bodyBytes := params.perturbBody(params.baseBodyBytes, attempt)

		upstreamReq, errReq := http.NewRequestWithContext(params.r.Context(), "POST", params.targetURL, bytes.NewReader(bodyBytes))
		if errReq != nil {
			http.Error(params.w, fmt.Sprintf("Failed to construct upstream request: %v", errReq), http.StatusInternalServerError)
			return
		}
		for k, v := range params.authHeaders {
			upstreamReq.Header.Set(k, v)
		}
		upstreamReq.Header.Set("Content-Type", "application/json")

		resp, errDo := params.client.Do(upstreamReq)
		if errDo != nil {
			if attempt == 0 {
				var errJSON []byte
				if params.logName == "Gemini" {
					errJSON, _ = json.Marshal(map[string]any{
						"error": map[string]any{
							"code":    502,
							"message": fmt.Sprintf("Failed to reach MAGPIE backend: %v", errDo),
							"status":  "UNAVAILABLE",
						},
					})
				} else {
					errJSON, _ = json.Marshal(map[string]any{
						"error": map[string]any{
							"message": fmt.Sprintf("Failed to reach MAGPIE backend: %v", errDo),
							"type":    "api_error",
							"code":    "bad_gateway",
						},
					})
				}
				_, _ = fmt.Fprintf(params.w, "data: %s\n\n", string(errJSON))
				if flusher != nil {
					flusher.Flush()
				}
			}
			return
		}

		cType := resp.Header.Get("Content-Type")
		if resp.StatusCode != http.StatusOK && !strings.Contains(cType, "text/event-stream") {
			params.w.WriteHeader(resp.StatusCode)
			_, _ = io.Copy(params.w, resp.Body)
			if errClose := resp.Body.Close(); errClose != nil {
				log.Debugf("failed to close response body: %v", errClose)
			}
			return
		}

		reader := bufio.NewReader(resp.Body)
		var preYieldFrames [][]byte
		committed := false
		sawMalformed := false

		for {
			line, errReadLine := reader.ReadString('\n')
			if len(line) > 0 {
				outLine := line
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "data:") {
					payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
					if payload == "[DONE]" {
						outLine = "data: [DONE]\n\n"
					} else if payload != "" {
						doc := ojson.NewDoc([]byte(payload))
						if doc.Get("@this").Exists() {
							params.transformDoc(doc, &committed, &sawMalformed, &acc)
							outLine = fmt.Sprintf("data: %s\n\n", doc.String())
						}
					}
				}

				if committed {
					_, _ = params.w.Write([]byte(outLine))
					if flusher != nil {
						flusher.Flush()
					}
				} else {
					preYieldFrames = append(preYieldFrames, []byte(outLine))
				}
			}
			if errReadLine != nil {
				break
			}
		}
		if errClose := resp.Body.Close(); errClose != nil {
			log.Debugf("failed to close streaming response body: %v", errClose)
		}

		if !committed && sawMalformed && attempt < MaxMalformedRetries {
			log.Warnf("[MAGPIE] %s SSE stream returned malformed reply with no valid output; retrying silently (%d/%d)", params.logName, attempt+1, MaxMalformedRetries)
			continue
		}

		for _, f := range preYieldFrames {
			_, _ = params.w.Write(f)
		}
		if flusher != nil {
			flusher.Flush()
		}

		coreusage.PublishRecord(params.r.Context(), coreusage.Record{
			RequestID:   params.reqID,
			TraceID:     params.traceID,
			Provider:    params.provider,
			Model:       params.prefixedModel,
			APIKey:      params.apiKeyPrincipal,
			Stream:      true,
			RequestedAt: params.startTime,
			Latency:     time.Since(params.startTime),
			Detail: coreusage.Detail{
				InputTokens:     acc.inputTokens,
				OutputTokens:    acc.outputTokens,
				ReasoningTokens: acc.reasoningTokens,
				TotalTokens:     acc.totalTokens,
				CachedTokens:    acc.cachedTokens,
			},
		})
		return
	}
}

type jsonForwardParams struct {
	w               http.ResponseWriter
	r               *http.Request
	client          *http.Client
	targetURL       string
	authHeaders     map[string]string
	baseBodyBytes   []byte
	perturbBody     func(base []byte, attempt int) []byte
	isMalformedDoc  func(doc *ojson.Doc) bool
	transformDoc    func(doc *ojson.Doc, acc *tokenAccounting)
	logName         string
	provider        string
	prefixedModel   string
	apiKeyPrincipal string
	reqID           string
	traceID         string
	startTime       time.Time
}

func forwardJSONRequest(params jsonForwardParams) {
	for attempt := 0; attempt <= MaxMalformedRetries; attempt++ {
		bodyBytes := params.perturbBody(params.baseBodyBytes, attempt)

		upstreamReq, errReq := http.NewRequestWithContext(params.r.Context(), "POST", params.targetURL, bytes.NewReader(bodyBytes))
		if errReq != nil {
			http.Error(params.w, fmt.Sprintf("Failed to construct upstream request: %v", errReq), http.StatusInternalServerError)
			return
		}
		for k, v := range params.authHeaders {
			upstreamReq.Header.Set(k, v)
		}
		upstreamReq.Header.Set("Content-Type", "application/json")

		resp, errDo := params.client.Do(upstreamReq)
		if errDo != nil {
			http.Error(params.w, fmt.Sprintf("Failed to reach MAGPIE backend: %v", errDo), http.StatusBadGateway)
			return
		}

		respBytes, errRead := io.ReadAll(resp.Body)
		if errClose := resp.Body.Close(); errClose != nil {
			log.Debugf("failed to close response body: %v", errClose)
		}
		if errRead != nil {
			http.Error(params.w, fmt.Sprintf("Failed to read response: %v", errRead), http.StatusBadGateway)
			return
		}

		doc := ojson.NewDoc(respBytes)
		if doc.Get("@this").Exists() && resp.StatusCode == http.StatusOK {
			if params.isMalformedDoc(doc) && attempt < MaxMalformedRetries {
				log.Warnf("[MAGPIE] %s JSON response returned malformed reply with no valid output; retrying silently (%d/%d)", params.logName, attempt+1, MaxMalformedRetries)
				continue
			}

			var acc tokenAccounting
			params.transformDoc(doc, &acc)

			coreusage.PublishRecord(params.r.Context(), coreusage.Record{
				RequestID:   params.reqID,
				TraceID:     params.traceID,
				Provider:    params.provider,
				Model:       params.prefixedModel,
				APIKey:      params.apiKeyPrincipal,
				Stream:      false,
				RequestedAt: params.startTime,
				Latency:     time.Since(params.startTime),
				Detail: coreusage.Detail{
					InputTokens:     acc.inputTokens,
					OutputTokens:    acc.outputTokens,
					ReasoningTokens: acc.reasoningTokens,
					TotalTokens:     acc.totalTokens,
					CachedTokens:    acc.cachedTokens,
				},
			})

			params.w.Header().Set("Content-Type", "application/json")
			params.w.WriteHeader(resp.StatusCode)
			_, _ = params.w.Write(doc.Bytes())
			return
		}

		for k, vv := range resp.Header {
			for _, v := range vv {
				params.w.Header().Add(k, v)
			}
		}
		params.w.WriteHeader(resp.StatusCode)
		_, _ = params.w.Write(respBytes)
		return
	}
}

// ForwardGeminiRequest handles forwarding of Gemini requests to the local MAGPIE daemon.
func ForwardGeminiRequest(w http.ResponseWriter, r *http.Request, rawModel, method string, rawBody []byte, inheritedProxy string, apiKeyPrincipal string) {
	baseURL, apiKey, magpieCfg, errInternal := InternalBaseURLAndKey(r.Context(), inheritedProxy)
	if errInternal != nil {
		http.Error(w, fmt.Sprintf("MAGPIE backend error: %v", errInternal), http.StatusBadGateway)
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
	baseBodyBytes, _ := json.Marshal(bodyMap)

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

	client := getForwardHTTPClient()
	isStreaming := (method == "streamGenerateContent")
	startTime := time.Now()

	prefixedModel := PrefixMagpieModel(cleanRawModel)
	reqID := coreusage.ExecutionRequestIDFromContext(r.Context())
	if reqID == "" {
		reqID = uuid.New().String()
	}
	traceID := coreusage.TraceIDFromContext(r.Context())

	authHeaders := map[string]string{
		"Authorization":  fmt.Sprintf("Bearer %s", apiKey),
		"x-goog-api-key": apiKey,
	}

	if isStreaming {
		forwardSSEStream(sseForwardParams{
			w:               w,
			r:               r,
			client:          client,
			targetURL:       targetURL,
			authHeaders:     authHeaders,
			baseBodyBytes:   baseBodyBytes,
			perturbBody:     BuildGeminiRetryBodyBytes,
			logName:         "Gemini",
			provider:        "magpie",
			prefixedModel:   prefixedModel,
			apiKeyPrincipal: apiKeyPrincipal,
			reqID:           reqID,
			traceID:         traceID,
			startTime:       startTime,
			transformDoc: func(doc *ojson.Doc, isCommitCandidate *bool, isMalformed *bool, acc *tokenAccounting) {
				RestoreToolAliasesDoc(doc, reverseToolMap)
				rewriteGeminiDocModel(doc)
				extractGeminiDocUsage(doc, acc)
				if !*isCommitCandidate {
					if GeminiDocHasValidOutput(doc) {
						*isCommitCandidate = true
					} else if isGeminiMalformedDoc(doc) {
						*isMalformed = true
					}
				}
			},
		})
		return
	}

	forwardJSONRequest(jsonForwardParams{
		w:               w,
		r:               r,
		client:          client,
		targetURL:       targetURL,
		authHeaders:     authHeaders,
		baseBodyBytes:   baseBodyBytes,
		perturbBody:     BuildGeminiRetryBodyBytes,
		isMalformedDoc:  isGeminiMalformedDoc,
		logName:         "Gemini",
		provider:        "magpie",
		prefixedModel:   prefixedModel,
		apiKeyPrincipal: apiKeyPrincipal,
		reqID:           reqID,
		traceID:         traceID,
		startTime:       startTime,
		transformDoc: func(doc *ojson.Doc, acc *tokenAccounting) {
			RestoreToolAliasesDoc(doc, reverseToolMap)
			rewriteGeminiDocModel(doc)
			extractGeminiDocUsage(doc, acc)
		},
	})
}

// ForwardOpenAIChatRequest handles forwarding of OpenAI chat completion requests to the local MAGPIE daemon.
func ForwardOpenAIChatRequest(w http.ResponseWriter, r *http.Request, rawModel string, rawBody []byte, inheritedProxy string, apiKeyPrincipal string) {
	baseURL, apiKey, magpieCfg, errInternal := InternalBaseURLAndKey(r.Context(), inheritedProxy)
	if errInternal != nil {
		http.Error(w, fmt.Sprintf("MAGPIE backend error: %v", errInternal), http.StatusBadGateway)
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

	bodyMap["model"] = cleanRawModel

	// Apply feature toggles
	if !magpieCfg.AllowTools {
		StripOpenAITools(bodyMap)
	}
	if !magpieCfg.AllowImages {
		StripOpenAIImages(bodyMap)
	}
	reverseToolMap := AliasOpenAITools(bodyMap)
	baseBodyBytes, _ := json.Marshal(bodyMap)

	targetURL := fmt.Sprintf("%s/v1/chat/completions", baseURL)
	client := getForwardHTTPClient()
	isStreamReq, _ := bodyMap["stream"].(bool)
	startTime := time.Now()

	prefixedModel := PrefixMagpieModel(cleanRawModel)
	reqID := coreusage.ExecutionRequestIDFromContext(r.Context())
	if reqID == "" {
		reqID = uuid.New().String()
	}
	traceID := coreusage.TraceIDFromContext(r.Context())

	authHeaders := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
	}

	if isStreamReq {
		forwardSSEStream(sseForwardParams{
			w:               w,
			r:               r,
			client:          client,
			targetURL:       targetURL,
			authHeaders:     authHeaders,
			baseBodyBytes:   baseBodyBytes,
			perturbBody:     BuildOpenAIRetryBodyBytes,
			logName:         "OpenAI",
			provider:        "magpie",
			prefixedModel:   prefixedModel,
			apiKeyPrincipal: apiKeyPrincipal,
			reqID:           reqID,
			traceID:         traceID,
			startTime:       startTime,
			transformDoc: func(doc *ojson.Doc, isCommitCandidate *bool, isMalformed *bool, acc *tokenAccounting) {
				RestoreToolAliasesDoc(doc, reverseToolMap)
				rewriteOpenAIDocModel(doc, prefixedModel)
				extractOpenAIDocUsage(doc, acc)
				if !*isCommitCandidate {
					if OpenAIDocHasValidOutput(doc) {
						*isCommitCandidate = true
					} else if isOpenAIMalformedDoc(doc) {
						*isMalformed = true
					}
				}
			},
		})
		return
	}

	forwardJSONRequest(jsonForwardParams{
		w:               w,
		r:               r,
		client:          client,
		targetURL:       targetURL,
		authHeaders:     authHeaders,
		baseBodyBytes:   baseBodyBytes,
		perturbBody:     BuildOpenAIRetryBodyBytes,
		isMalformedDoc:  isOpenAIMalformedDoc,
		logName:         "OpenAI",
		provider:        "magpie",
		prefixedModel:   prefixedModel,
		apiKeyPrincipal: apiKeyPrincipal,
		reqID:           reqID,
		traceID:         traceID,
		startTime:       startTime,
		transformDoc: func(doc *ojson.Doc, acc *tokenAccounting) {
			RestoreToolAliasesDoc(doc, reverseToolMap)
			rewriteOpenAIDocModel(doc, prefixedModel)
			extractOpenAIDocUsage(doc, acc)
		},
	})
}

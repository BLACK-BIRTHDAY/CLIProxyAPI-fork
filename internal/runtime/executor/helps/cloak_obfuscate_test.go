package helps

import (
	"strings"
	"testing"

	"github.com/MoeForge/moego/ojson"
)

func TestAntigravityIdentitySentenceReplacementInParts(t *testing.T) {
	raw := `{
		"request": {
			"systemInstruction": {
				"parts": [
					{
						"text": "You are a Claude agent, built on Anthropic's Claude Agent SDK."
					}
				]
			}
		}
	}`
	doc := ojson.NewDoc([]byte(raw))
	matcher := BuildAntigravitySensitiveWordMatcher(nil)
	modified := ObfuscateSensitiveWordsInSystemInstructionDoc(doc, matcher)
	if !modified {
		t.Fatalf("expected doc to be modified, but got false")
	}

	got := doc.Get("request.systemInstruction.parts.0.text").String()
	want := AntigravityCanonicalIdentitySentence
	if got != want {
		t.Fatalf("system instruction text = %q, want %q", got, want)
	}
}

func TestAntigravityIdentitySentenceCaseInsensitive(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "exact",
			input: "You are a Claude agent, built on Anthropic's Claude Agent SDK.",
		},
		{
			name:  "all lower",
			input: "you are a claude agent, built on anthropic's claude agent sdk.",
		},
		{
			name:  "all caps",
			input: "YOU ARE A CLAUDE AGENT, BUILT ON ANTHROPIC'S CLAUDE AGENT SDK.",
		},
		{
			name:  "mixed case",
			input: "You Are A Claude Agent, Built On Anthropic's Claude Agent SDK.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := `{"request":{"systemInstruction":{"parts":[{"text":"` + tc.input + `"}]}}}`
			doc := ojson.NewDoc([]byte(raw))
			matcher := BuildAntigravitySensitiveWordMatcher(nil)
			if !ObfuscateSensitiveWordsInSystemInstructionDoc(doc, matcher) {
				t.Fatalf("expected doc to be modified")
			}
			got := doc.Get("request.systemInstruction.parts.0.text").String()
			if got != AntigravityCanonicalIdentitySentence {
				t.Fatalf("system instruction text = %q, want %q", got, AntigravityCanonicalIdentitySentence)
			}
		})
	}
}

func TestAntigravityDefaultSensitiveWordsObfuscatedWithNilConfig(t *testing.T) {
	raw := `{
		"request": {
			"systemInstruction": {
				"parts": [
					{"text": "Connecting via Cherry Studio on Eta using Claude Code with RFC 2119 requirements."}
				]
			}
		}
	}`
	doc := ojson.NewDoc([]byte(raw))
	matcher := BuildAntigravitySensitiveWordMatcher(nil)
	if !ObfuscateSensitiveWordsInSystemInstructionDoc(doc, matcher) {
		t.Fatalf("expected doc to be modified")
	}

	got := doc.Get("request.systemInstruction.parts.0.text").String()

	expectedWords := map[string]string{
		"Cherry Studio": "C" + zeroWidthSpace + "herry Studio",
		"Eta":           "E" + zeroWidthSpace + "ta",
		"Claude Code":   "C" + zeroWidthSpace + "laude Code",
		"RFC 2119":      "R" + zeroWidthSpace + "FC 2119",
	}

	for original, obfuscated := range expectedWords {
		if strings.Contains(got, original) {
			t.Errorf("expected %q to be obfuscated, but still found in %q", original, got)
		}
		if !strings.Contains(got, obfuscated) {
			t.Errorf("expected %q to contain obfuscated version %q", got, obfuscated)
		}
	}
}

func TestAntigravityAllDefaultSensitiveWords(t *testing.T) {
	matcher := BuildAntigravitySensitiveWordMatcher(nil)
	if matcher == nil {
		t.Fatalf("expected non-nil matcher")
	}

	for _, word := range DefaultAntigravitySensitiveWords {
		t.Run(word, func(t *testing.T) {
			raw := `{"request":{"systemInstruction":{"parts":[{"text":"prefix ` + word + ` suffix"}]}}}`
			doc := ojson.NewDoc([]byte(raw))
			if !ObfuscateSensitiveWordsInSystemInstructionDoc(doc, matcher) {
				t.Fatalf("expected doc to be modified for %q", word)
			}
			got := doc.Get("request.systemInstruction.parts.0.text").String()
			if strings.Contains(got, word) {
				t.Fatalf("text still contains un-obfuscated %q: %q", word, got)
			}
			if !strings.Contains(got, zeroWidthSpace) {
				t.Fatalf("text does not contain zero-width space for %q: %q", word, got)
			}
		})
	}
}

func TestAntigravityContentsAndToolsUntouched(t *testing.T) {
	raw := `{
		"request": {
			"systemInstruction": {
				"parts": [
					{
						"text": "You are a Claude agent, built on Anthropic's Claude Agent SDK. Running Cherry Studio on Eta."
					}
				]
			},
			"contents": [
				{
					"role": "user",
					"parts": [
						{
							"text": "You are a Claude agent, built on Anthropic's Claude Agent SDK. Client is Cherry Studio on Eta with Claude Code and RFC 2119."
						}
					]
				},
				{
					"role": "model",
					"parts": [
						{
							"text": "Anthropic and Claude Code are respected here."
						}
					]
				}
			],
			"tools": [
				{
					"functionDeclarations": [
						{
							"name": "run_claude_code",
							"description": "Tool for Claude Code and Anthropic clients adhering to RFC 2119."
						}
					]
				}
			]
		}
	}`
	doc := ojson.NewDoc([]byte(raw))
	matcher := BuildAntigravitySensitiveWordMatcher(nil)
	modified := ObfuscateSensitiveWordsInSystemInstructionDoc(doc, matcher)
	if !modified {
		t.Fatalf("expected doc to be modified")
	}

	// 1. Verify system instruction identity was replaced and sensitive words obfuscated
	systemText := doc.Get("request.systemInstruction.parts.0.text").String()
	if !strings.HasPrefix(systemText, AntigravityCanonicalIdentitySentence) {
		t.Errorf("system instruction identity not replaced: %q", systemText)
	}
	if strings.Contains(systemText, "Cherry Studio") {
		t.Errorf("system instruction contains un-obfuscated 'Cherry Studio': %q", systemText)
	}
	expectedCherry := "C" + zeroWidthSpace + "herry Studio"
	if !strings.Contains(systemText, expectedCherry) {
		t.Errorf("system instruction missing obfuscated 'Cherry Studio': %q", systemText)
	}
	expectedEta := "E" + zeroWidthSpace + "ta"
	if !strings.Contains(systemText, expectedEta) {
		t.Errorf("system instruction missing obfuscated 'Eta': %q", systemText)
	}

	// 2. Verify contents user turn is 100% untouched
	userText := doc.Get("request.contents.0.parts.0.text").String()
	expectedUserText := "You are a Claude agent, built on Anthropic's Claude Agent SDK. Client is Cherry Studio on Eta with Claude Code and RFC 2119."
	if userText != expectedUserText {
		t.Errorf("user message was mutated!\ngot:  %q\nwant: %q", userText, expectedUserText)
	}
	if strings.Contains(userText, zeroWidthSpace) {
		t.Errorf("user message must not contain zero-width space")
	}

	// 3. Verify contents model turn is 100% untouched
	modelText := doc.Get("request.contents.1.parts.0.text").String()
	expectedModelText := "Anthropic and Claude Code are respected here."
	if modelText != expectedModelText {
		t.Errorf("model message was mutated!\ngot:  %q\nwant: %q", modelText, expectedModelText)
	}
	if strings.Contains(modelText, zeroWidthSpace) {
		t.Errorf("model message must not contain zero-width space")
	}

	// 4. Verify tools is 100% untouched
	toolDesc := doc.Get("request.tools.0.functionDeclarations.0.description").String()
	expectedToolDesc := "Tool for Claude Code and Anthropic clients adhering to RFC 2119."
	if toolDesc != expectedToolDesc {
		t.Errorf("tools description was mutated!\ngot:  %q\nwant: %q", toolDesc, expectedToolDesc)
	}
	if strings.Contains(toolDesc, zeroWidthSpace) {
		t.Errorf("tools description must not contain zero-width space")
	}
}

func TestAntigravityStringInstructionAndSnakeCase(t *testing.T) {
	matcher := BuildAntigravitySensitiveWordMatcher(nil)

	// String format on request.systemInstruction
	raw1 := `{"request":{"systemInstruction":"You are a Claude agent, built on Anthropic's Claude Agent SDK. Using Eta."}}`
	doc1 := ojson.NewDoc([]byte(raw1))
	if !ObfuscateSensitiveWordsInSystemInstructionDoc(doc1, matcher) {
		t.Fatalf("expected doc1 to be modified")
	}
	got1 := doc1.Get("request.systemInstruction").String()
	want1 := AntigravityCanonicalIdentitySentence + " Using E" + zeroWidthSpace + "ta."
	if got1 != want1 {
		t.Errorf("string format got = %q, want %q", got1, want1)
	}

	// Snake_case on request.system_instruction
	raw2 := `{"request":{"system_instruction":{"parts":[{"text":"You are a Claude agent, built on Anthropic's Claude Agent SDK."}]}}}`
	doc2 := ojson.NewDoc([]byte(raw2))
	if !ObfuscateSensitiveWordsInSystemInstructionDoc(doc2, matcher) {
		t.Fatalf("expected doc2 to be modified")
	}
	got2 := doc2.Get("request.system_instruction.parts.0.text").String()
	if got2 != AntigravityCanonicalIdentitySentence {
		t.Errorf("snake_case got = %q, want %q", got2, AntigravityCanonicalIdentitySentence)
	}
}

func TestAntigravityMergesConfiguredWords(t *testing.T) {
	customWords := []string{"custom-agent", "secret-code"}
	matcher := BuildAntigravitySensitiveWordMatcher(customWords)
	raw := `{"request":{"systemInstruction":{"parts":[{"text":"Running Cherry Studio with custom-agent and secret-code."}]}}}`
	doc := ojson.NewDoc([]byte(raw))
	if !ObfuscateSensitiveWordsInSystemInstructionDoc(doc, matcher) {
		t.Fatalf("expected doc to be modified")
	}
	got := doc.Get("request.systemInstruction.parts.0.text").String()
	if !strings.Contains(got, "C"+zeroWidthSpace+"herry Studio") {
		t.Errorf("default word Cherry Studio was not obfuscated: %q", got)
	}
	if !strings.Contains(got, "c"+zeroWidthSpace+"ustom-agent") {
		t.Errorf("custom word custom-agent was not obfuscated: %q", got)
	}
	if !strings.Contains(got, "s"+zeroWidthSpace+"ecret-code") {
		t.Errorf("custom word secret-code was not obfuscated: %q", got)
	}
}

func TestAntigravityNilMatcherStillReplacesIdentity(t *testing.T) {
	raw := `{"request":{"systemInstruction":{"parts":[{"text":"You are a Claude agent, built on Anthropic's Claude Agent SDK. Welcome!"}]}}}`
	doc := ojson.NewDoc([]byte(raw))
	if !ObfuscateSensitiveWordsInSystemInstructionDoc(doc, nil) {
		t.Fatalf("expected doc to be modified even with nil matcher")
	}
	got := doc.Get("request.systemInstruction.parts.0.text").String()
	want := AntigravityCanonicalIdentitySentence + " Welcome!"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAntigravityObfuscatePayloadBytes(t *testing.T) {
	raw := []byte(`{"request":{"systemInstruction":{"parts":[{"text":"You are a Claude agent, built on Anthropic's Claude Agent SDK. Client: Cline."}]}}}`)
	matcher := BuildAntigravitySensitiveWordMatcher(nil)
	gotBytes := ObfuscateSensitiveWordsInSystemInstruction(raw, matcher)

	doc := ojson.NewDoc(gotBytes)
	text := doc.Get("request.systemInstruction.parts.0.text").String()
	want := AntigravityCanonicalIdentitySentence + " Client: C" + zeroWidthSpace + "line."
	if text != want {
		t.Fatalf("got %q, want %q", text, want)
	}
}

func TestAntigravityNoInstructionUnchanged(t *testing.T) {
	raw := `{"request":{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}}`
	doc := ojson.NewDoc([]byte(raw))
	matcher := BuildAntigravitySensitiveWordMatcher(nil)
	if ObfuscateSensitiveWordsInSystemInstructionDoc(doc, matcher) {
		t.Fatalf("expected no modification when systemInstruction is absent")
	}
	if ObfuscateSensitiveWordsInSystemInstructionDoc(nil, matcher) {
		t.Fatalf("expected false on nil doc")
	}
}

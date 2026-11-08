package openai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/clienterror"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
)

const (
	aistudioTranscriptionProvider  = "aistudio2api"
	defaultAIStudioTranscribeModel = "gemini-3.5-transcribe"
	openAITranscriptionsPath       = "/v1/audio/transcriptions"
	openAITranslationsPath         = "/v1/audio/translations"
)

// AudioTranscriptions handles OpenAI-compatible POST /v1/audio/transcriptions requests.
func (h *OpenAIAPIHandler) AudioTranscriptions(c *gin.Context) {
	h.handleAudioMultipart(c, openAITranscriptionsPath)
}

// AudioTranslations handles OpenAI-compatible POST /v1/audio/translations requests.
func (h *OpenAIAPIHandler) AudioTranslations(c *gin.Context) {
	h.handleAudioMultipart(c, openAITranslationsPath)
}

func (h *OpenAIAPIHandler) handleAudioMultipart(c *gin.Context, targetPath string) {
	raw, errRead := handlers.ReadRequestBody(c)
	if errRead != nil {
		writeSpeechError(c, clienterror.HTTPStatusFromErrorOr(errRead, http.StatusBadRequest), fmt.Sprintf("Invalid request: %v", errRead))
		return
	}
	if h == nil || h.AuthManager == nil {
		writeSpeechError(c, http.StatusServiceUnavailable, "Audio transcription service is unavailable")
		return
	}

	body, contentType := normalizeAudioMultipartRequest(raw, c.GetHeader("Content-Type"))
	auth := h.selectAIStudioTranscriptionAuth()

	cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())
	httpReq, errReq := http.NewRequestWithContext(cliCtx, http.MethodPost, targetPath, bytes.NewReader(body))
	if errReq != nil {
		writeSpeechError(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", errReq))
		cliCancel(errReq)
		return
	}
	if c.Request != nil {
		for k, vals := range c.Request.Header {
			if strings.EqualFold(k, "Content-Type") || strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Authorization") {
				continue
			}
			for _, v := range vals {
				httpReq.Header.Add(k, v)
			}
		}
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	resp, errDo := h.AuthManager.HttpRequest(cliCtx, auth, httpReq)
	if errDo != nil {
		writeSpeechError(c, clienterror.HTTPStatusFromErrorOr(errDo, http.StatusBadGateway), errDo.Error())
		cliCancel(errDo)
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	respBody, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		writeSpeechError(c, http.StatusBadGateway, fmt.Sprintf("Failed to read transcription response: %v", errBody))
		cliCancel(errBody)
		return
	}

	handlers.WriteUpstreamHeaders(c.Writer.Header(), resp.Header)
	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		c.Header("Content-Type", ct)
	} else {
		c.Header("Content-Type", "application/json")
	}
	c.Writer.Header().Del("Content-Length")
	c.Status(resp.StatusCode)
	_, _ = c.Writer.Write(respBody)
	cliCancel(nil)
}

func (h *OpenAIAPIHandler) selectAIStudioTranscriptionAuth() *coreauth.Auth {
	if h != nil && h.AuthManager != nil {
		for _, candidate := range h.AuthManager.List() {
			if candidate == nil || candidate.Disabled {
				continue
			}
			prov := strings.ToLower(strings.TrimSpace(candidate.Provider))
			if prov == aistudioTranscriptionProvider || prov == "aistudio" || prov == "google-aistudio" {
				return candidate
			}
		}
	}
	return &coreauth.Auth{
		ID:       "aistudio2api-default",
		Provider: aistudioTranscriptionProvider,
		Status:   coreauth.StatusActive,
	}
}

func stripZeroWidthModelChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 0x200b, 0x200c, 0x200d, 0x200e, 0x200f, 0x2060, 0xfeff, 0x00ad:
			continue
		}
		if unicode.IsControl(r) || (!unicode.IsGraphic(r) && !unicode.IsSpace(r)) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func normalizeAudioTranscriptionModel(model string) string {
	trimmed := stripZeroWidthModelChars(model)
	if trimmed == "" {
		return defaultAIStudioTranscribeModel
	}
	trimmed = strings.TrimPrefix(trimmed, "models/")
	for _, prefix := range []string{"aistudio/", "aistudio2api/", "google/", "gemini/", "openai/"} {
		if len(trimmed) >= len(prefix) && strings.EqualFold(trimmed[:len(prefix)], prefix) {
			trimmed = stripZeroWidthModelChars(trimmed[len(prefix):])
			break
		}
	}
	trimmed = strings.TrimPrefix(trimmed, "models/")
	if open := strings.LastIndex(trimmed, "("); open > 0 && strings.HasSuffix(trimmed, ")") {
		trimmed = stripZeroWidthModelChars(trimmed[:open])
	}
	lower := strings.ToLower(trimmed)
	if lower == "" || strings.Contains(lower, "transcribe") || lower == "whisper-1" {
		return defaultAIStudioTranscribeModel
	}
	return trimmed
}

func normalizeAudioMultipartRequest(raw []byte, contentType string) ([]byte, string) {
	mediaType, params, errParse := mime.ParseMediaType(contentType)
	if errParse != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		return raw, contentType
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return raw, contentType
	}

	reader := multipart.NewReader(bytes.NewReader(raw), boundary)
	var out bytes.Buffer
	writer := multipart.NewWriter(&out)
	if errBoundary := writer.SetBoundary(boundary); errBoundary != nil {
		return raw, contentType
	}

	sawModel := false
	for {
		part, errPart := reader.NextPart()
		if errPart == io.EOF {
			break
		}
		if errPart != nil {
			return raw, contentType
		}
		partBytes, errRead := io.ReadAll(part)
		_ = part.Close()
		if errRead != nil {
			return raw, contentType
		}
		if part.FormName() == "model" && part.FileName() == "" {
			sawModel = true
			partBytes = []byte(normalizeAudioTranscriptionModel(string(partBytes)))
			part.Header.Del("Content-Length")
		}
		dst, errCreate := writer.CreatePart(part.Header)
		if errCreate != nil {
			return raw, contentType
		}
		if _, errWrite := dst.Write(partBytes); errWrite != nil {
			return raw, contentType
		}
	}

	if !sawModel {
		if errField := writer.WriteField("model", defaultAIStudioTranscribeModel); errField != nil {
			return raw, contentType
		}
	}
	if errClose := writer.Close(); errClose != nil {
		return raw, contentType
	}
	return out.Bytes(), writer.FormDataContentType()
}

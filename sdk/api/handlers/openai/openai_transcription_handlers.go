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
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	log "github.com/sirupsen/logrus"
)

const (
	aistudioTranscriptionProvider  = "aistudio2api"
	defaultAIStudioTranscribeModel = "gemini-3.5-transcribe"
	openAITranscriptionsPath       = "/v1/audio/transcriptions"
	openAITranslationsPath         = "/v1/audio/translations"
)

var audioBridgeBlockedRequestHeaders = map[string]struct{}{
	"Accept-Encoding":     {},
	"Authorization":       {},
	"Connection":          {},
	"Content-Encoding":    {},
	"Content-Length":      {},
	"Content-Type":        {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
	"X-Api-Key":           {},
	"X-Goog-Api-Key":      {},
}

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

	body, contentType, model, requestedModel := normalizeAudioMultipartRequest(raw, c.GetHeader("Content-Type"))

	cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())
	var reqHeaders http.Header
	if c.Request != nil && c.Request.Header != nil {
		reqHeaders = c.Request.Header.Clone()
	}
	selectionOpts := coreexecutor.Options{
		Headers:         reqHeaders,
		OriginalRequest: raw,
	}
	homeSelection, auth, selectionModel, errSelect := h.selectAIStudioTranscriptionAuth(cliCtx, model, requestedModel, selectionOpts)
	if errSelect != nil {
		writeSpeechError(c, clienterror.HTTPStatusFromErrorOr(errSelect, http.StatusServiceUnavailable), errSelect.Error())
		cliCancel(errSelect)
		return
	}
	if auth == nil {
		if homeSelection != nil {
			homeSelection.End("missing_auth")
		}
		writeSpeechError(c, http.StatusServiceUnavailable, "Audio transcription auth is unavailable")
		cliCancel(nil)
		return
	}
	if homeSelection != nil {
		attemptCtx, releaseAttempt, errBind := homeSelection.AttemptContext(cliCtx)
		if errBind != nil {
			homeSelection.End("attempt_bind_failed")
			writeSpeechError(c, http.StatusServiceUnavailable, errBind.Error())
			cliCancel(errBind)
			return
		}
		cliCtx = attemptCtx
		defer releaseAttempt()
		defer func() {
			if homeSelection.Active() && !homeSelection.Retained() {
				homeSelection.End("request_closed")
			}
		}()
	}
	routeModel := requestedModel
	if routeModel == "" {
		routeModel = model
	}

	targetURL := resolveAudioBridgeTargetURL(auth, targetPath)
	httpReq, errReq := http.NewRequestWithContext(cliCtx, http.MethodPost, targetURL, bytes.NewReader(body))
	if errReq != nil {
		if homeSelection != nil {
			homeSelection.End("invalid_request")
		}
		writeSpeechError(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", errReq))
		cliCancel(errReq)
		return
	}
	if c.Request != nil {
		copyFilteredAudioBridgeRequestHeaders(httpReq.Header, c.Request.Header)
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	resp, errDo := h.AuthManager.HttpRequest(cliCtx, auth, httpReq)
	if errDo != nil {
		if homeSelection != nil {
			homeSelection.End("request_failed")
		}
		status := clienterror.HTTPStatusFromErrorOr(errDo, http.StatusBadGateway)
		h.AuthManager.MarkResult(cliCtx, coreauth.Result{
			AuthID:            auth.ID,
			Provider:          auth.Provider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           false,
			Error:             &coreauth.Error{Message: errDo.Error(), HTTPStatus: status},
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
		})
		writeSpeechError(c, status, errDo.Error())
		cliCancel(errDo)
		return
	}
	closeResponseBody := func() error {
		if resp == nil || resp.Body == nil {
			return nil
		}
		errClose := resp.Body.Close()
		if errClose != nil {
			log.Errorf("openai audio transcription: close response body error: %v", errClose)
		}
		return errClose
	}
	if homeSelection != nil {
		if errBind := homeSelection.Bind(closeResponseBody); errBind != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				h.AuthManager.ReportHomeUnauthorized(cliCtx, auth, auth.Provider, routeModel)
			}
			homeSelection.End("response_bind_failed")
			writeSpeechError(c, http.StatusServiceUnavailable, errBind.Error())
			cliCancel(errBind)
			return
		}
		defer homeSelection.End("response_closed")
	} else {
		defer func() {
			_ = closeResponseBody()
		}()
	}

	respBody, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		if homeSelection != nil && resp.StatusCode == http.StatusUnauthorized {
			h.AuthManager.ReportHomeUnauthorized(cliCtx, auth, auth.Provider, routeModel, respBody)
		}
		h.AuthManager.MarkResult(cliCtx, coreauth.Result{
			AuthID:            auth.ID,
			Provider:          auth.Provider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           false,
			Error:             &coreauth.Error{Message: errBody.Error(), HTTPStatus: http.StatusBadGateway},
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
		})
		writeSpeechError(c, http.StatusBadGateway, fmt.Sprintf("Failed to read transcription response: %v", errBody))
		cliCancel(errBody)
		return
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if homeSelection != nil && resp.StatusCode == http.StatusUnauthorized {
			h.AuthManager.ReportHomeUnauthorized(cliCtx, auth, auth.Provider, routeModel, respBody)
		}
		errMsg := strings.TrimSpace(string(respBody))
		if errMsg == "" {
			errMsg = http.StatusText(resp.StatusCode)
		}
		h.AuthManager.MarkResult(cliCtx, coreauth.Result{
			AuthID:            auth.ID,
			Provider:          auth.Provider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           false,
			Error:             &coreauth.Error{Message: errMsg, HTTPStatus: resp.StatusCode},
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
		})
	} else {
		h.AuthManager.MarkResult(cliCtx, coreauth.Result{
			AuthID:            auth.ID,
			Provider:          auth.Provider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           true,
			CompletedResponse: true,
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
		})
	}

	filteredHeaders := handlers.FilterUpstreamHeaders(resp.Header)
	handlers.WriteUpstreamHeaders(c.Writer.Header(), filteredHeaders)
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

func copyFilteredAudioBridgeRequestHeaders(dst, src http.Header) {
	if dst == nil || src == nil {
		return
	}
	connectionScoped := make(map[string]struct{})
	for _, rawValue := range src.Values("Connection") {
		for _, token := range strings.Split(rawValue, ",") {
			if headerName := strings.TrimSpace(token); headerName != "" {
				connectionScoped[http.CanonicalHeaderKey(headerName)] = struct{}{}
			}
		}
	}
	for k, vals := range src {
		canonicalKey := http.CanonicalHeaderKey(k)
		if _, blocked := audioBridgeBlockedRequestHeaders[canonicalKey]; blocked {
			continue
		}
		if _, scoped := connectionScoped[canonicalKey]; scoped {
			continue
		}
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

func resolveAudioBridgeTargetURL(auth *coreauth.Auth, targetPath string) string {
	if auth == nil || auth.Attributes == nil {
		return targetPath
	}
	baseURL := strings.TrimRight(strings.TrimSpace(auth.Attributes["base_url"]), "/")
	if baseURL == "" {
		return targetPath
	}
	if strings.HasSuffix(strings.ToLower(baseURL), "/v1") && strings.HasPrefix(targetPath, "/v1/") {
		return baseURL + strings.TrimPrefix(targetPath, "/v1")
	}
	return baseURL + targetPath
}

func (h *OpenAIAPIHandler) selectAIStudioTranscriptionAuth(ctx context.Context, model, requestedModel string, opts coreexecutor.Options) (*coreauth.HomeDispatchSelection, *coreauth.Auth, string, error) {
	if h == nil || h.AuthManager == nil {
		return nil, nil, "", &coreauth.Error{Code: "provider_not_found", Message: "auth manager is nil", HTTPStatus: http.StatusServiceUnavailable}
	}
	lookupModel := model
	registryProviders := registry.GetGlobalRegistry().GetModelProviders(lookupModel)
	if len(registryProviders) == 0 && requestedModel != "" && !strings.EqualFold(requestedModel, lookupModel) {
		if reqProviders := registry.GetGlobalRegistry().GetModelProviders(requestedModel); len(reqProviders) > 0 {
			lookupModel = requestedModel
			registryProviders = reqProviders
		}
	}
	selectionModel := ""
	providers := []string{aistudioTranscriptionProvider, "aistudio", "google-aistudio"}
	if len(registryProviders) > 0 {
		selectionModel = lookupModel
		providers = registryProviders
	}
	if h.AuthManager.HomeEnabled() {
		homeModel := selectionModel
		if homeModel == "" {
			homeModel = lookupModel
		}
		var lastErr error
		for _, prov := range providers {
			selection, errSelect := h.AuthManager.SelectHomeAuth(ctx, prov, homeModel, opts)
			if errSelect == nil && selection != nil {
				return selection, selection.CloneAuth(), homeModel, nil
			}
			if errSelect != nil {
				lastErr = errSelect
			}
		}
		if lastErr != nil {
			return nil, nil, homeModel, lastErr
		}
		return nil, nil, homeModel, &coreauth.Error{Code: "auth_not_found", Message: "no auth available for audio transcription", HTTPStatus: http.StatusServiceUnavailable}
	}
	var lastErr error
	for _, prov := range providers {
		selected, errSelect := h.AuthManager.SelectAuth(ctx, prov, selectionModel, opts)
		if errSelect == nil && selected != nil {
			return nil, selected, selectionModel, nil
		}
		if errSelect != nil {
			lastErr = errSelect
		}
	}
	if lastErr != nil {
		return nil, nil, selectionModel, lastErr
	}
	return nil, nil, selectionModel, &coreauth.Error{Code: "auth_not_found", Message: "no auth available for audio transcription", HTTPStatus: http.StatusServiceUnavailable}
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
	clean := strings.TrimPrefix(trimmed, "models/")
	hasAIStudioPrefix := false
	for _, prefix := range []string{"aistudio/", "aistudio2api/"} {
		if len(clean) >= len(prefix) && strings.EqualFold(clean[:len(prefix)], prefix) {
			clean = stripZeroWidthModelChars(clean[len(prefix):])
			hasAIStudioPrefix = true
			break
		}
	}
	clean = strings.TrimPrefix(clean, "models/")
	base := clean
	if open := strings.LastIndex(base, "("); open > 0 && strings.HasSuffix(base, ")") {
		base = stripZeroWidthModelChars(base[:open])
	}
	switch strings.ToLower(base) {
	case "", "gemini-3.5-transcribe", "gemini-3.5-transcribe-live":
		return defaultAIStudioTranscribeModel
	case "whisper-1":
		if !hasAIStudioPrefix && len(registry.GetGlobalRegistry().GetModelProviders(trimmed)) > 0 {
			return trimmed
		}
		return defaultAIStudioTranscribeModel
	default:
		if hasAIStudioPrefix {
			return clean
		}
		return trimmed
	}
}

func normalizeAudioMultipartRequest(raw []byte, contentType string) ([]byte, string, string, string) {
	mediaType, params, errParse := mime.ParseMediaType(contentType)
	if errParse != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		return raw, contentType, defaultAIStudioTranscribeModel, defaultAIStudioTranscribeModel
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return raw, contentType, defaultAIStudioTranscribeModel, defaultAIStudioTranscribeModel
	}

	reader := multipart.NewReader(bytes.NewReader(raw), boundary)
	var out bytes.Buffer
	writer := multipart.NewWriter(&out)
	if errBoundary := writer.SetBoundary(boundary); errBoundary != nil {
		return raw, contentType, defaultAIStudioTranscribeModel, defaultAIStudioTranscribeModel
	}

	sawModel := false
	normalizedModel := defaultAIStudioTranscribeModel
	requestedModel := defaultAIStudioTranscribeModel
	for {
		part, errPart := reader.NextPart()
		if errPart == io.EOF {
			break
		}
		if errPart != nil {
			return raw, contentType, normalizedModel, requestedModel
		}
		partBytes, errRead := io.ReadAll(part)
		_ = part.Close()
		if errRead != nil {
			return raw, contentType, normalizedModel, requestedModel
		}
		if part.FormName() == "model" && part.FileName() == "" {
			sawModel = true
			requestedModel = stripZeroWidthModelChars(string(partBytes))
			normalizedModel = normalizeAudioTranscriptionModel(string(partBytes))
			partBytes = []byte(normalizedModel)
			part.Header.Del("Content-Length")
		}
		dst, errCreate := writer.CreatePart(part.Header)
		if errCreate != nil {
			return raw, contentType, normalizedModel, requestedModel
		}
		if _, errWrite := dst.Write(partBytes); errWrite != nil {
			return raw, contentType, normalizedModel, requestedModel
		}
	}

	if !sawModel {
		if errField := writer.WriteField("model", defaultAIStudioTranscribeModel); errField != nil {
			return raw, contentType, normalizedModel, requestedModel
		}
	}
	if errClose := writer.Close(); errClose != nil {
		return raw, contentType, normalizedModel, requestedModel
	}
	if requestedModel == "" {
		requestedModel = normalizedModel
	}
	return out.Bytes(), writer.FormDataContentType(), normalizedModel, requestedModel
}

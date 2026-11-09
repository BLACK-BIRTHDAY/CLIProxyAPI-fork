package openai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/clienterror"
	internalconfig "github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/runtime/executor/helps"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/thinking"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
	log "github.com/sirupsen/logrus"
)

const (
	openAITranscriptionsPath = "/v1/audio/transcriptions"
	openAITranslationsPath   = "/v1/audio/translations"
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

	body, contentType, model, requestedModel, streamRequested := normalizeAudioMultipartRequest(raw, c.GetHeader("Content-Type"))
	if strings.TrimSpace(model) == "" {
		writeSpeechError(c, http.StatusBadRequest, "Invalid request: model is required")
		return
	}

	cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())
	var reqHeaders http.Header
	if c.Request != nil && c.Request.Header != nil {
		reqHeaders = c.Request.Header.Clone()
	} else {
		reqHeaders = make(http.Header)
	}
	if reqHeaders.Get("Content-Type") == "" && c.GetHeader("Content-Type") != "" {
		reqHeaders.Set("Content-Type", c.GetHeader("Content-Type"))
	}
	routeModel := requestedModel
	if routeModel == "" {
		routeModel = model
	}
	selectionOpts := coreexecutor.Options{
		Stream:          streamRequested,
		Headers:         reqHeaders,
		OriginalRequest: raw,
		SourceFormat:    sdktranslator.FromString("openai"),
		Metadata: map[string]any{
			coreexecutor.RequestedModelMetadataKey: routeModel,
			coreexecutor.RequestPathMetadataKey:    targetPath,
		},
	}
	homeSelection, auth, selectionModel, errSelect := h.selectTranscriptionAuth(cliCtx, model, requestedModel, selectionOpts)
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

	upstreamModel, stateModel, okUpstream := h.AuthManager.ResolveExecutionModelForRequest(cliCtx, auth, selectionModel, homeSelection == nil, reqHeaders)
	if !okUpstream {
		if homeSelection != nil {
			homeSelection.End("no_execution_models")
		}
		errUnavailable := &coreauth.Error{Code: "auth_unavailable", Message: "no execution models available", HTTPStatus: http.StatusServiceUnavailable}
		writeSpeechError(c, http.StatusServiceUnavailable, errUnavailable.Error())
		cliCancel(errUnavailable)
		return
	}
	baseUpstreamModel := strings.TrimSpace(thinking.ParseSuffix(upstreamModel).ModelName)
	if baseUpstreamModel == "" {
		baseUpstreamModel = strings.TrimSpace(upstreamModel)
	}
	if baseUpstreamModel == "" {
		baseUpstreamModel = model
	}
	if strings.TrimSpace(stateModel) != "" {
		selectionModel = stateModel
	}
	if baseUpstreamModel != model {
		body, contentType = rewriteAudioMultipartModelField(body, contentType, baseUpstreamModel)
	}

	runtimeCfg := h.AuthManager.RuntimeConfigSnapshot()
	if runtimeCfg == nil {
		if sdkCfg := h.CurrentConfig(); sdkCfg != nil {
			runtimeCfg = &internalconfig.Config{SDKConfig: *sdkCfg}
		}
	}
	finalizedBody, finalizedContentType, errFinalize := helps.ApplyMediaPayloadConfig(
		runtimeCfg,
		strings.TrimSpace(auth.Provider),
		baseUpstreamModel,
		"openai",
		body,
		contentType,
		coreexecutor.Request{Model: baseUpstreamModel, Payload: raw},
		selectionOpts,
	)
	if errFinalize != nil {
		if homeSelection != nil {
			homeSelection.End("invalid_request")
		}
		writeSpeechError(c, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", errFinalize))
		cliCancel(errFinalize)
		return
	}
	body = finalizedBody
	contentType = finalizedContentType

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

	ephemeralResult := homeSelection != nil
	resultProvider := auth.Provider
	if homeSelection != nil && strings.TrimSpace(homeSelection.Provider) != "" {
		resultProvider = homeSelection.Provider
	}
	recordResult := func(res coreauth.Result) {
		if res.Provider == "" {
			res.Provider = resultProvider
		}
		h.AuthManager.RecordExecutionResult(cliCtx, res, auth, ephemeralResult)
	}

	executionSeq := h.AuthManager.NextExecutionSeq()
	var (
		resp  *http.Response
		errDo error
	)
	if homeSelection != nil {
		var preparedAuth *coreauth.Auth
		preparedAuth, resp, errDo = h.AuthManager.ExecuteHomeHttpRequest(cliCtx, homeSelection, httpReq)
		if preparedAuth != nil {
			auth = preparedAuth
		}
	} else {
		resp, errDo = h.AuthManager.HttpRequest(cliCtx, auth, httpReq)
	}
	if errDo != nil {
		if homeSelection != nil {
			homeSelection.End("request_failed")
		}
		status := clienterror.HTTPStatusFromErrorOr(errDo, http.StatusBadGateway)
		recordResult(coreauth.Result{
			AuthID:            auth.ID,
			Provider:          resultProvider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           false,
			Error:             classifyAudioBridgeError(errDo, status),
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
			ExecutionSeq:      executionSeq,
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
				h.AuthManager.ReportHomeUnauthorized(cliCtx, auth, resultProvider, routeModel)
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

	retryAfter := parseAudioBridgeRetryAfterHeader(resp.Header, time.Now())
	if resp.StatusCode < http.StatusBadRequest && (streamRequested || isAudioEventStreamContentType(resp.Header.Get("Content-Type"))) {
		filteredHeaders := handlers.FilterUpstreamHeaders(resp.Header)
		handlers.WriteUpstreamHeaders(c.Writer.Header(), filteredHeaders)
		if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
			c.Header("Content-Type", ct)
		} else {
			c.Header("Content-Type", "text/event-stream")
		}
		c.Writer.Header().Del("Content-Length")
		c.Status(resp.StatusCode)
		c.Writer.Flush()

		buf := make([]byte, 4096)
		for {
			n, errStream := resp.Body.Read(buf)
			if n > 0 {
				if _, errWrite := c.Writer.Write(buf[:n]); errWrite != nil {
					recordResult(coreauth.Result{
						AuthID:            auth.ID,
						Provider:          resultProvider,
						Model:             selectionModel,
						RouteModel:        routeModel,
						Success:           false,
						Error:             classifyAudioBridgeStreamWriteError(cliCtx, errWrite),
						RetryAfter:        retryAfter,
						Options:           selectionOpts,
						CredentialVersion: auth.CredentialVersion,
						RegistrationEpoch: auth.RegistrationEpoch,
						ExecutionSeq:      executionSeq,
					})
					cliCancel(errWrite)
					return
				}
				c.Writer.Flush()
			}
			if errStream == io.EOF {
				break
			}
			if errStream != nil {
				recordResult(coreauth.Result{
					AuthID:            auth.ID,
					Provider:          resultProvider,
					Model:             selectionModel,
					RouteModel:        routeModel,
					Success:           false,
					Error:             classifyAudioBridgeError(errStream, http.StatusBadGateway),
					RetryAfter:        retryAfter,
					Options:           selectionOpts,
					CredentialVersion: auth.CredentialVersion,
					RegistrationEpoch: auth.RegistrationEpoch,
					ExecutionSeq:      executionSeq,
				})
				cliCancel(errStream)
				return
			}
		}

		recordResult(coreauth.Result{
			AuthID:            auth.ID,
			Provider:          resultProvider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           true,
			CompletedResponse: true,
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
			ExecutionSeq:      executionSeq,
		})
		cliCancel(nil)
		return
	}

	respBody, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		if homeSelection != nil && resp.StatusCode == http.StatusUnauthorized {
			h.AuthManager.ReportHomeUnauthorized(cliCtx, auth, resultProvider, routeModel, respBody)
		}
		status := clienterror.HTTPStatusFromErrorOr(errBody, http.StatusBadGateway)
		recordResult(coreauth.Result{
			AuthID:            auth.ID,
			Provider:          resultProvider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           false,
			Error:             classifyAudioBridgeError(errBody, status),
			RetryAfter:        retryAfter,
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
			ExecutionSeq:      executionSeq,
		})
		writeSpeechError(c, status, fmt.Sprintf("Failed to read transcription response: %v", errBody))
		cliCancel(errBody)
		return
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if homeSelection != nil && resp.StatusCode == http.StatusUnauthorized {
			h.AuthManager.ReportHomeUnauthorized(cliCtx, auth, resultProvider, routeModel, respBody)
		}
		errMsg := strings.TrimSpace(string(respBody))
		if errMsg == "" {
			errMsg = http.StatusText(resp.StatusCode)
		}
		recordResult(coreauth.Result{
			AuthID:            auth.ID,
			Provider:          resultProvider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           false,
			Error:             &coreauth.Error{Message: errMsg, HTTPStatus: resp.StatusCode},
			RetryAfter:        retryAfter,
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
			ExecutionSeq:      executionSeq,
		})
	} else {
		recordResult(coreauth.Result{
			AuthID:            auth.ID,
			Provider:          resultProvider,
			Model:             selectionModel,
			RouteModel:        routeModel,
			Success:           true,
			CompletedResponse: true,
			Options:           selectionOpts,
			CredentialVersion: auth.CredentialVersion,
			RegistrationEpoch: auth.RegistrationEpoch,
			ExecutionSeq:      executionSeq,
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

func classifyAudioBridgeError(err error, fallbackStatus int) *coreauth.Error {
	if err == nil {
		return nil
	}
	classified := coreauth.ResultErrorFromError(err)
	if classified == nil {
		return &coreauth.Error{Message: err.Error(), HTTPStatus: fallbackStatus}
	}
	if classified.HTTPStatus == 0 {
		classified.HTTPStatus = fallbackStatus
	}
	return classified
}

func classifyAudioBridgeStreamWriteError(ctx context.Context, err error) *coreauth.Error {
	if err == nil {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		classified := coreauth.ResultErrorFromError(ctx.Err())
		if classified != nil {
			if classified.HTTPStatus == 0 {
				classified.HTTPStatus = clienterror.StatusClientClosedRequest
			}
			return classified
		}
	}
	classified := coreauth.ResultErrorFromError(err)
	if classified == nil {
		classified = &coreauth.Error{Message: err.Error()}
	}
	if classified.Code == "" {
		classified.Code = coreauth.ErrorCodeConnectionLifecycle
	}
	if classified.HTTPStatus == 0 {
		classified.HTTPStatus = clienterror.StatusClientClosedRequest
	}
	return classified
}

func rewriteAudioMultipartModelField(raw []byte, contentType, targetModel string) ([]byte, string) {
	targetModel = strings.TrimSpace(targetModel)
	if targetModel == "" {
		return raw, contentType
	}
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
		if part.FileName() == "" && part.FormName() == "model" {
			if sawModel {
				continue
			}
			sawModel = true
			partBytes = []byte(targetModel)
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
	if errClose := writer.Close(); errClose != nil {
		return raw, contentType
	}
	return out.Bytes(), writer.FormDataContentType()
}

func isAudioEventStreamContentType(contentType string) bool {
	mediaType, _, errParse := mime.ParseMediaType(strings.TrimSpace(contentType))
	if errParse != nil {
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "text/event-stream")
	}
	return strings.EqualFold(mediaType, "text/event-stream")
}

func isAudioStreamFieldTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1":
		return true
	default:
		return false
	}
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

func (h *OpenAIAPIHandler) selectTranscriptionAuth(ctx context.Context, model, requestedModel string, opts coreexecutor.Options) (*coreauth.HomeDispatchSelection, *coreauth.Auth, string, error) {
	if h == nil || h.AuthManager == nil {
		return nil, nil, "", &coreauth.Error{Code: "provider_not_found", Message: "auth manager is nil", HTTPStatus: http.StatusServiceUnavailable}
	}
	lookupModel := strings.TrimSpace(model)
	if lookupModel == "" {
		lookupModel = strings.TrimSpace(requestedModel)
	}
	modelReg := registry.GetGlobalRegistry()
	registryProviders := modelReg.GetModelProviders(lookupModel)
	if len(registryProviders) == 0 && lookupModel != "" {
		if parsed := thinking.ParseSuffix(lookupModel); parsed.ModelName != "" && !strings.EqualFold(parsed.ModelName, lookupModel) {
			if baseProviders := modelReg.GetModelProviders(strings.TrimSpace(parsed.ModelName)); len(baseProviders) > 0 {
				lookupModel = strings.TrimSpace(parsed.ModelName)
				registryProviders = baseProviders
			}
		}
	}
	if len(registryProviders) == 0 && requestedModel != "" && !strings.EqualFold(requestedModel, lookupModel) {
		if reqProviders := modelReg.GetModelProviders(requestedModel); len(reqProviders) > 0 {
			lookupModel = requestedModel
			registryProviders = reqProviders
		}
	}
	selectionModel := lookupModel
	if selectionModel == "" {
		selectionModel = requestedModel
	}
	if h.AuthManager.HomeEnabled() {
		providers := registryProviders
		if len(providers) == 0 {
			providers = []string{""}
		}
		var lastErr error
		for _, prov := range providers {
			selection, errSelect := h.AuthManager.SelectHomeAuth(ctx, prov, selectionModel, opts)
			if errSelect == nil && selection != nil {
				return selection, selection.CloneAuthForRoute(selectionModel), selectionModel, nil
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
	if len(registryProviders) == 0 {
		return nil, nil, selectionModel, &coreauth.Error{
			Code:       "model_not_found",
			Message:    "unknown provider for model " + selectionModel,
			HTTPStatus: http.StatusBadRequest,
		}
	}
	var lastErr error
	for _, prov := range registryProviders {
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

func parseAudioBridgeRetryAfterHeader(headers http.Header, now time.Time) *time.Duration {
	if headers == nil {
		return nil
	}
	raw := strings.TrimSpace(headers.Get("Retry-After"))
	if raw == "" {
		return nil
	}
	if seconds, errParse := strconv.ParseInt(raw, 10, 64); errParse == nil && seconds >= 0 {
		if seconds > int64(math.MaxInt64)/int64(time.Second) {
			delay := time.Duration(math.MaxInt64)
			return &delay
		}
		delay := time.Duration(seconds) * time.Second
		return &delay
	}
	if deadline, errParse := http.ParseTime(raw); errParse == nil {
		delay := deadline.Sub(now)
		if delay < 0 {
			delay = 0
		}
		return &delay
	}
	return nil
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
		return ""
	}
	return strings.TrimPrefix(trimmed, "models/")
}

func normalizeAudioMultipartRequest(raw []byte, contentType string) ([]byte, string, string, string, bool) {
	mediaType, params, errParse := mime.ParseMediaType(contentType)
	if errParse != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		return raw, contentType, "", "", false
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return raw, contentType, "", "", false
	}

	reader := multipart.NewReader(bytes.NewReader(raw), boundary)
	var out bytes.Buffer
	writer := multipart.NewWriter(&out)
	if errBoundary := writer.SetBoundary(boundary); errBoundary != nil {
		return raw, contentType, "", "", false
	}

	sawModel := false
	normalizedModel := ""
	requestedModel := ""
	streamRequested := false
	for {
		part, errPart := reader.NextPart()
		if errPart == io.EOF {
			break
		}
		if errPart != nil {
			return raw, contentType, normalizedModel, requestedModel, streamRequested
		}
		partBytes, errRead := io.ReadAll(part)
		_ = part.Close()
		if errRead != nil {
			return raw, contentType, normalizedModel, requestedModel, streamRequested
		}
		if part.FileName() == "" {
			switch part.FormName() {
			case "model":
				if sawModel {
					continue
				}
				sawModel = true
				requestedModel = stripZeroWidthModelChars(string(partBytes))
				normalizedModel = normalizeAudioTranscriptionModel(string(partBytes))
				partBytes = []byte(normalizedModel)
				part.Header.Del("Content-Length")
			case "stream":
				if !streamRequested {
					streamRequested = isAudioStreamFieldTruthy(string(partBytes))
				}
			}
		}
		dst, errCreate := writer.CreatePart(part.Header)
		if errCreate != nil {
			return raw, contentType, normalizedModel, requestedModel, streamRequested
		}
		if _, errWrite := dst.Write(partBytes); errWrite != nil {
			return raw, contentType, normalizedModel, requestedModel, streamRequested
		}
	}

	if errClose := writer.Close(); errClose != nil {
		return raw, contentType, normalizedModel, requestedModel, streamRequested
	}
	if requestedModel == "" {
		requestedModel = normalizedModel
	}
	return out.Bytes(), writer.FormDataContentType(), normalizedModel, requestedModel, streamRequested
}

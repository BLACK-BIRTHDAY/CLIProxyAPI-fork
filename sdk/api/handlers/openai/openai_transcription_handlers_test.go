package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	internalconfig "github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	apihandlers "github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executionregistry"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
)

const (
	testAIStudioTranscriptionProvider = "aistudio2api"
	testAIStudioTranscribeModel       = "gemini-3.5-transcribe"
)

type transcriptionCaptureExecutor struct {
	provider    string
	mu          sync.Mutex
	urls        []string
	paths       []string
	models      []string
	prompts     []string
	files       []string
	fileData    [][]byte
	authIDs     []string
	reqHeaders  []http.Header
	failAuthIDs map[string]int
	respHeaders http.Header
	customResp  func(auth *coreauth.Auth, req *http.Request) (*http.Response, error)
}

func (e *transcriptionCaptureExecutor) Identifier() string {
	if e.provider != "" {
		return e.provider
	}
	return testAIStudioTranscriptionProvider
}

func (e *transcriptionCaptureExecutor) Execute(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, &coreauth.Error{Code: "not_implemented", Message: "Execute not implemented"}
}

func (e *transcriptionCaptureExecutor) ExecuteStream(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (*coreexecutor.StreamResult, error) {
	return nil, &coreauth.Error{Code: "not_implemented", Message: "ExecuteStream not implemented"}
}

func (e *transcriptionCaptureExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *transcriptionCaptureExecutor) CountTokens(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, &coreauth.Error{Code: "not_implemented", Message: "CountTokens not implemented"}
}

func (e *transcriptionCaptureExecutor) HttpRequest(_ context.Context, auth *coreauth.Auth, req *http.Request) (*http.Response, error) {
	if errParse := req.ParseMultipartForm(1 << 20); errParse != nil {
		return nil, errParse
	}
	model := req.FormValue("model")
	file, hdr, errFile := req.FormFile("file")
	if errFile != nil {
		return nil, errFile
	}
	defer func() { _ = file.Close() }()
	data, errRead := io.ReadAll(file)
	if errRead != nil {
		return nil, errRead
	}
	authID := ""
	if auth != nil {
		authID = auth.ID
	}
	e.mu.Lock()
	e.urls = append(e.urls, req.URL.String())
	e.paths = append(e.paths, req.URL.Path)
	e.models = append(e.models, model)
	e.prompts = append(e.prompts, req.FormValue("prompt"))
	e.files = append(e.files, hdr.Filename)
	e.fileData = append(e.fileData, data)
	e.authIDs = append(e.authIDs, authID)
	e.reqHeaders = append(e.reqHeaders, req.Header.Clone())
	failStatus := 0
	if e.failAuthIDs != nil {
		failStatus = e.failAuthIDs[authID]
	}
	respHeaders := http.Header{"Content-Type": []string{"application/json"}}
	for k, vals := range e.respHeaders {
		respHeaders[k] = append([]string(nil), vals...)
	}
	customResp := e.customResp
	e.mu.Unlock()

	if customResp != nil {
		return customResp(auth, req)
	}

	if failStatus >= http.StatusBadRequest {
		return &http.Response{
			StatusCode: failStatus,
			Header:     respHeaders,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"upstream quota exceeded"}}`)),
			Request:    req,
		}, nil
	}

	body := `{"text":"hello live transcription"}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     respHeaders,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func buildTestAudioMultipart(t *testing.T, modelName string, fileName string, audioBytes []byte, extraFields ...map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if modelName != "" {
		if errField := mw.WriteField("model", modelName); errField != nil {
			t.Fatalf("WriteField model: %v", errField)
		}
	}
	for _, fields := range extraFields {
		for k, v := range fields {
			if errField := mw.WriteField(k, v); errField != nil {
				t.Fatalf("WriteField %s: %v", k, errField)
			}
		}
	}
	fw, errCreate := mw.CreateFormFile("file", fileName)
	if errCreate != nil {
		t.Fatalf("CreateFormFile: %v", errCreate)
	}
	if _, errWrite := fw.Write(audioBytes); errWrite != nil {
		t.Fatalf("Write audio: %v", errWrite)
	}
	if errClose := mw.Close(); errClose != nil {
		t.Fatalf("Close multipart writer: %v", errClose)
	}
	return buf.Bytes(), mw.FormDataContentType()
}

func TestAudioTranscriptionsAndTranslationsStandardRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := &transcriptionCaptureExecutor{
		respHeaders: http.Header{
			"Set-Cookie":        []string{"sid=secret; HttpOnly"},
			"Connection":        []string{"X-Internal-Hop"},
			"X-Internal-Hop":    []string{"should-be-filtered"},
			"X-Upstream-Custom": []string{"preserved-value"},
		},
	}
	manager := coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)
	manager.RegisterExecutor(exec)
	auth := &coreauth.Auth{ID: "aistudio-auth-1", Provider: testAIStudioTranscriptionProvider, Status: coreauth.StatusActive}
	if _, errReg := manager.Register(context.Background(), auth); errReg != nil {
		t.Fatalf("manager.Register: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{
		{ID: testAIStudioTranscribeModel},
		{ID: "gpt-4o-transcribe"},
	})
	manager.RefreshSchedulerEntry(auth.ID)
	t.Cleanup(func() {
		modelReg.UnregisterClient(auth.ID)
	})

	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)

	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)
	router.POST("/v1/audio/translations", handler.AudioTranslations)

	cases := []struct {
		name      string
		path      string
		model     string
		wantPath  string
		wantModel string
	}{
		{
			name:      "standard transcriptions with gemini-3.5-transcribe",
			path:      "/v1/audio/transcriptions",
			model:     testAIStudioTranscribeModel,
			wantPath:  openAITranscriptionsPath,
			wantModel: testAIStudioTranscribeModel,
		},
		{
			name:      "zero-width space prefixed gemini-3.5-transcribe",
			path:      "/v1/audio/transcriptions",
			model:     string([]rune{0x200b}) + testAIStudioTranscribeModel + string([]rune{0x200b}),
			wantPath:  openAITranscriptionsPath,
			wantModel: testAIStudioTranscribeModel,
		},
		{
			name:      "models/ prefixed gemini-3.5-transcribe on /v1/audio/transcriptions",
			path:      "/v1/audio/transcriptions",
			model:     "models/" + testAIStudioTranscribeModel,
			wantPath:  openAITranscriptionsPath,
			wantModel: testAIStudioTranscribeModel,
		},
		{
			name:      "gpt-4o-transcribe model preserved",
			path:      "/v1/audio/transcriptions",
			model:     "gpt-4o-transcribe",
			wantPath:  openAITranscriptionsPath,
			wantModel: "gpt-4o-transcribe",
		},
		{
			name:      "audio translations with gemini-3.5-transcribe",
			path:      "/v1/audio/translations",
			model:     testAIStudioTranscribeModel,
			wantPath:  openAITranslationsPath,
			wantModel: testAIStudioTranscribeModel,
		},
	}

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, ct := buildTestAudioMultipart(t, tc.model, "audio.wav", sampleWAV)
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(payload))
			req.Header.Set("Content-Type", ct)
			req.Header.Set("Accept-Encoding", "gzip, zstd")
			req.Header.Set("X-Api-Key", "cpa-client-secret")
			req.Header.Set("X-Goog-Api-Key", "cpa-goog-secret")
			req.Header.Set("Proxy-Authorization", "Basic secret")
			req.Header.Set("Connection", "X-Hop-Secret")
			req.Header.Set("X-Hop-Secret", "hop-value")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "hello live transcription") {
				t.Fatalf("unexpected body: %s", rec.Body.String())
			}
			if got := rec.Header().Get("Set-Cookie"); got != "" {
				t.Fatalf("Set-Cookie leaked: %q", got)
			}
			if got := rec.Header().Get("X-Internal-Hop"); got != "" {
				t.Fatalf("Connection-scoped header leaked: %q", got)
			}
			if got := rec.Header().Get("X-Upstream-Custom"); got != "preserved-value" {
				t.Fatalf("X-Upstream-Custom = %q, want preserved-value", got)
			}
			exec.mu.Lock()
			gotPath := exec.paths[i]
			gotModel := exec.models[i]
			gotFile := exec.files[i]
			gotData := exec.fileData[i]
			gotReqHeader := exec.reqHeaders[i]
			exec.mu.Unlock()
			if gotPath != tc.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tc.wantPath)
			}
			if gotModel != tc.wantModel {
				t.Fatalf("model = %q, want %q", gotModel, tc.wantModel)
			}
			if gotFile != "audio.wav" {
				t.Fatalf("filename = %q, want audio.wav", gotFile)
			}
			if !bytes.Equal(gotData, sampleWAV) {
				t.Fatalf("audio bytes mismatch: %q != %q", gotData, sampleWAV)
			}
			for _, forbidden := range []string{"Content-Encoding", "Accept-Encoding", "X-Api-Key", "X-Goog-Api-Key", "Proxy-Authorization", "X-Hop-Secret"} {
				if got := gotReqHeader.Get(forbidden); got != "" {
					t.Fatalf("forwarded %s = %q, want empty", forbidden, got)
				}
			}
		})
	}
}

func TestAudioTranscriptionsSchedulerSelectsHealthyAuthAfterCooldown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := &transcriptionCaptureExecutor{
		failAuthIDs: map[string]int{
			"aistudio-auth-bad": http.StatusTooManyRequests,
		},
		respHeaders: http.Header{
			"Retry-After": []string{"120"},
		},
	}
	manager := coreauth.NewManager(nil, &coreauth.FillFirstSelector{}, nil)
	manager.RegisterExecutor(exec)
	badAuth := &coreauth.Auth{ID: "aistudio-auth-bad", Provider: testAIStudioTranscriptionProvider, Status: coreauth.StatusActive}
	goodAuth := &coreauth.Auth{ID: "aistudio-auth-good", Provider: testAIStudioTranscriptionProvider, Status: coreauth.StatusActive}
	if _, errReg := manager.Register(context.Background(), badAuth); errReg != nil {
		t.Fatalf("manager.Register badAuth: %v", errReg)
	}
	if _, errReg := manager.Register(context.Background(), goodAuth); errReg != nil {
		t.Fatalf("manager.Register goodAuth: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(badAuth.ID, badAuth.Provider, []*registry.ModelInfo{{ID: testAIStudioTranscribeModel}})
	modelReg.RegisterClient(goodAuth.ID, goodAuth.Provider, []*registry.ModelInfo{{ID: testAIStudioTranscribeModel}})
	manager.RefreshSchedulerEntry(badAuth.ID)
	manager.RefreshSchedulerEntry(goodAuth.ID)
	t.Cleanup(func() {
		modelReg.UnregisterClient(badAuth.ID)
		modelReg.UnregisterClient(goodAuth.ID)
	})

	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)

	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	payload, ct := buildTestAudioMultipart(t, testAIStudioTranscribeModel, "audio.wav", sampleWAV)

	// First request hits badAuth, returns 429 with Retry-After: 120, and marks it in model-scoped cooldown.
	req1 := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req1.Header.Set("Content-Type", ct)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusTooManyRequests {
		t.Fatalf("first request status = %d, want 429", rec1.Code)
	}

	updatedBad, ok := manager.GetByID(badAuth.ID)
	if !ok || updatedBad == nil {
		t.Fatalf("%s not found", badAuth.ID)
	}
	state := updatedBad.ModelStates[testAIStudioTranscribeModel]
	if state == nil || !state.Unavailable || state.NextRetryAfter.Before(time.Now().Add(90*time.Second)) {
		t.Fatalf("expected %s model state to be cooled for ~120s from Retry-After, got %+v", testAIStudioTranscribeModel, state)
	}

	// Second request selects through the scheduler, skipping the cooled-down badAuth and using goodAuth.
	req2 := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", ct)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second request status = %d, want 200: %s", rec2.Code, rec2.Body.String())
	}

	exec.mu.Lock()
	gotAuths := append([]string(nil), exec.authIDs...)
	exec.mu.Unlock()
	if len(gotAuths) != 2 || gotAuths[0] != "aistudio-auth-bad" || gotAuths[1] != "aistudio-auth-good" {
		t.Fatalf("auth sequence = %v, want [aistudio-auth-bad aistudio-auth-good]", gotAuths)
	}
}

func TestAudioTranscriptionsResolvesBaseURLForHTTPBackedProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := &transcriptionCaptureExecutor{provider: "openai-compatibility"}
	manager := coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)
	manager.RegisterExecutor(exec)
	compatAuth := &coreauth.Auth{
		ID:       "compat-whisper-auth",
		Provider: "openai-compatibility",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"base_url": "https://api.example.com/v1",
		},
	}
	if _, errReg := manager.Register(context.Background(), compatAuth); errReg != nil {
		t.Fatalf("manager.Register: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(compatAuth.ID, compatAuth.Provider, []*registry.ModelInfo{{ID: "whisper-1"}})
	manager.RefreshSchedulerEntry(compatAuth.ID)
	t.Cleanup(func() {
		modelReg.UnregisterClient(compatAuth.ID)
	})

	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)
	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	payload, ct := buildTestAudioMultipart(t, "whisper-1", "audio.wav", sampleWAV)
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	exec.mu.Lock()
	gotURLs := append([]string(nil), exec.urls...)
	gotModels := append([]string(nil), exec.models...)
	exec.mu.Unlock()
	if len(gotURLs) != 1 || gotURLs[0] != "https://api.example.com/v1/audio/transcriptions" {
		t.Fatalf("urls = %v, want [https://api.example.com/v1/audio/transcriptions]", gotURLs)
	}
	if len(gotModels) != 1 || gotModels[0] != "whisper-1" {
		t.Fatalf("models = %v, want [whisper-1]", gotModels)
	}
}

func TestAudioTranscriptionsAppliesMediaPayloadRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := &transcriptionCaptureExecutor{}
	manager := coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)
	manager.RegisterExecutor(exec)
	auth := &coreauth.Auth{
		ID:       "aistudio-payload-rule-auth",
		Provider: testAIStudioTranscriptionProvider,
		Status:   coreauth.StatusActive,
	}
	if _, errReg := manager.Register(context.Background(), auth); errReg != nil {
		t.Fatalf("manager.Register: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{ID: testAIStudioTranscribeModel}})
	manager.RefreshSchedulerEntry(auth.ID)
	t.Cleanup(func() {
		modelReg.UnregisterClient(auth.ID)
	})

	manager.SetConfig(&internalconfig.Config{
		Payload: internalconfig.PayloadConfig{
			Override: []internalconfig.PayloadRule{
				{
					Models: []internalconfig.PayloadModelRule{{Name: testAIStudioTranscribeModel}},
					Params: map[string]any{"prompt": "configured subtitle prompt"},
				},
			},
		},
	})
	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)
	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	payload, ct := buildTestAudioMultipart(t, testAIStudioTranscribeModel, "audio.wav", sampleWAV)
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	exec.mu.Lock()
	gotPrompts := append([]string(nil), exec.prompts...)
	exec.mu.Unlock()
	if len(gotPrompts) != 1 || gotPrompts[0] != "configured subtitle prompt" {
		t.Fatalf("prompts = %v, want [configured subtitle prompt]", gotPrompts)
	}
}

type homeAudioFallbackDispatcher struct{}

func (homeAudioFallbackDispatcher) HeartbeatOK() bool { return true }

func (homeAudioFallbackDispatcher) RPopAuth(context.Context, string, string, http.Header, int) ([]byte, error) {
	return json.Marshal(coreauth.Auth{
		ID:       "home-custom-compat-auth",
		Provider: "custom-whisper-provider",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"base_url": "https://home-compat.example.com/v1",
		},
	})
}

func (homeAudioFallbackDispatcher) AbortAmbiguousDispatch() {}

func TestAudioTranscriptionsHomeSelectionUsesRetainedExecutor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	compatExec := &transcriptionCaptureExecutor{provider: "openai-compatibility"}
	manager := coreauth.NewManager(nil, nil, nil)
	manager.SetConfig(&internalconfig.Config{Home: internalconfig.HomeConfig{Enabled: true}})
	manager.PublishHomeDispatch(homeAudioFallbackDispatcher{}, executionregistry.New(), 1)
	manager.RegisterExecutor(compatExec)

	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)
	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	payload, ct := buildTestAudioMultipart(t, "whisper-1", "audio.wav", sampleWAV)
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	compatExec.mu.Lock()
	gotAuths := append([]string(nil), compatExec.authIDs...)
	gotURLs := append([]string(nil), compatExec.urls...)
	compatExec.mu.Unlock()
	if len(gotAuths) != 1 || gotAuths[0] != "home-custom-compat-auth" {
		t.Fatalf("authIDs = %v, want [home-custom-compat-auth]", gotAuths)
	}
	if len(gotURLs) != 1 || gotURLs[0] != "https://home-compat.example.com/v1/audio/transcriptions" {
		t.Fatalf("urls = %v, want [https://home-compat.example.com/v1/audio/transcriptions]", gotURLs)
	}
}

func TestAudioTranscriptionsStreamsEventStreamResponsesIncrementally(t *testing.T) {
	gin.SetMode(gin.TestMode)
	firstChunkArrived := make(chan struct{})
	releaseSecondChunk := make(chan struct{})
	exec := &transcriptionCaptureExecutor{
		provider: "openai-compatibility",
		customResp: func(_ *coreauth.Auth, req *http.Request) (*http.Response, error) {
			pr, pw := io.Pipe()
			go func() {
				defer func() { _ = pw.Close() }()
				_, _ = pw.Write([]byte("data: {\"type\":\"transcript.text.delta\",\"delta\":\"hello \"}\n\n"))
				close(firstChunkArrived)
				<-releaseSecondChunk
				_, _ = pw.Write([]byte("data: {\"type\":\"transcript.text.done\",\"text\":\"hello world\"}\n\n"))
			}()
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       pr,
				Request:    req,
			}, nil
		},
	}
	manager := coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)
	manager.RegisterExecutor(exec)
	auth := &coreauth.Auth{
		ID:       "stream-transcribe-auth",
		Provider: "openai-compatibility",
		Status:   coreauth.StatusActive,
	}
	if _, errReg := manager.Register(context.Background(), auth); errReg != nil {
		t.Fatalf("manager.Register: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{ID: "gpt-4o-transcribe"}})
	manager.RefreshSchedulerEntry(auth.ID)
	t.Cleanup(func() {
		modelReg.UnregisterClient(auth.ID)
	})

	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)
	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)
	srv := httptest.NewServer(router)
	defer srv.Close()

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	payload, ct := buildTestAudioMultipart(t, "gpt-4o-transcribe", "audio.wav", sampleWAV, map[string]string{"stream": "true"})
	req, errReq := http.NewRequest(http.MethodPost, srv.URL+"/v1/audio/transcriptions", bytes.NewReader(payload))
	if errReq != nil {
		t.Fatalf("http.NewRequest: %v", errReq)
	}
	req.Header.Set("Content-Type", ct)

	resp, errDo := srv.Client().Do(req)
	if errDo != nil {
		t.Fatalf("client.Do: %v", errDo)
	}
	defer func() { _ = resp.Body.Close() }()

	<-firstChunkArrived
	buf := make([]byte, 256)
	n, errReadFirst := resp.Body.Read(buf)
	if errReadFirst != nil {
		t.Fatalf("first stream read failed before EOF: %v", errReadFirst)
	}
	if !strings.Contains(string(buf[:n]), "transcript.text.delta") {
		t.Fatalf("expected first delta chunk before EOF, got %q", string(buf[:n]))
	}

	close(releaseSecondChunk)
	rest, errReadRest := io.ReadAll(resp.Body)
	if errReadRest != nil {
		t.Fatalf("read remaining stream: %v", errReadRest)
	}
	if !strings.Contains(string(rest), "transcript.text.done") {
		t.Fatalf("expected done chunk in stream remainder, got %q", string(rest))
	}
}

func TestAudioTranscriptionsRewritesUpstreamAliasModelBeforeFinalization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := &transcriptionCaptureExecutor{provider: "openai-compatibility"}
	manager := coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)
	manager.RegisterExecutor(exec)
	manager.SetConfig(&internalconfig.Config{
		OpenAICompatibility: []internalconfig.OpenAICompatibility{{
			Name: "groq-compat",
			Models: []internalconfig.OpenAICompatibilityModel{
				{Name: "whisper-large-v3", Alias: "whisper-1"},
			},
		}},
		Payload: internalconfig.PayloadConfig{
			Override: []internalconfig.PayloadRule{
				{
					Models: []internalconfig.PayloadModelRule{{Name: "whisper-large-v3"}},
					Params: map[string]any{"prompt": "upstream alias rule matched"},
				},
			},
		},
	})
	auth := &coreauth.Auth{
		ID:       "compat-alias-auth",
		Provider: "openai-compatibility",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"api_key":      "test-key",
			"base_url":     "https://api.example.com/v1",
			"compat_name":  "groq-compat",
			"provider_key": "openai-compatibility",
		},
	}
	if _, errReg := manager.Register(context.Background(), auth); errReg != nil {
		t.Fatalf("manager.Register: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{ID: "whisper-1"}})
	manager.RefreshSchedulerEntry(auth.ID)
	t.Cleanup(func() {
		modelReg.UnregisterClient(auth.ID)
	})

	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)
	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	payload, ct := buildTestAudioMultipart(t, "whisper-1", "audio.wav", sampleWAV)
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	exec.mu.Lock()
	gotModels := append([]string(nil), exec.models...)
	gotPrompts := append([]string(nil), exec.prompts...)
	exec.mu.Unlock()
	if len(gotModels) != 1 || gotModels[0] != "whisper-large-v3" {
		t.Fatalf("upstream models = %v, want [whisper-large-v3]", gotModels)
	}
	if len(gotPrompts) != 1 || gotPrompts[0] != "upstream alias rule matched" {
		t.Fatalf("prompts = %v, want [upstream alias rule matched]", gotPrompts)
	}
}

func TestAudioTranscriptionsCanceledRequestPreservesLifecycleAndSkipsCooldown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exec := &transcriptionCaptureExecutor{
		provider: "openai-compatibility",
		customResp: func(_ *coreauth.Auth, _ *http.Request) (*http.Response, error) {
			return nil, context.Canceled
		},
	}
	manager := coreauth.NewManager(nil, &coreauth.FillFirstSelector{}, nil)
	manager.RegisterExecutor(exec)
	auth := &coreauth.Auth{
		ID:       "sole-audio-auth",
		Provider: "openai-compatibility",
		Status:   coreauth.StatusActive,
	}
	if _, errReg := manager.Register(context.Background(), auth); errReg != nil {
		t.Fatalf("manager.Register: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{ID: "whisper-1"}})
	manager.RefreshSchedulerEntry(auth.ID)
	t.Cleanup(func() {
		modelReg.UnregisterClient(auth.ID)
	})

	base := apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager)
	handler := NewOpenAIAPIHandler(base)
	router := gin.New()
	router.POST("/v1/audio/transcriptions", handler.AudioTranscriptions)

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	payload, ct := buildTestAudioMultipart(t, "whisper-1", "audio.wav", sampleWAV)
	req := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 499 {
		t.Fatalf("status = %d, want 499: %s", rec.Code, rec.Body.String())
	}

	updated, ok := manager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatalf("auth %s not found", auth.ID)
	}
	if updated.Unavailable {
		t.Fatalf("expected sole credential to remain available after client cancellation, got unavailable=%v", updated.Unavailable)
	}
	if state := updated.ModelStates["whisper-1"]; state != nil && (state.Unavailable || !state.NextRetryAfter.IsZero()) {
		t.Fatalf("expected whisper-1 model state not to cool after client cancellation, got %+v", state)
	}
}

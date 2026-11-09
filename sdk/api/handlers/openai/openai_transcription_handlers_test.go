package openai

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/registry"
	apihandlers "github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
)

type transcriptionCaptureExecutor struct {
	provider    string
	mu          sync.Mutex
	paths       []string
	models      []string
	files       []string
	fileData    [][]byte
	authIDs     []string
	reqHeaders  []http.Header
	failAuthIDs map[string]int
	respHeaders http.Header
}

func (e *transcriptionCaptureExecutor) Identifier() string {
	if e.provider != "" {
		return e.provider
	}
	return aistudioTranscriptionProvider
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
	e.paths = append(e.paths, req.URL.Path)
	e.models = append(e.models, model)
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
	e.mu.Unlock()

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

func buildTestAudioMultipart(t *testing.T, modelName string, fileName string, audioBytes []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if modelName != "" {
		if errField := mw.WriteField("model", modelName); errField != nil {
			t.Fatalf("WriteField model: %v", errField)
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
	auth := &coreauth.Auth{ID: "aistudio-auth-1", Provider: aistudioTranscriptionProvider, Status: coreauth.StatusActive}
	if _, errReg := manager.Register(context.Background(), auth); errReg != nil {
		t.Fatalf("manager.Register: %v", errReg)
	}
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
			name:      "standard transcriptions with gemini-3.5-transcribe-live",
			path:      "/v1/audio/transcriptions",
			model:     "gemini-3.5-transcribe-live",
			wantPath:  openAITranscriptionsPath,
			wantModel: defaultAIStudioTranscribeModel,
		},
		{
			name:      "zero-width space prefixed gemini-3.5-transcribe-live",
			path:      "/v1/audio/transcriptions",
			model:     string([]rune{0x200b}) + "gemini-3.5-transcribe-live" + string([]rune{0x200b}),
			wantPath:  openAITranscriptionsPath,
			wantModel: defaultAIStudioTranscribeModel,
		},
		{
			name:      "prefixed AIStudio/gemini-3.5-transcribe-live on /v1/audio/transcriptions",
			path:      "/v1/audio/transcriptions",
			model:     "AIStudio/gemini-3.5-transcribe-live",
			wantPath:  openAITranscriptionsPath,
			wantModel: defaultAIStudioTranscribeModel,
		},
		{
			name:      "non-alias transcription model preserved",
			path:      "/v1/audio/transcriptions",
			model:     "gpt-4o-transcribe",
			wantPath:  openAITranscriptionsPath,
			wantModel: "gpt-4o-transcribe",
		},
		{
			name:      "audio translations with gemini-3.5-transcribe-live",
			path:      "/v1/audio/translations",
			model:     "gemini-3.5-transcribe-live",
			wantPath:  openAITranslationsPath,
			wantModel: defaultAIStudioTranscribeModel,
		},
	}

	sampleWAV := []byte("RIFF\x24\x00\x00\x00WAVEfmt ")
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, ct := buildTestAudioMultipart(t, tc.model, "audio.wav", sampleWAV)
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(payload))
			req.Header.Set("Content-Type", ct)
			req.Header.Set("Accept-Encoding", "gzip, zstd")
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
			if got := gotReqHeader.Get("Content-Encoding"); got != "" {
				t.Fatalf("forwarded Content-Encoding = %q, want empty", got)
			}
			if got := gotReqHeader.Get("Accept-Encoding"); got != "" {
				t.Fatalf("forwarded Accept-Encoding = %q, want empty", got)
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
	}
	manager := coreauth.NewManager(nil, &coreauth.FillFirstSelector{}, nil)
	manager.RegisterExecutor(exec)
	badAuth := &coreauth.Auth{ID: "aistudio-auth-bad", Provider: aistudioTranscriptionProvider, Status: coreauth.StatusActive}
	goodAuth := &coreauth.Auth{ID: "aistudio-auth-good", Provider: aistudioTranscriptionProvider, Status: coreauth.StatusActive}
	if _, errReg := manager.Register(context.Background(), badAuth); errReg != nil {
		t.Fatalf("manager.Register badAuth: %v", errReg)
	}
	if _, errReg := manager.Register(context.Background(), goodAuth); errReg != nil {
		t.Fatalf("manager.Register goodAuth: %v", errReg)
	}
	modelReg := registry.GetGlobalRegistry()
	modelReg.RegisterClient(badAuth.ID, badAuth.Provider, []*registry.ModelInfo{{ID: defaultAIStudioTranscribeModel}})
	modelReg.RegisterClient(goodAuth.ID, goodAuth.Provider, []*registry.ModelInfo{{ID: defaultAIStudioTranscribeModel}})
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
	payload, ct := buildTestAudioMultipart(t, "gemini-3.5-transcribe-live", "audio.wav", sampleWAV)

	// First request hits badAuth, returns 429, and marks it in model-scoped cooldown.
	req1 := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(payload))
	req1.Header.Set("Content-Type", ct)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusTooManyRequests {
		t.Fatalf("first request status = %d, want 429", rec1.Code)
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

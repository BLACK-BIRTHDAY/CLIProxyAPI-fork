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
	apihandlers "github.com/router-for-me/CLIProxyAPI/v8/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
)

type transcriptionCaptureExecutor struct {
	mu       sync.Mutex
	paths    []string
	models   []string
	files    []string
	fileData [][]byte
}

func (e *transcriptionCaptureExecutor) Identifier() string { return aistudioTranscriptionProvider }

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

func (e *transcriptionCaptureExecutor) HttpRequest(_ context.Context, _ *coreauth.Auth, req *http.Request) (*http.Response, error) {
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
	e.mu.Lock()
	e.paths = append(e.paths, req.URL.Path)
	e.models = append(e.models, model)
	e.files = append(e.files, hdr.Filename)
	e.fileData = append(e.fileData, data)
	e.mu.Unlock()

	body := `{"text":"hello live transcription"}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
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
	exec := &transcriptionCaptureExecutor{}
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
			name:      "prefixed AIStudio/gemini-3.5-transcribe-live on /v1/audio/transcriptions",
			path:      "/v1/audio/transcriptions",
			model:     "AIStudio/gemini-3.5-transcribe-live",
			wantPath:  openAITranscriptionsPath,
			wantModel: defaultAIStudioTranscribeModel,
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
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "hello live transcription") {
				t.Fatalf("unexpected body: %s", rec.Body.String())
			}
			exec.mu.Lock()
			gotPath := exec.paths[i]
			gotModel := exec.models[i]
			gotFile := exec.files[i]
			gotData := exec.fileData[i]
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
		})
	}
}

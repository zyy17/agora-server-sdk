package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRecognizeProtocolAndTranscripts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverDone := make(chan struct{})
	pcm := bytes.Repeat([]byte{1, 2}, 1600)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(serverDone)
		wantQuery := map[string]string{
			"model_id": "scribe_v2_realtime", "audio_format": "pcm_16000", "commit_strategy": "vad",
			"vad_silence_threshold_secs": "0.7", "keepalive_interval_ms": "1000", "language_code": "en",
		}
		for key, want := range wantQuery {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("query %s = %q, want %q", key, got, want)
			}
		}
		if got := r.Header.Get("Xi-Api-Key"); got != "test-eleven-key" {
			t.Errorf("ASR authorization = %q", got)
		}
		conn := upgradeTestSocket(t, w, r)
		if conn == nil {
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			t.Error(err)
			return
		}
		if messageType != websocket.TextMessage {
			t.Errorf("audio frame type = %d, want text JSON", messageType)
		}
		var input map[string]any
		if err := json.Unmarshal(message, &input); err != nil {
			t.Error(err)
			return
		}
		wantInput := map[string]any{"message_type": "input_audio_chunk", "audio_base_64": base64.StdEncoding.EncodeToString(pcm), "sample_rate": float64(16000), "commit": false}
		if !reflect.DeepEqual(input, wantInput) {
			t.Error("ASR input does not match the PCM JSON contract")
		}
		for _, event := range []string{
			`{"message_type":"session_started"}`,
			`{"message_type":"session_started"}`,
			`{"message_type":"partial_transcript","text":""}`,
			`{"message_type":"partial_transcript","text":"hello"}`,
			`{"message_type":"partial_transcript","text":"hello"}`,
			`{"message_type":"committed_transcript","text":"hello"}`,
		} {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(event)); err != nil {
				t.Error(err)
				return
			}
		}
		// The adapter must close the socket when the callback stops the reader.
		if _, _, err := conn.ReadMessage(); err == nil {
			t.Error("connection remained open")
		}
	}))
	defer server.Close()
	p := providers{asrURL: websocketURL(server), elevenKey: "test-eleven-key", language: "en"}
	audio := make(chan []byte, 1)
	audio <- pcm
	stop := errors.New("finished collecting transcripts")
	var got []transcript
	err := p.recognize(ctx, audio, func(value transcript) error {
		got = append(got, value)
		if value.final {
			return stop
		}
		return nil
	})
	if !errors.Is(err, stop) {
		t.Fatalf("recognize = %v, want callback error", err)
	}
	want := []transcript{{ready: true}, {}, {text: "hello"}, {text: "hello"}, {text: "hello", final: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("transcripts = %#v, want %#v", got, want)
	}
	waitProviderSignal(t, ctx, serverDone)
}

func TestRecognizeCancellationClosesSocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connected, serverDone := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(serverDone)
		conn := upgradeTestSocket(t, w, r)
		if conn == nil {
			return
		}
		defer conn.Close()
		close(connected)
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, _, err := conn.ReadMessage(); err == nil {
			t.Error("expected cancellation to close the socket")
		}
	}))
	defer server.Close()
	p := providers{asrURL: websocketURL(server)}
	result := make(chan error, 1)
	go func() { result <- p.recognize(ctx, nil, func(transcript) error { return nil }) }()
	waitProviderSignal(t, ctx, connected)
	cancel()
	if err := waitProviderResult(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("recognize = %v, want canceled", err)
	}
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Fatal("server socket was not closed")
	}
}

func TestRecognizeErrors(t *testing.T) {
	for _, event := range []string{
		`{"message_type":"auth_error","error":"private provider detail"}`,
		`{"message_type":"rate_limited","error":"private provider detail"}`,
		`{"message_type":"unaccepted_terms","error":"private provider detail"}`,
		`{"message_type":"future_error","error":"private provider detail"}`,
		`not json private provider detail`,
	} {
		t.Run(event, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn := upgradeTestSocket(t, w, r)
				if conn == nil {
					return
				}
				defer conn.Close()
				_ = conn.WriteMessage(websocket.TextMessage, []byte(event))
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, _, _ = conn.ReadMessage()
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			p := providers{asrURL: websocketURL(server)}
			err := p.recognize(ctx, nil, func(transcript) error { return nil })
			if err == nil || strings.Contains(err.Error(), "private") {
				t.Fatalf("expected sanitized ASR error, got %v", err)
			}
		})
	}
}

func TestRecognizeAudioClosureAndInvalidSamples(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(fmt.Sprint(invalid), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn := upgradeTestSocket(t, w, r)
				if conn == nil {
					return
				}
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, _, _ = conn.ReadMessage()
			}))
			defer server.Close()
			audio := make(chan []byte, 1)
			if invalid {
				audio <- []byte{1}
			} else {
				close(audio)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := (providers{asrURL: websocketURL(server)}).recognize(ctx, audio, func(transcript) error { return nil })
			if (err != nil) != invalid {
				t.Fatalf("recognize invalid=%v: %v", invalid, err)
			}
		})
	}
}

func TestCompleteStreamsBeforeResponseEnds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	firstDelta := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-openai-key" {
			t.Error("invalid completion request method or authorization")
		}
		var request struct {
			Model               string              `json:"model"`
			Messages            []map[string]string `json:"messages"`
			Stream              bool                `json:"stream"`
			MaxCompletionTokens int                 `json:"max_completion_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Model != "gpt-4.1-mini" || !request.Stream || request.MaxCompletionTokens != 256 || !reflect.DeepEqual(request.Messages, []map[string]string{{"role": "system", "content": "Be brief."}, {"role": "user", "content": "Hello"}}) {
			t.Errorf("invalid completion request: %+v", request)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, ": ping\r\n\r\ndata: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":null}}]}\r\n\r\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi!\"}}]}\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-firstDelta:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: {\"choices\":[],\"usage\":{\"total_tokens\":5}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	p := providers{llmURL: server.URL, openAIKey: "test-openai-key", systemPrompt: "Be brief."}
	var text string
	err := p.complete(ctx, "Hello", func(delta string) error {
		text += delta
		close(firstDelta)
		return nil
	})
	if err != nil || text != "Hi!" {
		t.Fatalf("complete = %q, %v", text, err)
	}
}

func TestCompletionFragmentedSSE(t *testing.T) {
	stream := ": heartbeat\r\nid: test\r\nevent: message\r\ndata: {\"choices\": [\r\ndata: {\"index\":0,\"delta\":{\"content\":\"你好\"}}]}\r\n\r\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]"
	var got strings.Builder
	err := readCompletion(context.Background(), &fragmentReader{data: []byte(stream), max: 1}, func(delta string) error {
		got.WriteString(delta)
		return nil
	})
	if err != nil || got.String() != "你好" {
		t.Fatalf("fragmented completion = %q, %v", got.String(), err)
	}
}

func TestCompletionRejectsIncompleteAndInvalidStreams(t *testing.T) {
	for name, stream := range map[string]string{
		"empty":          "",
		"missing done":   "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n",
		"missing stop":   "data: [DONE]\n\n",
		"partial":        "data: {\"choices\":[{\"delta\":{\"content\":\"unfinished\"}}]}\n\n",
		"provider error": "data: {\"error\":{\"message\":\"private provider detail\"}}\n\n",
		"error event":    "event: error\ndata: private provider detail\n\n",
		"invalid json":   "data: private provider detail\n\n",
		"length":         "data: {\"choices\":[{\"finish_reason\":\"length\"}]}\n\n",
		"filter":         "data: {\"choices\":[{\"finish_reason\":\"content_filter\"}]}\n\n",
		"tool finish":    "data: {\"choices\":[{\"finish_reason\":\"tool_calls\"}]}\n\n",
		"tool delta":     "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{}]}}]}\n\n",
		"function delta": "data: {\"choices\":[{\"delta\":{\"function_call\":{}}}]}\n\n",
		"refusal delta":  "data: {\"choices\":[{\"delta\":{\"refusal\":\"private refusal detail\"}}]}\n\ndata: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n",
		"empty reply":    "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":null,\"refusal\":null},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n",
		"blank reply":    "data: {\"choices\":[{\"delta\":{\"content\":\" \\n\\t\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n",
		"after stop":     "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"extra\"}}]}\n\n",
		"large event":    "data: " + strings.Repeat("x", 1<<20) + "\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			err := readCompletion(context.Background(), strings.NewReader(stream), func(string) error { return nil })
			if err == nil || strings.Contains(err.Error(), "private") {
				t.Fatalf("expected sanitized stream error, got %v", err)
			}
			if name == "refusal delta" && !strings.Contains(err.Error(), "refused") {
				t.Fatalf("refusal should fail explicitly, got %v", err)
			}
			if (name == "empty reply" || name == "blank reply") && !strings.Contains(err.Error(), "no text") {
				t.Fatalf("empty completion should fail explicitly, got %v", err)
			}
		})
	}
}

func TestSpeakStreamsBeforeResponseEnds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	firstAudio := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Xi-Api-Key") != "test-eleven-key" || r.URL.Query().Get("output_format") != "pcm_16000" {
			t.Error("invalid speech request method, authorization or format")
		}
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(request, map[string]string{"text": "Hello", "model_id": "eleven_flash_v2_5", "language_code": "en"}) {
			t.Errorf("speech request = %#v", request)
		}
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write([]byte{1, 2, 3})
		w.(http.Flusher).Flush()
		select {
		case <-firstAudio:
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write([]byte{4, 5, 6})
	}))
	defer server.Close()
	p := providers{ttsURL: server.URL, elevenKey: "test-eleven-key", language: "en"}
	var chunks [][]byte
	err := p.speak(ctx, "Hello", func(pcm []byte) error {
		if len(chunks) == 0 {
			close(firstAudio)
		}
		if len(pcm)%2 != 0 {
			t.Error("speech emitted an incomplete PCM16 sample")
		}
		chunks = append(chunks, pcm)
		return nil
	})
	if err != nil || !bytes.Equal(bytes.Join(chunks, nil), []byte{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("speech = %v, %v", chunks, err)
	}
}

func TestSpeakPreservesOddReadBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, 2, 5, 6, 10000} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			pcm := make([]byte, size)
			for i := range pcm {
				pcm[i] = byte(i)
			}
			p := providers{ttsURL: "http://offline.invalid", httpClient: &http.Client{Transport: providerTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(&fragmentReader{data: pcm, max: 3}), Header: make(http.Header)}, nil
			})}}
			var chunks [][]byte
			err := p.speak(context.Background(), "Hello", func(chunk []byte) error {
				chunks = append(chunks, chunk)
				if len(chunk)%2 != 0 {
					t.Error("odd audio callback")
				}
				return nil
			})
			wantErr := size == 0 || size%2 != 0
			if (err != nil) != wantErr {
				t.Fatalf("speak = %v, want error %v", err, wantErr)
			}
			if !bytes.Equal(bytes.Join(chunks, nil), pcm[:size&^1]) {
				t.Fatal("PCM samples changed or callback slices were reused")
			}
		})
	}
}

func TestSpeakRejectsTruncatedHTTPBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = w.Write([]byte{1, 2, 3})
	}))
	defer server.Close()
	var got []byte
	err := (providers{ttsURL: server.URL}).speak(context.Background(), "Hello", func(pcm []byte) error {
		got = append(got, pcm...)
		return nil
	})
	if err == nil || !bytes.Equal(got, []byte{1, 2}) {
		t.Fatalf("truncated HTTP audio = %v, %v", got, err)
	}
}

func TestProviderDefaultHTTPEndpoints(t *testing.T) {
	stop := errors.New("checked endpoint")
	for _, kind := range []string{"LLM", "TTS"} {
		t.Run(kind, func(t *testing.T) {
			called := false
			p := providers{voiceID: "voice-id", model: "custom-model", httpClient: &http.Client{Transport: providerTransport(func(r *http.Request) (*http.Response, error) {
				called = true
				want := "https://api.openai.com/v1/chat/completions"
				if kind == "TTS" {
					want = "https://api.elevenlabs.io/v1/text-to-speech/voice-id/stream?output_format=pcm_16000"
				}
				if r.URL.String() != want {
					t.Errorf("URL = %q, want %q", r.URL, want)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if kind == "LLM" && body["model"] != "custom-model" {
					t.Errorf("model override = %v", body["model"])
				}
				if _, ok := body["language_code"]; ok {
					t.Error("empty language must be omitted")
				}
				return nil, stop
			})}}
			var err error
			if kind == "LLM" {
				err = p.complete(context.Background(), "Hello", func(string) error { return nil })
			} else {
				err = p.speak(context.Background(), "Hello", func([]byte) error { return nil })
			}
			if !called || err == nil {
				t.Fatalf("request invoked=%v, error=%v", called, err)
			}
		})
	}
}

func TestHTTPProviderCancellationAndCallbackErrors(t *testing.T) {
	for _, kind := range []string{"LLM", "TTS"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			p := providers{llmURL: server.URL, ttsURL: server.URL}
			result := make(chan error, 1)
			go func() {
				if kind == "LLM" {
					result <- p.complete(ctx, "Hello", func(string) error { return nil })
				} else {
					result <- p.speak(ctx, "Hello", func([]byte) error { return nil })
				}
			}()
			waitProviderSignal(t, ctx, started)
			cancel()
			if err := waitProviderResult(t, result); !errors.Is(err, context.Canceled) {
				t.Fatalf("%s = %v, want canceled", kind, err)
			}
		})
	}
	stop := errors.New("callback stopped")
	err := readCompletion(context.Background(), strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n"), func(string) error { return stop })
	if !errors.Is(err, stop) {
		t.Fatalf("completion callback error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte{1, 2}) }))
	defer server.Close()
	err = (providers{ttsURL: server.URL}).speak(context.Background(), "Hello", func([]byte) error { return stop })
	if !errors.Is(err, stop) {
		t.Fatalf("speech callback error = %v", err)
	}
}

func TestProviderHTTPFailuresAreSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "private input test-api-key")
	}))
	defer server.Close()
	p := providers{asrURL: websocketURL(server), llmURL: server.URL, ttsURL: server.URL, elevenKey: "test-api-key", openAIKey: "test-api-key"}
	for _, call := range []func() error{
		func() error { return p.recognize(context.Background(), nil, func(transcript) error { return nil }) },
		func() error {
			return p.complete(context.Background(), "private input", func(string) error { return nil })
		},
		func() error { return p.speak(context.Background(), "private input", func([]byte) error { return nil }) },
	} {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "test-api-key") {
			t.Fatalf("HTTP failure = %v, want sanitized status", err)
		}
	}
}

func TestProvidersDoNotFollowRedirects(t *testing.T) {
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("provider followed a redirect with private request data")
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	p := providers{ttsURL: source.URL, llmURL: source.URL}
	for _, call := range []func() error{
		func() error {
			return p.complete(context.Background(), "private input", func(string) error { return nil })
		},
		func() error { return p.speak(context.Background(), "private input", func([]byte) error { return nil }) },
	} {
		if err := call(); err == nil || !strings.Contains(err.Error(), "307") {
			t.Fatalf("redirect = %v", err)
		}
	}
}

type fragmentReader struct {
	data []byte
	max  int
}

func (r *fragmentReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := min(len(p), len(r.data), r.max)
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

type providerTransport func(*http.Request) (*http.Response, error)

func (f providerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func upgradeTestSocket(t *testing.T, w http.ResponseWriter, r *http.Request) *websocket.Conn {
	t.Helper()
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.Error(err)
	}
	return conn
}

func websocketURL(server *httptest.Server) string {
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

func waitProviderSignal(t *testing.T, ctx context.Context, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-ctx.Done():
		t.Fatal("timed out waiting for provider test signal")
	}
}

func waitProviderResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("provider did not join its workers")
		return nil
	}
}

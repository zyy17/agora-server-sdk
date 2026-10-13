package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type transcript struct {
	text  string
	final bool
	ready bool
}

// These are deliberately small adapters for this example, not a provider SDK.
// URL overrides are full endpoints, useful for the offline tests below.
type providers struct {
	httpClient                    *http.Client
	asrURL, llmURL, ttsURL        string
	elevenKey, openAIKey, voiceID string
	language, model, systemPrompt string
}

// recognize expects paced PCM16LE, mono, 16 kHz, including silence. Closing
// audio ends the session; the caller must keep it open to receive VAD commits.
// emit runs on the reader and must not block indefinitely.
func (p providers) recognize(ctx context.Context, audio <-chan []byte, emit func(transcript) error) error {
	endpoint := p.asrURL
	if endpoint == "" {
		endpoint = "wss://api.elevenlabs.io/v1/speech-to-text/realtime"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return errors.New("ASR: invalid endpoint")
	}
	query := u.Query()
	query.Set("model_id", "scribe_v2_realtime")
	query.Set("audio_format", "pcm_16000")
	query.Set("commit_strategy", "vad")
	query.Set("vad_silence_threshold_secs", "0.7")
	// Keepalive partials require audio to keep flowing, including silence.
	query.Set("keepalive_interval_ms", "1000")
	if p.language != "" {
		query.Set("language_code", p.language)
	}
	u.RawQuery = query.Encode()
	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment, HandshakeTimeout: 10 * time.Second}
	conn, response, err := dialer.DialContext(ctx, u.String(), http.Header{"Xi-Api-Key": []string{p.elevenKey}})
	if err != nil {
		if response != nil {
			defer response.Body.Close()
			return fmt.Errorf("ASR: websocket handshake HTTP %d", response.StatusCode)
		}
		return providerIOError(ctx, "ASR: websocket connection failed")
	}
	conn.SetReadLimit(1 << 20)
	workerCtx, cancel := context.WithCancel(ctx)
	readDone := make(chan error, 1)
	writeDone := make(chan error, 1)
	go func() { readDone <- readTranscripts(workerCtx, conn, emit) }()
	go func() { writeDone <- writeAudio(workerCtx, conn, audio) }()
	// Exactly one reader and one writer own their respective deadlines. Close
	// is safe concurrently and unblocks network I/O; always join both workers.
	var result error
	select {
	case result = <-readDone:
		readDone = nil
	case result = <-writeDone:
		writeDone = nil
	case <-ctx.Done():
		result = ctx.Err()
	}
	cancel()
	_ = conn.Close()
	if readDone != nil {
		<-readDone
	}
	if writeDone != nil {
		<-writeDone
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return result
}

func writeAudio(ctx context.Context, conn *websocket.Conn, audio <-chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pcm, ok := <-audio:
			if !ok {
				return nil
			}
			if len(pcm) == 0 || len(pcm)%2 != 0 {
				return errors.New("ASR: audio chunk must contain complete PCM16 samples")
			}
			message := struct {
				MessageType string `json:"message_type"`
				Audio       string `json:"audio_base_64"`
				SampleRate  int    `json:"sample_rate"`
				Commit      bool   `json:"commit"`
			}{"input_audio_chunk", base64.StdEncoding.EncodeToString(pcm), 16000, false}
			if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return providerIOError(ctx, "ASR: setting write deadline failed")
			}
			if err := conn.WriteJSON(message); err != nil {
				return providerIOError(ctx, "ASR: websocket write failed")
			}
		}
	}
}

func readTranscripts(ctx context.Context, conn *websocket.Conn, emit func(transcript) error) error {
	ready := false
	for {
		if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
			return providerIOError(ctx, "ASR: setting read deadline failed")
		}
		var event struct {
			Type  string          `json:"message_type"`
			Text  string          `json:"text"`
			Error json.RawMessage `json:"error"`
		}
		if err := conn.ReadJSON(&event); err != nil {
			return providerIOError(ctx, "ASR: websocket read failed or invalid event")
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		switch event.Type {
		case "session_started":
			if !ready {
				ready = true
				if err := emit(transcript{ready: true}); err != nil {
					return err
				}
			}
		case "partial_transcript", "committed_transcript":
			if err := emit(transcript{text: event.Text, final: event.Type == "committed_transcript"}); err != nil {
				return err
			}
		case "error", "auth_error", "quota_exceeded", "commit_throttled", "transcriber_error",
			"unaccepted_terms", "unaccepted_terms_error", "rate_limited", "input_error", "invalid_request",
			"queue_overflow", "resource_exhausted", "session_time_limit_exceeded", "chunk_size_exceeded",
			"insufficient_audio_activity":
			// Do not print provider error text: it can contain private input.
			return fmt.Errorf("ASR: provider event %s", event.Type)
		default:
			if len(event.Error) > 0 && string(event.Error) != "null" {
				return errors.New("ASR: provider error event")
			}
			// Future informational events need no action.
		}
	}
}

func (p providers) complete(ctx context.Context, text string, emit func(string) error) error {
	endpoint := p.llmURL
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	model := p.model
	if model == "" {
		model = "gpt-4.1-mini"
	}
	request := struct {
		Model               string              `json:"model"`
		Messages            []map[string]string `json:"messages"`
		Stream              bool                `json:"stream"`
		MaxCompletionTokens int                 `json:"max_completion_tokens"`
	}{model, []map[string]string{{"role": "system", "content": p.systemPrompt}, {"role": "user", "content": text}}, true, 256}
	response, err := p.post(ctx, "LLM", endpoint, "Authorization", "Bearer "+p.openAIKey, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return readCompletion(ctx, response.Body, emit)
}

func readCompletion(ctx context.Context, body io.Reader, emit func(string) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var data strings.Builder
	var eventType string
	var stopped, done, receivedText bool
	dispatch := func() error {
		payload := strings.TrimSpace(data.String())
		data.Reset()
		kind := eventType
		eventType = ""
		if kind == "error" {
			return errors.New("LLM: provider error event")
		}
		if payload == "" {
			return nil
		}
		if payload == "[DONE]" {
			if !stopped {
				return errors.New("LLM: stream ended without a normal stop")
			}
			if !receivedText {
				return errors.New("LLM: completion contained no text")
			}
			done = true
			return nil
		}
		var chunk struct {
			Error   json.RawMessage `json:"error"`
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Content      string          `json:"content"`
					Refusal      string          `json:"refusal"`
					ToolCalls    json.RawMessage `json:"tool_calls"`
					FunctionCall json.RawMessage `json:"function_call"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return errors.New("LLM: invalid streaming JSON")
		}
		if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
			return errors.New("LLM: provider error event")
		}
		// Empty choices (including usage-only events) and role-only deltas are valid.
		for _, choice := range chunk.Choices {
			if choice.Index != 0 || stopped {
				return errors.New("LLM: unexpected completion choice")
			}
			if hasJSONValue(choice.Delta.ToolCalls) || hasJSONValue(choice.Delta.FunctionCall) {
				return errors.New("LLM: tool calls are not supported")
			}
			if choice.Delta.Refusal != "" {
				return errors.New("LLM: provider refused the completion")
			}
			if choice.FinishReason != nil && *choice.FinishReason != "stop" {
				return errors.New("LLM: completion did not finish normally (length, filter, or unsupported output)")
			}
			if choice.Delta.Content != "" {
				receivedText = receivedText || strings.TrimSpace(choice.Delta.Content) != ""
				if err := emit(choice.Delta.Content); err != nil {
					return err
				}
			}
			if choice.FinishReason != nil {
				stopped = true
			}
		}
		return nil
	}
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		if line == "" {
			if err := dispatch(); err != nil {
				return err
			}
			if done {
				return nil
			}
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "data":
			if data.Len()+len(value)+1 > 1<<20 {
				return errors.New("LLM: streaming event exceeds size limit")
			}
			data.WriteString(value)
			data.WriteByte('\n')
		case "event":
			eventType = value
		}
	}
	if scanner.Err() != nil {
		return providerIOError(ctx, "LLM: stream read failed")
	}
	if err := dispatch(); err != nil {
		return err
	}
	if !done {
		return providerIOError(ctx, "LLM: truncated stream (missing [DONE])")
	}
	return nil
}

func hasJSONValue(value json.RawMessage) bool {
	return len(value) > 0 && string(value) != "null" && string(value) != "[]"
}

func (p providers) speak(ctx context.Context, text string, emit func([]byte) error) error {
	endpoint := p.ttsURL
	if endpoint == "" {
		endpoint = "https://api.elevenlabs.io/v1/text-to-speech/" + url.PathEscape(p.voiceID) + "/stream"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return errors.New("TTS: invalid endpoint")
	}
	query := u.Query()
	query.Set("output_format", "pcm_16000")
	u.RawQuery = query.Encode()
	request := struct {
		Text     string `json:"text"`
		Model    string `json:"model_id"`
		Language string `json:"language_code,omitempty"`
	}{text, "eleven_flash_v2_5", p.language}
	response, err := p.post(ctx, "TTS", u.String(), "Xi-Api-Key", p.elevenKey, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	// HTTP read boundaries can split a PCM16 sample. Carry the odd byte into
	// the next read and give emit ownership of each complete-sample slice.
	buffer := make([]byte, 4097)
	carry, total := 0, 0
	for {
		n, readErr := response.Body.Read(buffer[carry:])
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n += carry
		even := n &^ 1
		if even > 0 {
			if err := emit(bytes.Clone(buffer[:even])); err != nil {
				return err
			}
			total += even
		}
		carry = n - even
		if carry > 0 {
			buffer[0] = buffer[even]
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return providerIOError(ctx, "TTS: audio stream read failed")
			}
			if carry != 0 {
				return errors.New("TTS: truncated PCM16 sample")
			}
			if total == 0 {
				return errors.New("TTS: returned no audio")
			}
			return nil
		}
	}
}

func (p providers) post(ctx context.Context, name, endpoint, header, key string, payload any) (*http.Response, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: encode request failed", name)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("%s: invalid endpoint", name)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(header, key)
	client := http.Client{Timeout: 60 * time.Second}
	if p.httpClient != nil {
		client = *p.httpClient
	}
	// Never forward credentials or user input to a redirect destination.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return nil, providerIOError(ctx, name+": HTTP request failed")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_ = response.Body.Close()
		return nil, fmt.Errorf("%s: HTTP %d", name, response.StatusCode)
	}
	return response, nil
}

func providerIOError(ctx context.Context, message string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Transport errors may contain URLs, credentials or server close reasons.
	return errors.New(message)
}

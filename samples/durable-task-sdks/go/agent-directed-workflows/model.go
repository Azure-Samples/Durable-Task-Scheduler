package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type toolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type modelMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type chatModel interface {
	Complete(context.Context, []modelMessage, func(string)) (modelMessage, error)
}

type modelConfig struct {
	Endpoint, Deployment, APIVersion, APIKey string
}

var (
	deploymentPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,80}$`)
	versionPattern    = regexp.MustCompile(`^20[0-9]{2}-[0-9]{2}-[0-9]{2}(-preview)?$`)
)

func loadModelConfig(getenv func(string) string) (modelConfig, error) {
	config := modelConfig{
		Endpoint: strings.TrimSpace(getenv("AZURE_OPENAI_ENDPOINT")), Deployment: strings.TrimSpace(getenv("AZURE_OPENAI_DEPLOYMENT")),
		APIVersion: strings.TrimSpace(getenv("AZURE_OPENAI_API_VERSION")), APIKey: getenv("AZURE_OPENAI_API_KEY"),
	}
	if config.APIVersion == "" {
		config.APIVersion = "2024-10-21"
	}
	if _, err := azureEndpoint(config.Endpoint); err != nil {
		return config, err
	}
	if !deploymentPattern.MatchString(config.Deployment) {
		return config, errors.New("AZURE_OPENAI_DEPLOYMENT must be a deployment name (1–80 letters, digits, '.', '_' or '-')")
	}
	if !versionPattern.MatchString(config.APIVersion) {
		return config, errors.New("invalid AZURE_OPENAI_API_VERSION")
	}
	return config, nil
}

func azureEndpoint(value string) (*url.URL, error) {
	address, err := url.Parse(value)
	if err != nil || address.Scheme != "https" || address.User != nil ||
		address.RawQuery != "" || address.Fragment != "" ||
		(address.Path != "" && address.Path != "/") || (address.Port() != "" && address.Port() != "443") {
		return nil, errors.New("AZURE_OPENAI_ENDPOINT must be an HTTPS Azure resource root URL without credentials, query or fragment")
	}
	host := strings.ToLower(address.Hostname())
	for _, suffix := range []string{
		".openai.azure.com", ".cognitiveservices.azure.com", ".services.ai.azure.com",
		".openai.azure.us", ".cognitiveservices.azure.us", ".openai.azure.cn", ".cognitiveservices.azure.cn",
	} {
		if strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
			return address, nil
		}
	}
	return nil, errors.New("AZURE_OPENAI_ENDPOINT must name an Azure OpenAI/AI Services resource")
}

type openAIModel struct {
	endpoint   string
	deployment string
	apiVersion string
	key        string
	credential azcore.TokenCredential
	client     *http.Client
}

func newOpenAIModel(config modelConfig) (*openAIModel, error) {
	address, err := azureEndpoint(config.Endpoint)
	if err != nil {
		return nil, err
	}
	model := &openAIModel{
		endpoint: strings.TrimRight(address.String(), "/"), deployment: config.Deployment,
		apiVersion: config.APIVersion, key: config.APIKey,
		client: &http.Client{
			Timeout:       25 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
	if model.key == "" {
		model.credential, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("initialize Azure OpenAI identity: %w", err)
		}
	}
	return model, nil
}

func (m *openAIModel) Complete(ctx context.Context, messages []modelMessage, emit func(string)) (modelMessage, error) {
	body, err := json.Marshal(struct {
		Messages  []modelMessage `json:"messages"`
		Tools     any            `json:"tools"`
		Stream    bool           `json:"stream"`
		MaxTokens int            `json:"max_tokens"`
	}{
		Messages: messages, Stream: true, MaxTokens: 2048,
		Tools: []any{map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": "get_weather", "description": "Get synthetic sample weather for a location (not live weather)",
				"parameters": map[string]any{
					"type": "object", "additionalProperties": false,
					"properties": map[string]any{"location": map[string]any{"type": "string", "description": "City or location name"}},
					"required":   []string{"location"},
				},
			},
		}},
	})
	if err != nil {
		return modelMessage{}, err
	}
	address := m.endpoint + "/openai/deployments/" + url.PathEscape(m.deployment) +
		"/chat/completions?api-version=" + url.QueryEscape(m.apiVersion)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return modelMessage{}, errors.New("invalid Azure OpenAI request URL")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	if m.key != "" {
		request.Header.Set("api-key", m.key)
	} else {
		if m.credential == nil {
			return modelMessage{}, errors.New("Azure OpenAI credential is not configured")
		}
		token, err := m.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://cognitiveservices.azure.com/.default"}})
		if err != nil {
			if ctx.Err() != nil {
				return modelMessage{}, ctx.Err()
			}
			return modelMessage{}, errors.New("Azure OpenAI authentication failed")
		}
		request.Header.Set("Authorization", "Bearer "+token.Token)
	}
	response, err := m.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return modelMessage{}, ctx.Err()
		}
		return modelMessage{}, errors.New("Azure OpenAI request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return modelMessage{}, fmt.Errorf("Azure OpenAI returned HTTP %d", response.StatusCode)
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		return modelMessage{}, errors.New("Azure OpenAI did not return an SSE stream")
	}
	return parseModelStream(response.Body, emit)
}

type modelFrame struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int          `json:"index"`
				ID       string       `json:"id"`
				Type     string       `json:"type"`
				Function toolFunction `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

func parseModelStream(reader io.Reader, emit func(string)) (modelMessage, error) {
	limited := &io.LimitedReader{R: reader, N: 1024*1024 + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	result := modelMessage{Role: "assistant"}
	calls := make(map[int]toolCall)
	finished := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			if !finished {
				return modelMessage{}, errors.New("Azure OpenAI stream ended without a finish reason")
			}
			keys := make([]int, 0, len(calls))
			for key := range calls {
				keys = append(keys, key)
			}
			sort.Ints(keys)
			seenIDs := map[string]bool{}
			for index, key := range keys {
				call := calls[key]
				if key != index || call.ID == "" || seenIDs[call.ID] || call.Type != "function" || call.Function.Name == "" {
					return modelMessage{}, errors.New("invalid streamed tool call")
				}
				seenIDs[call.ID] = true
				result.ToolCalls = append(result.ToolCalls, call)
			}
			return result, nil
		}
		var frame modelFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil || (len(frame.Error) > 0 && string(frame.Error) != "null") {
			return modelMessage{}, errors.New("invalid Azure OpenAI stream frame")
		}
		if len(frame.Choices) == 0 {
			continue
		}
		choice := frame.Choices[0]
		if choice.FinishReason != "" {
			if choice.FinishReason != "stop" && choice.FinishReason != "tool_calls" {
				return modelMessage{}, errors.New("Azure OpenAI response was truncated or filtered")
			}
			finished = true
		}
		if len(result.Content)+len(choice.Delta.Content) > maxReplyBytes {
			return modelMessage{}, errors.New("Azure OpenAI response exceeded its byte budget")
		}
		result.Content += choice.Delta.Content
		if choice.Delta.Content != "" {
			emit(choice.Delta.Content)
		}
		for _, delta := range choice.Delta.ToolCalls {
			if delta.Index < 0 || delta.Index >= maxToolCalls {
				return modelMessage{}, errors.New("too many streamed tool calls")
			}
			call := calls[delta.Index]
			if delta.ID != "" {
				call.ID = delta.ID
			}
			if delta.Type != "" {
				call.Type = delta.Type
			}
			call.Function.Name += delta.Function.Name
			call.Function.Arguments += delta.Function.Arguments
			if len(call.ID) > 128 || len(call.Function.Name) > 100 || len(call.Function.Arguments) > 4096 {
				return modelMessage{}, errors.New("streamed tool call exceeded its byte budget")
			}
			calls[delta.Index] = call
		}
	}
	if err := scanner.Err(); err != nil {
		return modelMessage{}, errors.New("failed to read Azure OpenAI stream")
	}
	return modelMessage{}, errors.New("Azure OpenAI stream ended before [DONE]")
}

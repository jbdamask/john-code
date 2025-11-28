package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const DefaultOpenAIEndpoint = "https://api.openai.com/v1/chat/completions"

type OpenAIClient struct {
	apiKey   string
	endpoint string
	model    string
	client   *http.Client
}

func NewOpenAIClient(apiKey string, model string) *OpenAIClient {
	if model == "" {
		model = "gpt-5"
	}

	return &OpenAIClient{
		apiKey:   apiKey,
		endpoint: DefaultOpenAIEndpoint,
		model:    model,
		client:   &http.Client{},
	}
}

// OpenAI API structures
type openAIRequest struct {
	Model       string            `json:"model"`
	Messages    []openAIMessage   `json:"messages"`
	Tools       []openAITool      `json:"tools,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role       string             `json:"role"`
	Content    interface{}        `json:"content"` // string or []openAIContentPart
	ToolCalls  []openAIToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
}

type openAIContentPart struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *openAIImageURL   `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAITool struct {
	Type     string           `json:"type"`
	Function openAIFunction   `json:"function"`
}

type openAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Streaming structures
type openAIStreamChunk struct {
	ID      string              `json:"id"`
	Choices []openAIStreamChoice `json:"choices"`
}

type openAIStreamChoice struct {
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

type openAIStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

func (c *OpenAIClient) Generate(ctx context.Context, messages []Message, tools []interface{}) (*Message, error) {
	return c.GenerateStream(ctx, messages, tools, nil)
}

func (c *OpenAIClient) GenerateStream(ctx context.Context, messages []Message, tools []interface{}, outputChan chan<- string) (*Message, error) {
	apiMessages := make([]openAIMessage, 0, len(messages))

	for _, msg := range messages {
		apiMsg := openAIMessage{
			Role: string(msg.Role),
		}

		switch msg.Role {
		case RoleSystem:
			apiMsg.Content = msg.Content

		case RoleUser:
			if len(msg.Images) > 0 {
				var parts []openAIContentPart
				if msg.Content != "" {
					parts = append(parts, openAIContentPart{
						Type: "text",
						Text: msg.Content,
					})
				}
				for _, imgPath := range msg.Images {
					data, err := os.ReadFile(imgPath)
					if err != nil {
						continue
					}
					ext := strings.ToLower(filepath.Ext(imgPath))
					var mediaType string
					switch ext {
					case ".jpg", ".jpeg":
						mediaType = "image/jpeg"
					case ".png":
						mediaType = "image/png"
					case ".gif":
						mediaType = "image/gif"
					case ".webp":
						mediaType = "image/webp"
					default:
						mediaType = "image/jpeg"
					}
					encoded := base64.StdEncoding.EncodeToString(data)
					parts = append(parts, openAIContentPart{
						Type: "image_url",
						ImageURL: &openAIImageURL{
							URL: fmt.Sprintf("data:%s;base64,%s", mediaType, encoded),
						},
					})
				}
				apiMsg.Content = parts
			} else {
				apiMsg.Content = msg.Content
			}

		case RoleAssistant:
			apiMsg.Content = msg.Content
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Args)
					apiMsg.ToolCalls = append(apiMsg.ToolCalls, openAIToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: openAIFunctionCall{
							Name:      tc.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}

		case RoleTool:
			apiMsg.Role = "tool"
			apiMsg.Content = msg.ToolResult.Content
			apiMsg.ToolCallID = msg.ToolResult.ToolCallID
		}

		apiMessages = append(apiMessages, apiMsg)
	}

	// Convert tools to OpenAI format
	var openAITools []openAITool
	for _, t := range tools {
		var name, desc string
		var schema interface{}

		// Handle both ToolDefinition struct and map[string]interface{}
		switch tool := t.(type) {
		case map[string]interface{}:
			name, _ = tool["name"].(string)
			desc, _ = tool["description"].(string)
			schema = tool["input_schema"]
		default:
			// Try to extract via JSON marshaling (handles ToolDefinition)
			data, err := json.Marshal(t)
			if err != nil {
				continue
			}
			var toolMap map[string]interface{}
			if err := json.Unmarshal(data, &toolMap); err != nil {
				continue
			}
			name, _ = toolMap["name"].(string)
			desc, _ = toolMap["description"].(string)
			schema = toolMap["input_schema"]
		}

		if name != "" {
			openAITools = append(openAITools, openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        name,
					Description: desc,
					Parameters:  schema,
				},
			})
		}
	}

	reqBody := openAIRequest{
		Model:     c.model,
		Messages:  apiMessages,
		Tools:     openAITools,
		MaxTokens: 8192,
		Stream:    true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	finalMsg := &Message{
		Role:      RoleAssistant,
		ToolCalls: []ToolCall{},
	}

	// Track tool calls being built
	type toolBuilder struct {
		ID         string
		Name       string
		ArgsBuffer string
	}
	toolBuilders := make(map[int]*toolBuilder)

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			delta := choice.Delta

			// Text content
			if delta.Content != "" {
				finalMsg.Content += delta.Content
				if outputChan != nil {
					outputChan <- delta.Content
				}
			}

			// Tool calls
			for i, tc := range delta.ToolCalls {
				if _, exists := toolBuilders[i]; !exists {
					toolBuilders[i] = &toolBuilder{}
				}
				tb := toolBuilders[i]

				if tc.ID != "" {
					tb.ID = tc.ID
				}
				if tc.Function.Name != "" {
					tb.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					tb.ArgsBuffer += tc.Function.Arguments
				}
			}
		}
	}

	// Finalize tool calls
	for _, tb := range toolBuilders {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tb.ArgsBuffer), &args); err != nil {
			args = make(map[string]interface{})
		}
		finalMsg.ToolCalls = append(finalMsg.ToolCalls, ToolCall{
			ID:   tb.ID,
			Name: tb.Name,
			Args: args,
		})
	}

	return finalMsg, nil
}

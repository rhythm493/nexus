package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	openai "github.com/sashabaranov/go-openai"
)

// --- Our types → SDK types ---

// messagesToSDK converts our Message slice to the SDK's ChatCompletionMessage slice.
func messagesToSDK(msgs []Message) []openai.ChatCompletionMessage {
	out := make([]openai.ChatCompletionMessage, len(msgs))
	for i, m := range msgs {
		sdkMsg := openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}

		if len(m.ToolCalls) > 0 {
			sdkMsg.ToolCalls = make([]openai.ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				sdkMsg.ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		out[i] = sdkMsg
	}
	return out
}

// toolsToSDK converts our Tool slice to the SDK's Tool slice.
func toolsToSDK(tools []Tool) []openai.Tool {
	out := make([]openai.Tool, len(tools))
	for i, t := range tools {
		out[i] = openai.Tool{
			Type: openai.ToolType(t.Type),
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters, // map[string]interface{} satisfies any
			},
		}
	}
	return out
}

// --- SDK types → Our types ---

// responseFromSDK converts the SDK's ChatCompletionResponse to our Response.
func responseFromSDK(resp openai.ChatCompletionResponse) *Response {
	if len(resp.Choices) == 0 {
		return &Response{}
	}

	choice := resp.Choices[0]
	return &Response{
		Text:      choice.Message.Content,
		ToolCalls: toolCallsFromSDK(choice.Message.ToolCalls),
	}
}

// toolCallsFromSDK converts SDK ToolCall slice to our ToolCall slice.
func toolCallsFromSDK(calls []openai.ToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]ToolCall, len(calls))
	for i, tc := range calls {
		out[i] = ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return out
}

// modelsFromSDK converts the SDK's ModelsList to our Model slice.
func modelsFromSDK(list openai.ModelsList) []Model {
	out := make([]Model, len(list.Models))
	for i, m := range list.Models {
		out[i] = Model{
			ID:      m.ID,
			Name:    m.ID,
			OwnedBy: m.OwnedBy,
		}
	}
	return out
}

// extractErrorMessage tries to extract a human-readable error message from a
// raw API response body. Non-OpenAI providers (e.g. Gemini) can return errors
// in formats that the go-openai SDK cannot parse (e.g. JSON arrays, Google RPC
// style), causing confusing "cannot unmarshal array" wrapping errors.
func extractErrorMessage(body []byte) string {
	// Try OpenAI format: {"error": {"message": "..."}}
	var openAIErr struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &openAIErr) == nil && openAIErr.Error.Message != "" {
		return openAIErr.Error.Message
	}

	// Try Gemini array format: [{"error": {"message": "..."}}]
	var arr []json.RawMessage
	if json.Unmarshal(body, &arr) == nil && len(arr) > 0 {
		var elemErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(arr[0], &elemErr) == nil && elemErr.Error.Message != "" {
			return elemErr.Error.Message
		}
	}

	// Try Google RPC format: {"error": {"message": "...", "status": "...", "details": [...]}}
	var googleErr struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &googleErr) == nil && googleErr.Error.Message != "" {
		msg := googleErr.Error.Message
		if googleErr.Error.Status != "" {
			msg += " (" + googleErr.Error.Status + ")"
		}
		return msg
	}

	return ""
}

// convertSDKError converts SDK errors to our error types.
// Maps HTTP 429 to RateLimitError for the rotating provider.
func convertSDKError(err error) error {
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		if apiErr.HTTPStatusCode == http.StatusTooManyRequests {
			return &RateLimitError{
				RetryAfter: parseRetryAfter(""),
				Message:    apiErr.Message,
			}
		}
		return fmt.Errorf("API error (%d): %s", apiErr.HTTPStatusCode, apiErr.Message)
	}

	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) {
		if reqErr.HTTPStatusCode == http.StatusTooManyRequests {
			return &RateLimitError{
				RetryAfter: parseRetryAfter(""),
				Message:    reqErr.Error(),
			}
		}
		msg := extractErrorMessage(reqErr.Body)
		if msg != "" {
			return fmt.Errorf("API error (%d): %s", reqErr.HTTPStatusCode, msg)
		}
		return fmt.Errorf("request error (%d): %s", reqErr.HTTPStatusCode, reqErr.Error())
	}

	return err
}

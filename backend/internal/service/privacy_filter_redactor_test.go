package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type fakePrivacyFilterClient struct {
	calls int
}

func (f *fakePrivacyFilterClient) RedactBatch(ctx context.Context, texts []string) ([]PrivacyFilterTextResult, error) {
	f.calls++
	out := make([]PrivacyFilterTextResult, len(texts))
	for i, text := range texts {
		out[i] = PrivacyFilterTextResult{Redacted: "[redacted:" + text + "]", Hit: true, Count: 1}
	}
	return out, nil
}

func TestRedactPrivacyFilterBodyAnthropicMessagesOnlyTextFields(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{
		"model":"claude-sonnet-4-5",
		"system":"contact me at milk@example.com",
		"messages":[
			{"role":"user","content":[{"type":"text","text":"my token is sk-test"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgoAAA"}}]}
		],
		"metadata":{"trace":"keep"}
	}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolAnthropicMessages, body, client)

	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, 1, client.calls)
	require.Equal(t, "claude-sonnet-4-5", gjson.GetBytes(result.Body, "model").String())
	require.Equal(t, "keep", gjson.GetBytes(result.Body, "metadata.trace").String())
	require.Equal(t, "iVBORw0KGgoAAA", gjson.GetBytes(result.Body, "messages.0.content.1.source.data").String())
	require.Equal(t, "[redacted:contact me at milk@example.com]", gjson.GetBytes(result.Body, "system").String())
	require.Equal(t, "[redacted:my token is sk-test]", gjson.GetBytes(result.Body, "messages.0.content.0.text").String())
}

func TestRedactPrivacyFilterBodyAnthropicSystemBlocks(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{
		"model":"claude-sonnet-4-5",
		"system":[{"type":"text","text":"admin email root@example.com"},{"type":"cache_control","ttl":"1h"}],
		"messages":[{"role":"user","content":"hello"}]
	}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolAnthropicMessages, body, client)

	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, "[redacted:admin email root@example.com]", gjson.GetBytes(result.Body, "system.0.text").String())
	require.Equal(t, "1h", gjson.GetBytes(result.Body, "system.1.ttl").String())
	require.Equal(t, "[redacted:hello]", gjson.GetBytes(result.Body, "messages.0.content").String())
}

func TestRedactPrivacyFilterBodyOpenAIEmbeddingsInputString(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{"model":"text-embedding-3-large","input":"email milk@example.com"}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolOpenAIEmbeddings, body, client)

	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, 1, client.calls)
	require.Equal(t, "text-embedding-3-large", gjson.GetBytes(result.Body, "model").String())
	require.Equal(t, "[redacted:email milk@example.com]", gjson.GetBytes(result.Body, "input").String())
}

func TestRedactPrivacyFilterBodyOpenAIEmbeddingsInputStringArray(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{"model":"text-embedding-3-large","input":["email milk@example.com","phone 123456"]}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolOpenAIEmbeddings, body, client)

	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, 1, client.calls)
	require.Equal(t, "text-embedding-3-large", gjson.GetBytes(result.Body, "model").String())
	require.Equal(t, "[redacted:email milk@example.com]", gjson.GetBytes(result.Body, "input.0").String())
	require.Equal(t, "[redacted:phone 123456]", gjson.GetBytes(result.Body, "input.1").String())
}

func TestRedactPrivacyFilterBodyOpenAIEmbeddingsTokenArraysSkipClient(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{"model":"text-embedding-3-large","input":[[1,2,3],[4,5,6]]}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolOpenAIEmbeddings, body, client)

	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, body, result.Body)
	require.Zero(t, client.calls)
}

func TestRedactPrivacyFilterBodyOpenAIResponsesInputString(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{"model":"gpt-5.4","input":"email milk@example.com"}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolOpenAIResponses, body, client)

	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, "gpt-5.4", gjson.GetBytes(result.Body, "model").String())
	require.Equal(t, "[redacted:email milk@example.com]", gjson.GetBytes(result.Body, "input").String())
}

func TestRedactPrivacyFilterBodyOpenAIResponses(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{
		"model":"gpt-5.4",
		"instructions":"system phone 123456",
		"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"email milk@example.com"},{"type":"input_image","image_url":"data:image/png;base64,abc"}]}]
	}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolOpenAIResponses, body, client)

	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, "gpt-5.4", gjson.GetBytes(result.Body, "model").String())
	require.Equal(t, "data:image/png;base64,abc", gjson.GetBytes(result.Body, "input.0.content.1.image_url").String())
	require.Equal(t, "[redacted:system phone 123456]", gjson.GetBytes(result.Body, "instructions").String())
	require.Equal(t, "[redacted:email milk@example.com]", gjson.GetBytes(result.Body, "input.0.content.0.text").String())
}

func TestRedactPrivacyFilterBodyGemini(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{
		"contents":[{"role":"user","parts":[{"text":"secret abc"},{"inline_data":{"mime_type":"image/png","data":"base64-image"}}]}],
		"system_instruction":{"parts":[{"text":"system secret"}]}
	}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolGemini, body, client)

	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, "base64-image", gjson.GetBytes(result.Body, "contents.0.parts.1.inline_data.data").String())
	require.Equal(t, "[redacted:secret abc]", gjson.GetBytes(result.Body, "contents.0.parts.0.text").String())
	require.Equal(t, "[redacted:system secret]", gjson.GetBytes(result.Body, "system_instruction.parts.0.text").String())
}

func TestRedactPrivacyFilterBodyNoTextSkipsClient(t *testing.T) {
	client := &fakePrivacyFilterClient{}
	body := []byte(`{"model":"gpt-5.4","input":[{"type":"message","role":"user","content":[{"type":"input_image","image_url":"data:image/png;base64,abc"}]}]}`)

	result, err := RedactPrivacyFilterBody(t.Context(), ContentModerationProtocolOpenAIResponses, body, client)

	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, body, result.Body)
	require.Zero(t, client.calls)
}

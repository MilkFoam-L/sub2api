package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type PrivacyFilterBodyResult struct {
	Body        []byte
	Changed     bool
	HitCount    int
	EntityTypes []string
}

type privacyFilterTextTarget struct {
	text string
	set  func(string)
}

func RedactPrivacyFilterBody(ctx context.Context, protocol string, body []byte, client PrivacyFilterClient) (*PrivacyFilterBodyResult, error) {
	result := &PrivacyFilterBodyResult{Body: body}
	if len(body) == 0 {
		return result, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse privacy filter body: %w", err)
	}

	var targets []privacyFilterTextTarget
	collectPrivacyFilterTargets(protocol, root, &targets)
	if len(targets) == 0 {
		return result, nil
	}
	if client == nil {
		return nil, fmt.Errorf("privacy filter client is not configured")
	}

	texts := make([]string, 0, len(targets))
	for _, target := range targets {
		texts = append(texts, target.text)
	}
	redacted, err := client.RedactBatch(ctx, texts)
	if err != nil {
		return nil, err
	}
	if len(redacted) != len(targets) {
		return nil, fmt.Errorf("privacy filter returned %d results for %d texts", len(redacted), len(targets))
	}

	entitySeen := make(map[string]struct{})
	for i := range targets {
		item := redacted[i]
		if item.Hit {
			result.HitCount += item.Count
			for _, entity := range item.Entities {
				entityType := strings.TrimSpace(entity.Type)
				if entityType == "" {
					continue
				}
				if _, ok := entitySeen[entityType]; !ok {
					entitySeen[entityType] = struct{}{}
					result.EntityTypes = append(result.EntityTypes, entityType)
				}
			}
		}
		if item.Redacted != targets[i].text {
			result.Changed = true
			targets[i].set(item.Redacted)
		}
	}
	if !result.Changed {
		return result, nil
	}

	out, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("marshal privacy filtered body: %w", err)
	}
	result.Body = out
	return result, nil
}

func collectPrivacyFilterTargets(protocol string, root any, targets *[]privacyFilterTextTarget) {
	obj, _ := root.(map[string]any)
	if obj == nil {
		return
	}
	switch protocol {
	case ContentModerationProtocolAnthropicMessages:
		collectContentValueTargets(obj["system"], func(redacted any) { obj["system"] = redacted }, targets)
		collectMessageContentTargets(obj["messages"], targets)
	case ContentModerationProtocolOpenAIChat:
		collectMessageContentTargets(obj["messages"], targets)
	case ContentModerationProtocolOpenAIEmbeddings:
		collectEmbeddingsInputTargets(obj, targets)
	case ContentModerationProtocolOpenAIResponses:
		collectObjectStringField(obj, "instructions", targets)
		collectResponsesInputTargets(obj, targets)
	case ContentModerationProtocolGemini:
		collectGeminiSystemInstructionTargets(obj["system_instruction"], targets)
		collectGeminiContentsTargets(obj["contents"], targets)
	case ContentModerationProtocolOpenAIImages:
		collectObjectStringField(obj, "prompt", targets)
	default:
		collectObjectStringField(obj, "instructions", targets)
		collectResponsesInputTargets(obj, targets)
		collectMessageContentTargets(obj["messages"], targets)
		collectGeminiContentsTargets(obj["contents"], targets)
	}
}

func collectObjectStringField(obj map[string]any, key string, targets *[]privacyFilterTextTarget) {
	value, ok := obj[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return
	}
	collectPrivacyTextTarget(value, func(redacted string) { obj[key] = redacted }, targets)
}

func collectMessageContentTargets(value any, targets *[]privacyFilterTextTarget) {
	messages, ok := value.([]any)
	if !ok {
		return
	}
	for _, item := range messages {
		msg, ok := item.(map[string]any)
		if !ok {
			continue
		}
		collectContentValueTargets(msg["content"], func(redacted any) { msg["content"] = redacted }, targets)
	}
}

func collectEmbeddingsInputTargets(obj map[string]any, targets *[]privacyFilterTextTarget) {
	input, exists := obj["input"]
	if !exists {
		return
	}
	switch typed := input.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return
		}
		collectPrivacyTextTarget(typed, func(redacted string) { obj["input"] = redacted }, targets)
	case []any:
		for i, item := range typed {
			text, ok := item.(string)
			if !ok || strings.TrimSpace(text) == "" {
				continue
			}
			idx := i
			collectPrivacyTextTarget(text, func(redacted string) { typed[idx] = redacted }, targets)
		}
	}
}

func collectResponsesInputTargets(value any, targets *[]privacyFilterTextTarget) {
	if obj, ok := value.(map[string]any); ok {
		input, exists := obj["input"]
		if !exists {
			return
		}
		collectContentValueTargets(input, func(redacted any) { obj["input"] = redacted }, targets)
		return
	}
	collectContentValueTargets(value, nil, targets)
}

func collectGeminiSystemInstructionTargets(value any, targets *[]privacyFilterTextTarget) {
	collectGeminiPartsContainerTargets(value, targets)
}

func collectGeminiContentsTargets(value any, targets *[]privacyFilterTextTarget) {
	contents, ok := value.([]any)
	if !ok {
		return
	}
	for _, item := range contents {
		collectGeminiPartsContainerTargets(item, targets)
	}
}

func collectGeminiPartsContainerTargets(value any, targets *[]privacyFilterTextTarget) {
	obj, ok := value.(map[string]any)
	if !ok {
		return
	}
	parts, ok := obj["parts"].([]any)
	if !ok {
		return
	}
	for _, partValue := range parts {
		part, ok := partValue.(map[string]any)
		if !ok {
			continue
		}
		collectObjectStringField(part, "text", targets)
	}
}

func collectContentValueTargets(value any, setValue func(any), targets *[]privacyFilterTextTarget) {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" || setValue == nil {
			return
		}
		collectPrivacyTextTarget(typed, func(redacted string) { setValue(redacted) }, targets)
	case []any:
		for _, itemValue := range typed {
			item, ok := itemValue.(map[string]any)
			if !ok {
				continue
			}
			collectTextFieldsInObject(item, targets)
			if nested, ok := item["content"]; ok {
				collectContentValueTargets(nested, func(redacted any) { item["content"] = redacted }, targets)
			}
		}
	case map[string]any:
		collectTextFieldsInObject(typed, targets)
		if nested, ok := typed["content"]; ok {
			collectContentValueTargets(nested, func(redacted any) { typed["content"] = redacted }, targets)
		}
	}
}

func collectTextFieldsInObject(obj map[string]any, targets *[]privacyFilterTextTarget) {
	collectObjectStringField(obj, "text", targets)
	collectObjectStringField(obj, "input_text", targets)
}

func collectPrivacyTextTarget(text string, set func(string), targets *[]privacyFilterTextTarget) {
	if strings.TrimSpace(text) == "" || set == nil {
		return
	}
	*targets = append(*targets, privacyFilterTextTarget{text: text, set: set})
}

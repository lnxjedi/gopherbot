package googlechat

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const diagnosticPreviewLimit = 120

func diagnosticPreview(text string) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= diagnosticPreviewLimit {
		return text
	}
	return string(runes[:diagnosticPreviewLimit]) + " ..."
}

func diagnosticPreviewBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if !utf8.Valid(data) {
		limit := len(data)
		if limit > 32 {
			limit = 32
		}
		return fmt.Sprintf("<%d bytes: %x>", len(data), data[:limit])
	}
	return diagnosticPreview(string(data))
}

func summarizePubSubAttributes(attrs map[string]string) string {
	if len(attrs) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, diagnosticPreview(attrs[key])))
	}
	return strings.Join(parts, ", ")
}

func summarizeChatEvent(event *chatEvent) string {
	if event == nil {
		return "nil"
	}
	var (
		userID    string
		spaceID   string
		direct    bool
		messageID string
		text      string
		arg       string
		slash     bool
	)
	if event.User != nil {
		userID = normalizeUserResource(event.User.Name)
	}
	if event.Message != nil {
		messageID = strings.TrimSpace(event.Message.Name)
		text = diagnosticPreview(event.Message.Text)
		arg = diagnosticPreview(event.Message.ArgumentText)
		slash = event.Message.SlashCommand != nil
		if userID == "" && event.Message.Sender != nil {
			userID = normalizeUserResource(event.Message.Sender.Name)
		}
		if event.Message.Space != nil {
			spaceID = strings.TrimSpace(event.Message.Space.Name)
			direct = strings.EqualFold(event.Message.Space.SpaceType, "DIRECT_MESSAGE")
		}
	}
	if event.Space != nil {
		spaceID = strings.TrimSpace(event.Space.Name)
		direct = strings.EqualFold(event.Space.SpaceType, "DIRECT_MESSAGE")
	}
	return fmt.Sprintf("type=%q message=%q user=%q space=%q direct=%t slash=%t appCommand=%t text=%q argument=%q",
		strings.TrimSpace(event.Type),
		messageID,
		userID,
		spaceID,
		direct,
		slash,
		event.AppCommandMetadata != nil,
		text,
		arg,
	)
}

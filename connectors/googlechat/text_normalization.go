package googlechat

import (
	"sort"
	"strings"

	chatapi "google.golang.org/api/chat/v1"
)

type mentionSpan struct {
	start       int
	length      int
	replacement string
}

func (gc *googleChatConnector) normalizeEventText(msg *chatEventMessage, explicit bool) string {
	if msg == nil {
		return ""
	}
	if explicit {
		if text := strings.TrimSpace(msg.ArgumentText); text != "" {
			return text
		}
	}
	return strings.TrimSpace(applyMentionRewrites(msg.Text, gc.eventMentionSpans(msg.Annotations)))
}

func (gc *googleChatConnector) normalizeAPIText(message *chatapi.Message, explicit bool) string {
	if message == nil {
		return ""
	}
	if explicit {
		if text := strings.TrimSpace(message.ArgumentText); text != "" {
			return text
		}
	}
	return strings.TrimSpace(applyMentionRewrites(message.Text, gc.apiMentionSpans(message.Annotations)))
}

func (gc *googleChatConnector) eventMentionSpans(annotations []*chatEventAnnotation) []mentionSpan {
	if len(annotations) == 0 {
		return nil
	}
	spans := make([]mentionSpan, 0, len(annotations))
	for _, annotation := range annotations {
		if annotation == nil || !strings.EqualFold(strings.TrimSpace(annotation.Type), "USER_MENTION") {
			continue
		}
		if annotation.StartIndex < 0 || annotation.Length <= 0 || annotation.UserMention == nil || annotation.UserMention.User == nil {
			continue
		}
		name := gc.canonicalMentionName(annotation.UserMention.User.Name)
		if name == "" {
			continue
		}
		spans = append(spans, mentionSpan{
			start:       annotation.StartIndex,
			length:      annotation.Length,
			replacement: "@" + name,
		})
	}
	return spans
}

func (gc *googleChatConnector) apiMentionSpans(annotations []*chatapi.Annotation) []mentionSpan {
	if len(annotations) == 0 {
		return nil
	}
	spans := make([]mentionSpan, 0, len(annotations))
	for _, annotation := range annotations {
		if annotation == nil || !strings.EqualFold(strings.TrimSpace(annotation.Type), "USER_MENTION") {
			continue
		}
		if annotation.StartIndex < 0 || annotation.Length <= 0 || annotation.UserMention == nil || annotation.UserMention.User == nil {
			continue
		}
		name := gc.canonicalMentionName(annotation.UserMention.User.Name)
		if name == "" {
			continue
		}
		spans = append(spans, mentionSpan{
			start:       int(annotation.StartIndex),
			length:      int(annotation.Length),
			replacement: "@" + name,
		})
	}
	return spans
}

func (gc *googleChatConnector) canonicalMentionName(user string) string {
	resource := normalizeUserResource(user)
	if resource == "" {
		return ""
	}
	if gc.isBotResource(resource) {
		return strings.ToLower(strings.TrimSpace(gc.botName))
	}

	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if name, ok := gc.configuredUsers[resource]; ok && strings.TrimSpace(name) != "" {
		return name
	}
	if record, ok := gc.usersByID[resource]; ok && strings.TrimSpace(record.CanonicalName) != "" {
		return record.CanonicalName
	}
	return ""
}

func (gc *googleChatConnector) eventMentionedBotID(message *chatEventMessage) string {
	if gc == nil || message == nil {
		return ""
	}
	return gc.findMentionedBotID(message.Text, func(yield func(resource, userType string, start, length int)) {
		for _, annotation := range message.Annotations {
			if annotation == nil || !strings.EqualFold(strings.TrimSpace(annotation.Type), "USER_MENTION") {
				continue
			}
			if annotation.StartIndex < 0 || annotation.Length <= 0 || annotation.UserMention == nil || annotation.UserMention.User == nil {
				continue
			}
			yield(annotation.UserMention.User.Name, annotation.UserMention.User.Type, annotation.StartIndex, annotation.Length)
		}
	})
}

func (gc *googleChatConnector) apiMentionedBotID(message *chatapi.Message) string {
	if gc == nil || message == nil {
		return ""
	}
	return gc.findMentionedBotID(message.Text, func(yield func(resource, userType string, start, length int)) {
		for _, annotation := range message.Annotations {
			if annotation == nil || !strings.EqualFold(strings.TrimSpace(annotation.Type), "USER_MENTION") {
				continue
			}
			if annotation.StartIndex < 0 || annotation.Length <= 0 || annotation.UserMention == nil || annotation.UserMention.User == nil {
				continue
			}
			yield(annotation.UserMention.User.Name, annotation.UserMention.User.Type, int(annotation.StartIndex), int(annotation.Length))
		}
	})
}

func (gc *googleChatConnector) findMentionedBotID(text string, walk func(func(resource, userType string, start, length int))) string {
	if gc == nil || walk == nil {
		return ""
	}
	botName := strings.ToLower(strings.TrimSpace(gc.botName))
	runes := []rune(text)
	found := ""
	walk(func(resource, userType string, start, length int) {
		normalized := normalizeUserResource(resource)
		if normalized == "" || found != "" {
			return
		}
		if gc.isBotResource(normalized) {
			found = normalized
			return
		}
		if !strings.EqualFold(strings.TrimSpace(userType), "BOT") {
			return
		}
		if start < 0 || length <= 0 || start+length > len(runes) {
			return
		}
		mentionText := strings.ToLower(strings.TrimSpace(string(runes[start : start+length])))
		mentionText = strings.TrimPrefix(mentionText, "@")
		if botName != "" && strings.Contains(mentionText, botName) {
			found = normalized
		}
	})
	return found
}

func (gc *googleChatConnector) isBotResource(resource string) bool {
	resource = normalizeUserResource(resource)
	if resource == "" {
		return false
	}
	if resource == "users/app" {
		return true
	}
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return resource != "" && resource == gc.selfID
}

func applyMentionRewrites(text string, spans []mentionSpan) string {
	if text == "" || len(spans) == 0 {
		return text
	}
	runes := []rune(text)
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].start > spans[j].start
	})
	for _, span := range spans {
		if span.start < 0 || span.length <= 0 || span.start > len(runes) {
			continue
		}
		end := span.start + span.length
		if end > len(runes) {
			continue
		}
		replacement := []rune(span.replacement)
		runes = append(runes[:span.start], append(replacement, runes[end:]...)...)
	}
	return string(runes)
}

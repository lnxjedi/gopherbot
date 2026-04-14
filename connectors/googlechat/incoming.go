package googlechat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type chatEvent struct {
	Type               string                   `json:"type"`
	Message            *chatEventMessage        `json:"message"`
	User               *chatEventUser           `json:"user"`
	Thread             *chatEventThread         `json:"thread"`
	Space              *chatEventSpace          `json:"space"`
	AppCommandMetadata *chatEventAppCommandMeta `json:"appCommandMetadata"`
}

type chatEventMessage struct {
	Name         string                 `json:"name"`
	Text         string                 `json:"text"`
	ArgumentText string                 `json:"argumentText"`
	Annotations  []*chatEventAnnotation `json:"annotations"`
	Thread       *chatEventThread       `json:"thread"`
	ThreadReply  bool                   `json:"threadReply"`
	Sender       *chatEventUser         `json:"sender"`
	SlashCommand *chatEventSlashCommand `json:"slashCommand"`
	Space        *chatEventSpace        `json:"space"`
}

type chatEventUser struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Type        string `json:"type"`
}

type chatEventThread struct {
	Name      string `json:"name"`
	ThreadKey string `json:"threadKey"`
}

type chatEventSpace struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	SpaceType   string `json:"spaceType"`
}

type chatEventAppCommandMeta struct {
	AppCommandId   jsonInt64 `json:"appCommandId"`
	AppCommandType string    `json:"appCommandType"`
}

type chatEventSlashCommand struct {
	CommandId jsonInt64 `json:"commandId"`
}

type chatEventAnnotation struct {
	Type        string                    `json:"type"`
	StartIndex  int                       `json:"startIndex"`
	Length      int                       `json:"length"`
	UserMention *chatEventUserMentionMeta `json:"userMention"`
}

type chatEventUserMentionMeta struct {
	Type string         `json:"type"`
	User *chatEventUser `json:"user"`
}

// jsonInt64 accepts either a JSON number or a quoted decimal string.
// Google Chat currently sends command IDs as strings in some live payloads.
type jsonInt64 int64

func (v *jsonInt64) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*v = 0
		return nil
	}

	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*v = 0
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid quoted integer %q: %w", s, err)
		}
		*v = jsonInt64(n)
		return nil
	}

	var n int64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*v = jsonInt64(n)
	return nil
}

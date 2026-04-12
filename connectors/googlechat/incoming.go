package googlechat

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
	AppCommandId   int64  `json:"appCommandId"`
	AppCommandType string `json:"appCommandType"`
}

type chatEventSlashCommand struct {
	CommandId int64 `json:"commandId"`
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

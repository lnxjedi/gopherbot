package googlechat

import (
	"context"
	"log"
	"strings"
	"time"

	chat "cloud.google.com/go/chat/apiv1"
	"cloud.google.com/go/chat/apiv1/chatpb"
	"cloud.google.com/go/pubsub"
	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/internal/gcloud"
	chatapi "google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
)

const (
	defaultCredentialsEncryptedFile = "gopherbot-key.json.enc"
	chatBotScope                    = "https://www.googleapis.com/auth/chat.bot"
	chatAppMessagesReadonlyScope    = "https://www.googleapis.com/auth/chat.app.messages.readonly"
)

type config struct {
	ProjectID                string
	SubscriptionID           string
	CredentialsEncryptedFile string
	AmbientMessages          *bool
	ThreadResponses          *bool
	SlashCommand             string
	UserMap                  map[string]string
}

func normalizeConfiguredUserMap(in map[string]string, h robot.Handler) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for user, id := range in {
		name := strings.TrimSpace(user)
		uid := normalizeUserResource(id)
		if name == "" || uid == "" {
			h.Log(robot.Warn, "Ignoring invalid Google Chat UserMap entry (empty username or user ID): %q -> %q", user, id)
			continue
		}
		if strings.ToLower(name) != name {
			h.Log(robot.Warn, "Ignoring Google Chat UserMap entry with uppercase username: %q", user)
			continue
		}
		out[name] = uid
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeSubscriptionID(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	if idx := strings.LastIndex(in, "/subscriptions/"); idx >= 0 {
		return strings.TrimSpace(in[idx+len("/subscriptions/"):])
	}
	if idx := strings.LastIndex(in, "subscriptions/"); idx >= 0 {
		return strings.TrimSpace(in[idx+len("subscriptions/"):])
	}
	return in
}

func normalizeSlashCommand(in string) string {
	in = strings.TrimSpace(in)
	in = strings.TrimPrefix(in, "/")
	if in == "" {
		return ""
	}
	return strings.ToLower(in)
}

func boolValueOrDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func Initialize(handler robot.Handler, l *log.Logger) robot.InitializedConnector {
	var c config
	if err := handler.GetProtocolConfig(&c); err != nil {
		handler.Log(robot.Fatal, "Unable to retrieve googlechat protocol configuration: %v", err)
	}

	credentialsPath := strings.TrimSpace(c.CredentialsEncryptedFile)
	if credentialsPath == "" {
		credentialsPath = defaultCredentialsEncryptedFile
	}
	creds, err := gcloud.LoadServiceAccountCredentials(handler.ReadEncryptedFile, credentialsPath)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to load Google Chat credentials: %v", err)
	}
	projectID, err := gcloud.ResolveProjectID(c.ProjectID, creds)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to determine Google Chat project ID: %v", err)
	}
	subscriptionID := normalizeSubscriptionID(c.SubscriptionID)
	if subscriptionID == "" {
		handler.Log(robot.Fatal, "Google Chat protocol config requires SubscriptionID")
	}

	opts := gcloud.ServiceAccountClientOptions(creds)
	ctx := context.Background()

	pubsubClient, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to create Google Pub/Sub client: %v", err)
	}
	subscription := pubsubClient.Subscription(subscriptionID)
	subscriptionConfig, err := subscription.Config(ctx)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to inspect Google Pub/Sub subscription %q: %v", subscriptionID, err)
	}
	chatOpts := append([]option.ClientOption{}, opts...)
	chatOpts = append(chatOpts, option.WithScopes(chatBotScope))
	chatClient, err := chat.NewClient(ctx, chatOpts...)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to create Google Chat client: %v", err)
	}
	chatAPIService, err := chatapi.NewService(ctx, chatOpts...)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to create Google Chat REST service: %v", err)
	}
	workspaceHTTPClient, err := gcloud.NewScopedHTTPClient(ctx, creds, chatBotScope, chatAppMessagesReadonlyScope)
	if err != nil {
		handler.Log(robot.Fatal, "Unable to create Google Workspace Events HTTP client: %v", err)
	}

	botInfo := handler.GetBotInfo()
	botName := strings.TrimSpace(botInfo.UserName)
	if botName == "" {
		botName = "gopherbot"
	}

	connector := &googleChatConnector{
		Handler:           handler,
		projectID:         projectID,
		subscriptionID:    subscriptionID,
		subscription:      subscription,
		subscriptionTopic: subscriptionConfig.Topic.String(),
		pubsubClient:      pubsubClient,
		chatClient:        chatClient,
		chatAPI:           chatAPIService,
		workspaceEvents:   newWorkspaceEventsClient(workspaceHTTPClient),
		ambientMessages:   boolValueOrDefault(c.AmbientMessages, false),
		threadResponses:   boolValueOrDefault(c.ThreadResponses, true),
		slashCommand:      normalizeSlashCommand(firstNonEmpty(c.SlashCommand, botName)),
		botName:           botName,
		botUserMap:        normalizeConfiguredUserMap(c.UserMap, handler),
		usersByID:         make(map[string]chatUserRecord),
		usersByName:       make(map[string]chatUserRecord),
		channelsByID:      make(map[string]chatChannelRecord),
		channelIDsByName:  make(map[string]string),
		unmappedUsers:     make(map[string]bool),
		recentMessages:    make(map[string]time.Time),
	}
	connector.rebuildConfiguredUserIndexes()
	connector.createMessage = func(ctx context.Context, req *chatpb.CreateMessageRequest) (*chatpb.Message, error) {
		return connector.chatClient.CreateMessage(ctx, req)
	}
	connector.findDirectMessage = func(ctx context.Context, req *chatpb.FindDirectMessageRequest) (*chatpb.Space, error) {
		return connector.chatClient.FindDirectMessage(ctx, req)
	}

	handler.SetBotID("users/app")
	return robot.InitializedConnector{
		Connector:    connector,
		Capabilities: robot.ConnectorCapabilities{HiddenCommands: connector.slashCommand != ""},
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

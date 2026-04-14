package googlechat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	chat "cloud.google.com/go/chat/apiv1"
	"cloud.google.com/go/chat/apiv1/chatpb"
	"cloud.google.com/go/pubsub"
	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/robot/util"
	chatapi "google.golang.org/api/chat/v1"
)

const (
	sendTimeout             = 20 * time.Second
	dmFindTimeout           = 20 * time.Second
	maxMessageSize          = 32000
	ambientSyncTimeout      = 45 * time.Second
	ambientRenewInterval    = 90 * time.Minute
	recentMessageWindow     = 2 * time.Minute
	ambientSubscriptionLead = 90 * time.Minute
)

type chatpbCreateMessageRequest struct {
	request *chatpb.CreateMessageRequest
}

type chatpbFindDirectMessageRequest struct {
	request *chatpb.FindDirectMessageRequest
}

type createMessageFunc func(context.Context, *chatpb.CreateMessageRequest) (*chatpb.Message, error)
type findDirectMessageFunc func(context.Context, *chatpb.FindDirectMessageRequest) (*chatpb.Space, error)

type chatUserRecord struct {
	ResourceName  string
	DisplayName   string
	Email         string
	CanonicalName string
}

type chatChannelRecord struct {
	ResourceName string
	DisplayName  string
	Direct       bool
}

type googleChatConnector struct {
	robot.Handler

	projectID         string
	subscriptionID    string
	subscriptionTopic string
	botName           string
	slashCommand      string

	ambientMessages bool
	threadResponses bool

	pubsubClient *pubsub.Client
	subscription *pubsub.Subscription
	chatClient   *chat.Client
	chatAPI      *chatapi.Service

	createMessage     createMessageFunc
	findDirectMessage findDirectMessageFunc
	workspaceEvents   *workspaceEventsClient

	mu               sync.RWMutex
	botUserMap       map[string]string
	configuredUsers  map[string]string
	usersByID        map[string]chatUserRecord
	usersByName      map[string]chatUserRecord
	channelsByID     map[string]chatChannelRecord
	channelIDsByName map[string]string
	unmappedUsers    map[string]bool
	recentMessages   map[string]time.Time
}

func (gc *googleChatConnector) rebuildConfiguredUserIndexes() {
	configured := make(map[string]string, len(gc.botUserMap))
	for name, id := range gc.botUserMap {
		configured[id] = name
	}
	gc.configuredUsers = configured
}

func (gc *googleChatConnector) Run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gc.subscription.ReceiveSettings.NumGoroutines = 1
	gc.subscription.ReceiveSettings.MaxOutstandingMessages = 1
	gc.startAmbientLoop(ctx)

	go func() {
		<-stop
		cancel()
	}()

	err := gc.subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		if err := gc.handlePubSubMessage(msg); err != nil {
			gc.Log(robot.Error, "Google Chat: failed to process Pub/Sub message %q: %v", msg.ID, err)
		}
		msg.Ack()
	})
	if err != nil && ctx.Err() == nil {
		gc.Log(robot.Error, "Google Chat Pub/Sub receive loop exited with error: %v", err)
	}
}

func (gc *googleChatConnector) handlePubSubMessage(msg *pubsub.Message) error {
	attrSummary := summarizePubSubAttributes(msg.Attributes)
	if eventType := strings.TrimSpace(msg.Attributes["ce-type"]); eventType != "" {
		gc.Log(robot.Info, "Google Chat Pub/Sub workspace event received: id=%q type=%q attrs=%s bytes=%d", msg.ID, eventType, attrSummary, len(msg.Data))
		return gc.handleWorkspaceEventMessage(msg, eventType)
	}
	var event chatEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return fmt.Errorf("parsing Google Chat event JSON (attrs=%s preview=%q): %w", attrSummary, diagnosticPreviewBytes(msg.Data), err)
	}
	gc.Log(robot.Info, "Google Chat interaction event received: id=%q %s attrs=%s bytes=%d", msg.ID, summarizeChatEvent(&event), attrSummary, len(msg.Data))
	return gc.handleEvent(&event)
}

func (gc *googleChatConnector) handleEvent(event *chatEvent) error {
	if event == nil {
		return nil
	}
	switch strings.ToUpper(strings.TrimSpace(event.Type)) {
	case "MESSAGE", "APP_COMMAND":
		msg, ok := gc.normalizeIncomingMessage(event)
		if !ok {
			return nil
		}
		if !gc.shouldProcessMessage(msg.MessageID) {
			return nil
		}
		gc.IncomingMessage(msg)
		return nil
	case "ADDED_TO_SPACE", "REMOVED_FROM_SPACE":
		return gc.handleSpaceLifecycleEvent(event)
	default:
		gc.Log(robot.Debug, "Ignoring Google Chat event type %q", event.Type)
		return nil
	}
}

func (gc *googleChatConnector) normalizeIncomingMessage(event *chatEvent) (*robot.ConnectorMessage, bool) {
	if event == nil {
		return nil, false
	}
	msg := event.Message
	if msg == nil {
		return nil, false
	}

	user := event.User
	if user == nil {
		user = msg.Sender
	}
	if user == nil {
		gc.Log(robot.Warn, "Ignoring Google Chat event without user information")
		return nil, false
	}

	userID := normalizeUserResource(user.Name)
	if userID == "" {
		gc.Log(robot.Warn, "Ignoring Google Chat event with empty user name: %+v", user)
		return nil, false
	}
	if userID == "users/app" {
		return &robot.ConnectorMessage{
			Protocol:      "googlechat",
			UserID:        userID,
			MessageID:     msg.Name,
			MessageText:   gc.normalizeEventText(msg, true),
			SelfMessage:   true,
			BotMessage:    true,
			MessageObject: event,
			Client:        gc.chatClient,
		}, true
	}

	space := event.Space
	if space == nil {
		space = msg.Space
	}
	direct := space != nil && strings.EqualFold(space.SpaceType, "DIRECT_MESSAGE")
	threadID, threaded := resolveThreadContext(msg, event.Thread)

	channelID := ""
	channelName := ""
	if !direct && space != nil {
		channelID = strings.TrimSpace(space.Name)
		channelName = strings.TrimSpace(space.DisplayName)
	}

	canonicalUser := gc.cacheUser(user)
	if space != nil {
		gc.cacheChannel(space)
	}
	if canonicalUser == "" {
		gc.logUnmappedUser(user)
	}

	botMessage := gc.isExplicitBotMessage(event)
	hidden := gc.isHiddenInteraction(event)
	text := gc.normalizeEventText(msg, botMessage)

	connectorMsg := &robot.ConnectorMessage{
		Protocol:        "googlechat",
		UserID:          userID,
		ChannelID:       channelID,
		MessageID:       strings.TrimSpace(msg.Name),
		ThreadID:        threadID,
		ThreadedMessage: threaded,
		DirectMessage:   direct,
		BotMessage:      botMessage,
		HiddenMessage:   hidden,
		MessageText:     text,
		MessageObject:   event,
		Client:          gc.chatClient,
	}
	if canonicalUser != "" {
		connectorMsg.UserName = canonicalUser
	}
	if channelName != "" {
		connectorMsg.ChannelName = channelName
	}
	return connectorMsg, true
}

func resolveThreadContext(message *chatEventMessage, eventThread *chatEventThread) (string, bool) {
	if message != nil && message.Thread != nil && strings.TrimSpace(message.Thread.Name) != "" {
		return strings.TrimSpace(message.Thread.Name), message.ThreadReply
	}
	if eventThread != nil && strings.TrimSpace(eventThread.Name) != "" {
		return strings.TrimSpace(eventThread.Name), false
	}
	return "", false
}

func (gc *googleChatConnector) isHiddenInteraction(event *chatEvent) bool {
	if event == nil {
		return false
	}
	if event.Message != nil && event.Message.SlashCommand != nil {
		return true
	}
	if event.AppCommandMetadata != nil && strings.EqualFold(event.AppCommandMetadata.AppCommandType, "SLASH_COMMAND") {
		return true
	}
	return false
}

func (gc *googleChatConnector) isExplicitBotMessage(event *chatEvent) bool {
	if event == nil || event.Message == nil {
		return false
	}
	return gc.isHiddenInteraction(event) || strings.EqualFold(strings.TrimSpace(event.Type), "APP_COMMAND")
}

func (gc *googleChatConnector) cacheUser(user *chatEventUser) string {
	if user == nil {
		return ""
	}
	resource := normalizeUserResource(user.Name)
	if resource == "" {
		return ""
	}
	record := chatUserRecord{
		ResourceName: resource,
		DisplayName:  strings.TrimSpace(user.DisplayName),
		Email:        strings.TrimSpace(strings.ToLower(user.Email)),
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()
	if canonical, ok := gc.configuredUsers[resource]; ok {
		record.CanonicalName = canonical
	}
	if existing, ok := gc.usersByID[resource]; ok {
		if record.DisplayName == "" {
			record.DisplayName = existing.DisplayName
		}
		if record.Email == "" {
			record.Email = existing.Email
		}
		if record.CanonicalName == "" {
			record.CanonicalName = existing.CanonicalName
		}
	}
	gc.usersByID[resource] = record
	if record.CanonicalName != "" {
		gc.usersByName[record.CanonicalName] = record
	}
	return record.CanonicalName
}

func (gc *googleChatConnector) cacheChannel(space *chatEventSpace) {
	if space == nil {
		return
	}
	resource := strings.TrimSpace(space.Name)
	if resource == "" {
		return
	}
	record := chatChannelRecord{
		ResourceName: resource,
		DisplayName:  strings.TrimSpace(space.DisplayName),
		Direct:       strings.EqualFold(space.SpaceType, "DIRECT_MESSAGE"),
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.channelsByID[resource] = record
	if record.DisplayName != "" {
		nameKey := strings.ToLower(record.DisplayName)
		if existing, ok := gc.channelIDsByName[nameKey]; !ok || existing == resource {
			gc.channelIDsByName[nameKey] = resource
		}
	}
}

func (gc *googleChatConnector) logUnmappedUser(user *chatEventUser) {
	resource := normalizeUserResource(user.Name)
	if resource == "" {
		return
	}
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if gc.unmappedUsers[resource] {
		return
	}
	gc.unmappedUsers[resource] = true
	gc.Log(robot.Warn, "Google Chat user is not mapped in ProtocolConfig.UserMap: display=%q email=%q id=%q", user.DisplayName, user.Email, resource)
}

func (gc *googleChatConnector) GetProtocolUserAttribute(u, attr string) (string, robot.RetVal) {
	record, ok := gc.lookupUserRecord(u)
	if !ok {
		return "", robot.UserNotFound
	}

	switch strings.ToLower(strings.TrimSpace(attr)) {
	case "name":
		if record.CanonicalName != "" {
			return record.CanonicalName, robot.Ok
		}
		if record.DisplayName != "" {
			return record.DisplayName, robot.Ok
		}
	case "fullname", "realname":
		if record.DisplayName != "" {
			return record.DisplayName, robot.Ok
		}
	case "email":
		if record.Email != "" {
			return record.Email, robot.Ok
		}
	case "internalid":
		if record.ResourceName != "" {
			return record.ResourceName, robot.Ok
		}
	}
	return "", robot.AttributeNotFound
}

func (gc *googleChatConnector) lookupUserRecord(u string) (chatUserRecord, bool) {
	id, isID := util.ExtractID(u)
	id = normalizeUserResource(id)

	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if isID && id != "" {
		record, ok := gc.usersByID[id]
		return record, ok
	}
	key := strings.ToLower(strings.TrimSpace(u))
	if key == "" {
		return chatUserRecord{}, false
	}
	if id, ok := gc.botUserMap[key]; ok {
		record, ok := gc.usersByID[id]
		if ok {
			return record, true
		}
		return chatUserRecord{ResourceName: id, CanonicalName: key}, true
	}
	record, ok := gc.usersByName[key]
	return record, ok
}

func (gc *googleChatConnector) MessageHeard(user, channel string) {}

func (gc *googleChatConnector) DefaultHelp() []string { return nil }

func (gc *googleChatConnector) JoinChannel(c string) robot.RetVal {
	gc.Log(robot.Warn, "JoinChannel is not implemented for Google Chat spaces: %s", c)
	return robot.FailedChannelJoin
}

func (gc *googleChatConnector) SendProtocolChannelThreadMessage(channelname, threadid, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	channelID, ok := gc.resolveChannelID(channelname)
	if !ok {
		gc.Log(robot.Error, "Google Chat channel not found for: %s", channelname)
		return robot.ChannelNotFound
	}
	threadID := gc.resolveThreadForContext(channelID, "", threadid, msgObject)
	return gc.sendMessage(channelID, "", threadID, msg, format, msgObject)
}

func (gc *googleChatConnector) SendProtocolUserChannelThreadMessage(userid, username, channelname, threadid, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	channelID, ok := gc.resolveChannelID(channelname)
	if !ok {
		gc.Log(robot.Error, "Google Chat channel not found for: %s", channelname)
		return robot.ChannelNotFound
	}
	userID, ok := gc.resolveUserID(userid, username)
	if !ok {
		gc.Log(robot.Error, "Google Chat user not found for: %s", username)
		return robot.UserNotFound
	}
	threadID := gc.resolveThreadForContext(channelID, userID, threadid, msgObject)
	return gc.sendMessage(channelID, userID, threadID, msg, format, msgObject)
}

func (gc *googleChatConnector) SendProtocolUserMessage(user, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	userID, ok := gc.resolveUserID(user, user)
	if !ok {
		gc.Log(robot.Error, "Google Chat user not found for DM: %s", user)
		return robot.UserNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), dmFindTimeout)
	defer cancel()
	space, err := gc.findDirectMessage(ctx, &chatpb.FindDirectMessageRequest{Name: userID})
	if err != nil {
		gc.Log(robot.Error, "Google Chat direct message lookup failed for %s: %v", userID, err)
		return robot.FailedMessageSend
	}
	if space == nil || strings.TrimSpace(space.Name) == "" {
		gc.Log(robot.Error, "Google Chat direct message space missing for %s", userID)
		return robot.FailedMessageSend
	}
	return gc.sendMessage(space.Name, "", "", msg, format, msgObject)
}

func (gc *googleChatConnector) resolveChannelID(channel string) (string, bool) {
	if id, ok := util.ExtractID(channel); ok {
		id = strings.TrimSpace(id)
		return id, id != ""
	}
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return "", false
	}
	if strings.HasPrefix(channel, "spaces/") {
		return channel, true
	}
	key := strings.ToLower(channel)
	gc.mu.RLock()
	id, ok := gc.channelIDsByName[key]
	gc.mu.RUnlock()
	if ok {
		return id, true
	}
	return "", false
}

func (gc *googleChatConnector) resolveUserID(uid, username string) (string, bool) {
	if id, ok := util.ExtractID(uid); ok {
		id = normalizeUserResource(id)
		return id, id != ""
	}
	uid = normalizeUserResource(uid)
	if uid != "" {
		return uid, true
	}
	key := strings.ToLower(strings.TrimSpace(username))
	if key == "" {
		return "", false
	}
	gc.mu.RLock()
	id, ok := gc.botUserMap[key]
	gc.mu.RUnlock()
	if ok {
		return id, true
	}
	return "", false
}

func (gc *googleChatConnector) resolveThreadForContext(channelID, userID, explicitThreadID string, msgObject *robot.ConnectorMessage) string {
	threadID := strings.TrimSpace(explicitThreadID)
	if threadID != "" || !gc.threadResponses || msgObject == nil {
		return threadID
	}
	if strings.TrimSpace(msgObject.ThreadID) == "" {
		return ""
	}
	if msgObject.DirectMessage {
		return ""
	}
	if strings.TrimSpace(msgObject.ChannelID) != channelID {
		return ""
	}
	if userID != "" {
		incomingUser := normalizeUserResource(msgObject.UserID)
		if incomingUser != userID {
			return ""
		}
	}
	return strings.TrimSpace(msgObject.ThreadID)
}

func (gc *googleChatConnector) sendMessage(channelID, userID, threadID, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	message, replyOption := gc.buildOutgoingMessage(channelID, userID, threadID, msg, format, msgObject)
	if message == nil {
		gc.Log(robot.Error, "Google Chat: refusing to send empty message")
		return robot.Failed
	}
	if len(message.Text) > maxMessageSize {
		gc.Log(robot.Error, "Google Chat message exceeds maximum size (%d bytes)", maxMessageSize)
		return robot.FailedMessageSend
	}
	req := &chatpb.CreateMessageRequest{
		Parent:             channelID,
		Message:            message,
		MessageReplyOption: replyOption,
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	if _, err := gc.createMessage(ctx, req); err != nil {
		gc.Log(robot.Error, "Google Chat send failed to %s: %v", channelID, err)
		return robot.FailedMessageSend
	}
	return robot.Ok
}

func (gc *googleChatConnector) buildOutgoingMessage(channelID, userID, threadID, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) (*chatpb.Message, chatpb.CreateMessageRequest_MessageReplyOption) {
	body := strings.TrimSpace(gc.renderMessageText(msg, format))
	if body == "" {
		return nil, chatpb.CreateMessageRequest_MESSAGE_REPLY_OPTION_UNSPECIFIED
	}

	hiddenUserID, sameHiddenContext := gc.hiddenReplyContext(channelID, userID, msgObject)
	if userID != "" && !sameHiddenContext {
		body = gc.prefixMention(userID, body)
	}

	message := &chatpb.Message{Text: body}
	replyOption := chatpb.CreateMessageRequest_MESSAGE_REPLY_OPTION_UNSPECIFIED
	if threadID != "" {
		message.Thread = &chatpb.Thread{Name: threadID}
		replyOption = chatpb.CreateMessageRequest_REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD
	}
	if sameHiddenContext {
		message.PrivateMessageViewer = &chatpb.User{Name: hiddenUserID}
	}
	return message, replyOption
}

func (gc *googleChatConnector) hiddenReplyContext(channelID, userID string, msgObject *robot.ConnectorMessage) (string, bool) {
	if msgObject == nil || !msgObject.HiddenMessage {
		return "", false
	}
	if msgObject.DirectMessage {
		return "", false
	}
	if strings.TrimSpace(msgObject.ChannelID) != strings.TrimSpace(channelID) {
		return "", false
	}
	incomingUserID := normalizeUserResource(msgObject.UserID)
	if incomingUserID == "" {
		return "", false
	}
	if userID == "" {
		return incomingUserID, true
	}
	userID = normalizeUserResource(userID)
	if userID != incomingUserID {
		return "", false
	}
	return incomingUserID, true
}

func (gc *googleChatConnector) prefixMention(userID, msg string) string {
	userID = normalizeUserResource(userID)
	if userID == "" {
		return msg
	}
	return "<" + userID + ">: " + msg
}

func normalizeUserResource(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	if strings.HasPrefix(in, "users/") {
		return in
	}
	return "users/" + in
}

func (gc *googleChatConnector) shouldProcessMessage(messageID string) bool {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return true
	}
	now := time.Now()
	cutoff := now.Add(-recentMessageWindow)

	gc.mu.Lock()
	defer gc.mu.Unlock()
	for id, seen := range gc.recentMessages {
		if seen.Before(cutoff) {
			delete(gc.recentMessages, id)
		}
	}
	if _, ok := gc.recentMessages[messageID]; ok {
		return false
	}
	gc.recentMessages[messageID] = now
	return true
}

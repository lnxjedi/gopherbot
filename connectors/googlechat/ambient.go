package googlechat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/lnxjedi/gopherbot/robot"
	chatapi "google.golang.org/api/chat/v1"
)

const (
	workspaceEventMessageCreated      = "google.workspace.chat.message.v1.created"
	workspaceEventMessageBatchCreated = "google.workspace.chat.message.v1.batchCreated"
)

var ambientEventTypes = []string{workspaceEventMessageCreated}

type workspaceMessageCreatedEventData struct {
	Message *chatapi.Message `json:"message,omitempty"`
}

type workspaceMessageBatchCreatedEventData struct {
	Messages []*workspaceMessageCreatedEventData `json:"messages,omitempty"`
}

func (gc *googleChatConnector) startAmbientLoop(ctx context.Context) {
	if gc == nil || !gc.ambientMessages || gc.chatAPI == nil || gc.workspaceEvents == nil {
		return
	}
	go func() {
		gc.runAmbientSync(ctx)
		ticker := time.NewTicker(ambientRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				gc.runAmbientSync(ctx)
			}
		}
	}()
}

func (gc *googleChatConnector) runAmbientSync(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, ambientSyncTimeout)
	defer cancel()
	if err := gc.syncAmbientSubscriptions(ctx); err != nil && ctx.Err() == nil {
		gc.Log(robot.Warn, "Google Chat ambient subscription sync failed: %v", err)
	}
}

func (gc *googleChatConnector) syncAmbientSubscriptions(ctx context.Context) error {
	spaces, err := gc.listAmbientCandidateSpaces(ctx)
	if err != nil {
		return err
	}
	for _, space := range spaces {
		if err := gc.ensureAmbientSubscriptionForSpace(ctx, space); err != nil {
			gc.Log(robot.Warn, "Google Chat ambient subscription ensure failed for %s: %v", space.Name, err)
		}
	}
	return nil
}

func (gc *googleChatConnector) listAmbientCandidateSpaces(ctx context.Context) ([]*chatapi.Space, error) {
	if gc == nil || gc.chatAPI == nil {
		return nil, fmt.Errorf("google chat REST service is not configured")
	}
	var spaces []*chatapi.Space
	call := gc.chatAPI.Spaces.List().PageSize(1000).Filter(`space_type = "SPACE" OR space_type = "GROUP_CHAT"`)
	pageToken := ""
	for {
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, space := range resp.Spaces {
			if isAmbientCandidateSpace(space) {
				spaces = append(spaces, space)
				gc.cacheAPIChannel(space)
			}
		}
		if strings.TrimSpace(resp.NextPageToken) == "" {
			return spaces, nil
		}
		pageToken = resp.NextPageToken
		call = gc.chatAPI.Spaces.List().PageSize(1000).Filter(`space_type = "SPACE" OR space_type = "GROUP_CHAT"`)
	}
}

func isAmbientCandidateSpace(space *chatapi.Space) bool {
	if space == nil {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(space.SpaceType)) {
	case "SPACE", "GROUP_CHAT":
		return strings.TrimSpace(space.Name) != ""
	default:
		return false
	}
}

func (gc *googleChatConnector) ensureAmbientSubscriptionForSpace(ctx context.Context, space *chatapi.Space) error {
	if gc == nil || gc.workspaceEvents == nil || space == nil {
		return nil
	}
	target := targetResourceForSpace(space.Name)
	existing, err := gc.listAmbientSubscriptionsForTarget(ctx, target)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		_, err := gc.workspaceEvents.createSubscription(ctx, &workspaceSubscription{
			TargetResource: target,
			EventTypes:     append([]string(nil), ambientEventTypes...),
			NotificationEndpoint: &workspaceNotificationEndpoint{
				PubsubTopic: gc.subscriptionTopic,
			},
			PayloadOptions: &workspacePayloadOptions{
				IncludeResource: true,
			},
			Ttl: durationString(4 * time.Hour),
		})
		if err == nil {
			gc.Log(robot.Info, "Google Chat ambient subscription created for %s", space.Name)
		}
		return err
	}
	subscription := existing[0]
	if subscription.NotificationEndpoint == nil || strings.TrimSpace(subscription.NotificationEndpoint.PubsubTopic) != strings.TrimSpace(gc.subscriptionTopic) {
		return fmt.Errorf("existing workspace subscription %s targets %q instead of %q", subscription.Name, topicName(subscription.NotificationEndpoint), gc.subscriptionTopic)
	}
	if strings.EqualFold(subscription.State, "SUSPENDED") {
		if _, err := gc.workspaceEvents.reactivateSubscription(ctx, subscription.Name); err != nil {
			return err
		}
	}
	needsPatch := !subscriptionIncludesEvent(subscription, workspaceEventMessageCreated) || subscriptionExpiresSoon(subscription)
	if !needsPatch {
		return nil
	}
	updated := workspaceSubscription{
		Name:       subscription.Name,
		Etag:       subscription.Etag,
		EventTypes: mergeEventTypes(subscription.EventTypes, ambientEventTypes),
		Ttl:        durationString(4 * time.Hour),
	}
	_, err = gc.workspaceEvents.updateSubscription(ctx, &updated, "ttl,eventTypes")
	return err
}

func (gc *googleChatConnector) deleteAmbientSubscriptionForSpace(ctx context.Context, spaceName string) error {
	if gc == nil || gc.workspaceEvents == nil || strings.TrimSpace(spaceName) == "" {
		return nil
	}
	target := targetResourceForSpace(spaceName)
	existing, err := gc.listAmbientSubscriptionsForTarget(ctx, target)
	if err != nil {
		return err
	}
	for _, subscription := range existing {
		if err := gc.workspaceEvents.deleteSubscription(ctx, subscription.Name, true); err != nil {
			return err
		}
		gc.Log(robot.Info, "Google Chat ambient subscription deleted for %s", spaceName)
	}
	return nil
}

func (gc *googleChatConnector) listAmbientSubscriptionsForTarget(ctx context.Context, target string) ([]workspaceSubscription, error) {
	if gc == nil || gc.workspaceEvents == nil {
		return nil, fmt.Errorf("workspace events client is not configured")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("workspace events target resource is required")
	}
	listed, err := gc.workspaceEvents.listSubscriptions(ctx, fmt.Sprintf(`event_types:%q`, workspaceEventMessageCreated))
	if err != nil {
		return nil, err
	}
	matching := make([]workspaceSubscription, 0, len(listed))
	for _, subscription := range listed {
		if strings.TrimSpace(subscription.TargetResource) == target {
			matching = append(matching, subscription)
		}
	}
	return matching, nil
}

func (gc *googleChatConnector) handleSpaceLifecycleEvent(event *chatEvent) error {
	spaceName := ""
	if event != nil && event.Space != nil {
		spaceName = strings.TrimSpace(event.Space.Name)
		gc.cacheChannel(event.Space)
	}
	if event == nil {
		return nil
	}
	gc.Log(robot.Info, "Google Chat event %s for %s", event.Type, spaceName)
	if !gc.ambientMessages || strings.TrimSpace(spaceName) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), ambientSyncTimeout)
	defer cancel()
	switch strings.ToUpper(strings.TrimSpace(event.Type)) {
	case "ADDED_TO_SPACE":
		return gc.ensureAmbientSubscriptionForSpace(ctx, &chatapi.Space{Name: spaceName, SpaceType: "SPACE", DisplayName: channelName(event.Space)})
	case "REMOVED_FROM_SPACE":
		return gc.deleteAmbientSubscriptionForSpace(ctx, spaceName)
	default:
		return nil
	}
}

func (gc *googleChatConnector) handleWorkspaceEventMessage(msg *pubsub.Message, eventType string) error {
	switch strings.TrimSpace(eventType) {
	case workspaceEventMessageCreated:
		return gc.handleWorkspaceMessageCreated(msg.Data)
	case workspaceEventMessageBatchCreated:
		return gc.handleWorkspaceMessageBatchCreated(msg.Data)
	default:
		if strings.HasPrefix(strings.TrimSpace(eventType), "google.workspace.events.subscription.v1.") {
			return gc.handleWorkspaceSubscriptionLifecycle(msg.Data, eventType)
		}
		gc.Log(robot.Debug, "Ignoring Google Workspace event type %q", eventType)
		return nil
	}
}

func (gc *googleChatConnector) handleWorkspaceMessageCreated(data []byte) error {
	var payload workspaceMessageCreatedEventData
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parsing workspace message event payload: %w", err)
	}
	connectorMsg, ok := gc.normalizeAmbientMessage(payload.Message)
	if !ok {
		return nil
	}
	if !gc.shouldProcessMessage(connectorMsg.MessageID) {
		return nil
	}
	gc.IncomingMessage(connectorMsg)
	return nil
}

func (gc *googleChatConnector) handleWorkspaceMessageBatchCreated(data []byte) error {
	var payload workspaceMessageBatchCreatedEventData
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parsing workspace batch event payload: %w", err)
	}
	for _, item := range payload.Messages {
		if item == nil {
			continue
		}
		connectorMsg, ok := gc.normalizeAmbientMessage(item.Message)
		if !ok || !gc.shouldProcessMessage(connectorMsg.MessageID) {
			continue
		}
		gc.IncomingMessage(connectorMsg)
	}
	return nil
}

func (gc *googleChatConnector) handleWorkspaceSubscriptionLifecycle(data []byte, eventType string) error {
	var payload struct {
		Subscription *workspaceSubscription `json:"subscription,omitempty"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parsing workspace subscription lifecycle payload: %w", err)
	}
	if payload.Subscription == nil || strings.TrimSpace(payload.Subscription.Name) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), ambientSyncTimeout)
	defer cancel()
	switch {
	case strings.HasSuffix(eventType, ".expirationWarning"):
		updated := workspaceSubscription{
			Name: payload.Subscription.Name,
			Etag: payload.Subscription.Etag,
			Ttl:  durationString(4 * time.Hour),
		}
		_, err := gc.workspaceEvents.updateSubscription(ctx, &updated, "ttl")
		return err
	case strings.HasSuffix(eventType, ".expired"), strings.HasSuffix(eventType, ".suspended"):
		_, err := gc.workspaceEvents.reactivateSubscription(ctx, payload.Subscription.Name)
		return err
	default:
		return nil
	}
}

func (gc *googleChatConnector) normalizeAmbientMessage(message *chatapi.Message) (*robot.ConnectorMessage, bool) {
	if gc == nil || message == nil {
		return nil, false
	}
	sender := message.Sender
	if sender == nil || strings.TrimSpace(sender.Name) == "" {
		gc.Log(robot.Warn, "Ignoring ambient Google Chat message without sender")
		return nil, false
	}
	userID := normalizeUserResource(sender.Name)
	if userID == "users/app" {
		return &robot.ConnectorMessage{
			Protocol:      "googlechat",
			UserID:        userID,
			MessageID:     strings.TrimSpace(message.Name),
			MessageText:   ambientMessageText(message),
			SelfMessage:   true,
			BotMessage:    true,
			MessageObject: message,
			Client:        gc.chatClient,
		}, true
	}

	canonicalUser := gc.cacheAPIUser(sender)
	if message.Space != nil {
		gc.cacheAPIChannel(message.Space)
	}
	if canonicalUser == "" {
		gc.logUnmappedAPIUser(sender)
	}

	channelID := ""
	channelName := ""
	direct := false
	if message.Space != nil {
		if !strings.EqualFold(message.Space.SpaceType, "DIRECT_MESSAGE") {
			channelID = strings.TrimSpace(message.Space.Name)
			channelName = strings.TrimSpace(message.Space.DisplayName)
		} else {
			direct = true
		}
	}
	threadID := ""
	threaded := false
	if message.Thread != nil {
		threadID = strings.TrimSpace(message.Thread.Name)
		threaded = message.ThreadReply
	}
	connectorMsg := &robot.ConnectorMessage{
		Protocol:        "googlechat",
		UserID:          userID,
		ChannelID:       channelID,
		MessageID:       strings.TrimSpace(message.Name),
		ThreadID:        threadID,
		ThreadedMessage: threaded,
		DirectMessage:   direct,
		BotMessage:      true,
		MessageText:     ambientMessageText(message),
		MessageObject:   message,
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

func ambientMessageText(message *chatapi.Message) string {
	if message == nil {
		return ""
	}
	if text := strings.TrimSpace(message.ArgumentText); text != "" {
		return text
	}
	return strings.TrimSpace(message.Text)
}

func (gc *googleChatConnector) cacheAPIUser(user *chatapi.User) string {
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
		if record.CanonicalName == "" {
			record.CanonicalName = existing.CanonicalName
		}
		if record.Email == "" {
			record.Email = existing.Email
		}
	}
	gc.usersByID[resource] = record
	if record.CanonicalName != "" {
		gc.usersByName[record.CanonicalName] = record
	}
	return record.CanonicalName
}

func (gc *googleChatConnector) logUnmappedAPIUser(user *chatapi.User) {
	if user == nil {
		return
	}
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
	gc.Log(robot.Warn, "Google Chat user is not mapped in ProtocolConfig.UserMap: display=%q id=%q", user.DisplayName, resource)
}

func (gc *googleChatConnector) cacheAPIChannel(space *chatapi.Space) {
	if space == nil {
		return
	}
	gc.cacheChannel(&chatEventSpace{
		Name:        space.Name,
		DisplayName: space.DisplayName,
		SpaceType:   space.SpaceType,
	})
}

func targetResourceForSpace(spaceName string) string {
	spaceName = strings.TrimSpace(spaceName)
	if spaceName == "" {
		return ""
	}
	if strings.HasPrefix(spaceName, "//chat.googleapis.com/") {
		return spaceName
	}
	return "//chat.googleapis.com/" + strings.TrimPrefix(spaceName, "/")
}

func mergeEventTypes(existing, desired []string) []string {
	seen := make(map[string]bool, len(existing)+len(desired))
	out := make([]string, 0, len(existing)+len(desired))
	for _, eventType := range append(append([]string(nil), existing...), desired...) {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" || seen[eventType] {
			continue
		}
		seen[eventType] = true
		out = append(out, eventType)
	}
	return out
}

func subscriptionIncludesEvent(subscription workspaceSubscription, eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	for _, candidate := range subscription.EventTypes {
		if strings.TrimSpace(candidate) == eventType {
			return true
		}
	}
	return false
}

func subscriptionExpiresSoon(subscription workspaceSubscription) bool {
	if strings.TrimSpace(subscription.ExpireTime) == "" {
		return true
	}
	expireTime, err := time.Parse(time.RFC3339, subscription.ExpireTime)
	if err != nil {
		return true
	}
	return time.Until(expireTime) < ambientSubscriptionLead
}

func durationString(d time.Duration) string {
	seconds := int64(d / time.Second)
	if seconds <= 0 {
		seconds = 1
	}
	return fmt.Sprintf("%ds", seconds)
}

func topicName(endpoint *workspaceNotificationEndpoint) string {
	if endpoint == nil {
		return ""
	}
	return endpoint.PubsubTopic
}

func channelName(space *chatEventSpace) string {
	if space == nil {
		return ""
	}
	return strings.TrimSpace(space.DisplayName)
}

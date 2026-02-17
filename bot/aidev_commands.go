package bot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const defaultAIDevCommandBufferSize = 256

type aidevCommandEvent struct {
	Cursor    uint64 `json:"cursor"`
	Timestamp string `json:"timestamp"`
	Protocol  string `json:"protocol"`
	UserName  string `json:"user_name"`
	UserID    string `json:"user_id,omitempty"`
	Channel   string `json:"channel"`
	ThreadID  string `json:"thread_id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	Text      string `json:"text"`
	Command   string `json:"command"`
}

type aidevCommandQuery struct {
	AfterCursor uint64
	All         bool
	Limit       int
	TimeoutMS   int
}

type aidevCommandBatch struct {
	User       string              `json:"user"`
	Commands   []aidevCommandEvent `json:"commands"`
	NextCursor uint64              `json:"next_cursor"`
	Latest     uint64              `json:"latest"`
	TimedOut   bool                `json:"timed_out"`
	HasMore    bool                `json:"has_more"`
}

type aidevCommandThreadRef struct {
	set      bool
	protocol string
	user     string
	channel  string
	threadID string
}

var aidevCommands = struct {
	sync.RWMutex
	enabled bool
	user    string
	prefix  byte
	consume bool
	active  aidevCommandThreadRef
	buffer  []aidevCommandEvent
	bufIdx  int
	filled  bool
	nextSeq uint64
	waiters map[chan struct{}]struct{}
}{
	prefix:  '>',
	waiters: map[chan struct{}]struct{}{},
}

func configureAIDevCommandConduitFromEnv() {
	user, _ := lookupEnv("GOPHER_AIDEV_COMMAND_USER")
	user = normalizeAIDevCommandUser(user)
	prefix := defaultAIDevCommandPrefix()

	aidevCommands.Lock()
	defer aidevCommands.Unlock()

	aidevCommands.enabled = user != ""
	aidevCommands.user = user
	aidevCommands.prefix = prefix
	aidevCommands.consume = defaultAIDevCommandConsume()
	aidevCommands.active = aidevCommandThreadRef{}
	aidevCommands.buffer = make([]aidevCommandEvent, defaultAIDevCommandBufferSize)
	aidevCommands.bufIdx = 0
	aidevCommands.filled = false
	aidevCommands.nextSeq = 0
	aidevCommands.waiters = map[chan struct{}]struct{}{}

	if aidevCommands.enabled {
		Log(robot.Info, "AI-dev command conduit enabled for user '%s' with prefix '%c' (consume=%t)", user, prefix, aidevCommands.consume)
	}
}

func normalizeAIDevCommandUser(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
}

func defaultAIDevCommandPrefix() byte {
	prefix, ok := lookupEnv("GOPHER_AIDEV_COMMAND_PREFIX")
	if !ok {
		return '>'
	}
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return '>'
	}
	return trimmed[0]
}

func defaultAIDevCommandConsume() bool {
	consumeRaw, ok := lookupEnv("GOPHER_AIDEV_COMMAND_CONSUME")
	if !ok {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(consumeRaw)) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func aidevCommandConduitInfo() (enabled bool, user string, prefix string, consume bool) {
	aidevCommands.RLock()
	defer aidevCommands.RUnlock()
	if !aidevCommands.enabled {
		return false, "", "", false
	}
	return true, aidevCommands.user, string([]byte{aidevCommands.prefix}), aidevCommands.consume
}

func captureAIDevCommandIfMatched(resolvedUser string, inc *robot.ConnectorMessage, isCommand bool, parsedMessage string) bool {
	if inc == nil {
		return false
	}
	msg := strings.TrimSpace(parsedMessage)
	if msg == "" {
		return false
	}
	conduitUser := normalizeAIDevCommandUser(resolvedUser)
	protocol := normalizeProtocolName(inc.Protocol)
	channel := strings.TrimSpace(inc.ChannelName)
	threadID := strings.TrimSpace(inc.ThreadID)
	var closedThreadNotice *aidevCommandThreadRef
	var newThreadNotice *aidevCommandThreadRef

	aidevCommands.Lock()
	if !aidevCommands.enabled {
		aidevCommands.Unlock()
		return false
	}
	if conduitUser != aidevCommands.user {
		aidevCommands.Unlock()
		return false
	}

	prefixedCommand := isCommand && msg[0] == aidevCommands.prefix
	threadFollowup := aidevCommands.active.set &&
		aidevCommands.active.protocol == protocol &&
		aidevCommands.active.user == conduitUser &&
		aidevCommands.active.channel == channel &&
		aidevCommands.active.threadID == threadID
	if !prefixedCommand && !threadFollowup {
		aidevCommands.Unlock()
		return false
	}

	command := msg
	if prefixedCommand {
		command = strings.TrimSpace(msg[1:])
		if !inc.DirectMessage {
			newActive := aidevCommandThreadRef{
				set:      true,
				protocol: protocol,
				user:     conduitUser,
				channel:  channel,
				threadID: threadID,
			}
			if aidevCommands.active.set &&
				(aidevCommands.active.protocol != newActive.protocol ||
					aidevCommands.active.user != newActive.user ||
					aidevCommands.active.channel != newActive.channel ||
					aidevCommands.active.threadID != newActive.threadID) {
				prev := aidevCommands.active
				closedThreadNotice = &prev
				newThreadNotice = &newActive
			}
			aidevCommands.active = newActive
		}
	}
	aidevCommands.nextSeq++
	event := aidevCommandEvent{
		Cursor:    aidevCommands.nextSeq,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Protocol:  protocol,
		UserName:  conduitUser,
		UserID:    strings.TrimSpace(inc.UserID),
		Channel:   channel,
		ThreadID:  threadID,
		MessageID: strings.TrimSpace(inc.MessageID),
		Text:      msg,
		Command:   command,
	}
	if event.Protocol == "" {
		event.Protocol = "unknown"
	}

	aidevCommands.buffer[aidevCommands.bufIdx] = event
	aidevCommands.bufIdx = (aidevCommands.bufIdx + 1) % len(aidevCommands.buffer)
	if aidevCommands.bufIdx == 0 {
		aidevCommands.filled = true
	}
	for ch := range aidevCommands.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	aidevCommands.Unlock()

	if closedThreadNotice != nil && newThreadNotice != nil {
		notifyClosedAIDevCommandThread(*closedThreadNotice, *newThreadNotice)
	}
	return true
}

func notifyClosedAIDevCommandThread(oldThread, newThread aidevCommandThreadRef) {
	conn := getConnectorForProtocol(oldThread.protocol)
	if conn == nil {
		return
	}
	msgObject := &robot.ConnectorMessage{Protocol: oldThread.protocol}
	text := fmt.Sprintf("(thread closed, now active in new thread from %s/%s)", newThread.protocol, newThread.channel)
	ret := conn.SendProtocolChannelThreadMessage(oldThread.channel, oldThread.threadID, text, robot.Raw, msgObject)
	if ret != robot.Ok {
		Log(robot.Warn, "Unable to send AI-dev thread closed notice to '%s/%s' thread '%s': %s", oldThread.protocol, oldThread.channel, oldThread.threadID, ret.String())
	}
}

func aidevCommandConduitConsumes() bool {
	aidevCommands.RLock()
	defer aidevCommands.RUnlock()
	return aidevCommands.enabled && aidevCommands.consume
}

func getAIDevCommands(query aidevCommandQuery) aidevCommandBatch {
	limit := query.Limit
	if limit <= 0 {
		limit = 128
	}
	if limit > 512 {
		limit = 512
	}
	waitForever := query.TimeoutMS < 0
	timeout := query.TimeoutMS
	if timeout < 0 {
		timeout = 1400
	}
	if timeout == 0 {
		timeout = 1400
	}

	collect := func() aidevCommandBatch {
		return collectAIDevCommands(query.AfterCursor, query.All, limit)
	}

	batch := collect()
	if query.All || len(batch.Commands) > 0 || query.TimeoutMS == 0 {
		return batch
	}

	waiter := registerAIDevCommandWaiter()
	defer unregisterAIDevCommandWaiter(waiter)

	batch = collect()
	if len(batch.Commands) > 0 {
		return batch
	}

	if waitForever {
		for {
			<-waiter
			batch = collect()
			if len(batch.Commands) > 0 {
				return batch
			}
		}
	}

	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-waiter:
		batch = collect()
	case <-timer.C:
		batch = collect()
		batch.TimedOut = len(batch.Commands) == 0
	}
	return batch
}

func collectAIDevCommands(afterCursor uint64, all bool, limit int) aidevCommandBatch {
	msgs, latest, conduitUser := snapshotAIDevCommands()
	filtered := make([]aidevCommandEvent, 0, len(msgs))
	for _, msg := range msgs {
		if !all && msg.Cursor <= afterCursor {
			continue
		}
		filtered = append(filtered, msg)
	}
	hasMore := false
	if len(filtered) > limit {
		hasMore = true
		filtered = filtered[:limit]
	}

	nextCursor := afterCursor
	if len(filtered) > 0 {
		nextCursor = filtered[len(filtered)-1].Cursor
	}
	return aidevCommandBatch{
		User:       conduitUser,
		Commands:   filtered,
		NextCursor: nextCursor,
		Latest:     latest,
		HasMore:    hasMore,
	}
}

func snapshotAIDevCommands() ([]aidevCommandEvent, uint64, string) {
	aidevCommands.RLock()
	defer aidevCommands.RUnlock()

	if !aidevCommands.enabled || len(aidevCommands.buffer) == 0 {
		return nil, aidevCommands.nextSeq, aidevCommands.user
	}
	out := make([]aidevCommandEvent, 0, len(aidevCommands.buffer))
	if aidevCommands.filled {
		out = append(out, aidevCommands.buffer[aidevCommands.bufIdx:]...)
	}
	out = append(out, aidevCommands.buffer[:aidevCommands.bufIdx]...)
	return out, aidevCommands.nextSeq, aidevCommands.user
}

func registerAIDevCommandWaiter() chan struct{} {
	ch := make(chan struct{}, 1)
	aidevCommands.Lock()
	if aidevCommands.waiters == nil {
		aidevCommands.waiters = map[chan struct{}]struct{}{}
	}
	aidevCommands.waiters[ch] = struct{}{}
	aidevCommands.Unlock()
	return ch
}

func unregisterAIDevCommandWaiter(ch chan struct{}) {
	aidevCommands.Lock()
	delete(aidevCommands.waiters, ch)
	aidevCommands.Unlock()
	close(ch)
}

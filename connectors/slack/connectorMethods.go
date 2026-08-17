package slack

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/robot/util"
	"github.com/slack-go/slack"
)

// Message send delay; slack has problems with scrolling if messages fly out
// too fast.
const msgDelay = 1 * time.Second

const (
	slackSendTimeout  = 15 * time.Second
	slackSendAttempts = 3
)

// Bursting constants; we allow the robot to send a maximum of `burstMessages`
// in a `burstWindow` window; above the burst limit we slow messages down to
// 1 / sec.
const burstMessages = 14            // maximum burst
const burstWindow = 4 * time.Second // window in which to allow the burst
const coolDown = 21 * time.Second   // cooldown time after bursting

// GetProtocolUserAttribute returns a string attribute or "" if slack doesn't
// have that information
func (s *slackConnector) GetProtocolUserAttribute(u, attr string) (value string, ret robot.RetVal) {
	var userID string
	var ok bool
	var user *slack.User
	if userID, ok = util.ExtractID(u); !ok {
		userID, ok = s.userID(u, false)
	}
	if ok {
		s.RLock()
		user, ok = s.userIDInfo[userID]
		s.RUnlock()
	}
	if !ok {
		return "", robot.UserNotFound
	}
	switch attr {
	case "email":
		return user.Profile.Email, robot.Ok
	case "internalid":
		return user.ID, robot.Ok
	case "realname", "fullname", "real name", "full name":
		return user.RealName, robot.Ok
	case "firstname", "first name":
		return user.Profile.FirstName, robot.Ok
	case "lastname", "last name":
		return user.Profile.LastName, robot.Ok
	case "phone":
		return user.Profile.Phone, robot.Ok
	// that's all the attributes we can currently get from slack
	default:
		return "", robot.AttributeNotFound
	}
}

type sendMessage struct {
	message, markdownText, user, channel, thread string
	blocks                                       []slack.Block
	format                                       robot.MessageFormat
	mtype                                        msgType
}

type sendRequest struct {
	ctx      context.Context
	messages []*sendMessage
	result   chan robot.RetVal
}

const sendQueueSize = 256

// Send a typing notifier letting the user know the message has been heard by
// the robot.
func (s *slackConnector) MessageHeard(user, channel string) {
	var chanID string
	var ok bool
	if chanID, ok = util.ExtractID(channel); ok {
		if s.socketMode {
			// TODO someday - socketmode doesn't support typing notifications :-(
			// Two problems with what's below:
			// - doesn't show up in thread
			// - never disappears
			// if userID, ok := s.ExtractID(user); ok {
			// 	opts := []slack.MsgOption{
			// 		slack.MsgOptionText(":speech_balloon:", false),
			// 		slack.MsgOptionAsUser(true),
			// 		slack.MsgOptionDisableLinkUnfurl(),
			// 	}
			// 	s.api.PostEphemeral(chanID, userID, opts...)
			// }
		} else {
			s.RLock()
			conn := s.conn
			s.RUnlock()
			if conn == nil {
				s.Log(robot.Warn, "Skipping typing indicator before Slack RTM connection is ready")
				return
			}
			conn.SendMessage(conn.NewTypingMessage(chanID))
		}
	}
}

func (s *slackConnector) queueSendRequest(req *sendRequest) robot.RetVal {
	s.RLock()
	q := s.sendQueue
	stop := s.sendStop
	running := s.running
	s.RUnlock()
	if !running || q == nil || stop == nil {
		s.Log(robot.Warn, "Dropping Slack outbound message while connector is stopped")
		return robot.FailedMessageSend
	}
	select {
	case <-req.ctx.Done():
		return robot.FailedMessageSend
	case <-stop:
		return robot.FailedMessageSend
	case q <- req:
	default:
		channel := ""
		if len(req.messages) > 0 {
			channel = req.messages[0].channel
		}
		s.Log(robot.Warn, "Slack outbound queue is full; dropping message for channel '%s'", channel)
		return robot.FailedMessageSend
	}
	select {
	case ret := <-req.result:
		return ret
	case <-req.ctx.Done():
		select {
		case ret := <-req.result:
			return ret
		default:
		}
		s.Log(robot.Error, "Slack outbound message timed out before delivery completed")
		return robot.FailedMessageSend
	case <-stop:
		select {
		case ret := <-req.result:
			return ret
		default:
		}
		return robot.FailedMessageSend
	}
}

func slackErrorRetryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	type retryableError interface {
		Retryable() bool
	}
	var retryable retryableError
	if errors.As(err, &retryable) {
		return retryable.Retryable()
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func slackRetryDelay(err error, attempt int) time.Duration {
	var rateLimited *slack.RateLimitedError
	if errors.As(err, &rateLimited) && rateLimited.RetryAfter > 0 {
		return rateLimited.RetryAfter
	}
	return time.Second << attempt
}

func (s *slackConnector) sleepRetry(ctx context.Context, delay time.Duration) error {
	if s.retrySleep != nil {
		return s.retrySleep(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func sleepSlackBurst(stop <-chan struct{}, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-stop:
		return false
	}
}

func (s *slackConnector) postSlackMessage(ctx context.Context, send *sendMessage) robot.RetVal {
	opts := []slack.MsgOption{
		slack.MsgOptionAsUser(true),
		slack.MsgOptionDisableLinkUnfurl(),
	}
	if send.markdownText != "" {
		opts = append(opts, slack.MsgOptionMarkdownText(send.markdownText))
	} else {
		opts = append(opts, slack.MsgOptionText(send.message, false))
	}
	if len(send.blocks) > 0 {
		opts = append(opts, slack.MsgOptionBlocks(send.blocks...))
	}
	// Slash commands are hidden, so we respond with an ephemeral message.
	if len(send.user) > 0 && send.mtype == msgSlashCmd {
		opts = append(opts, slack.MsgOptionPostEphemeral(send.user))
	}
	if len(send.thread) > 0 {
		opts = append(opts, slack.MsgOptionTS(send.thread))
	}
	if send.format == robot.Variable || len(send.blocks) > 0 {
		opts = append(opts, slack.MsgOptionDisableMarkdown(), slack.MsgOptionParse(false))
	}

	postMessage := s.postMessage
	if postMessage == nil && s.api != nil {
		postMessage = s.api.PostMessageContext
	}
	if postMessage == nil {
		s.Log(robot.Error, "Slack send failed: Web API client is unavailable")
		return robot.FailedMessageSend
	}

	s.Log(robot.Trace, "Bot message in slack send loop for channel %s, size: %d", send.channel, len(send.message))
	var lastErr error
	for attempt := 0; attempt < slackSendAttempts; attempt++ {
		_, _, err := postMessage(ctx, send.channel, opts...)
		if err == nil {
			return robot.Ok
		}
		lastErr = err
		if attempt == slackSendAttempts-1 || !slackErrorRetryable(err) {
			break
		}
		delay := slackRetryDelay(err, attempt)
		s.Log(robot.Warn, "Sending Slack message to channel '%s' failed (attempt %d/%d); retrying in %v: %v", send.channel, attempt+1, slackSendAttempts, delay, err)
		if err := s.sleepRetry(ctx, delay); err != nil {
			lastErr = err
			break
		}
	}
	s.Log(robot.Error, "Failed sending Slack message to channel '%s' after delivery attempts: %v", send.channel, lastErr)
	return robot.FailedMessageSend
}

func completeSlackSend(req *sendRequest, ret robot.RetVal) {
	select {
	case req.result <- ret:
	default:
	}
}

func (s *slackConnector) startSendLoop(q <-chan *sendRequest, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	// See bursting constants above.
	var burstTime time.Time
	mtimes := make([]time.Time, burstMessages)
	current := 0 // index of the current message send time
	for {
		var req *sendRequest
		select {
		case <-stop:
			return
		case req = <-q:
		}
		ret := robot.Ok
		for i, send := range req.messages {
			if req.ctx.Err() != nil {
				ret = robot.FailedMessageSend
				break
			}
			msgTime := time.Now()
			mtimes[current] = msgTime
			windowStartMsg := current + 1
			if windowStartMsg == (burstMessages - 1) {
				windowStartMsg = 0
			}
			current++
			if current == (burstMessages - 1) {
				current = 0
			}
			if ret = s.postSlackMessage(req.ctx, send); ret != robot.Ok {
				break
			}

			lastMessage := i == len(req.messages)-1
			if lastMessage {
				completeSlackSend(req, robot.Ok)
			}
			timeSinceBurst := msgTime.Sub(burstTime)
			if msgTime.Sub(mtimes[windowStartMsg]) < burstWindow || timeSinceBurst < coolDown {
				if timeSinceBurst > coolDown {
					burstTime = msgTime
				}
				s.Log(robot.Debug, "Slack burst limit exceeded, delaying next message by %v", msgDelay)
				if !sleepSlackBurst(stop, msgDelay) {
					return
				}
			}
		}
		if ret != robot.Ok {
			completeSlackSend(req, ret)
		}
	}
}

func (s *slackConnector) sendMessages(msgs []slackOutgoingPayload, userID, chanID, threadID string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	mtype := getMsgType(msgObject)
	if mtype == msgSlashCmd { // could also check msgObject.Hidden
		slashCmd := msgObject.MessageObject.(*slack.SlashCommand)
		if (userID == "" || userID == slashCmd.UserID) && chanID == slashCmd.ChannelID {
			// Make sure a blank userID is replaced by the original userID
			userID = slashCmd.UserID
			threadID = ""
		} else {
			// If the user or channel has changed, don't send ephemeral/hidden reply
			mtype = msgNone
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), slackSendTimeout)
	defer cancel()
	req := &sendRequest{
		ctx:      ctx,
		messages: make([]*sendMessage, 0, len(msgs)),
		result:   make(chan robot.RetVal, 1),
	}
	for _, msg := range msgs {
		req.messages = append(req.messages, &sendMessage{
			message:      msg.text,
			markdownText: msg.markdown,
			blocks:       msg.blocks,
			user:         userID,
			channel:      chanID,
			thread:       threadID,
			format:       f,
			mtype:        mtype,
		})
	}
	if len(req.messages) == 0 {
		return robot.FailedMessageSend
	}
	return s.queueSendRequest(req)
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolChannelThreadMessage(ch, thr, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) (ret robot.RetVal) {
	msgs := s.slackifyMessage("", "", "", msg, f, msgObject)
	if chanID, ok := util.ExtractID(ch); ok {
		return s.sendMessages(msgs, "", chanID, thr, f, msgObject)
	}
	if chanID, ok := s.chanID(ch); ok {
		return s.sendMessages(msgs, "", chanID, thr, f, msgObject)
	}
	s.Log(robot.Error, "Slack channel ID not found for: %s", ch)
	return robot.ChannelNotFound
}

// SendProtocolUserChannelMessage sends a message to a user in a channel
func (s *slackConnector) SendProtocolUserChannelThreadMessage(uid, u, ch, thr, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) (ret robot.RetVal) {
	var userID, chanID string
	var ok bool
	if chanID, ok = util.ExtractID(ch); !ok {
		chanID, ok = s.chanID(ch)
	}
	if !ok {
		s.Log(robot.Error, "Slack channel ID not found for: %s", ch)
		return robot.ChannelNotFound
	}
	userID, ok = util.ExtractID(uid)
	if !ok {
		userID, ok = s.userID(u, false)
	}
	if !ok {
		s.Log(robot.Error, "Slack user ID not found for: %s", uid)
		return robot.UserNotFound
	}
	if strings.TrimSpace(u) == "" {
		if readable, found := s.userName(userID); found {
			u = readable
		} else {
			u = userID
		}
	}
	// Block-backed sends use a readable literal prefix instead of exposing Slack's internal mention token.
	legacyPrefix := "<@" + userID + ">: "
	blockPrefix := "@" + u + ": "
	msgs := s.slackifyMessage(userID, legacyPrefix, blockPrefix, msg, f, msgObject)
	return s.sendMessages(msgs, userID, chanID, thr, f, msgObject)
}

// SendProtocolUserMessage sends a direct message to a user
func (s *slackConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) (ret robot.RetVal) {
	var userID string
	var ok bool
	if userID, ok = util.ExtractID(u); !ok {
		userID, ok = s.userID(u, false)
	}
	if !ok {
		s.Log(robot.Error, "No slack user ID found for user: %s", u)
		return robot.UserNotFound
	}
	var userIMchanstr string
	var userIMchan *slack.Channel
	var err error
	userIMchanstr, ok = s.userIMID(userID)
	if !ok {
		s.Log(robot.Warn, "No slack IM channel found for user: %s, ID: %s trying to open IM", u, userID)
		ocParam := slack.OpenConversationParameters{
			ChannelID: "",
			ReturnIM:  false,
			Users:     []string{userID},
		}
		userIMchan, _, _, err = s.api.OpenConversation(&ocParam)
		if err != nil {
			s.Log(robot.Error, "Unable to open a slack IM channel to user: %s, ID: %s", u, userID)
			return robot.FailedMessageSend
		}
		userIMchanstr = userIMchan.Conversation.ID
	}
	msgs := s.slackifyMessage(userID, "", "", msg, f, msgObject)
	return s.sendMessages(msgs, "", userIMchanstr, "", f, msgObject)
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) (ret robot.RetVal) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.Log(robot.Error, "Slack channel ID not found for: %s", c)
		return robot.ChannelNotFound
	}
	if s.socketMode {
		_, _, _, err := s.api.JoinConversation(chanID)
		if err != nil {
			s.Log(robot.Error, "Joining channel '%s': %v", c, err)
		} else {
			s.Log(robot.Debug, "Joined channel %s/%s", c, chanID)
		}
	} else {
		s.Log(robot.Debug, "Slack RTM robots can't join channels, skipping join for %s/%s", c, chanID)
	}
	return robot.Ok
}

func (s *slackConnector) DefaultHelp() []string {
	return nil
}

func formatHiddenCommand(botName, input string) string {
	name := strings.TrimSpace(strings.TrimPrefix(botName, "/"))
	if name == "" {
		return ""
	}
	fields := strings.Fields(name)
	if len(fields) > 0 {
		name = fields[0]
	}
	command := strings.TrimSpace(input)
	name = strings.ToLower(name)
	if command == "" {
		return "/" + name
	}
	return "/" + name + " " + command
}

func (s *slackConnector) FormatHiddenCommand(input string) string {
	return formatHiddenCommand(s.slashCommand, input)
}

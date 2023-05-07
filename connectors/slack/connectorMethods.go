package slack

import (
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
)

const typingDelay = 200 * time.Millisecond

// Message send delay; slack has problems with scrolling if messages fly out
// too fast.
const msgDelay = 1 * time.Second

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
	if userID, ok = s.ExtractID(u); !ok {
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
	message, channel, thread string
	format                   robot.MessageFormat
}

var messages = make(chan *sendMessage)

// Send a typing notifier letting the user know the message has been heard by
// the robot.
func (s *slackConnector) MessageHeard(user, channel string) {
	var chanID string
	var ok bool
	if chanID, ok = s.ExtractID(channel); ok {
		if socketmodeEnabled {
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
			s.conn.SendMessage(s.conn.NewTypingMessage(chanID))
		}
	}
}

func (s *slackConnector) startSendLoop() {
	// See bursting constants above.
	var burstTime time.Time
	mtimes := make([]time.Time, burstMessages)
	current := 0 // index of the current message send time
	for {
		send := <-messages
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
		opts := []slack.MsgOption{
			slack.MsgOptionText(send.message, false),
			slack.MsgOptionAsUser(true),
			slack.MsgOptionDisableLinkUnfurl(),
		}
		if len(send.thread) > 0 {
			opts = append(opts, slack.MsgOptionTS(send.thread))
		}
		if send.format == robot.Variable {
			opts = append(opts, slack.MsgOptionDisableMarkdown(), slack.MsgOptionParse(false))
		}
		s.Log(robot.Trace, "Bot message in slack send loop for channel %s, size: %d", send.channel, len(send.message))
		time.Sleep(typingDelay)
		sent := false
		for p := range []int{1, 2, 4} {
			_, _, err := s.api.PostMessage(send.channel, opts...)
			if err != nil && p == 1 {
				s.Log(robot.Warn, "Sending slack message '%s' initiating backoff: %v", send.message, err)
			}
			if err != nil {
				time.Sleep(time.Second * time.Duration(p))
			} else {
				sent = true
				break
			}
		}
		if !sent {
			if socketmodeEnabled {
				s.Log(robot.Error, "Failed sending slack message '%s' to channel '%s' after 3 tries", send.message, send.channel)
				// There doesn't appear to be a fallback available with socket mode
			} else {
				s.Log(robot.Error, "Failed sending slack message '%s' to channel '%s' after 3 tries, attempting fallback to RTM", send.message, send.channel)
				s.conn.SendMessage(s.conn.NewOutgoingMessage(send.message, send.channel))
			}
		}
		timeSinceBurst := msgTime.Sub(burstTime)
		if msgTime.Sub(mtimes[windowStartMsg]) < burstWindow || timeSinceBurst < coolDown {
			if timeSinceBurst > coolDown {
				burstTime = msgTime
			}
			s.Log(robot.Debug, "Slack burst limit exceeded, delaying next message by %v", msgDelay)
			// if we've sent `burstMessages` messages in less than the `burstWindow`
			// window, delay the next message by `msgDelay`.
			time.Sleep(msgDelay)
		}
	}
}

func (s *slackConnector) sendMessages(msgs []string, chanID, threadID string, f robot.MessageFormat) {
	for _, msg := range msgs {
		messages <- &sendMessage{
			message: msg,
			channel: chanID,
			thread:  threadID,
			format:  f,
		}
	}
}

// SetUserMap takes a map of username to userID mappings, built from the UserRoster
// of robot.yaml
func (s *slackConnector) SetUserMap(umap map[string]string) {
	s.Lock()
	s.botUserMap = umap
	s.Unlock()
	s.updateUserList("")
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolChannelThreadMessage(ch, thr, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	msgs := s.slackifyMessage("", msg, f)
	if chanID, ok := s.ExtractID(ch); ok {
		s.sendMessages(msgs, chanID, thr, f)
		return
	}
	if chanID, ok := s.chanID(ch); ok {
		s.sendMessages(msgs, chanID, thr, f)
		return
	}
	s.Log(robot.Error, "Slack channel ID not found for: %s", ch)
	return robot.ChannelNotFound
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolUserChannelThreadMessage(uid, u, ch, thr, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var userID, chanID string
	var ok bool
	if chanID, ok = s.ExtractID(ch); !ok {
		chanID, ok = s.chanID(ch)
	}
	if !ok {
		s.Log(robot.Error, "Slack channel ID not found for: %s", ch)
		return robot.ChannelNotFound
	}
	if userID, ok = s.ExtractID(uid); !ok {
		userID, ok = s.userID(u, false)
	}
	if !ok {
		s.Log(robot.Error, "Slack user ID not found for: %s", uid)
		return robot.UserNotFound
	}
	// This gets converted to <@userID> in slackifyMessage
	prefix := "<@" + userID + ">: "
	msgs := s.slackifyMessage(prefix, msg, f)
	s.sendMessages(msgs, chanID, thr, f)
	return
}

// SendProtocolUserMessage sends a direct message to a user
func (s *slackConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var userID string
	var ok bool
	if userID, ok = s.ExtractID(u); !ok {
		userID, ok = s.userID(u, false)
	}
	if !ok {
		s.Log(robot.Error, "No slack user ID found for user: %s", u)
		ret = robot.UserNotFound
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
		userIMchanstr = userIMchan.Conversation.ID

		if err != nil {
			s.Log(robot.Error, "Unable to open a slack IM channel to user: %s, ID: %s", u, userID)
			ret = robot.FailedMessageSend
		}
	}
	if ret != robot.Ok {
		return
	}
	msgs := s.slackifyMessage("", msg, f)
	s.sendMessages(msgs, userIMchanstr, "", f)
	return robot.Ok
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) (ret robot.RetVal) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.Log(robot.Error, "Slack channel ID not found for: %s", c)
		return robot.ChannelNotFound
	}
	if socketmodeEnabled {
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

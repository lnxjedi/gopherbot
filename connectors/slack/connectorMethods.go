package slack

import (
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/nlopes/slack"
)

// Message send delay; slack has problems with scrolling if messages fly out
// too fast.
const typingDelay = 200 * time.Millisecond
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
		userID, ok = s.userID(u)
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
	message, channel string
	format           robot.MessageFormat
}

var messages = make(chan *sendMessage)

// Send a typing notifier letting the user know the message has been heard by
// the robot.
func (s *slackConnector) MessageHeard(user, channel string) {
	var chanID string
	var ok bool
	if chanID, ok = s.ExtractID(channel); ok {
		s.conn.SendMessage(s.conn.NewTypingMessage(chanID))
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
		s.Log(robot.Trace, "bot message in slack send loop for channel %s, size: %d", send.channel, len(send.message))
		time.Sleep(typingDelay)
		sent := false
		for p := range []int{1, 2, 4} {
			unfurl := slack.MsgOptionEnableLinkUnfurl()
			if send.format == robot.Variable {
				unfurl = slack.MsgOptionDisableLinkUnfurl()
			}
			_, _, err := s.api.PostMessage(send.channel, slack.MsgOptionText(send.message, false), slack.MsgOptionAsUser(true), unfurl)
			if err != nil && p == 1 {
				s.Log(robot.Warn, "sending slack message '%s' initiating backoff: %v", send.message, err)
			}
			if err != nil {
				time.Sleep(time.Second * time.Duration(p))
			} else {
				sent = true
				break
			}
		}
		if !sent {
			s.Log(robot.Error, "failed sending slack message '%s' to channel '%s' after 3 tries, attempting fallback to RTM", send.message, send.channel)
			s.conn.SendMessage(s.conn.NewOutgoingMessage(send.message, send.channel))
		}
		timeSinceBurst := msgTime.Sub(burstTime)
		if msgTime.Sub(mtimes[windowStartMsg]) < burstWindow || timeSinceBurst < coolDown {
			if timeSinceBurst > coolDown {
				burstTime = msgTime
			}
			s.Log(robot.Debug, "slack burst limit exceeded, delaying next message by %v", msgDelay)
			// if we've sent `burstMessages` messages in less than the `burstWindow`
			// window, delay the next message by `msgDelay`.
			time.Sleep(msgDelay)
		}
	}
}

func (s *slackConnector) sendMessages(msgs []string, chanID string, f robot.MessageFormat) {
	for _, msg := range msgs {
		messages <- &sendMessage{
			message: msg,
			channel: chanID,
			format:  f,
		}
	}
}

// SetUserMap takes a map of username to userID mappings, built from the UserRoster
// of gopherbot.yaml
func (s *slackConnector) SetUserMap(umap map[string]string) {
	s.Lock()
	s.botUserMap = umap
	s.Unlock()
	s.updateUserList("")
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolChannelMessage(ch string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	msgs := s.slackifyMessage("", msg, f)
	if chanID, ok := s.ExtractID(ch); ok {
		s.sendMessages(msgs, chanID, f)
		return
	}
	if chanID, ok := s.chanID(ch); ok {
		s.sendMessages(msgs, chanID, f)
		return
	}
	s.Log(robot.Error, "slack channel ID not found for: %s", ch)
	return robot.ChannelNotFound
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolUserChannelMessage(uid, u, ch, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var userID, chanID string
	var ok bool
	if chanID, ok = s.ExtractID(ch); !ok {
		chanID, ok = s.chanID(ch)
	}
	if !ok {
		s.Log(robot.Error, "slack channel ID not found for: %s", ch)
		return robot.ChannelNotFound
	}
	if userID, ok = s.ExtractID(uid); !ok {
		userID, ok = s.userID(u)
	}
	if !ok {
		s.Log(robot.Error, "slack user ID not found for: %s", uid)
		return robot.UserNotFound
	}
	// This gets converted to <@userID> in slackifyMessage
	prefix := "<@" + userID + ">: "
	msgs := s.slackifyMessage(prefix, msg, f)
	s.sendMessages(msgs, chanID, f)
	return
}

// SendProtocolUserMessage sends a direct message to a user
func (s *slackConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat) (ret robot.RetVal) {
	var userID string
	var ok bool
	if userID, ok = s.ExtractID(u); !ok {
		userID, ok = s.userID(u)
	}
	if !ok {
		s.Log(robot.Error, "no slack user ID found for user: %s", u)
		ret = robot.UserNotFound
	}
	var userIMchan string
	var err error
	userIMchan, ok = s.userIMID(userID)
	if !ok {
		s.Log(robot.Warn, "no slack IM channel found for user: %s, ID: %s trying to open IM", u, userID)
		_, _, userIMchan, err = s.conn.OpenIMChannel(userID)
		if err != nil {
			s.Log(robot.Error, "unable to open a slack IM channel to user: %s, ID: %s", u, userID)
			ret = robot.FailedMessageSend
		}
	}
	if ret != robot.Ok {
		return
	}
	msgs := s.slackifyMessage("", msg, f)
	s.sendMessages(msgs, userIMchan, f)
	return robot.Ok
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) (ret robot.RetVal) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.Log(robot.Error, "slack channel ID not found for: %s", c)
		return robot.ChannelNotFound
	}
	_, err := s.api.JoinChannel(chanID)
	if err != nil {
		s.Log(robot.Error, "failed to join slack channel %s: %v; (try inviting the bot to the channel)", c, err)
		return robot.FailedChannelJoin
	}
	return robot.Ok
}

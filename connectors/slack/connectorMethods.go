package slack

import (
	"fmt"
	"time"

	"github.com/lnxjedi/gopherbot/bot"
	// "github.com/nlopes/slack"
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

// GetUserAttribute returns a string attribute or nil if slack doesn't
// have that information
func (s *slackConnector) GetProtocolUserAttribute(u, attr string) (value string, ret bot.RetVal) {
	user, ok := s.getUser(u)
	if !ok {
		return "", bot.UserNotFound
	}
	switch attr {
	case "email":
		return user.Profile.Email, bot.Ok
	case "internalid":
		return user.ID, bot.Ok
	case "realname", "fullname", "real name", "full name":
		return user.RealName, bot.Ok
	case "firstname", "first name":
		return user.Profile.FirstName, bot.Ok
	case "lastname", "last name":
		return user.Profile.LastName, bot.Ok
	case "phone":
		return user.Profile.Phone, bot.Ok
	// that's all the attributes we can currently get from slack
	default:
		return "", bot.AttributeNotFound
	}
}

type sendMessage struct {
	message, channel string
	format           bot.MessageFormat
}

var messages = make(chan *sendMessage)

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
		current += 1
		if current == (burstMessages - 1) {
			current = 0
		}
		s.Log(bot.Trace, fmt.Sprintf("Bot message in send loop for channel %s, size: %d", send.channel, len(send.message)))
		s.conn.SendMessage(s.conn.NewTypingMessage(send.channel))
		time.Sleep(typingDelay)
		s.conn.SendMessage(s.conn.NewOutgoingMessage(send.message, send.channel))
		/* NOTE: The commented out code below doesn't work. Long story:

		To implement a proper 'Variable' format that preserves _, * and `, I tried
		disabling markdown. However, messages came through as a generic 'bot', not
		the bot user with an icon and name. When I set as_user (AsUser) 'true', it
		forced Markdown to 'true' regardless of passing explicit 'false'. In the end,
		I resorted to a solution found on stackoverflow, and for format == Variable
		I 'escape' _, *, ` by surrounding them with a "\x00" (null) char.
		*/
		// time.Sleep(typingDelay) // the minimum time between message sends
		// params := slack.PostMessageParameters{
		// 	AsUser:      true,
		// 	UnfurlMedia: true,
		// 	Markdown:    false,
		// }
		// if send.format == bot.Raw {
		// 	params.Markdown = true
		// }
		// _, _, err := s.api.PostMessage(send.channel, send.message, params)
		// if err != nil {
		// 	s.Log(bot.Error, fmt.Sprintf("Error sending message '%s': %v", send.message, err))
		// }
		timeSinceBurst := msgTime.Sub(burstTime)
		if msgTime.Sub(mtimes[windowStartMsg]) < burstWindow || timeSinceBurst < coolDown {
			if timeSinceBurst > coolDown {
				burstTime = msgTime
			}
			s.Log(bot.Debug, fmt.Sprintf("Burst limit exceeded, delaying next message by %v", msgDelay))
			// if we've sent `burstMessages` messages in less than the `burstWindow`
			// window, delay the next message by `msgDelay`.
			time.Sleep(msgDelay)
		}
	}
}

func (s *slackConnector) sendMessages(msgs []string, chanID string, f bot.MessageFormat) {
	for _, msg := range msgs {
		messages <- &sendMessage{
			message: msg,
			channel: chanID,
			format:  f,
		}
	}
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolChannelMessage(ch string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	chanID, ok := s.chanID(ch)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", ch)
		return bot.ChannelNotFound
	}
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, chanID, f)
	return
}

// SendProtocolChannelMessage sends a message to a channel
func (s *slackConnector) SendProtocolUserChannelMessage(u, ch, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	chanID, ok := s.chanID(ch)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", ch)
		ret = bot.ChannelNotFound
	} else if _, ok := s.userID(u); !ok {
		ret = bot.UserNotFound
	}
	if ret != bot.Ok {
		return
	}
	msg = "@" + u + ": " + msg
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, chanID, f)
	return
}

// SendProtocolUserMessage sends a direct message to a user
func (s *slackConnector) SendProtocolUserMessage(u string, msg string, f bot.MessageFormat) (ret bot.RetVal) {
	userID, ok := s.userID(u)
	if !ok {
		s.Log(bot.Error, "No user ID found for user:", u)
		ret = bot.UserNotFound
	}
	var userIMchan string
	var err error
	userIMchan, ok = s.userIMID(userID)
	if !ok {
		s.Log(bot.Warn, "No IM channel found for user:", u, "ID:", userID, "trying to open IM")
		_, _, userIMchan, err = s.conn.OpenIMChannel(userID)
		if err != nil {
			s.Log(bot.Error, "Unable to open an IM channel to user:", u, "ID:", userID)
			ret = bot.FailedUserDM
		}
	}
	if ret != bot.Ok {
		return
	}
	msgs := s.slackifyMessage(msg, f)
	s.sendMessages(msgs, userIMchan, f)
	return bot.Ok
}

// JoinChannel joins a channel given it's human-readable name, e.g. "general"
func (s *slackConnector) JoinChannel(c string) (ret bot.RetVal) {
	chanID, ok := s.chanID(c)
	if !ok {
		s.Log(bot.Error, "Channel ID not found for:", c)
		return bot.ChannelNotFound
	}
	_, err := s.api.JoinChannel(chanID)
	if err != nil {
		s.Log(bot.Error, "Failed to join channel", c, ":", err, "(try inviting the bot)")
		return bot.FailedChannelJoin
	}
	return bot.Ok
}

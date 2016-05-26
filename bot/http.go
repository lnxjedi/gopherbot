package bot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type JSONFunction struct {
	FuncName string
	User     string
	Channel  string
	Format   string
	PluginID string
	FuncArgs json.RawMessage
}

type Attr struct {
	Attribute string
}

type UserAttr struct {
	User      string
	Attribute string
}

type ChannelMessage struct {
	Channel string
	Message string
	Format  string
}

type UserMessage struct {
	User    string
	Message string
	Format  string
}

type UserChannelMessage struct {
	User    string
	Channel string
	Message string
	Format  string
}

type ReplyRequest struct {
	RegExId string
	Timeout int
}

func (b *robot) listenHttpJSON() {
	if len(b.port) > 0 {
		http.Handle("/json", b)
		b.Log(Fatal, http.ListenAndServe(b.port, nil))
	}
}

// decode looks for a base64: prefix, then removes it and tries to decode the message
func (b *robot) decode(msg string) string {
	if strings.HasPrefix(msg, "base64:") {
		msg = strings.TrimPrefix(msg, "base64:")
		decoded, err := base64.StdEncoding.DecodeString(msg)
		if err != nil {
			b.Log(Error, fmt.Errorf("Unable to decode base64 message %s: %v", msg, err))
			return msg
		}
		return string(decoded)
	} else {
		return msg
	}
}

func (b *robot) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.Log(Fatal, err)
	}
	defer r.Body.Close()

	var f JSONFunction
	err = json.Unmarshal(data, &f)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		b.Log(Error, "Couldn't decipher JSON command: ", err)
		return
	}

	// Generate a synthetic Robot for access to it's methods
	bot := Robot{
		User:     f.User,
		Channel:  f.Channel,
		Format:   setFormat(f.Format),
		pluginID: f.PluginID,
		robot:    b,
	}

	switch f.FuncName {
	case "GetSenderAttribute", "GetBotAttribute":
		var a Attr
		err := json.Unmarshal(f.FuncArgs, &a)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		if f.FuncName == "GetBotAttribute" {
			fmt.Fprintln(rw, bot.GetBotAttribute(a.Attribute))
		} else {
			fmt.Fprintln(rw, bot.GetSenderAttribute(a.Attribute))
		}
	case "GetUserAttribute":
		var ua UserAttr
		err := json.Unmarshal(f.FuncArgs, &ua)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		fmt.Fprintln(rw, bot.GetUserAttribute(ua.User, ua.Attribute))
	case "SendChannelMessage":
		var cm ChannelMessage
		err := json.Unmarshal(f.FuncArgs, &cm)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.Channel = cm.Channel
		bot.SendChannelMessage(cm.Channel, b.decode(cm.Message))
	case "SendUserChannelMessage":
		var ucm UserChannelMessage
		err := json.Unmarshal(f.FuncArgs, &ucm)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.User = ucm.User
		bot.Channel = ucm.Channel
		bot.SendUserChannelMessage(ucm.User, ucm.Channel, b.decode(ucm.Message))
	case "SendUserMessage":
		var um UserMessage
		err := json.Unmarshal(f.FuncArgs, &um)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.User = um.User
		bot.SendUserMessage(um.User, b.decode(um.Message))
	case "WaitForReply":
		var rr ReplyRequest
		err := json.Unmarshal(f.FuncArgs, &rr)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		_, _, reply, err := bot.WaitForReply(rr.RegExId, rr.Timeout)
		if err != nil {
			rw.WriteHeader(http.StatusServiceUnavailable)
			b.Log(Error, "Waiting for reply: ", err)
			return
		}
		fmt.Fprintln(rw, reply)
	// NOTE: "Say" and "Reply" are implemented in shellLib.sh or other scripting library
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

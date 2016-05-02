package bot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type JSONCommand struct {
	Command  string
	PluginID string
	CmdArgs  json.RawMessage
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

type ReplyRequest struct {
	User        string
	Channel     string
	RegExId     string
	Timeout     int
	NeedCommand bool
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

func (b *robot) listenHttpJSON() {
	if len(b.port) > 0 {
		http.Handle("/json", b)
		log.Fatal(http.ListenAndServe(b.port, nil))
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
		log.Fatal(err)
	}
	defer r.Body.Close()

	var c JSONCommand
	err = json.Unmarshal(data, &c)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		b.Log(Error, "Couldn't decipher JSON command: ", err)
		return
	}

	// Generate a synthetic Robot for access to it's methods
	bot := Robot{
		User:     "",
		Channel:  "",
		Format:   Variable,
		pluginID: c.PluginID,
		robot:    b,
	}

	switch c.Command {
	case "GetAttribute":
		var a Attr
		err := json.Unmarshal(c.CmdArgs, &a)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		fmt.Fprintln(rw, bot.GetAttribute(a.Attribute))
	case "GetUserAttribute":
		var ua UserAttr
		err := json.Unmarshal(c.CmdArgs, &ua)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.User = ua.User
		fmt.Fprintln(rw, bot.GetUserAttribute(ua.Attribute))
	case "SendChannelMessage":
		var cm ChannelMessage
		err := json.Unmarshal(c.CmdArgs, &cm)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.Channel = cm.Channel
		bot.Format = setFormat(cm.Format)
		bot.SendChannelMessage(b.decode(cm.Message))
	case "SendUserChannelMessage":
		var ucm UserChannelMessage
		err := json.Unmarshal(c.CmdArgs, &ucm)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.User = ucm.User
		bot.Channel = ucm.Channel
		bot.Format = setFormat(ucm.Format)
		bot.SendUserChannelMessage(b.decode(ucm.Message))
	case "SendUserMessage":
		var um UserMessage
		err := json.Unmarshal(c.CmdArgs, &um)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.User = um.User
		bot.Format = setFormat(um.Format)
		bot.SendUserMessage(b.decode(um.Message))
	case "WaitForReply":
		var rr ReplyRequest
		err := json.Unmarshal(c.CmdArgs, &rr)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		bot.Log(Trace, "")
		bot.User = rr.User
		bot.Channel = rr.Channel
		reply, err := bot.WaitForReply(rr.RegExId, rr.Timeout, rr.NeedCommand)
		if err != nil {
			rw.WriteHeader(http.StatusServiceUnavailable)
			b.Log(Error, "Waiting for reply: ", err)
			//			rw.WriteHeader(http.StatusServiceUnavailable)
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

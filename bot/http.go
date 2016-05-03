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

type Message struct {
	Message string
}

type ReplyRequest struct {
	RegExId     string
	Timeout     int
	NeedCommand bool
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
	case "GetAttribute", "GetUserAttribute":
		var a Attr
		err := json.Unmarshal(f.FuncArgs, &a)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		if f.FuncName == "GetAttribute" {
			fmt.Fprintln(rw, bot.GetAttribute(a.Attribute))
		} else {
			fmt.Fprintln(rw, bot.GetUserAttribute(a.Attribute))
		}
	case "SendChannelMessage", "SendUserChannelMessage", "SendUserMessage":
		var m Message
		err := json.Unmarshal(f.FuncArgs, &m)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		switch f.FuncName {
		case "SendChannelMessage":
			bot.SendChannelMessage(b.decode(m.Message))
		case "SendUserChannelMessage":
			bot.SendUserChannelMessage(b.decode(m.Message))
		case "SendUserMessage":
			bot.SendUserMessage(b.decode(m.Message))
		}
	case "WaitForReply":
		var rr ReplyRequest
		err := json.Unmarshal(f.FuncArgs, &rr)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			b.Log(Error, "Couldn't decipher JSON command data: ", err)
			return
		}
		reply, err := bot.WaitForReply(rr.RegExId, rr.Timeout, rr.NeedCommand)
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

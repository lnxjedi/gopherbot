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
	Command string
	CmdArgs json.RawMessage
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

func (b *Bot) listenHttpJSON() {
	if len(b.port) > 0 {
		http.Handle("/json", b)
		log.Fatal(http.ListenAndServe(b.port, nil))
	}
}

// decode looks for a base64: prefix, then removes it and tries to decode the message
func (b *Bot) decode(msg string) string {
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

func (b *Bot) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()

	var c JSONCommand
	err = json.Unmarshal(data, &c)
	if err != nil {
		fmt.Fprintln(rw, "Couldn't decipher JSON command: ", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	switch c.Command {
	case "SendChannelMessage":
		var cm ChannelMessage
		err := json.Unmarshal(c.CmdArgs, &cm)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		f := setFormat(cm.Format)
		b.SendProtocolChannelMessage(cm.Channel, b.decode(cm.Message), f)
	case "SendUserChannelMessage":
		var ucm UserChannelMessage
		err := json.Unmarshal(c.CmdArgs, &ucm)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		f := setFormat(ucm.Format)
		b.SendProtocolUserChannelMessage(ucm.User, ucm.Channel, b.decode(ucm.Message), f)
	case "SendUserMessage":
		var um UserMessage
		err := json.Unmarshal(c.CmdArgs, &um)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		f := setFormat(um.Format)
		b.SendProtocolUserMessage(um.User, b.decode(um.Message), f)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

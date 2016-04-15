package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type JSONCommand struct {
	Command string
	CmdArgs json.RawMessage
}

type ChannelMessage struct {
	Channel string
	Message string
}

type UserMessage struct {
	User    string
	Message string
}

func (b *Bot) listenHttpJSON() {
	if len(b.port) > 0 {
		http.Handle("/json", b)
		log.Fatal(http.ListenAndServe(b.port, nil))
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
		b.SendChannelMessage(cm.Channel, cm.Message)
	case "SendUserMessage":
		var um UserMessage
		err := json.Unmarshal(c.CmdArgs, &um)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		b.SendUserMessage(um.User, um.Message)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

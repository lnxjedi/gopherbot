package gobot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type JSONCommand struct {
	Command string
	CmdData json.RawMessage
}

type ChannelMessage struct {
	ChanID  string
	Message string
}

func (b *Bot) ListenHttpJSON() {
	if len(b.port) > 0 {
		http.Handle("/json", b)
		log.Fatal(http.ListenAndServe(b.port, nil))
	}
}

func (b *Bot) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	b.Debug("Read: ", data)
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
	case "ChannelMessage":
		var cm ChannelMessage
		err := json.Unmarshal(c.CmdData, &cm)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		b.conn.SendChannelMsg(cm.ChanID, cm.Message)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

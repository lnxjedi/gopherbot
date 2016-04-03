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
	CmdArgs json.RawMessage
}

type ChannelMessage struct {
	ChanID  string
	Message string
}

func (l *botListener) listenHttpJSON() {
	if len(l.port) > 0 {
		http.Handle("/json", l)
		log.Fatal(http.ListenAndServe(l.port, nil))
	}
}

func (l *botListener) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()

	b := l.owner
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
		b.conn.SendChannelMessage(cm.ChanID, cm.Message)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

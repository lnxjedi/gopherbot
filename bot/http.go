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

type attribute struct {
	Attribute string
}

type userattr struct {
	User      string
	Attribute string
}

type logmessage struct {
	Level   string
	Message string
}

type channelmessage struct {
	Channel string
	Message string
	Format  string
}

type usermessage struct {
	User    string
	Message string
	Format  string
}

type userchannelmessage struct {
	User    string
	Channel string
	Message string
	Format  string
}

type replyrequest struct {
	RegExId string
	Timeout int
}

// Types for returning values

// BotAttrRet implements Stringer so it can be interpolated with fmt if
// the plugin author is ok with ignoring the BotRetVal.
type BotAttrRet struct {
	Attribute string
	BotRetVal
}

func (bar *BotAttrRet) String() string {
	return bar.Attribute
}

// These are only for json marshalling
type botretvalresponse struct {
	BotRetVal int
}

type waitreplyresponse struct {
	Reply     string
	BotRetVal int
}

func listenHttpJSON() {
	if len(b.port) > 0 {
		h := handler{}
		http.Handle("/json", h)
		Log(Fatal, http.ListenAndServe(b.port, nil))
	}
}

// decode looks for a base64: prefix, then removes it and tries to decode the message
func decode(msg string) string {
	if strings.HasPrefix(msg, "base64:") {
		msg = strings.TrimPrefix(msg, "base64:")
		decoded, err := base64.StdEncoding.DecodeString(msg)
		if err != nil {
			Log(Error, fmt.Errorf("Unable to decode base64 message %s: %v", msg, err))
			return msg
		}
		return string(decoded)
	} else {
		return msg
	}
}

func encode(arg string) string {
	return "base64:" + base64.StdEncoding.EncodeToString([]byte(arg))
}

func getArgs(rw http.ResponseWriter, jsonargs *json.RawMessage, args interface{}) bool {
	err := json.Unmarshal(*jsonargs, args)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		Log(Error, "Couldn't deciper JSON args: ", err)
		return false
	}
	return true
}

func sendReturn(rw http.ResponseWriter, ret interface{}) {
	d, err := json.Marshal(ret)
	if err != nil { // this should never happen
		Log(Fatal, fmt.Sprintf("BUG in bot/http.go:sendReturn, error marshalling JSON: %v", err))
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(d)
}

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log(Fatal, err)
	}
	defer r.Body.Close()

	var f JSONFunction
	err = json.Unmarshal(data, &f)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		Log(Error, "Couldn't decipher JSON command: ", err)
		return
	}

	// Generate a synthetic Robot for access to it's methods
	bot := Robot{
		User:     f.User,
		Channel:  f.Channel,
		Format:   setFormat(f.Format),
		pluginID: f.PluginID,
	}

	var (
		attr  *BotAttrRet
		reply string
		ret   BotRetVal
	)
	switch f.FuncName {
	case "GetSenderAttribute", "GetBotAttribute":
		var a attribute
		if !getArgs(rw, &f.FuncArgs, &a) {
			return
		}
		if f.FuncName == "GetBotAttribute" {
			attr = bot.GetBotAttribute(a.Attribute)
		} else {
			attr = bot.GetSenderAttribute(a.Attribute)
		}
		attr.Attribute = encode(attr.Attribute)
		sendReturn(rw, attr)
		return
	case "GetUserAttribute":
		var ua userattr
		if !getArgs(rw, &f.FuncArgs, &ua) {
			return
		}
		attr = bot.GetUserAttribute(ua.User, ua.Attribute)
		attr.Attribute = encode(attr.Attribute)
		sendReturn(rw, attr)
		return
	case "LogMessage":
		var lm logmessage
		if !getArgs(rw, &f.FuncArgs, &lm) {
			return
		}
		l := logStrToLevel(lm.Level)
		Log(l, lm.Message)
		sendReturn(rw, &botretvalresponse{int(Ok)})
		return
	case "SendChannelMessage":
		var cm channelmessage
		if !getArgs(rw, &f.FuncArgs, &cm) {
			return
		}
		sendReturn(rw, &botretvalresponse{
			int(bot.SendChannelMessage(cm.Channel, decode(cm.Message))),
		})
		return
	case "SendUserChannelMessage":
		var ucm userchannelmessage
		if !getArgs(rw, &f.FuncArgs, &ucm) {
			return
		}
		sendReturn(rw, &botretvalresponse{
			int(bot.SendUserChannelMessage(ucm.User, ucm.Channel, decode(ucm.Message))),
		})
		return
	case "SendUserMessage":
		var um usermessage
		if !getArgs(rw, &f.FuncArgs, &um) {
			return
		}
		sendReturn(rw, &botretvalresponse{
			int(bot.SendUserMessage(um.User, decode(um.Message))),
		})
		return
	case "WaitForReply":
		var rr replyrequest
		if !getArgs(rw, &f.FuncArgs, &rr) {
			return
		}
		reply, ret = bot.WaitForReply(rr.RegExId, rr.Timeout)
		sendReturn(rw, &waitreplyresponse{encode(reply), int(ret)})
		return
	// NOTE: "Say" and "Reply" are implemented in shellLib.sh or other scripting library
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
}

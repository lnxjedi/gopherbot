package bot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type jsonFunction struct {
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

type elevate struct {
	Immediate bool
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
}

// Something to be placed in short-term memory
type shorttermmemory struct {
	Key, Value string
}

// Something to be recalled from short-term memory
type shorttermrecollection struct {
	Key string
}

// Something to be remembered in long term memory
type memory struct {
	Key   string
	Token string
	Datum json.RawMessage
}

// Something to be recalled from long term memory
type recollection struct {
	Key string
	RW  bool
}

type usermessage struct {
	User    string
	Message string
}

type userchannelmessage struct {
	User    string
	Channel string
	Message string
}

type replyrequest struct {
	RegexID string
	Timeout int
}

type regexreplyrequest struct {
	RegEx   string
	Timeout int
}

// Types for returning values

// AttrRet implements Stringer so it can be interpolated with fmt if
// the plugin author is ok with ignoring the RetVal.
type AttrRet struct {
	Attribute string
	RetVal
}

func (bar *AttrRet) String() string {
	return bar.Attribute
}

// These are only for json marshalling
type boolresponse struct {
	Boolean bool
}

type stringresponse struct {
	StrVal string
}

type boolretresponse struct {
	Boolean bool
	RetVal  int
}

type botretvalresponse struct {
	RetVal int
}

type checkoutresponse struct {
	LockToken string
	Exists    bool
	Datum     interface{}
	RetVal    int
}

type waitreplyresponse struct {
	Reply  string
	RetVal int
}

func listenHTTPJSON() {
	if len(robot.port) > 0 {
		h := handler{}
		http.Handle("/json", h)
		Log(Fatal, http.ListenAndServe(robot.port, nil))
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
	}
	return msg
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

	var f jsonFunction
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
		attr  *AttrRet
		reply string
		ret   RetVal
	)
	switch f.FuncName {
	case "CheckAdmin":
		bret := bot.CheckAdmin()
		sendReturn(rw, boolresponse{Boolean: bret})
		return
	case "Elevate":
		var e elevate
		if !getArgs(rw, &f.FuncArgs, &e) {
			return
		}
		success := bot.Elevate(e.Immediate)
		sendReturn(rw, boolresponse{Boolean: success})
		return
	case "CheckoutDatum":
		var r recollection
		if !getArgs(rw, &f.FuncArgs, &r) {
			return
		}
		var datum interface{}
		l, e, brv := bot.CheckoutDatum(r.Key, &datum, r.RW)
		sendReturn(rw, checkoutresponse{
			LockToken: l,
			Exists:    e,
			Datum:     datum,
			RetVal:    int(brv),
		})
		return
	case "CheckinDatum":
		var m memory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		bot.CheckinDatum(m.Key, m.Token)
		sendReturn(rw, &botretvalresponse{int(Ok)})
		return
	case "UpdateDatum":
		plugMapLock.Lock()
		pluginName := plugIDNameMap[bot.pluginID]
		plugMapLock.Unlock()
		var m memory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		// Since we're getting raw JSON (=[]byte), we call update directly.
		// See brain.go
		ret = update(pluginName+":"+m.Key, m.Token, (*[]byte)(&m.Datum))
		sendReturn(rw, &botretvalresponse{int(ret)})
		return
	case "Remember":
		var m shorttermmemory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		bot.Remember(m.Key, m.Value)
		sendReturn(rw, &botretvalresponse{int(Ok)})
	case "Recall":
		var m shorttermrecollection
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		s := bot.Recall(m.Key)
		sendReturn(rw, &stringresponse{s})
	case "GetPluginConfig":
		robot.RLock()
		defer robot.RUnlock()
		plugin := plugins[plugIDmap[bot.pluginID]]
		if plugin.Config == nil {
			Log(Error, fmt.Sprintf("GetPluginConfig called by external plugin \"%s\", but no config found.", plugin.name))
			sendReturn(rw, handler{})
			return
		}
		sendReturn(rw, plugin.Config)
		return
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
	case "Log":
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
		reply, ret = bot.WaitForReply(rr.RegexID, rr.Timeout)
		sendReturn(rw, &waitreplyresponse{encode(reply), int(ret)})
		return
	case "WaitForReplyRegex":
		var rr regexreplyrequest
		if !getArgs(rw, &f.FuncArgs, &rr) {
			return
		}
		reply, ret = bot.WaitForReplyRegex(rr.RegEx, rr.Timeout)
		sendReturn(rw, &waitreplyresponse{encode(reply), int(ret)})
		return
	// NOTE: "Say" and "Reply" are implemented in shellLib.sh or other scripting library
	default:
		Log(Error, fmt.Sprintf("Bad function name: %s", f.FuncName))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
}

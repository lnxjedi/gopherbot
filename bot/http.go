package bot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

type jsonFunction struct {
	FuncName string
	User     string
	Channel  string
	Format   string
	Protocol string
	CallerID string
	FuncArgs json.RawMessage
}

// Types for FuncArgs

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
	Base64  bool
}

type channelmessage struct {
	Channel string
	Message string
	Base64  bool
}

type plugincall struct {
	PluginName string
}

// Something to be placed in short-term memory
type shorttermmemory struct {
	Key, Value string
	Base64     bool
}

// Something to be recalled from short-term memory
type shorttermrecollection struct {
	Key    string
	Base64 bool
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
	Base64  bool
}

type userchannelmessage struct {
	User    string
	Channel string
	Message string
	Base64  bool
}

type replyrequest struct {
	RegexID string
	User    string
	Channel string
	Prompt  string
	Base64  bool
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

type replyresponse struct {
	Reply  string
	RetVal int
}

var botHttpListener struct {
	listening bool
	sync.Mutex
}

func listenHTTPJSON() {
	robot.RLock()
	port := robot.port
	robot.RUnlock()
	if len(port) > 0 {
		h := handler{}
		http.Handle("/json", h)
		Log(Fatal, http.ListenAndServe(port, nil))
	}
}

// decode decodes a base64 string, primarily for the bash library
func decode(msg string) string {
	decoded, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		Log(Error, fmt.Errorf("Unable to decode base64 message %s: %v", msg, err))
		return msg
	}
	return string(decoded)
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

	if f.CallerID == "" {
		rw.WriteHeader(http.StatusBadRequest)
		Log(Error, fmt.Sprintf("JSON function '%s' called with empty CallerID; args: %v", f.FuncName, f.FuncArgs))
		return
	}

	// TODO: This should do a lookup from a global hash of running plugins
	activeRobots.RLock()
	bot, ok := activeRobots.m[f.CallerID]
	activeRobots.RUnlock()
	if !ok {
		rw.WriteHeader(http.StatusBadRequest)
		Log(Error, fmt.Sprintf("JSON function '%s' called with invalid CallerID '%s'; args: %s", f.FuncName, f.CallerID, f.FuncArgs))
		return
	}
	task, _, _ := currentTasks.getTaskByID(f.CallerID)
	if task == nil {
	}
	Log(Trace, fmt.Sprintf("Task '%s' calling function '%s' in channel '%s' for user '%s'", task.name, f.FuncName, f.Channel, f.User))

	// Generate a synthetic Robot for access to it's methods
	bot := Robot{
		User:     f.User,
		Channel:  f.Channel,
		Protocol: setProtocol(f.Protocol),
		callerID: f.CallerID,
	}
	if len(f.Format) > 0 {
		bot.Format = bot.setFormat(f.Format)
	} else {
		robot.RLock()
		bot.Format = robot.defaultMessageFormat
		robot.RUnlock()
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
		var m memory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		// Since we're getting raw JSON (=[]byte), we call update directly.
		// See brain.go
		// TODO: Use the namespace from the Robot here
		ret = update(task.name+":"+m.Key, m.Token, (*[]byte)(&m.Datum))
		sendReturn(rw, &botretvalresponse{int(ret)})
		return
	case "Remember":
		var m shorttermmemory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		if m.Base64 {
			m.Key = decode(m.Key)
			m.Value = decode(m.Value)
		}
		bot.Remember(m.Key, m.Value)
		sendReturn(rw, &botretvalresponse{int(Ok)})
	case "Recall":
		var m shorttermrecollection
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		if m.Base64 {
			m.Key = decode(m.Key)
		}
		s := bot.Recall(m.Key)
		sendReturn(rw, &stringresponse{s})
	case "GetTaskConfig":
		if task.Config == nil {
			Log(Error, fmt.Sprintf("GetTaskConfig called by external script '%s', but no config found.", task.name))
			sendReturn(rw, handler{})
			return
		}
		sendReturn(rw, task.Config)
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
		sendReturn(rw, attr)
		return
	case "GetUserAttribute":
		var ua userattr
		if !getArgs(rw, &f.FuncArgs, &ua) {
			return
		}
		attr = bot.GetUserAttribute(ua.User, ua.Attribute)
		sendReturn(rw, attr)
		return
	case "Log":
		var lm logmessage
		if !getArgs(rw, &f.FuncArgs, &lm) {
			return
		}
		l := logStrToLevel(lm.Level)
		if lm.Base64 {
			lm.Message = decode(lm.Message)
		}
		Log(l, lm.Message)
		sendReturn(rw, &botretvalresponse{int(Ok)})
		return
	case "SendChannelMessage":
		var cm channelmessage
		if !getArgs(rw, &f.FuncArgs, &cm) {
			return
		}
		if cm.Base64 {
			cm.Message = decode(cm.Message)
		}
		sendReturn(rw, &botretvalresponse{
			int(bot.SendChannelMessage(cm.Channel, cm.Message)),
		})
		return
	case "SendUserChannelMessage":
		var ucm userchannelmessage
		if !getArgs(rw, &f.FuncArgs, &ucm) {
			return
		}
		if ucm.Base64 {
			ucm.Message = decode(ucm.Message)
		}
		sendReturn(rw, &botretvalresponse{
			int(bot.SendUserChannelMessage(ucm.User, ucm.Channel, ucm.Message)),
		})
		return
	case "SendUserMessage":
		var um usermessage
		if !getArgs(rw, &f.FuncArgs, &um) {
			return
		}
		if um.Base64 {
			um.Message = decode(um.Message)
		}
		sendReturn(rw, &botretvalresponse{
			int(bot.SendUserMessage(um.User, um.Message)),
		})
		return
	case "PromptUserChannelForReply":
		var rr replyrequest
		if !getArgs(rw, &f.FuncArgs, &rr) {
			return
		}
		if rr.Base64 {
			rr.Prompt = decode(rr.Prompt)
		}
		reply, ret = bot.promptInternal(rr.RegexID, rr.User, rr.Channel, rr.Prompt)
		sendReturn(rw, &replyresponse{reply, int(ret)})
		return
	// NOTE: "Say", "Reply", PromptForReply and PromptUserForReply are implemented
	// in the scripting libraries
	default:
		Log(Error, fmt.Sprintf("Bad function name: %s", f.FuncName))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
}

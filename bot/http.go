package bot

/* http.go translates posted JSON to Robot method calls, then packages
   and returns the JSON response.
*/

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/lnxjedi/gopherbot/robot"
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

type secname struct {
	Secret string
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

type taskcall struct {
	Name    string
	CmdArgs []string
}

type cmdcall struct {
	Plugin  string
	Command string
}

type paramcall struct {
	Name, Value string
}

type wdcall struct {
	Path string
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

// Request for exclusive execution
type exclusive struct {
	Tag       string
	QueueTask bool
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

type extns struct {
	Extend    string
	Histories int
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

// decode decodes a base64 string, primarily for the bash library
func decode(msg string) string {
	decoded, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		Log(robot.Error, "Unable to decode base64 message %s: %v", msg, err)
		return msg
	}
	return string(decoded)
}

func getArgs(rw http.ResponseWriter, jsonargs *json.RawMessage, args interface{}) bool {
	err := json.Unmarshal(*jsonargs, args)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		Log(robot.Error, "Couldn't deciper JSON args: ", err)
		return false
	}
	return true
}

func sendReturn(rw http.ResponseWriter, ret interface{}) {
	d, err := json.Marshal(ret)
	if err != nil { // this should never happen
		Log(robot.Fatal, "BUG in bot/http.go:sendReturn, error marshalling JSON: %v", err)
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(d)
}

func (h handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		Log(robot.Fatal, err.Error())
	}
	defer req.Body.Close()

	var f jsonFunction
	err = json.Unmarshal(data, &f)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		Log(robot.Error, "Couldn't decipher JSON command: ", err)
		return
	}

	if f.CallerID == "" {
		rw.WriteHeader(http.StatusBadRequest)
		Log(robot.Error, "JSON function '%s' called with empty CallerID; args: %v", f.FuncName, f.FuncArgs)
		return
	}

	// Look up the botContext
	c := getBotContextStr(f.CallerID)
	if c == nil {
		rw.WriteHeader(http.StatusBadRequest)
		Log(robot.Error, "JSON function '%s' called with invalid CallerID '%s'; args: %s", f.FuncName, f.CallerID, f.FuncArgs)
		return
	}
	privCheck(fmt.Sprintf("http method %s", f.FuncName))

	// Generate a synthetic Robot for access to it's methods
	proto, _ := getProtocol(f.Protocol)
	r := Robot{
		&robot.Message{
			User:            f.User,
			ProtocolUser:    c.ProtocolUser,
			Channel:         f.Channel,
			ProtocolChannel: c.ProtocolChannel,
			Protocol:        proto,
			Incoming:        c.Incoming,
		},
		c.id,
	}
	if len(f.Format) > 0 {
		r.Format = setFormat(f.Format)
	} else {
		botCfg.RLock()
		r.Format = botCfg.defaultMessageFormat
		botCfg.RUnlock()
	}
	task, _, _ := getTask(c.currentTask)
	Log(robot.Trace, "Task '%s' calling function '%s' in channel '%s' for user '%s'", task.name, f.FuncName, f.Channel, f.User)

	if len(f.Format) > 0 {
		r.Format = setFormat(f.Format)
	} else {
		botCfg.RLock()
		r.Format = botCfg.defaultMessageFormat
		botCfg.RUnlock()
	}

	var (
		attr  *robot.AttrRet
		reply string
		ret   robot.RetVal
	)
	switch f.FuncName {
	case "CheckAdmin":
		bret := r.CheckAdmin()
		sendReturn(rw, boolresponse{Boolean: bret})
		return
	case "GetRepoData":
		sendReturn(rw, r.GetRepoData())
		return
	case "AddTask", "AddJob", "FinalTask", "FailTask", "SpawnJob":
		var ts taskcall
		if !getArgs(rw, &f.FuncArgs, &ts) {
			return
		}
		var ret robot.RetVal
		switch f.FuncName {
		case "AddJob":
			ret = r.AddJob(ts.Name, ts.CmdArgs...)
		case "AddTask":
			ret = r.AddTask(ts.Name, ts.CmdArgs...)
		case "FinalTask":
			ret = r.FinalTask(ts.Name, ts.CmdArgs...)
		case "FailTask":
			ret = r.FailTask(ts.Name, ts.CmdArgs...)
		case "SpawnJob":
			ret = r.SpawnJob(ts.Name, ts.CmdArgs...)
		default:
			return
		}
		sendReturn(rw, &botretvalresponse{int(ret)})
		return
	case "AddCommand", "FinalCommand", "FailCommand":
		var cc cmdcall
		if !getArgs(rw, &f.FuncArgs, &cc) {
			return
		}
		var ret robot.RetVal
		switch f.FuncName {
		case "AddCommand":
			ret = r.AddCommand(cc.Plugin, cc.Command)
		case "FinalCommand":
			ret = r.FinalCommand(cc.Plugin, cc.Command)
		case "FailCommand":
			ret = r.FailCommand(cc.Plugin, cc.Command)
		default:
			return
		}
		sendReturn(rw, &botretvalresponse{int(ret)})
		return
	case "SetParameter":
		var param paramcall
		if !getArgs(rw, &f.FuncArgs, &param) {
			return
		}
		success := r.SetParameter(param.Name, param.Value)
		sendReturn(rw, boolresponse{Boolean: success})
	case "SetWorkingDirectory":
		var wd wdcall
		if !getArgs(rw, &f.FuncArgs, &wd) {
			return
		}
		success := r.SetWorkingDirectory(wd.Path)
		sendReturn(rw, boolresponse{Boolean: success})
	case "ExtendNamespace":
		var en extns
		if !getArgs(rw, &f.FuncArgs, &en) {
			return
		}
		success := r.ExtendNamespace(en.Extend, en.Histories)
		sendReturn(rw, boolresponse{Boolean: success})
	case "Exclusive":
		var e exclusive
		if !getArgs(rw, &f.FuncArgs, &e) {
			return
		}
		success := r.Exclusive(e.Tag, e.QueueTask)
		sendReturn(rw, boolresponse{Boolean: success})
	case "Elevate":
		var e elevate
		if !getArgs(rw, &f.FuncArgs, &e) {
			return
		}
		success := r.Elevate(e.Immediate)
		sendReturn(rw, boolresponse{Boolean: success})
		return
	case "CheckoutDatum":
		var rec recollection
		if !getArgs(rw, &f.FuncArgs, &rec) {
			return
		}
		var datum interface{}
		l, e, brv := r.CheckoutDatum(rec.Key, &datum, rec.RW)
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
		r.CheckinDatum(m.Key, m.Token)
		sendReturn(rw, &botretvalresponse{int(robot.Ok)})
		return
	case "UpdateDatum":
		var m memory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		var key string
		task, _, _ := getTask(c.currentTask)
		key = task.NameSpace + ":" + m.Key
		// Since we're getting raw JSON (=[]byte), we call update directly.
		// See brain.go
		ret = update(key, m.Token, (*[]byte)(&m.Datum))
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
		r.Remember(m.Key, m.Value)
		sendReturn(rw, &botretvalresponse{int(robot.Ok)})
		return
	case "Recall":
		var m shorttermrecollection
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		if m.Base64 {
			m.Key = decode(m.Key)
		}
		s := r.Recall(m.Key)
		sendReturn(rw, &stringresponse{s})
		return
	case "GetSecret":
		var sarg secname
		if !getArgs(rw, &f.FuncArgs, &sarg) {
			return
		}
		s := r.GetSecret(sarg.Secret)
		sendReturn(rw, &stringresponse{s})
		return
	case "GetTaskConfig":
		if task.Config == nil {
			Log(robot.Error, "GetTaskConfig called by external script '%s', but no config found.", task.name)
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
			attr = r.GetBotAttribute(a.Attribute)
		} else {
			attr = r.GetSenderAttribute(a.Attribute)
		}
		sendReturn(rw, attr)
		return
	case "GetUserAttribute":
		var ua userattr
		if !getArgs(rw, &f.FuncArgs, &ua) {
			return
		}
		attr = r.GetUserAttribute(ua.User, ua.Attribute)
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
		r.Log(l, lm.Message)
		sendReturn(rw, &botretvalresponse{int(robot.Ok)})
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
			int(r.SendChannelMessage(cm.Channel, cm.Message)),
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
			int(r.SendUserChannelMessage(ucm.User, ucm.Channel, ucm.Message)),
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
			int(r.SendUserMessage(um.User, um.Message)),
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
		reply, ret = r.promptInternal(rr.RegexID, rr.User, rr.Channel, rr.Prompt)
		sendReturn(rw, &replyresponse{reply, int(ret)})
		return
	// NOTE: "Say", "Reply", PromptForReply and PromptUserForReply are implemented
	// in the scripting libraries
	default:
		Log(robot.Error, "Bad function name: %s", f.FuncName)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
}

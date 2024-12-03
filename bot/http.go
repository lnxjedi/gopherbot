package bot

/* http.go translates posted JSON to Robot method calls, then packages
   and returns the JSON response.
*/

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/lnxjedi/gopherbot/robot"
)

type jsonFunction struct {
	FuncName string
	Format   string
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

type channelthreadmessage struct {
	Channel string
	Thread  string
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
	Base64      bool
}

type wdcall struct {
	Path string
}

// Something to be placed in ephemeral memory
type ephemeralmemory struct {
	Key, Value string
	Base64     bool
	Shared     bool
}

// Something to be recalled from ephemeral memory
type ephemeralrecollection struct {
	Key    string
	Base64 bool
	Shared bool
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

type userchannelthreadmessage struct {
	User    string
	Channel string
	Thread  string
	Message string
	Base64  bool
}

type replyrequest struct {
	RegexID string
	User    string
	Channel string
	Thread  string
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

func sendReturn(r robot.Robot, rw http.ResponseWriter, ret interface{}) {
	d, err := json.Marshal(ret)
	if err != nil { // this should never happen
		Log(robot.Fatal, "BUG in bot/http.go:sendReturn, error marshalling JSON: %v", err)
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(d)
	var respJSON []byte
	if len(d) > 256 {
		respJSON = d[:242]
		respJSON = append(respJSON, "... (truncated)"...)
	} else {
		respJSON = d
	}
	r.Log(robot.Debug, "http sending JSON response: %s", respJSON)
}

func logJSON(r robot.Robot, d *[]byte) {
	var obj map[string]interface{}
	json.Unmarshal(*d, &obj)
	formattedJSON, _ := json.Marshal(obj)
	if len(formattedJSON) > 256 {
		formattedJSON = formattedJSON[:242]
		formattedJSON = append(formattedJSON, "... (truncated)"...)
	}
	r.Log(robot.Debug, "http received raw JSON: %s", formattedJSON)
}

func (h handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	data, err := io.ReadAll(req.Body)
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

	// Look up the Robot
	taskLookup.RLock()
	r, ok := taskLookup.e[f.CallerID]
	taskLookup.RUnlock()
	logJSON(r, &data)
	if !ok {
		rw.WriteHeader(http.StatusBadRequest)
		Log(robot.Error, "JSON function '%s' called with invalid CallerID '%s'; args: %s", f.FuncName, f.CallerID, f.FuncArgs)
		return
	}
	if len(f.Format) > 0 {
		r.Format = setFormat(f.Format)
	} else {
		r.Format = r.cfg.defaultMessageFormat
	}
	task, _, _ := getTask(r.currentTask)
	Log(robot.Trace, "Task '%s' calling function '%s' in channel '%s' for user '%s'", task.name, f.FuncName, r.Channel, r.User)

	var (
		attr  *robot.AttrRet
		reply string
		ret   robot.RetVal
	)
	switch f.FuncName {
	case "CheckAdmin":
		bret := r.CheckAdmin()
		sendReturn(r, rw, boolresponse{Boolean: bret})
		return
	case "Subscribe":
		bret := r.Subscribe()
		sendReturn(r, rw, boolresponse{Boolean: bret})
		return
	case "Unsubscribe":
		bret := r.Unsubscribe()
		sendReturn(r, rw, boolresponse{Boolean: bret})
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
		sendReturn(r, rw, &botretvalresponse{int(ret)})
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
		sendReturn(r, rw, &botretvalresponse{int(ret)})
		return
	case "SetParameter":
		var param paramcall
		if !getArgs(rw, &f.FuncArgs, &param) {
			return
		}
		if param.Base64 {
			param.Name = decode(param.Name)
			param.Value = decode(param.Value)
		}
		success := r.SetParameter(param.Name, param.Value)
		sendReturn(r, rw, boolresponse{Boolean: success})
	case "SetWorkingDirectory":
		var wd wdcall
		if !getArgs(rw, &f.FuncArgs, &wd) {
			return
		}
		success := r.SetWorkingDirectory(wd.Path)
		sendReturn(r, rw, boolresponse{Boolean: success})
	case "Exclusive":
		var e exclusive
		if !getArgs(rw, &f.FuncArgs, &e) {
			return
		}
		success := r.Exclusive(e.Tag, e.QueueTask)
		sendReturn(r, rw, boolresponse{Boolean: success})
	case "Elevate":
		var e elevate
		if !getArgs(rw, &f.FuncArgs, &e) {
			return
		}
		success := r.Elevate(e.Immediate)
		sendReturn(r, rw, boolresponse{Boolean: success})
		return
	case "CheckoutDatum":
		var rec recollection
		if !getArgs(rw, &f.FuncArgs, &rec) {
			return
		}
		var datum interface{}
		l, e, brv := r.CheckoutDatum(rec.Key, &datum, rec.RW)
		sendReturn(r, rw, checkoutresponse{
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
		sendReturn(r, rw, &botretvalresponse{int(robot.Ok)})
		return
	case "UpdateDatum":
		var m memory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		var key string
		w := getLockedWorker(r.tid)
		w.Unlock()
		ns := w.getNameSpace(r.currentTask)
		key = ns + ":" + m.Key
		// Since we're getting raw JSON (=[]byte), we call update directly.
		// See brain.go
		ret = update(key, m.Token, (*[]byte)(&m.Datum))
		sendReturn(r, rw, &botretvalresponse{int(ret)})
		return
	case "Remember":
		var m ephemeralmemory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		if m.Base64 {
			m.Key = decode(m.Key)
			m.Value = decode(m.Value)
		}
		r.Remember(m.Key, m.Value, m.Shared)
		sendReturn(r, rw, &botretvalresponse{int(robot.Ok)})
		return
	case "RememberThread":
		var m ephemeralmemory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		if m.Base64 {
			m.Key = decode(m.Key)
			m.Value = decode(m.Value)
		}
		r.RememberThread(m.Key, m.Value, m.Shared)
		sendReturn(r, rw, &botretvalresponse{int(robot.Ok)})
		return
	case "Recall":
		var m ephemeralrecollection
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		if m.Base64 {
			m.Key = decode(m.Key)
		}
		s := r.Recall(m.Key, m.Shared)
		sendReturn(r, rw, &stringresponse{s})
		return
	case "GetTaskConfig":
		if task.Config == nil {
			Log(robot.Error, "GetTaskConfig called by external script '%s', but no config found.", task.name)
			sendReturn(r, rw, handler{})
			return
		}
		sendReturn(r, rw, task.Config)
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
		sendReturn(r, rw, attr)
		return
	case "GetUserAttribute":
		var ua userattr
		if !getArgs(rw, &f.FuncArgs, &ua) {
			return
		}
		attr = r.GetUserAttribute(ua.User, ua.Attribute)
		sendReturn(r, rw, attr)
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
		sendReturn(r, rw, &botretvalresponse{int(robot.Ok)})
		return
	case "SendChannelThreadMessage":
		var ctm channelthreadmessage
		if !getArgs(rw, &f.FuncArgs, &ctm) {
			return
		}
		if ctm.Base64 {
			ctm.Message = decode(ctm.Message)
		}
		sendReturn(r, rw, &botretvalresponse{
			int(r.SendChannelThreadMessage(ctm.Channel, ctm.Thread, ctm.Message)),
		})
		return
	case "SendUserChannelThreadMessage":
		var uctm userchannelthreadmessage
		if !getArgs(rw, &f.FuncArgs, &uctm) {
			return
		}
		if uctm.Base64 {
			uctm.Message = decode(uctm.Message)
		}
		sendReturn(r, rw, &botretvalresponse{
			int(r.SendUserChannelThreadMessage(uctm.User, uctm.Channel, uctm.Thread, uctm.Message)),
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
		sendReturn(r, rw, &botretvalresponse{
			int(r.SendUserMessage(um.User, um.Message)),
		})
		return
	case "PromptUserChannelThreadForReply":
		var rr replyrequest
		if !getArgs(rw, &f.FuncArgs, &rr) {
			return
		}
		if rr.Base64 {
			rr.Prompt = decode(rr.Prompt)
		}
		reply, ret = r.promptInternal(rr.RegexID, rr.User, rr.Channel, rr.Thread, rr.Prompt)
		sendReturn(r, rw, &replyresponse{reply, int(ret)})
		return
	// NOTE: "Say", "Reply", PromptForReply and PromptUserForReply are implemented
	// in the scripting libraries
	default:
		Log(robot.Error, "Bad function name: %s", f.FuncName)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
}

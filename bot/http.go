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
	PluginID string
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

type callpluginresponse struct {
	InterpreterPath string
	PluginPath      string
	PluginID        string
	PlugRetVal      int
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

	if f.PluginID == "" {
		rw.WriteHeader(http.StatusBadRequest)
		Log(Error, fmt.Sprintf("JSON function \"%s\" called with empty PluginID; args: %v", f.FuncName, f.FuncArgs))
		return
	}

	plugin := currentPlugins.getPluginByID(f.PluginID)
	if plugin == nil {
		rw.WriteHeader(http.StatusBadRequest)
		Log(Error, fmt.Sprintf("JSON function \"%s\" called with invalid PluginID \"%s\"; args: %s", f.FuncName, f.PluginID, f.FuncArgs))
		return
	}
	Log(Trace, fmt.Sprintf("Plugin \"%s\" calling function \"%s\" in channel \"%s\" for user \"%s\"", plugin.name, f.FuncName, f.Channel, f.User))

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
		var m memory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		// Since we're getting raw JSON (=[]byte), we call update directly.
		// See brain.go
		ret = update(plugin.name+":"+m.Key, m.Token, (*[]byte)(&m.Datum))
		sendReturn(rw, &botretvalresponse{int(ret)})
		return
	case "CallPlugin":
		var p plugincall
		if !getArgs(rw, &f.FuncArgs, &p) {
			return
		}
		calledPlugin := currentPlugins.getPluginByName(p.PluginName)
		if calledPlugin == nil {
			sendReturn(rw, &callpluginresponse{"", "", "", int(ConfigurationError)})
			return
		}
		plugAllowed := false
		if len(calledPlugin.TrustedPlugins) > 0 {
			for _, allowed := range calledPlugin.TrustedPlugins {
				if plugin.name == allowed {
					plugAllowed = true
					break
				}
			}
		}
		if plugAllowed {
			plugPath, err := getPluginPath(calledPlugin)
			if err != nil {
				Log(Error, fmt.Sprintf("Configuration error calling plugin \"%s\" from \"%s\": %s", calledPlugin.name, plugin.name, err))
				sendReturn(rw, &callpluginresponse{"", "", "", int(ConfigurationError)})
				return
			}
			interpreterPath, ierr := getInterpreter(plugPath)
			if ierr != nil {
				Log(Error, fmt.Sprintf("Couldn't get interpreter while calling plugin \"%s\" from \"%s\": %s", calledPlugin.name, plugin.name, err))
				sendReturn(rw, &callpluginresponse{"", "", "", int(MechanismFail)})
				return
			}
			Log(Debug, fmt.Sprintf("External plugin \"%s\" calling external plugin \"%s\"", plugin.name, calledPlugin.name))
			sendReturn(rw, &callpluginresponse{interpreterPath, plugPath, calledPlugin.pluginID, int(Success)})
		} else {
			Log(Error, fmt.Sprintf("Unable to call plugin \"%s\" from \"%s\": untrusted", calledPlugin.name, plugin.name))
			sendReturn(rw, &callpluginresponse{"", "", "", int(UntrustedPlugin)})
			return
		}
	case "Remember":
		var m shorttermmemory
		if !getArgs(rw, &f.FuncArgs, &m) {
			return
		}
		if m.Base64 {
			m.Value = decode(m.Value)
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

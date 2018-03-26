import os
import json
import random
import subprocess
import sys
import time
import urllib2

class Attribute:
    "A Gopherbot Attribute return object"
    def __init__(self, ret):
        self.attr = ret["Attribute"]
        self.ret = ret["RetVal"]

    def __str__(self):
        return self.attr

class Reply:
    "A Gopherbot Reply return object"
    def __init__(self, ret):
        self.reply = ret["Reply"]
        self.ret = ret["RetVal"]

    def __str__(self):
        return self.reply

class Memory:
    "A Gopherbot long-term memory object"
    def __init__(self, key, ret):
        self.key = key
        self.lock_token = ret["LockToken"]
        self.exists = ret["Exists"]
        self.datum = ret["Datum"]
        self.ret = ret["RetVal"]

class Robot:
    "Instantiate a robot object for use with Gopherbot"

    # Return values for robot method calls
    Ok = 0
    UserNotFound = 1
    ChannelNotFound = 2
    AttributeNotFound = 3
    FailedUserDM = 4
    FailedChannelJoin = 5
    DatumNotFound = 6
    DatumLockExpired = 7
    DataFormatError = 8
    BrainFailed = 9
    InvalidDatumKey = 10
    InvalidDblPtr = 11
    InvalidCfgStruct = 12
    NoConfigFound = 13
    RetryPrompt = 14
    ReplyNotMatched = 15
    UseDefaultValue = 16
    TimeoutExpired = 17
    Interrupted = 18
    MatcherNotFound = 19
    NoUserEmail = 20
    NoBotEmail = 21
    MailError = 22
    InvalidPluginID = 23
    UntrustedPlugin = 24

    # Plugin return values / exit codes, return values from CallPlugin
    Normal = 0
    Fail = 1
    MechanismFail = 2
    ConfigurationError = 3
    Success = 7

    def __init__(self):
        random.seed()
        self.channel = os.getenv("GOPHER_CHANNEL")
        self.user = os.getenv("GOPHER_USER")
        self.plugin_id = os.getenv("GOPHER_PLUGIN_ID")

    def Direct(self):
        "Get a direct messaging instance of the robot"
        return DirectBot()

    def Call(self, func_name, func_args, format="variable"):
        func_call = { "FuncName": func_name, "User": self.user,
                    "Channel": self.channel, "Format": format,
                    "PluginID": self.plugin_id, "FuncArgs": func_args }
        func_json = json.dumps(func_call)
        req = urllib2.Request(url="%s/json" % os.getenv("GOPHER_HTTP_POST"),
            data=func_json)
        req.add_header('Content-Type', 'application/json')
        # sys.stderr.write("Sending: %s\n" % func_json)
        f = urllib2.urlopen(req)
        body = f.read()
        # sys.stderr.write("Got back: %s\n" % body)
        return json.loads(body)

    def CheckAdmin(self):
        return self.Call("CheckAdmin", {})["Boolean"]

    def Elevate(self, immediate=False):
        return self.Call("Elevate", { "Immediate": immediate })["Boolean"]

    def CallPlugin(self, plugName, *plugArgs):
        ret = self.Call("CallPlugin", { "PluginName": plugName })
        if ret["PlugRetVal"] != self.Success:
            return ret["PlugRetVal"]
        plugenv = { "GOPHER_PLUGIN_ID": ret["PluginID"], "GOPHER_CHANNEL": self.channel, "GOPHER_USER": self.user, "GOPHER_INSTALLDIR": os.getenv("GOPHER_INSTALLDIR"), "GOPHER_HTTP_POST": os.getenv("GOPHER_HTTP_POST") }
        status = subprocess.call( [ ret["PluginPath"] ] + list(plugArgs), env=plugenv )
        return status

    def Pause(self, s):
        time.sleep(s)

    def RandomString(self, sa):
        return sa[random.randint(0, (len(sa)-1))]

    def GetPluginConfig(self):
        return self.Call("GetPluginConfig", {})

    def CheckoutDatum(self, key, rw):
        ret = self.Call("CheckoutDatum", { "Key": key, "RW": rw })
        return Memory(key, ret)

    def Log(self, level, msg):
        self.Call("Log", { "Level": level, "Message": msg })

    def CheckinDatum(self, m):
        self.Call("CheckinDatum", { "Key": m.key, "Token": m.lock_token })

    def UpdateDatum(self, m):
        ret = self.Call("UpdateDatum", { "Key": m.key, "Token": m.lock_token,
        "Datum": m.datum })
        return ret["RetVal"]

    def GetSenderAttribute(self, attr):
        ret = self.Call("GetSenderAttribute", { "Attribute": attr })
        return Attribute(ret)

    def GetUserAttribute(self, user, attr):
        ret = self.Call("GetUserAttribute", { "User": user, "Attribute": attr })
        return Attribute(ret)

    def GetBotAttribute(self, attr):
        ret = self.Call("GetBotAttribute", { "Attribute": attr })
        return Attribute(ret)

    def PromptForReply(self, regex_id, prompt):
        return self.PromptUserChannelForReply(regex_id, self.user, self.channel, prompt)

    def PromptUserForReply(self, regex_id, user, prompt):
        return self.PromptUserChannelForReply(regex_id, user, "", prompt)

    def PromptUserChannelForReply(self, regex_id, user, channel, prompt):
        for i in range(0, 3):
            rep = self.Call("PromptUserChannelForReply", { "RegexID": regex_id, "User": user, "Channel": channel, "Prompt": prompt })
            if rep["RetVal"] == self.RetryPrompt:
                continue
            return Reply(rep)
        if rep["RetVal"] == self.RetryPrompt:
            rep["RetVal"] = self.Interrupted
        return Reply(rep)

    def SendChannelMessage(self, channel, message, format="variable"):
        ret = self.Call("SendChannelMessage", { "Channel": channel,
        "Message": message })
        return ret["RetVal"]

    def SendUserMessage(self, user, message, format="variable"):
        ret = self.Call("SendUserMessage", { "User": user,
        "Message": message })
        return ret["RetVal"]

    def SendUserChannelMessage(self, user, channel, message, format="variable"):
        ret = self.Call("SendUserChannelMessage", { "User": user,
        "Channel": channel, "Message": message })
        return ret["RetVal"]

    def Say(self, message, format="variable"):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            return self.SendChannelMessage(self.channel, message, format)

    def Reply(self, message, format="variable"):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            return self.SendUserChannelMessage(self.user, self.channel, message, format)

class DirectBot(Robot):
    "Instantiate a robot for direct messaging with the user"
    def __init__(self):
        self.channel = ""
        self.user = os.getenv("GOPHER_USER")
        self.plugin_id = os.getenv("GOPHER_PLUGIN_ID")

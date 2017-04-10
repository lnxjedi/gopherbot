import os
import json
import random
import sys
import time
import urllib2
from base64 import b64encode, b64decode

def enc64(s):
    return "base64:%s" % b64encode(s)

def dec64(s):
    f = s.split(":")
    if f[0] == "base64":
        return b64decode(f[1])
    else:
        return s

class Attribute:
    "A Gopherbot Attribute return object"
    def __init__(self, ret):
        self.attr = dec64(ret["Attribute"])
        self.ret = ret["RetVal"]

    def __str__(self):
        return self.attr

class Reply:
    "A Gopherbot Reply return object"
    def __init__(self, ret):
        self.reply = dec64(ret["Reply"])
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
    TechnicalProblem = 14
    GeneralError = 15
    ReplyNotMatched = 16
    UseDefaultValue = 17
    TimeoutExpired = 18
    Interrupted = 19
    MatcherNotFound = 20
    NoUserEmail = 21
    NoBotEmail = 22
    MailError = 23

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

    def Pause(self, s):
        time.sleep(s)

    def RandomString(self, sa):
        return sa[random.randint(0, (len(sa)-1))]

    def GetPluginConfig(self):
        return self.Call("GetPluginConfig", {})

    def CheckoutDatum(self, key, rw):
        ret = self.Call("CheckoutDatum", { "Key": key, "RW": rw })
        return Memory(key, ret)

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

    def WaitForReply(self, regex_id, timeout=60):
        ret = self.Call("WaitForReply", { "RegexID": regex_id, "Timeout": timeout })
        return Reply(ret)

    def WaitForReplyRegex(self, regex, timeout=60):
        ret = self.Call("WaitForReplyRegex", { "RegEx": regex, "Timeout": timeout })
        return Reply(ret)

    def SendChannelMessage(self, channel, message, format="variable"):
        ret = self.Call("SendChannelMessage", { "Channel": channel,
        "Message": enc64(message) })
        return ret["RetVal"]

    def SendUserMessage(self, user, message, format="variable"):
        ret = self.Call("SendUserMessage", { "User": user,
        "Message": enc64(message) })
        return ret["RetVal"]

    def SendUserChannelMessage(self, user, channel, message, format="variable"):
        ret = self.Call("SendUserChannelMessage", { "User": user,
        "Channel": channel, "Message": enc64(message) })
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

import os
import json
import random
import subprocess
import sys
import time
import urllib2

# python 2 version

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
    FailedMessageSend = 4
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
    TaskNotFound = 23
    MissingArguments = 24
    InvalidStage = 25

    # Plugin return values / exit codes
    Normal = 0
    Fail = 1
    MechanismFail = 2
    ConfigurationError = 3
    NotFound = 6
    Success = 7

    def __init__(self):
        random.seed()
        self.channel = os.getenv("GOPHER_CHANNEL")
        self.user = os.getenv("GOPHER_USER")
        self.plugin_id = os.getenv("GOPHER_CALLER_ID")
        self.format = ""
        self.protocol = os.getenv("GOPHER_PROTOCOL")

    def Direct(self):
        "Get a direct messaging instance of the robot"
        return DirectBot(self)

    def MessageFormat(self, format):
        "Get a bot with a non-default message format"
        return FormattedBot(self, format)

    def Call(self, func_name, func_args, format=""):
        if len(format) == 0:
            format = self.format
        func_call = { "FuncName": func_name, "User": self.user,
                    "Channel": self.channel, "Format": format,
                    "Protocol": self.protocol, "CallerID": self.plugin_id,
                    "FuncArgs": func_args }
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

    def GetTaskConfig(self):
        return self.Call("GetTaskConfig", {})

    def CheckoutDatum(self, key, rw):
        ret = self.Call("CheckoutDatum", { "Key": key, "RW": rw })
        return Memory(key, ret)

    def SpawnJob(self, name, args):
        return self.Call("SpawnJob", { "Name": name, "CmdArgs": args })["RetVal"]

    def AddJob(self, name, args):
        return self.Call("AddJob", { "Name": name, "CmdArgs": args })["RetVal"]

    def AddTask(self, name, args):
        return self.Call("AddTask", { "Name": name, "CmdArgs": args })["RetVal"]

    def FinalTask(self, name, args):
        return self.Call("FinalTask", { "Name": name, "CmdArgs": args })["RetVal"]

    def FailTask(self, name, args):
        return self.Call("FailTask", { "Name": name, "CmdArgs": args })["RetVal"]

    def AddCommand(self, plugin, cmd):
        return self.Call("AddCommand", { "Plugin": plugin, "Command": cmd })["RetVal"]

    def FinalCommand(self, plugin, cmd):
        return self.Call("FinalCommand", { "Plugin": plugin, "Command": cmd })["RetVal"]

    def FailCommand(self, plugin, cmd):
        return self.Call("FailCommand", { "Plugin": plugin, "Command": cmd })["RetVal"]

    def SetParameter(self, name, value):
        return self.Call("SetParameter", { "Name": name, "Value": value })["Boolean"]

    def Exclusive(self, tag, queue_task=False):
        return self.Call("Exclusive", { "Tag": tag, "QueueTask": queue_task })["Boolean"]

    def ExtendNamespace(self, ns, hist):
        return self.Call("ExtendNamespace", { "Extend": ns, "Histories": hist })["Boolean"]

    def SetWorkingDirectory(self, path):
        return self.Call("SetWorkingDirectory", { "Path": path })["Boolean"]

    def GetRepoData(self):
        return self.Call("GetRepoData", {})

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

    def Recall(self, memory):
        ret = self.Call("Recall", { "Key": memory })
        return ret["StrVal"]

    def PromptForReply(self, regex_id, prompt, format=""):
        return self.PromptUserChannelForReply(regex_id, self.user, self.channel, prompt, format)

    def PromptUserForReply(self, regex_id, user, prompt, format=""):
        return self.PromptUserChannelForReply(regex_id, user, "", prompt, format)

    def PromptUserChannelForReply(self, regex_id, user, channel, prompt, format=""):
        for i in range(0, 3):
            rep = self.Call("PromptUserChannelForReply", { "RegexID": regex_id, "User": user, "Channel": channel, "Prompt": prompt }, format)
            if rep["RetVal"] == self.RetryPrompt:
                continue
            return Reply(rep)
        if rep["RetVal"] == self.RetryPrompt:
            rep["RetVal"] = self.Interrupted
        return Reply(rep)

    def SendChannelMessage(self, channel, message, format=""):
        ret = self.Call("SendChannelMessage", { "Channel": channel,
        "Message": message }, format)
        return ret["RetVal"]

    def SendUserMessage(self, user, message, format=""):
        ret = self.Call("SendUserMessage", { "User": user,
        "Message": message }, format)
        return ret["RetVal"]

    def SendUserChannelMessage(self, user, channel, message, format=""):
        ret = self.Call("SendUserChannelMessage", { "User": user,
        "Channel": channel, "Message": message }, format)
        return ret["RetVal"]

    def Say(self, message, format=""):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            return self.SendChannelMessage(self.channel, message, format)

    def Reply(self, message, format=""):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            return self.SendUserChannelMessage(self.user, self.channel, message, format)

class DirectBot(Robot):
    "Instantiate a robot for direct messaging with the user"
    def __init__(self, bot):
        self.channel = ""
        self.user = bot.user
        self.protocol = bot.protocol
        self.format = bot.format
        self.plugin_id = bot.plugin_id

class FormattedBot(Robot):
    "Instantiate a robot with a non-default message format"
    def __init__(self, bot, format):
        self.channel = bot.channel
        self.user = bot.user
        self.protocol = bot.protocol
        self.format = format
        self.plugin_id = bot.plugin_id

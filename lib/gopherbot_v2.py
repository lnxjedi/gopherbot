import os
import json
import random
import sys
import time
import urllib.request

# python 3 version

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
    InvalidConfigPointer = 11
    ConfigUnmarshalError = 12
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
    InvalidTaskType = 26
    CommandNotMatched = 27
    TaskDisabled = 28
    PrivilegeViolation = 29
    Failed = 63

    # Plugin return values / exit codes
    Normal = 0
    Fail = 1
    MechanismFail = 2
    ConfigurationError = 3
    NotFound = 6
    Success = 7

    # **Class-level attribute to store eid**
    _eid = None

    def __init__(self):
        random.seed()
        self.channel = os.getenv("GOPHER_CHANNEL")
        self.thread_id = os.getenv("GOPHER_THREAD_ID")
        self.threaded_message = os.getenv("GOPHER_THREADED_MESSAGE")
        self.user = os.getenv("GOPHER_USER")
        self.format = ""
        self.protocol = os.getenv("GOPHER_PROTOCOL")

        if Robot._eid is None:
            caller_id = os.getenv("GOPHER_CALLER_ID")
            if caller_id == "stdin":
                # Read the eid from stdin
                Robot._eid = sys.stdin.readline().strip()
                # Update the environment variable to indicate consumption
                os.environ["GOPHER_CALLER_ID"] = "read"
            else:
                Robot._eid = caller_id

        self.caller_id = Robot._eid

    def Call(self, func_name, func_args, format=""):
        if len(format) == 0:
            format = self.format
        func_call = { "FuncName": func_name, "Format": format,
                    "FuncArgs": func_args }
        data = json.dumps(func_call)
        data = bytes(data, 'utf-8')
        req = urllib.request.Request(url="%s/json" % os.getenv("GOPHER_HTTP_POST"),
            data=data)
        req.add_header('Content-Type', 'application/json')
        req.add_header('X-Caller-ID', self.caller_id)
        # sys.stderr.write("Sending: %s\n" % func_json)
        res = urllib.request.urlopen(req)
        body = res.read()
        # sys.stderr.write("Got back: %s\n" % body)
        return json.loads(body.decode("utf-8"))

    def CheckAdmin(self):
        return self.Call(sys._getframe().f_code.co_name, {})["Boolean"]

    def Subscribe(self):
        return self.Call(sys._getframe().f_code.co_name, {})["Boolean"]

    def Unsubscribe(self):
        return self.Call(sys._getframe().f_code.co_name, {})["Boolean"]

    def Elevate(self, immediate=False):
        return self.Call(sys._getframe().f_code.co_name, { "Immediate": immediate })["Boolean"]

    def Pause(self, s):
        time.sleep(s)

    def RandomString(self, sa):
        return sa[random.randint(0, (len(sa)-1))]

    def GetTaskConfig(self):
        return self.Call(sys._getframe().f_code.co_name, {})

    def CheckoutDatum(self, key, rw):
        ret = self.Call(sys._getframe().f_code.co_name, { "Key": key, "RW": rw })
        return Memory(key, ret)

    def SpawnJob(self, name, args):
        return self.Call(sys._getframe().f_code.co_name, { "Name": name, "CmdArgs": args })["RetVal"]

    def AddJob(self, name, args):
        return self.Call(sys._getframe().f_code.co_name, { "Name": name, "CmdArgs": args })["RetVal"]

    def AddTask(self, name, args):
        return self.Call(sys._getframe().f_code.co_name, { "Name": name, "CmdArgs": args })["RetVal"]

    def FinalTask(self, name, args):
        return self.Call(sys._getframe().f_code.co_name, { "Name": name, "CmdArgs": args })["RetVal"]

    def FailTask(self, name, args):
        return self.Call(sys._getframe().f_code.co_name, { "Name": name, "CmdArgs": args })["RetVal"]

    def AddCommand(self, plugin, cmd):
        return self.Call(sys._getframe().f_code.co_name, { "Plugin": plugin, "Command": cmd })["RetVal"]

    def FinalCommand(self, plugin, cmd):
        return self.Call(sys._getframe().f_code.co_name, { "Plugin": plugin, "Command": cmd })["RetVal"]

    def FailCommand(self, plugin, cmd):
        return self.Call(sys._getframe().f_code.co_name, { "Plugin": plugin, "Command": cmd })["RetVal"]

    def SetParameter(self, name, value):
        return self.Call(sys._getframe().f_code.co_name, { "Name": name, "Value": value })["Boolean"]

    def GetParameter(self, name):
        ret = self.Call(sys._getframe().f_code.co_name, { "Parameter": name })
        return ret["StrVal"]

    def Exclusive(self, tag, queue_task=False):
        return self.Call(sys._getframe().f_code.co_name, { "Tag": tag, "QueueTask": queue_task })["Boolean"]

    def SetWorkingDirectory(self, path):
        return self.Call(sys._getframe().f_code.co_name, { "Path": path })["Boolean"]

    def Log(self, level, msg):
        self.Call(sys._getframe().f_code.co_name, { "Level": level, "Message": msg })

    def CheckinDatum(self, m):
        self.Call(sys._getframe().f_code.co_name, { "Key": m.key, "Token": m.lock_token })

    def UpdateDatum(self, m):
        ret = self.Call(sys._getframe().f_code.co_name, { "Key": m.key, "Token": m.lock_token,
        "Datum": m.datum })
        return ret["RetVal"]

    def GetSenderAttribute(self, attr):
        ret = self.Call(sys._getframe().f_code.co_name, { "Attribute": attr })
        return Attribute(ret)

    def GetUserAttribute(self, user, attr):
        ret = self.Call(sys._getframe().f_code.co_name, { "User": user, "Attribute": attr })
        return Attribute(ret)

    def GetBotAttribute(self, attr):
        ret = self.Call(sys._getframe().f_code.co_name, { "Attribute": attr })
        return Attribute(ret)

    def Remember(k, v, shared=False):
        funcname = "RememberThread" if self.threaded_message else "Remember"
        ret = self.Call(funcname, { "Key": k, "Value": v, "Shared": shared })
        return ret["RetVal"]

    def RememberContext(k, v):
        return self.Remember("context:"+k, v, False)

    def RememberThread(k, v, shared=False):
        ret = self.Call(sys._getframe().f_code.co_name, { "Key": k, "Value": v, "Shared": shared })
        return ret["RetVal"]

    def RememberContextThread(k, v):
        return self.RememberThread("context:"+k, v, False)

    def Recall(self, memory, shared=False):
        ret = self.Call(sys._getframe().f_code.co_name, { "Key": memory, "Shared": shared })
        return ret["StrVal"]

    def PromptForReply(self, regex_id, prompt, format=""):
        thread = ""
        if self.threaded_message:
            thread = self.thread_id
        return self.PromptUserChannelThreadForReply(regex_id, self.user, self.channel, thread, prompt, format)

    def PromptThreadForReply(self, regex_id, prompt, format=""):
        return self.PromptUserChannelThreadForReply(regex_id, self.user, self.channel, self.thread_id, prompt, format)

    def PromptUserForReply(self, regex_id, user, prompt, format=""):
        return self.PromptUserChannelThreadForReply(regex_id, user, "", "", prompt, format)

    def PromptUserChannelThreadForReply(self, regex_id, user, channel, thread, prompt, format=""):
        for i in range(0, 3):
            rep = self.Call(sys._getframe().f_code.co_name, { "RegexID": regex_id, "User": user, "Channel": channel, "Thread": thread, "Prompt": prompt }, format)
            if rep["RetVal"] == self.RetryPrompt:
                continue
            return Reply(rep)
        if rep["RetVal"] == self.RetryPrompt:
            rep["RetVal"] = self.Interrupted
        return Reply(rep)

    def SendChannelMessage(self, channel, message, format=""):
        return self.SendChannelThreadMessage(channel, "", message, format)

    def SendChannelThreadMessage(self, channel, thread, message, format=""):
        ret = self.Call(sys._getframe().f_code.co_name, { "Channel": channel, "Thread": thread,
        "Message": message }, format)
        return ret["RetVal"]

    def SendUserMessage(self, user, message, format=""):
        ret = self.Call(sys._getframe().f_code.co_name, { "User": user,
        "Message": message }, format)
        return ret["RetVal"]

    def SendUserChannelMessage(self, user, channel, message, format=""):
        return self.SendUserChannelThreadMessage(user, channel, "", message, format)

    def SendProtocolUserChannelMessage(self, protocol, user, channel, message, format=""):
        ret = self.Call(sys._getframe().f_code.co_name, { "Protocol": protocol, "User": user, "Channel": channel, "Message": message }, format)
        return ret["RetVal"]

    def SendUserChannelThreadMessage(self, user, channel, thread, message, format=""):
        ret = self.Call(sys._getframe().f_code.co_name, { "User": user,
        "Channel": channel, "Thread": thread, "Message": message }, format)
        return ret["RetVal"]

    def Say(self, message, format=""):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            thread = ""
            if self.threaded_message:
                thread = self.thread_id
            return self.SendChannelThreadMessage(self.channel, thread, message, format)

    def SayThread(self, message, format=""):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            return self.SendChannelThreadMessage(self.channel, self.thread_id, message, format)

    def Reply(self, message, format=""):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            thread = ""
            if self.threaded_message:
                thread = self.thread_id
            return self.SendUserChannelThreadMessage(self.user, self.channel, thread, message, format)

    def ReplyThread(self, message, format=""):
        if self.channel == '':
            return self.SendUserMessage(self.user, message, format)
        else:
            return self.SendUserChannelThreadMessage(self.user, self.channel, self.thread_id, message, format)

    def Direct(self):
        "Get a direct messaging instance of the robot"
        return DirectBot(self)

    def MessageFormat(self, format):
        "Get a bot with a non-default message format"
        return FormattedBot(self, format)

    def Threaded(self):
        "Get a bot associated with the message thread"
        return ThreadedBot(self)

class DirectBot(Robot):
    "Instantiate a robot for direct messaging with the user"
    def __init__(self, bot):
        self.channel = ""
        self.thread_id = ""
        self.threaded_message = None
        self.user = bot.user
        self.protocol = bot.protocol
        self.format = bot.format
        self.caller_id = bot.caller_id

class FormattedBot(Robot):
    "Instantiate a robot with a non-default message format"
    def __init__(self, bot, format):
        self.channel = bot.channel
        self.thread_id = bot.thread_id
        self.threaded_message = bot.threaded_message
        self.user = bot.user
        self.protocol = bot.protocol
        self.format = format
        self.caller_id = bot.caller_id

class ThreadedBot(Robot):
    "Instantiate a robot with a non-default message format"
    def __init__(self, bot):
        self.channel = bot.channel
        self.thread_id = bot.thread_id
        if len(self.channel) > 0:
            self.threaded_message = "true"
        else:
            self.threaded_message = None
        self.user = bot.user
        self.protocol = bot.protocol
        self.format = bot.format
        self.caller_id = bot.caller_id

import os
import json
from base64 import b64encode, b64decode

def enc64(s):
    return "base64:%s" % b64encode(s)

def dec64(s):
    f = s.split(":")
    if f[0] == "base64":
        return b64decode(f[1])
    else:
        return s

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
        self.channel = os.getenv("GOPHER_CHANNEL")
        self.user = os.getenv("GOPHER_USER")
        self.plugin_id = os.getenv("GOPHER_PLUGIN_ID")

    def Direct(self):
        return DirectBot()

    def Call(self, func_name, func_args, format="variable"):
        func_call = { "FuncName": func_name, "User": self.user,
                    "Channel": self.channel, "Format": format,
                    "PluginID": self.plugin_id, "FuncArgs": func_args }
        func_json = json.dumps(func_call)
        print "Posting: %s" % func_json

class DirectBot(Robot):
    "Instantiate a robot for direct messaging with the user"
    def __init__(self):
        self.channel = ""
        self.user = os.getenv("GOPHER_USER")
        self.plugin_id = os.getenv("GOPHER_PLUGIN_ID")

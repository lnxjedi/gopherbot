require 'json'
require 'net/http'
require 'uri'

class Attribute
	def initialize(attr, ret)
		@attr = attr
		@ret = ret
	end

	attr_reader :attr, :ret

	def to_s
		@attr
	end
end

class Reply
	def initialize(reply, ret)
		@reply = reply
		@ret = ret
	end

	attr_reader :reply, :ret

	def to_s
		@reply
	end
end

class Memory
	def initialize(key, lt, exists, datum, ret)
		@key = key
		@lock_token = lt
		@exists = exists
		@datum = datum
		@ret = ret
	end

	attr_reader :key, :lock_token, :exists, :ret
	attr :datum, true
end

class BaseBot
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

	attr_reader :user, :channel

	def Direct()
	end

	def RandomString(sarr)
		return sarr[@prng.rand(sarr.size)]
	end

	def RandomInt(i)
		return @prng.rand(i)
	end

	def CheckAdmin()
		return callBotFunc("CheckAdmin", {})["Boolean"]
	end

	def Elevate(immediate=false)
		return callBotFunc("Elevate", { "Immediate" => immediate })["Boolean"]
	end

	def SpawnJob(name, args)
		return callBotFunc("SpawnJob", { "Name" => name, "CmdArgs" => args })["RetVal"]
	end

	def AddJob(name, args)
		return callBotFunc("AddJob", { "Name" => name, "CmdArgs" => args })["RetVal"]
	end

	def AddTask(name, args)
		return callBotFunc("AddTask", { "Name" => name, "CmdArgs" => args })["RetVal"]
	end

	def FinalTask(name, args)
		return callBotFunc("FinalTask", { "Name" => name, "CmdArgs" => args })["RetVal"]
	end

	def FailTask(name, args)
		return callBotFunc("FailTask", { "Name" => name, "CmdArgs" => args })["RetVal"]
	end

	def AddCommand(name, arg)
		return callBotFunc("AddCommand", { "Plugin" => name, "Command" => arg })["RetVal"]
	end

	def FinalCommand(name, arg)
		return callBotFunc("FinalCommand", { "Plugin" => name, "Command" => arg })["RetVal"]
	end

	def FailCommand(name, arg)
		return callBotFunc("FailCommand", { "Plugin" => name, "Command" => arg })["RetVal"]
	end

	def SetParameter(name, value)
		return callBotFunc("SetParameter", { "Name" => name, "Value" => value })["Boolean"]
	end

	def Exclusive(tag, queue_task=false)
		return callBotFunc("Exclusive", { "Tag" => tag, "QueueTask" => queue_task })["Boolean"]
	end

	def ExtendNamespace(ns, hist)
		return callBotFunc("ExtendNamespace", { "Extend" => ns, "Histories" => hist })["Boolean"]
	end

	def SetWorkingDirectory(path)
		return callBotFunc("SetWorkingDirectory", { "Path" => path })["Boolean"]
	end

	def GetRepoData()
		return callBotFunc("GetRepoData", {})
	end

	def CheckoutDatum(key, rw)
		args = { "Key" => key, "RW" => rw }
		ret = callBotFunc("CheckoutDatum", args)
		return Memory.new(key, ret["LockToken"], ret["Exists"], ret["Datum"], ret["RetVal"])
	end

	def CheckinDatum(m)
		args = { "Key" => m.key, "Token" => m.lock_token }
		callBotFunc("CheckinDatum", args)
		return 0
	end

	def UpdateDatum(m)
		args = { "Key" => m.key, "Token" => m.lock_token, "Datum" => m.datum }
		ret = callBotFunc("UpdateDatum", args)
		return ret["RetVal"]
	end

	def Remember(k, v)
		funcName = @threaded_message ? "RememberThread" : "Remember"
		args = { "Key" => k, "Value" => v }
		ret = callBotFunc(funcName, args)
		return ret["RetVal"]
	end

	def RememberContext(c, v)
		return Remember("context:"+c, v)
	end

	def RememberThread(k, v)
		args = { "Key" => k, "Value" => v }
		ret = callBotFunc("RememberThread", args)
		return ret["RetVal"]
	end

	def RememberContextThread(c, v)
		return RememberThread("context:"+c, v)
	end

	def Recall(k)
		args = { "Key" => k }
		return callBotFunc("Recall", args).StrVal
	end

	def GetTaskConfig()
		ret = callBotFunc("GetTaskConfig", {})
		return ret
	end

	def GetSenderAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return Attribute.new(ret["Attribute"], ret["RetVal"])
	end

	def GetUserAttribute(user, attr)
		args = { "User" => user, "Attribute" => attr }
		ret = callBotFunc("GetUserAttribute", args)
		return Attribute.new(ret["Attribute"], ret["RetVal"])
	end

	def GetBotAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetBotAttribute", args)
		return Attribute.new(ret["Attribute"], ret["RetVal"])
	end

	def Log(level, message)
		args = { "Level" => level, "Message" => message }
		callBotFunc("Log", args)
		return 0
	end

	def SendChannelMessage(channel, message, format="")
		return SendChannelThreadMessage(channel, "", message, format)
	end

	def SendChannelThreadMessage(channel, thread, message, format="")
		format = format.to_s if format.class == Symbol
		args = { "Channel" => channel, "Thread" => thread, "Message" => message }
		ret = callBotFunc("SendChannelThreadMessage", args, format)
		return ret["RetVal"]
	end

	def SendUserMessage(user, message, format="")
		format = format.to_s if format.class == Symbol
		args = { "User" => user, "Message" => message }
		ret = callBotFunc("SendUserMessage", args, format)
		return ret["RetVal"]
	end

	def SendUserChannelMessage(user, channel, message, format="")
		return SendUserChannelThreadMessage(user, channel, "", message, format)
	end

	def SendUserChannelThreadMessage(user, channel, thread, message, format="")
		format = format.to_s if format.class == Symbol
		args = { "User" => user, "Channel" => channel, "Thread" => thread, "Message" => message }
		ret = callBotFunc("SendUserChannelThreadMessage", args, format)
		return ret["RetVal"]
	end

	def Say(message, format="")
		format = format.to_s if format.class == Symbol
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			thread = @threaded_message ? @thread_id : ""
			return SendChannelThreadMessage(@channel, thread, message, format)
		end
	end

	def SayThread(message, format="")
		format = format.to_s if format.class == Symbol
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			return SendChannelThreadMessage(@channel, @thread_id, message, format)
		end
	end

	def Pause(seconds)
		sleep seconds
	end

	def Reply(message, format="")
		format = format.to_s if format.class == Symbol
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			thread = @threaded_message ? @thread_id : ""
			return SendUserChannelThreadMessage(@user, @channel, thread, message, format)
		end
	end

	def ReplyThread(message, format="")
		format = format.to_s if format.class == Symbol
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			return SendUserChannelThreadMessage(@user, @channel, @thread_id, message, format)
		end
	end

	def PromptForReply(regex_id, prompt)
		thread = @threaded_message ? @thread_id : ""
		return PromptUserChannelThreadForReply(regex_id, @user, @channel, thread, prompt)
	end

	def PromptThreadForReply(regex_id, prompt)
		return PromptUserChannelThreadForReply(regex_id, @user, @channel, @thread_id, prompt)
	end

	def PromptUserForReply(regex_id, prompt)
		return PromptUserChannelThreadForReply(regex_id, @user, "", "", prompt)
	end

	def PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt)
		args = { "RegexID" => regex_id, "User" => user, "Channel" => channel, "Thread" => thread, "Prompt" => prompt }
		for i in 1..3
			ret = callBotFunc("PromptUserChannelThreadForReply", args)
			next if ret["RetVal"] == RetryPrompt
			return Reply.new(ret["Reply"], ret["RetVal"])
		end
		if ret == RetryPrompt
			return Reply.new(ret["Reply"], Interrupted)
		else
			return Reply.new(ret["Reply"], ret["RetVal"])
		end
	end

	def callBotFunc(funcname, args, format="")
		if format.size == 0
			format = @format
		end
		func = {
			"FuncName" => funcname,
			"Format" => format,
			"CallerID" => @plugin_id,
			"FuncArgs" => args
		}
		uri = URI.parse(ENV["GOPHER_HTTP_POST"] + "/json")
		http = Net::HTTP.new(uri.host, uri.port)
		req = Net::HTTP::Post.new(uri, initheader = {'Content-Type' =>'application/json'})
		req.body = func.to_json
#		STDERR.puts "Sending:\n#{req.body}"
		res = http.request(req)
		body = res.body()
#		STDERR.puts "Got back:\n#{body}"
		return JSON.load(body)
	end
	private :callBotFunc
end

class Robot < BaseBot
	attr_accessor :channel, :thread_id, :threaded_message, :user, :plugin_id, :protocol, :format
	def initialize()
		@channel = ENV["GOPHER_CHANNEL"]
        @thread_id = ENV["GOPHER_THREAD_ID"]
		@threaded_message = ENV["GOPHER_THREADED_MESSAGE"]
		@user = ENV["GOPHER_USER"]
		@plugin_id = ENV["GOPHER_CALLER_ID"]
		@protocol = ENV["GOPHER_PROTOCOL"]
		@format = ""
		@prng = Random.new
	end

	def Direct()
		DirectBot.new
	end

	def Threaded()
		ThreadedBot.new
	end

	def MessageFormat(format)
		FormattedBot.new(format)
	end
end

class DirectBot < Robot
	def initialize()
		super
		@channel = ""
		@thread_id = ""
		@threaded_message = nil
	end
end

class ThreadedBot < Robot
	def initialize()
		super
		if @channel.length > 0
			@threaded_message = "true"
		else
			@threaded_message = nil
		end
	end
end

class FormattedBot < BaseBot
	def initialize(format)
		super
		@format = format
	end
end

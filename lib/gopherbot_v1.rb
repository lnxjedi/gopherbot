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

	def CallPlugin(plugName, *plugargs)
		args = { "PluginName" => plugName }
		ret = callBotFunc("CallPlugin", args)
		if ret["PlugRetVal"] != Success
			return ret["PlugRetVal"]
		end
		system({ 'GOPHER_PLUGIN_ID' => ret["PluginID"] }, ret["PluginPath"], *plugargs)
		return $?.exitstatus
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
		args = { "Key" => k, "Value" => v }
		ret = callBotFunc("Remember", args)
		return ret["RetVal"]
	end

	def RememberContext(c, v)
		return Remember("context:"+c, v)
	end

	def Recall(k)
		args = { "Key" => k }
		return callBotFunc("Recall", args).StrVal
	end

	def GetPluginConfig()
		ret = callBotFunc("GetPluginConfig", {})
		return ret
	end

	def GetSenderAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return Attribute.new(ret["Attribute"], ret["RetVal"])
	end

	def GetUserAttribute(user, attr)
		args = { "User" => user, "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return Attribute.new(ret["Attribute"], ret["RetVal"])
	end

	def GetBotAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return Attribute.new(ret["Attribute"], ret["RetVal"])
	end

	def Log(level, message)
		args = { "Level" => level, "Message" => message }
		callBotFunc("Log", args)
		return 0
	end

	def SendChannelMessage(channel, message, format="variable")
		format = format.to_s if format.class == Symbol
		args = { "Channel" => channel, "Message" => message }
		ret = callBotFunc("SendChannelMessage", args, format)
		return ret["RetVal"]
	end

	def SendUserMessage(user, message, format="variable")
		format = format.to_s if format.class == Symbol
		args = { "User" => user, "Message" => message }
		ret = callBotFunc("SendUserMessage", args, format)
		return ret["RetVal"]
	end

	def SendUserChannelMessage(user, channel, message, format="variable")
		format = format.to_s if format.class == Symbol
		args = { "User" => user, "Channel" => channel, "Message" => message }
		ret = callBotFunc("SendUserChannelMessage", args, format)
		return ret["RetVal"]
	end

	def Say(message, format="variable")
		format = format.to_s if format.class == Symbol
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			return SendChannelMessage(@channel, message, format)
		end
	end

	def Pause(seconds)
		sleep seconds
	end

	def Reply(message, format="variable")
		format = format.to_s if format.class == Symbol
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			return SendUserChannelMessage(@user, @channel, message, format)
		end
	end

	def PromptForReply(regex_id, prompt)
		return PromptUserChannelForReply(regex_id, @user, @channel, prompt)
	end

	def PromptUserForReply(regex_id, prompt)
		return PromptUserChannelForReply(regex_id, user, "", prompt)
	end

	def PromptUserChannelForReply(regex_id, user, channel, prompt)
		args = { "RegexID" => regex_id, "User" => user, "Channel" => channel, "Prompt" => prompt }
		for i in 1..3
			ret = callBotFunc("PromptUserChannelForReply", args)
			next if ret["RetVal"] == RetryPrompt
			return Reply.new(ret["Reply"], ret["RetVal"])
		end
		if ret == RetryPrompt
			return Reply.new(ret["Reply"], Interrupted)
		else
			return Reply.new(ret["Reply"], ret["RetVal"])
		end
	end

	def callBotFunc(funcname, args, format="variable")
		func = {
			"FuncName" => funcname,
			"User" => @user,
			"Channel" => @channel,
			"Protocol" => @protocol,
			"Format" => format,
			"PluginID" => @plugin_id,
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

	def initialize()
		@channel = ENV["GOPHER_CHANNEL"]
		@user = ENV["GOPHER_USER"]
		@plugin_id = ENV["GOPHER_PLUGIN_ID"]
		@protocol = ENV["GOPHER_PROTOCOL"]
		@prng = Random.new
	end

	def Direct()
		DirectBot.new(@user, @plugin_id, @prng)
	end
end

class DirectBot < BaseBot

	def initialize(user, plugin_id, prng)
		@channel = ""
		@user = user
		@plugin_id = plugin_id
		@prng = prng
	end

end

require 'base64'
require 'json'
require 'net/http'
require 'uri'

# Make base64 a little more accessible
class String
	def to_base64
		Base64.encode64(self).chomp
	end
end

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

class OTPRet
	def initialize(valid, ret)
		@valid = valid
		@ret= ret
	end

	attr_reader :valid, :ret
end

class Robot
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
	UntrustedPlugin = 13
	NoUserOTP = 14
	OTPError = 15
	ReplyNotMatched = 16
	TimeoutExpired = 17
	ReplyInProgress = 18
	MatcherNotFound = 19
	NoUserEmail = 20
	NoBotEmail = 21
	MailError = 22

	def initialize()
		@channel = ARGV[0]
		@user = ARGV[1]
		@plugin_id = ARGV[2]
		ARGV.shift(3)
		@prng = Random.new
	end

	attr_reader :user, :channel

	def RandomString(sarr)
		return sarr[@prng.rand(sarr.size)]
	end

	def RandomInt(i)
		return @prng.rand(i)
	end

	def CheckAdmin()
		return callBotFunc("CheckAdmin", {})["Boolean"]
	end

	def CheckOTP(code)
		args = { "Code" => code }
		ret = callBotFunc("CheckOTP", args)
		return OTPRet.new(ret["Boolean"], ret["BotRetVal"])
	end

	def CheckoutDatum(key, rw)
		args = { "Key" => key, "RW" => rw }
		ret = callBotFunc("CheckoutDatum", args)
		return Memory.new(key, ret["LockToken"], ret["Exists"], ret["Datum"], ret["BotRetVal"])
	end

	def CheckinDatum(m)
		args = { "Key" => m.key, "Token" => m.lock_token }
		callBotFunc("CheckinDatum", args)
		return 0
	end

	def UpdateDatum(m)
		args = { "Key" => m.key, "Token" => m.lock_token, "Datum" => m.datum }
		ret = callBotFunc("UpdateDatum", args)
		return ret["BotRetVal"]
	end

	def GetPluginConfig()
		ret = callBotFunc("GetPluginConfig", {})
		return ret
	end

	def GetSenderAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return Attribute.new(decode(ret["Attribute"]), ret["BotRetVal"])
	end

	def GetUserAttribute(user, attr)
		args = { "User" => user, "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return Attribute.new(decode(ret["Attribute"]), ret["BotRetVal"])
	end

	def GetBotAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return Attribute.new(decode(ret["Attribute"]), ret["BotRetVal"])
	end

	def Log(level, message)
		args = { "Level" => level, "Message" => message }
		callBotFunc("Log", args)
		return 0
	end

	def SendChannelMessage(channel, message, format="variable")
		args = { "Channel" => channel, "Message" => "base64:" + message.to_base64, "Format" => format }
		ret = callBotFunc("SendChannelMessage", args, format)
		return ret["BotRetVal"]
	end

	def SendUserMessage(user, message, format="variable")
		args = { "User" => user, "Message" => "base64:" + message.to_base64, "Format" => format }
		ret = callBotFunc("SendUserMessage", args, format)
		return ret["BotRetVal"]
	end

	def SendUserChannelMessage(user, channel, message, format="variable")
		args = { "User" => user, "Channel" => channel, "Message" => "base64:" + message.to_base64, "Format" => format }
		ret = callBotFunc("SendChannelMessage", args, format)
		return ret["BotRetVal"]
	end

	def Say(message, format="variable")
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
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			return SendUserChannelMessage(@user, @channel, message, format)
		end
	end

	def WaitForReply(re, timeout=30)
		args = { "RegExId" => re, "Timeout" => timeout }
		ret = callBotFunc("WaitForReply", args)
		return Reply.new(decode(ret["Reply"]), ret["BotRetVal"])
	end

	def decode(str)
		if str.start_with?("base64:")
			if bstr = str.split(':')[1]
				return Base64.decode64(bstr)
			else
				return ""
			end
		else
			return str
		end
	end
	private :decode

	def callBotFunc(funcname, args, format="variable")
		func = {
			"FuncName" => funcname,
			"User" => @user,
			"Channel" => @channel,
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
		return JSON.load(res.body())
	end
	private :callBotFunc
end

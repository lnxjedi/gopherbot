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

class GBAttribute
	def initialize(attr, ret)
		@attr = attr
		@ret = ret
	end

	attr_reader :attr, :ret

	def to_s
		@attr
	end
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
	end

	attr_reader :user, :channel

	def GetSenderAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return GBAttribute.new(decode(ret["Attribute"]), ret["BotRetVal"])
	end

	def GetUserAttribute(user, attr)
		args = { "User" => user, "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return GBAttribute.new(decode(ret["Attribute"]), ret["BotRetVal"])
	end

	def GetBotAttribute(attr)
		args = { "Attribute" => attr }
		ret = callBotFunc("GetSenderAttribute", args)
		return GBAttribute.new(decode(ret["Attribute"]), ret["BotRetVal"])
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
	
	def Reply(message, format="variable")
		if @channel.empty?
			return SendUserMessage(@user, message, format)
		else
			return SendUserChannelMessage(@user, @channel, message, format)
		end
	end

	def decode(str)
		if str.start_with?("base64:")
			return Base64.decode64(str.split(':')[1])
		else
			return str
		end
	end

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
		STDERR.puts "Sending:\n#{req.body}"
		res = http.request(req)
		body = res.body()
		STDERR.puts "Got back:\n#{body}"
		return JSON.parse(res.body())
	end

	private :callBotFunc
end

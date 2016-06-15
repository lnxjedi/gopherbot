require 'json'
require 'base64'

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
		args = { "Attribute" => "base64:" + attr.to_base64 }
		callBotFunc("GetSenderAttribute", args)
		return GBAttribute.new("David", Robot::Ok)
	end

	def callBotFunc(funcname, args)
		func = {
			"FuncName" => funcname,
			"User" => @user,
			"Channel" => @channel,
			"PluginID" => @plugin_id,
			"FuncArgs" => args
		}
		print JSON.pretty_generate(func)
		print func.to_json
	end

	private :callBotFunc
end

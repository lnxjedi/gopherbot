#!/usr/bin/ruby

require 'gopherbot_v1'

default_config = <<'DEFCONFIG'
---
AllChannels: true
AllowedPrivateCommands:
- "*"
DEFCONFIG

command = ARGV.shift()

case command
when "configure"
  puts default_config
  exit 0
when "init"
  exit 0
end

bot = Robot.new()

if command == "encryptsecret"
  ciphertext, ret = bot.EncryptSecret("test-secret")
  if ret == Robot::Ok && !ciphertext.to_s.empty?
    bot.Say("ENCRYPT SECRET: ok")
  else
    bot.Say("ENCRYPT SECRET: failed")
  end
  exit 0
end

exit 1

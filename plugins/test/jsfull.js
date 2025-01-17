// hello.js
// JavaScript plugin "Hello World" and boilerplate

// Define the default configuration as a YAML string
const defaultConfig = `---
Help:
  - Keywords: [ "say" ]
    Helptext: [ "(bot), say everything - full test of Say*/Reply*/Send* methods" ]
CommandMatchers:
  - Regex: (?i:say everything)
    Command: sendmsg
`;

// Require the Gopherbot JavaScript library
const { Robot, ret, task, log, fmt, proto } = require('gopherbot_v1')();

function handler(argv) {
  // 0: the path to gopherbot, the js interpreter
  // 1: the path to the *.js source file
  // 2+: arguments
  const cmd = argv.length > 2 ? argv[2] : '';

  switch (cmd) {
    case 'init':
      // Initialization logic if needed (usually not required for simple plugins)
      return task.Normal;

    case 'configure':
      // Return the default configuration
      return defaultConfig;

    case 'sendmsg':
      const bot = new Robot();
      bot.Say("Regular Say");
      bot.SayThread("SayThread, yeah")
      bot.Reply("Regular Reply")
      bot.ReplyThread("Reply in thread, yo")
      bot.SendChannelMessage(bot.channel, "Sending to the channel: " + bot.channel)
      bot.SendUserMessage(bot.user, "Sending this message to user: " + bot.user)
      bot.SendUserChannelMessage(bot.user, bot.channel, "Sending to user '" + bot.user + "' in channel: " + bot.channel)
      bot.SendChannelThreadMessage(bot.channel, bot.thread_id, "Sending to channel '" + bot.channel + "' in thread: " + bot.thread_id)
      bot.SendUserChannelThreadMessage(bot.user, bot.channel, bot.thread_id, "Sending to user '" + bot.user + "' in channel '" + bot.channel + "' in thread: " + bot.thread_id)
      return task.Normal;

    default:
      // const botDefault = robot.New();
      // botDefault.Log(log.Error, `JavaScript plugin received unknown command: ${cmd}`);
      return task.Fail;
  }
}

// @ts-ignore - the "process" object is created by goja
handler(process.argv || []); // Call the function with process.argv

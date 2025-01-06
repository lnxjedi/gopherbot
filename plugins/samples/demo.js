// demo.js
// Example JavaScript plugin that exercises more functionality.

// Define the default configuration as a YAML string
const defaultConfig = `---
Help:
- Keywords: [ "js" ]
  Helptext: [ "(bot), js (me!) - prove that JavaScript plugins work" ]
- Keywords: [ "listen" ]
  Helptext: [ "(bot), listen (to me?) - ask a question" ]
- Keywords: [ "thread" ]
  Helptext: [ "(bot), js-thread - ask the robot to start a new thread" ]
- Keywords: [ "thread" ]
  Helptext: [ "(bot), js-ask-thread - ask a question in a thread" ]
- Keywords: [ "remember", "memory" ]
  Helptext: [ "(bot), remember <anything> - prove the robot has a brain(tm)" ]
- Keywords: [ "recall", "memory" ]
  Helptext: [ "(bot), recall - list or recall a certain memory" ]
- Keywords: [ "forget", "memory" ]
  Helptext: [ "(bot), forget <#> - ask the robot to forget one of its remembered 'facts'" ]
- Keywords: [ "check" ]
  Helptext: [ "(bot), check me - get the bot to check you out" ]
CommandMatchers:
- Regex: (?i:js( me)?!?)
  Command: js
- Regex: (?i:js-thread)
  Command: thread
- Regex: (?i:js-ask-thread)
  Command: askthread
- Regex: (?i:listen( to me)?!?)
  Command: listen
- Regex: '(?i:remember(?: (slowly))? ([-\w .,!?:\/]+))'
  Command: remember
  Contexts: [ "", "item" ]
- Regex: (?i:recall ?([\d]+)?)
  Command: recall
- Regex: (?i:forget ([\d]{1,2}))
  Command: forget
- Regex: (?i:check me)
  Command: check
Config:
  Replies:
  - "Consider yourself caffeinated..."
  - "Waaaaaait a second... what do you mean by that?"
  - "I'll javascript you alright, but not right now - I'll wait 'til you're least expecting it..."
  - "Crap, sorry - all out of coffee beans"
`;

// Require the Gopherbot JavaScript library
const { Robot, ret, task, log, fmt, proto } = require('gopherbot_v1')();

function handler(argv) {
  // 0: the path to gopherbot, the js interpreter
  // 1: the path to the *.js source file
  // 2+: arguments
  const cmd = argv.length > 2 ? argv[2] : '';

  const bot = new Robot()

  switch (cmd) {
    case 'init':
      // Initialization logic if needed (usually not required for simple plugins)
      return task.Normal;

    case 'configure':
      // Return the default configuration
      return defaultConfig;

    case 'js': // js me!
      const cfg = bot.GetTaskConfig();
      if (cfg.retVal != ret.Ok) {
        bot.Say("Uh-oh, I wasn't able to find any configuration")
        return task.Normal
      }
      const replies = cfg.config.Replies
      if (replies) {
        bot.Say(bot.RandomString(replies))
      } else {
        bot.Say("I'm speechless")
      }
      return task.Normal
    case 'thread': // js-thread
      bot.ReplyThread("Sure, let's start a new thread")
      const nameAttr = bot.GetBotAttribute("name")
      bot.SayThread("We can talk about ANYTHING here - my name is " + nameAttr.attribute)
      return task.Normal
    case 'askthread': // js-ask-thread
      bot.Say("Lemme ask you a question ...")
      const repObj = bot.PromptThreadForReply("SimpleString", "Whadaya know, Jack?")
      if (repObj.retVal == ret.Ok) {
        bot.SayThread("I hear what you're saying - " + repObj.reply)
      } else {
        bot.SayThread("Eh, I had a problem hearing you? - " + ret.string(repObj.retVal))
      }
      return task.Normal
    case 'listen': // listen to me
    case 'remember': // remember <something>
    case 'recall': // recall everything or one item
    case 'forget': // forget an item
    case 'check': // check me
    default:
      // const botDefault = robot.New();
      // botDefault.Log(log.Error, `JavaScript plugin received unknown command: ${cmd}`);
      return task.Fail;
  }
}

// @ts-ignore - the "process" object is created by goja
handler(process.argv || []); // Call the function with process.argv

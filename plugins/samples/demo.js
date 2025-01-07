// demo.js
// Example JavaScript plugin that exercises more functionality using a 
// "switch" statement to dispatch commands. 

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
- Regex: '(?i:remember(?: (slowly))? ([-\\w .,!?:\\/]+))'
  Command: remember
  Contexts: [ "", "item" ]
- Regex: (?i:recall ?([\\d]+)?)
  Command: recall
- Regex: (?i:forget ([\\d]{1,2}))
  Command: forget
- Regex: (?i:check me)
  Command: check
Config:
  Replies:
  - "Consider yourself caffeinated..."
  - "Waaaaaait a second... what do you mean by that?"
  - "I'll javascript you alright, but not right now - I'll wait 'til you're least expecting it..."
  - "Crap, sorry - all out of coffee beans"
  - "Oh really! Have you considered asking HUBOT for that?!?"
`;

// Require the Gopherbot JavaScript library
const { Robot, ret, task, log, fmt, proto } = require('gopherbot_v1')();

function handler(argv) {
  // 0: the path to gopherbot, the js interpreter
  // 1: the path to the *.js source file
  // 2+: arguments
  const cmd = argv.length > 2 ? argv[2] : '';

  // Create a Robot instance for handling commands
  const bot = new Robot();

  switch (cmd) {
    // -------------------------------------------------------------------------
    // init & configure
    // -------------------------------------------------------------------------
    case 'init':
      // Initialization logic if needed
      return task.Normal;

    case 'configure':
      // Return the default YAML configuration
      return defaultConfig;

    // -------------------------------------------------------------------------
    // js (the primary command)
    // -------------------------------------------------------------------------
    case 'js':
      {
        // Example showing usage of config data
        const cfg = bot.GetTaskConfig();
        if (cfg.retVal != ret.Ok) {
          bot.Say("Uh-oh, I wasn't able to find any configuration");
          return task.Normal;
        }
        const replies = cfg.config.Replies;
        if (replies && replies.length) {
          bot.Say(bot.RandomString(replies));
        } else {
          bot.Say("I'm speechless");
        }
        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // thread (js-thread)
    // -------------------------------------------------------------------------
    case 'thread':
      {
        // Reply in a new thread
        bot.ReplyThread("Sure, let's start a new thread");
        const nameAttr = bot.GetBotAttribute("name");
        bot.SayThread(
          "We can talk about ANYTHING here - my name is " + nameAttr.attribute
        );
        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // askthread (js-ask-thread)
    // -------------------------------------------------------------------------
    case 'askthread':
      {
        bot.Say("Lemme ask you a question ...");
        const repObj = bot.PromptThreadForReply("SimpleString", "Whadaya know, Jack?");
        if (repObj.retVal === ret.Ok) {
          bot.SayThread("I hear what you're saying - " + repObj.reply);
        } else {
          bot.SayThread("Eh, I had a problem hearing you? - " + ret.string(repObj.retVal));
        }
        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // listen (listen to me)
    // -------------------------------------------------------------------------
    case 'listen':
      {
        // Demonstrate a DM-based prompt
        const directBot = bot.Direct();
        const result = directBot.PromptForReply(
          "SimpleString", 
          "Ok, what do you want to tell me?"
        );
        if (result.retVal === ret.Ok) {
          directBot.Say("I hear what you're saying: '" + result.reply + "'");
        } else {
          bot.Say(
            "Sorry, I'm not sure I caught that. Maybe you used funny characters?"
          );
        }
        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // remember (like "remember <anything>")
    // Optional: "remember slowly <anything>"
    // -------------------------------------------------------------------------
    case 'remember':
      {
        // Argv might look like: ["gopherbot", "demo.js", "remember", "slowly", "the cat?"]
        // or just ["...","...","remember","the cat?"]
        const speed = argv[3] && argv[3].toLowerCase() === "slowly" ? "slowly" : "";
        const thing = argv[4] || "";

        if (!thing) {
          bot.Say("Huh, not sure what you want me to remember?");
          return task.Normal;
        }

        // Use CheckoutDatum to read/write "memory"
        const memObj = bot.CheckoutDatum("memory", true);
        if (memObj.retVal != ret.Ok) {
          bot.Say("Sorry, I'm having trouble checking out my memory.");
          return task.Fail;
        }

        // If the datum doesn't exist, create it
        if (!memObj.exists) {
          memObj.datum = [];
        }

        // We'll store an array of strings
        const data = memObj.datum;
        // Check if we already have `thing`
        const found = data.includes(thing);

        if (found) {
          bot.Say("That's already one of my fondest memories.");
          bot.CheckinDatum(memObj); // We don't need to Update if we didn't change anything
        } else {
          data.push(thing);
          if (speed === "slowly") {
            bot.Say(`Ok, I'll remember "${thing}" ... but sloooowly`);
            bot.Pause(4);
          } else {
            bot.Say(`Ok, I'll remember "${thing}"`);
          }
          const updRet = bot.UpdateDatum(memObj);
          if (updRet === ret.Ok) {
            if (!speed) {
              bot.Say("committed to memory");
            }
          } else {
            if (!speed) {
              bot.Say("Dang it, having problems with my memory");
            }
          }
        }
        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // recall (recall everything or one item)
    // e.g., "recall" or "recall 2"
    // -------------------------------------------------------------------------
    case 'recall':
      {
        // The user might specify a number to recall a specific index
        const which = argv[3] || "";
        const memObj = bot.CheckoutDatum("memory", false);
        bot.Log(log.Info, "Checking in: " + JSON.stringify(memObj))
        if (memObj.retVal != ret.Ok) {
          bot.Say("Sorry - trouble checking memory!");
          return task.Fail;
        }
        const data = memObj.datum;
        if (data && data.length > 0) {
          if (which) {
            const idx = parseInt(which, 10);
            if (isNaN(idx) || idx < 1) {
              bot.Say("I can't make out which item you want me to recall.");
              bot.CheckinDatum(memObj);
              return task.Normal;
            }
            if (idx > data.length) {
              bot.Say("I don't remember that many things!");
              bot.CheckinDatum(memObj);
              return task.Normal;
            }
            const item = data[idx - 1]; // zero-based
            bot.CheckinDatum(memObj);
            bot.Say(item);
          } else {
            // If no index given, list them all
            let reply = "Here's what I remember:\n";
            data.forEach((item, i) => {
              reply += `${i + 1}: ${item}\n`;
            });
            bot.CheckinDatum(memObj);
            bot.Say(reply);
          }
        } else {
          bot.CheckinDatum(memObj);
          bot.Say("Sorry - I don't remember anything!");
        }
        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // forget (forget <itemIndex>)
    // e.g., "forget 2"
    // -------------------------------------------------------------------------
    case 'forget':
      {
        const which = argv[3] || "";
        const idx = parseInt(which, 10);
        if (isNaN(idx) || idx < 1) {
          bot.Say("I can't make out what you want me to forget.");
          return task.Normal;
        }

        // zero-based index
        const arrayIndex = idx - 1;

        const memObj = bot.CheckoutDatum("memory", true);
        if (memObj.retVal != ret.Ok) {
          bot.Say("Sorry - trouble checking memory!");
          return task.Fail;
        }
        const data = memObj.datum;
        if (data && data[arrayIndex]) {
          const item = data[arrayIndex];
          bot.Say(`Ok, I'll forget "${item}"`);
          data.splice(arrayIndex, 1);
          const updRet = bot.UpdateDatum(memObj);
          if (updRet != ret.Ok) {
            bot.Say("Hmm, having trouble forgetting that item for real, sorry.");
          }
        } else {
          bot.CheckinDatum(memObj);
          bot.Say("Gosh, I guess I never remembered that in the first place!");
        }
        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // check (check me)
    // -------------------------------------------------------------------------
    case 'check':
      {
        const isAdmin = bot.CheckAdmin();
        if (isAdmin) {
          bot.Say("It looks like you're an administrator.");
        } else {
          bot.Say("Well, you're not an administrator.");
        }
        bot.Pause(1);
        bot.Say("Now I'll request elevation...");

        const success = bot.Elevate(true);
        if (success) {
          bot.Say("Everything looks good, mac!");
        } else {
          bot.Say("You failed to elevate, I'm calling the cops!");
        }

        // Log some details
        bot.Log(
          log.Info,
          `Checked out ${bot.user}, admin: ${isAdmin}, elevate: ${success}`
        );

        return task.Normal;
      }

    // -------------------------------------------------------------------------
    // default / unrecognized command
    // -------------------------------------------------------------------------
    default:
      // We can optionally log an error if we want
      // bot.Log(log.Error, `JavaScript plugin received unknown command: ${cmd}`);
      return task.Fail;
  }
}

// @ts-ignore - the "process" object is created by goja
handler(process.argv || []); // Call the function with process.argv

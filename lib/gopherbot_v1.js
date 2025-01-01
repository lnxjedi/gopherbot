/**
 * gopherbot_v1.js
 *
 * This module defines constants and robot/bot methods required for JavaScript
 * Gopherbot extensions. It mirrors the structure of gopherbot_v1.lua so that
 * script authors can see usage patterns in a single file.
 */

// 1. ret (Robot method return values)
const ret = {
  // Connector Issues
  Ok: 0,
  UserNotFound: 1,
  ChannelNotFound: 2,
  AttributeNotFound: 3,
  FailedMessageSend: 4,
  FailedChannelJoin: 5,

  // Brain Maladies
  DatumNotFound: 6,
  DatumLockExpired: 7,
  DataFormatError: 8,
  BrainFailed: 9,
  InvalidDatumKey: 10,

  // GetTaskConfig
  InvalidConfigPointer: 11,
  ConfigUnmarshalError: 12,
  NoConfigFound: 13,

  // PromptForReply
  RetryPrompt: 14,
  ReplyNotMatched: 15,
  UseDefaultValue: 16,
  TimeoutExpired: 17,
  Interrupted: 18,
  MatcherNotFound: 19,

  // Email
  NoUserEmail: 20,
  NoBotEmail: 21,
  MailError: 22,

  // Pipeline Errors
  TaskNotFound: 23,
  MissingArguments: 24,
  InvalidStage: 25,
  InvalidTaskType: 26,
  CommandNotMatched: 27,
  TaskDisabled: 28,
  PrivilegeViolation: 29,

  // General Failure
  Failed: 63,
};

ret.string = function (val) {
  for (const k in this) {
    if (this.hasOwnProperty(k) && this[k] == val) {
      return k;
    }
  }
  return "UnknownRetVal";
}

// 2. task (Script return values)
const task = {
  Normal: 0,
  Fail: 1,
  MechanismFail: 2,
  ConfigurationError: 3,
  PipelineAborted: 4,
  RobotStopping: 5,
  NotFound: 6,
  Success: 7,
};

task.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownTaskRetVal";
};

// 3. log (LogLevel)
const log = {
  Trace: 0,
  Debug: 1,
  Info: 2,
  Audit: 3,
  Warn: 4,
  Error: 5,
  Fatal: 6,
};

log.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownLogLevel";
};

// 4. fmt (MessageFormat)
const fmt = {
  Raw: 0,
  Fixed: 1,
  Variable: 2,
};

fmt.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownMessageFormat";
};

// 5. proto (Protocol)
const proto = {
  Slack: 0,
  Rocket: 1,
  Terminal: 2,
  Test: 3,
  Null: 4,
};

proto.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownProtocol";
};

// 6. robot object, with .New(...)
const robot = {};

/**
 * robot.New(gbot?)
 *
 * In Lua, we allowed `robot:New()` or `robot.New()`.
 * Here, we simply do `robot.New(gbot)`. If no arg is given, default to GBOT global.
 *
 * Usage in JS scripts:
 *   const { robot } = require("gopherbot_v1");
 *   let bot = robot.New();
 *   bot.Say("Hello from JS!");
 */
robot.New = function (gbotArg) {
  // Use global GBOT if none is passed
  // @ts-ignore - GBOT is created in Go and added to the JS VM
  const gbot = gbotArg || (typeof GBOT !== "undefined" ? GBOT : null);
  if (!gbot) {
    throw new Error("No valid bot object provided, and GBOT is undefined.");
  }

  // Build a new "bot" object that wraps the underlying Go 'gbot' methods
  const newBot = {};

  // Keep a reference to the underlying gbot for advanced usage or debugging
  newBot.gbot = gbot;

  // Copy fields to show script authors what's available (informational only)
  newBot.user = gbot.user;
  newBot.user_id = gbot.user_id;
  newBot.channel = gbot.channel;
  newBot.channel_id = gbot.channel_id;
  newBot.thread_id = gbot.thread_id;
  newBot.message_id = gbot.message_id;
  newBot.protocol = gbot.protocol;
  newBot.brain = gbot.brain;
  newBot.type = "native";
  newBot.threaded_message = gbot.threaded_message;

  // -----------------------------
  // Send*/Say*/Reply* Methods
  // -----------------------------
  newBot.SendChannelMessage = function (channel, message, format) {
    return gbot.SendChannelMessage(channel, message, format);
  };

  newBot.SendChannelThreadMessage = function (channel, thread, message, format) {
    return gbot.SendChannelThreadMessage(channel, thread, message, format);
  };

  newBot.SendUserMessage = function (user, message, format) {
    return gbot.SendUserMessage(user, message, format);
  };

  newBot.SendUserChannelMessage = function (user, channel, message, format) {
    return gbot.SendUserChannelMessage(user, channel, message, format);
  };

  newBot.SendUserChannelThreadMessage = function (user, channel, thread, message, format) {
    return gbot.SendUserChannelThreadMessage(user, channel, thread, message, format);
  };

  newBot.Say = function (message, format) {
    return gbot.Say(message, format);
  };

  newBot.SayThread = function (message, format) {
    return gbot.SayThread(message, format);
  };

  newBot.Reply = function (message, format) {
    return gbot.Reply(message, format);
  };

  newBot.ReplyThread = function (message, format) {
    return gbot.ReplyThread(message, format);
  };

  // -----------------------------
  // Robot Modifier Methods
  // -----------------------------
  newBot.Direct = function () {
    const dbot = gbot.Direct();
    return robot.New(dbot);
  };

  newBot.Fixed = function () {
    const fbot = gbot.Fixed();
    return robot.New(fbot);
  };

  newBot.Threaded = function () {
    const tbot = gbot.Threaded();
    return robot.New(tbot);
  };

  newBot.MessageFormat = function (format) {
    const mbot = gbot.MessageFormat(format);
    return robot.New(mbot);
  };

  // -----------------------------
  // Prompting Methods
  // -----------------------------
  newBot.PromptForReply = function (regex_id, prompt, format) {
    return gbot.PromptForReply(regex_id, prompt, format);
  };

  newBot.PromptThreadForReply = function (regex_id, prompt, format) {
    return gbot.PromptThreadForReply(regex_id, prompt, format);
  };

  newBot.PromptUserForReply = function (regex_id, user, prompt, format) {
    return gbot.PromptUserForReply(regex_id, user, prompt, format);
  };

  newBot.PromptUserChannelForReply = function (regex_id, user, channel, prompt, format) {
    return gbot.PromptUserChannelForReply(regex_id, user, channel, prompt, format);
  };

  newBot.PromptUserChannelThreadForReply = function (regex_id, user, channel, thread, prompt, format) {
    return gbot.PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt, format);
  };

  // -----------------------------
  // Short-Term Memory Methods
  // -----------------------------
  newBot.Remember = function (key, value, shared) {
    return gbot.Remember(key, value, shared);
  };

  newBot.RememberThread = function (key, value, shared) {
    return gbot.RememberThread(key, value, shared);
  };

  newBot.RememberContext = function (context, value) {
    return gbot.RememberContext(context, value);
  };

  newBot.RememberContextThread = function (context, value) {
    return gbot.RememberContextThread(context, value);
  };

  newBot.Recall = function (key, shared) {
    return gbot.Recall(key, shared);
  };

  // -----------------------------
  // Pipeline Methods
  // -----------------------------
  newBot.GetParameter = function (name) {
    return gbot.GetParameter(name);
  };

  newBot.SetParameter = function (name, value) {
    return gbot.SetParameter(name, value);
  };

  newBot.Exclusive = function (tag, queueTask) {
    return gbot.Exclusive(tag, queueTask);
  };

  newBot.SpawnJob = function (name, ...args) {
    return gbot.SpawnJob(name, ...args);
  };

  newBot.AddTask = function (name, ...args) {
    return gbot.AddTask(name, ...args);
  };

  newBot.FinalTask = function (name, ...args) {
    return gbot.FinalTask(name, ...args);
  };

  newBot.FailTask = function (name, ...args) {
    return gbot.FailTask(name, ...args);
  };

  newBot.AddJob = function (name, ...args) {
    return gbot.AddJob(name, ...args);
  };

  newBot.AddCommand = function (pluginName, command) {
    return gbot.AddCommand(pluginName, command);
  };

  newBot.FinalCommand = function (pluginName, command) {
    return gbot.FinalCommand(pluginName, command);
  };

  newBot.FailCommand = function (pluginName, command) {
    return gbot.FailCommand(pluginName, command);
  };

  // -----------------------------
  // Attribute Methods
  // -----------------------------
  newBot.GetBotAttribute = function (attr) {
    const { attribute, retVal } = gbot.GetBotAttribute(attr);
    return { attribute, retVal };
  };

  newBot.GetUserAttribute = function (user, attr) {
    const result = gbot.GetUserAttribute(user, attr);
    return result.attribute, result.retVal
  };

  newBot.GetSenderAttribute = function (attr) {
    const result = gbot.GetSenderAttribute(attr);
    return result.attribute, result.retVal
  };

  // -----------------------------
  // Long-Term Memory Methods
  // -----------------------------
  // In JavaScript, we rely on the underlying Go methods in longterm_memories.go
  // but here we show how they'd be used. The usage is purely for doc/SDK convenience.
  newBot.CheckoutDatum = function (key, rw) {
    // In Go, `botCheckoutDatum` returns an object like:
    //   { retVal, exists, datum, token }
    // We’re not adding logic here; just pass through to the underlying goja method.
    return gbot.CheckoutDatum(key, rw);
  };

  newBot.UpdateDatum = function (memoryObj) {
    return gbot.UpdateDatum(memoryObj);
  };

  newBot.CheckinDatum = function (memoryObj) {
    return gbot.CheckinDatum(memoryObj);
  };

  // -----------------------------
  // Other Methods
  // -----------------------------
  newBot.GetTaskConfig = function () {
    return gbot.GetTaskConfig();
  };

  newBot.RandomInt = function (n) {
    return gbot.RandomInt(n);
  };

  newBot.RandomString = function (arr) {
    return gbot.RandomString(arr);
  };

  newBot.Pause = function (seconds) {
    return gbot.Pause(seconds);
  };

  newBot.CheckAdmin = function () {
    return gbot.CheckAdmin();
  };

  newBot.Elevate = function (immediate) {
    return gbot.Elevate(immediate);
  };

  newBot.Log = function (level, message) {
    return gbot.Log(level, message);
  };

  // Return our newly constructed bot object
  return newBot;
};

// Export everything so scripts can do:
//   const { robot, ret, task, log, fmt, proto } = require("gopherbot_v1");
module.exports = function () {
  return { robot, ret, task, log, fmt, proto };
};

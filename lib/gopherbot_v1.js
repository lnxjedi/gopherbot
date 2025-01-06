/**
 * gopherbot_v1.js
 *
 * This module defines constants and robot/bot methods required for JavaScript
 * Gopherbot extensions. It mirrors the structure of gopherbot_v1.lua so that
 * script authors can see usage patterns in a single file.
 */

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

/**
 * Converts a ret return value to its corresponding string constant.
 * @param {ret} val - The ret return value.
 * @returns {string} - The name of the ret return constant or "UnknownRetVal".
 * @example
 * const status = ret.string(ret.Ok); // "Ok"
 */
ret.string = function (val) {
  for (const k in this) {
    if (this.hasOwnProperty(k) && this[k] == val) {
      return k;
    }
  }
  return "UnknownRetVal";
};

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

/**
 * Converts a task return value to its corresponding string constant.
 * @param {task} val - The task return value.
 * @returns {string} - The name of the task return constant or "UnknownTaskRetVal".
 * @example
 * const taskStatus = task.string(task.Normal); // "Normal"
 */
task.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownTaskRetVal";
};

const log = {
  Trace: 0,
  Debug: 1,
  Info: 2,
  Audit: 3,
  Warn: 4,
  Error: 5,
  Fatal: 6,
};

/**
 * Converts a log level to its corresponding string constant.
 * @param {log} val - The log level value.
 * @returns {string} - The name of the log level or "UnknownLogLevel".
 * @example
 * const logLevel = log.string(log.Info); // "Info"
 */
log.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownLogLevel";
};

const fmt = {
  Raw: 0,
  Fixed: 1,
  Variable: 2,
};

/**
 * Converts a message format to its corresponding string constant.
 * @param {fmt} val - The message format value.
 * @returns {string} - The name of the message format or "UnknownMessageFormat".
 * @example
 * const format = fmt.string(fmt.Fixed); // "Fixed"
 */
fmt.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownMessageFormat";
};

const proto = {
  Slack: 0,
  Rocket: 1,
  Terminal: 2,
  Test: 3,
  Null: 4,
};

/**
 * Converts a protocol value to its corresponding string constant.
 * @param {proto} val - The protocol value.
 * @returns {string} - The name of the protocol or "UnknownProtocol".
 * @example
 * const protocolName = proto.string(proto.Slack); // "Slack"
 */
proto.string = function (val) {
  for (const k in this) {
    if (this[k] == val) {
      return k;
    }
  }
  return "UnknownProtocol";
};

// Robot object
/**
 * Usage in JS scripts:
 *   const { Robot } = require("gopherbot_v1");
 *   let bot = robot.New();
 *   bot.Say("Hello from JS!");
 */
function Robot(gbotArg) {
  // Use global GBOT if none is passed
  // @ts-ignore - GBOT is created in Go and added to the JS VM
  const gbot = gbotArg || (typeof GBOT !== "undefined" ? GBOT : null);
  if (!gbot) {
    throw new Error("No valid bot object provided, and GBOT is undefined.");
  }
  this.gbot = gbot
  this.user = gbot.user;
  this.user_id = gbot.user_id;
  this.channel = gbot.channel;
  this.channel_id = gbot.channel_id;
  this.thread_id = gbot.thread_id;
  this.message_id = gbot.message_id;
  this.protocol = gbot.protocol;
  this.brain = gbot.brain;
  this.type = "native";
  this.threaded_message = gbot.threaded_message;
}

// -----------------------------
// Send*/Say*/Reply* Methods
// -----------------------------
/**
 * Sends a channel message.
 *
 * @param {string} channel - Channel to send to
 * @param {string} message - The message to send
 * @param {fmt} [format] - Optional format (fmt.*) for the message
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.SendChannelMessage("#general", "Hello, Channel!", fmt.Raw);
 */
Robot.prototype.SendChannelMessage = function (channel, message, format) {
  return this.gbot.SendChannelMessage(channel, message, format);
};

/**
 * Sends a message in a channel thread.
 *
 * @param {string} channel - Channel to send to
 * @param {string} thread - The thread identifier.
 * @param {string} message - The message to send
 * @param {fmt} [format] - Optional format (fmt.*) for the message
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.SendChannelThreadMessage("#general", "thread123", "Hello, Thread!", fmt.Fixed);
 */
Robot.prototype.SendChannelThreadMessage = function (channel, thread, message, format) {
  return this.gbot.SendChannelThreadMessage(channel, thread, message, format);
};

/**
 * Sends a direct user message.
 *
 * @param {string} user - The user identifier.
 * @param {string} message - The message to send
 * @param {fmt} [format] - Optional format (fmt.*) for the message
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.SendUserMessage("user123", "Hello, User!", fmt.Variable);
 */
Robot.prototype.SendUserMessage = function (user, message, format) {
  return this.gbot.SendUserMessage(user, message, format);
};

/**
 * Sends a message to a user within a specific channel.
 *
 * @param {string} user - The user identifier.
 * @param {string} channel - Channel to send to
 * @param {string} message - The message to send
 * @param {fmt} [format] - Optional format (fmt.*) for the message
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.SendUserChannelMessage("user123", "#general", "Hello, Channel User!", fmt.Fixed);
 */
Robot.prototype.SendUserChannelMessage = function (user, channel, message, format) {
  return this.gbot.SendUserChannelMessage(user, channel, message, format);
};

/**
 * Sends a message to a user within a specific channel thread.
 *
 * @param {string} user - The user identifier.
 * @param {string} channel - Channel to send to
 * @param {string} thread - The thread identifier.
 * @param {string} message - The message to send
 * @param {fmt} [format] - Optional format (fmt.*) for the message
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.SendUserChannelThreadMessage("user123", "#general", "thread123", "Hello, Thread User!", fmt.Variable);
 */
Robot.prototype.SendUserChannelThreadMessage = function (user, channel, thread, message, format) {
  return this.gbot.SendUserChannelThreadMessage(user, channel, thread, message, format);
};

/**
 * Sends a message.
 *
 * @param {string} message - The message to send
 * @param {fmt} [format] - Optional format (fmt.*) for the message
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.Say("Hello, World!", fmt.Raw);
 */
Robot.prototype.Say = function (message, format) {
  return this.gbot.Say(message, format);
};

/**
 * Sends a threaded message.
 *
 * @param {string} message - The message to send
 * @param {fmt} [format] - Optional format (fmt.*) for the message
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.SayThread("Hello, Threaded World!", fmt.Fixed);
 */
Robot.prototype.SayThread = function (message, format) {
  return this.gbot.SayThread(message, format);
};

/**
 * Replies to a message.
 *
 * @param {string} message - The reply message.
 * @param {fmt} [format] - Optional format for the reply.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.Reply("Thanks for your message!", fmt.Variable);
 */
Robot.prototype.Reply = function (message, format) {
  return this.gbot.Reply(message, format);
};

/**
 * Replies to a message in a thread.
 *
 * @param {string} message - The reply message.
 * @param {fmt} [format] - Optional format for the reply.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.ReplyThread("Thanks for your thread message!", fmt.Fixed);
 */
Robot.prototype.ReplyThread = function (message, format) {
  return this.gbot.ReplyThread(message, format);
};

// -----------------------------
// Robot Modifier Methods
// -----------------------------
// Note that applying annotations mostly just breaks things.
Robot.prototype.Direct = function () {
  const dbot = this.gbot.Direct();
  return new Robot(dbot);
};

Robot.prototype.Fixed = function () {
  const fbot = this.gbot.Fixed();
  return new Robot(fbot);
};

Robot.prototype.Threaded = function () {
  const tbot = this.gbot.Threaded();
  return new Robot(tbot);
};

Robot.prototype.MessageFormat = function (format) {
  const mbot = this.gbot.MessageFormat(format);
  return new Robot(mbot);
};

// -----------------------------
// Prompting Methods
// -----------------------------
/**
 * Prompts for a reply matching a regex
 *
 * @param {string} regex_id - The regex identifier
 * @param {string} prompt - The prompt message
 * @param {fmt} [format] - Optional format (fmt.*) for the prompt
 * @returns {{ reply: string, retVal: number }} - The reply and retVal (ret.*)
 *
 * @example
 * const result = bot.PromptForReply("regex123", "Please reply with your name.", fmt.Raw);
 * bot.Say("Your name is: " + result.reply);
 */
Robot.prototype.PromptForReply = function (regex_id, prompt, format) {
  return this.gbot.PromptForReply(regex_id, prompt, format);
};

/**
 * Prompts for a reply in a thread matching a regex
 *
 * @param {string} regex_id - The regex identifier
 * @param {string} prompt - The prompt message
 * @param {fmt} [format] - Optional format (fmt.*) for the prompt
 * @returns {{ reply: string, retVal: number }} - The reply and retVal (ret.*)
 *
 * @example
 * const result = bot.PromptThreadForReply("regex123", "Please reply in the thread.", fmt.Fixed);
 * bot.SayThread("Your reply is: " + result.reply, fmt.Fixed);
 */
Robot.prototype.PromptThreadForReply = function (regex_id, prompt, format) {
  return this.gbot.PromptThreadForReply(regex_id, prompt, format);
};

/**
 * Prompts a specific user for a reply matching a regex
 *
 * @param {string} regex_id - The regex identifier
 * @param {string} user - The user identifier
 * @param {string} prompt - The prompt message
 * @param {fmt} [format] - Optional format (fmt.*) for the prompt
 * @returns {{ reply: string, retVal: number }} - The reply and retVal (ret.*)
 *
 * @example
 * const result = bot.PromptUserForReply("regex123", "user123", "Please reply with your email.", fmt.Variable);
 * bot.Say("Your email is: " + result.reply);
 */
Robot.prototype.PromptUserForReply = function (regex_id, user, prompt, format) {
  return this.gbot.PromptUserForReply(regex_id, user, prompt, format);
};

/**
 * Prompts a specific user in a channel for a reply matching a regex
 *
 * @param {string} regex_id - The regex identifier
 * @param {string} user - The user identifier
 * @param {string} channel - The channel identifier
 * @param {string} prompt - The prompt message
 * @param {fmt} [format] - Optional format (fmt.*) for the prompt
 * @returns {{ reply: string, retVal: number }} - The reply and retVal (ret.*)
 *
 * @example
 * const result = bot.PromptUserChannelForReply("regex123", "user123", "#general", "Please reply with your age.", fmt.Fixed);
 * bot.Say("Your age is: " + result.reply, fmt.Fixed);
 */
Robot.prototype.PromptUserChannelForReply = function (regex_id, user, channel, prompt, format) {
  return this.gbot.PromptUserChannelForReply(regex_id, user, channel, prompt, format);
};

/**
 * Prompts a specific user in a channel thread for a reply matching a regex
 *
 * @param {string} regex_id - The regex identifier
 * @param {string} user - The user identifier
 * @param {string} channel - The channel identifier
 * @param {string} thread - The thread identifier
 * @param {string} prompt - The prompt message
 * @param {fmt} [format] - Optional format (fmt.*) for the prompt
 * @returns {{ reply: string, retVal: ret }} - The reply and retVal (ret.*)
 *
 * @example
 * const result = bot.PromptUserChannelThreadForReply("regex123", "user123", "#general", "thread123", "Please reply with your location.", fmt.Variable);
 * bot.SayThread("Your location is: " + result.reply, fmt.Variable);
 */
Robot.prototype.PromptUserChannelThreadForReply = function (regex_id, user, channel, thread, prompt, format) {
  return this.gbot.PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt, format);
};

// -----------------------------
// Short-Term Memory Methods
// -----------------------------
/**
 * Remembers a key-value pair in short-term memory.
 *
 * @param {string} key - The key to remember.
 * @param {string} value - The value to associate with the key.
 * @param {boolean} [shared] - Whether the memory is shared.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.Remember("favoriteColor", "blue", true);
 */
Robot.prototype.Remember = function (key, value, shared) {
  return this.gbot.Remember(key, value, shared);
};

/**
 * Remembers a key-value pair in a thread's short-term memory.
 *
 * @param {string} key - The key to remember.
 * @param {string} value - The value to associate with the key.
 * @param {boolean} [shared] - Whether the memory is shared.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.RememberThread("sessionToken", "abc123", false);
 */
Robot.prototype.RememberThread = function (key, value, shared) {
  return this.gbot.RememberThread(key, value, shared);
};

/**
 * Remembers a value for a specific context, the next time a user
 * message refers to 'it' for that context, the value is used; example:
 * user: patch www.myhost.com
 * ... (patching happens, context "server" gets set to "www.myhost.com")
 * user: shut it down
 * ... (www.myhost.com gets shut down, because it in the context of a
 * server is "www.myhost.com")
 * n.b. - AFAIK, nobody uses contexts, not even me (the author!)
 *
 * @param {string} context - The context identifier.
 * @param {string} value - The value to associate with the context.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.RememberContext("server", "www.myhost.com");
 */
Robot.prototype.RememberContext = function (context, value) {
  return this.gbot.RememberContext(context, value);
};

/**
 * Remembers a context-value pair in a thread's short-term memory.
 *
 * @param {string} context - The context identifier.
 * @param {string} value - The value to associate with the context.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * bot.RememberContextThread("server", "www.myhost.com");
 */
Robot.prototype.RememberContextThread = function (context, value) {
  return this.gbot.RememberContextThread(context, value);
};

/**
 * Recalls a value from short-term memory.
 *
 * @param {string} key - The key to recall.
 * @param {boolean} [shared] - Whether to recall shared memory.
 * @returns {string} - The recalled value or "".
 *
 * @example
 * const recalled = bot.Recall("favoriteColor", true);
 * bot.Say("Your favorite color is: " + recalled, fmt.Fixed);
 */
Robot.prototype.Recall = function (key, shared) {
  return this.gbot.Recall(key, shared);
};

// -----------------------------
// Pipeline Methods
// -----------------------------

/**
 * Retrieves a pipeline parameter.
 *
 * @param {string} name - The parameter name.
 * @returns {string} - The value of the parameter.
 *
 * @example
 * const param = bot.GetParameter("maxRetries");
 */
Robot.prototype.GetParameter = function (name) {
  return this.gbot.GetParameter(name);
};

/**
 * Sets a pipeline parameter.
 *
 * @param {string} name - The parameter name.
 * @param {string} value - The value to set.
 * @returns {boolean} - `true` if the parameter was set successfully, otherwise `false`.
 *
 * @example
 * const success = bot.SetParameter("maxRetries", 5);
 */
Robot.prototype.SetParameter = function (name, value) {
  return this.gbot.SetParameter(name, value);
};

/**
 * Gets exclusive access to a resource for the script.
 *
 * @param {string} tag - The exclusive tag.
 * @param {boolean} queueTask - Whether to queue the task.
 * @returns {boolean} - `true` if the operation was successful, otherwise `false`.
 *
 * @example
 * const success = bot.Exclusive("prodbuild", true);
 * if (success) {
 *   bot.Log(log.Audit, "Exclusive access to prod build obtained, proceeding");
 * }
 */
Robot.prototype.Exclusive = function (tag, queueTask) {
  return this.gbot.Exclusive(tag, queueTask);
};

/**
 * Spawns a new job.
 *
 * @param {string} name - The job name.
 * @param {...string} args - Arguments for the job.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.SpawnJob("processData", "dataId", "options");
 * if (result === ret.Ok) {
 *   bot.Say("Job spawned successfully!");
 * }
 */
Robot.prototype.SpawnJob = function (name, ...args) {
  return this.gbot.SpawnJob(name, ...args);
};

/**
 * Adds a new task to the pipeline.
 *
 * @param {string} name - The task name.
 * @param {...string} args - Arguments for the task.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.AddTask("validateData", "data");
 * if (result === ret.Ok) {
 *   bot.Say("Task added successfully!");
 * }
 */
Robot.prototype.AddTask = function (name, ...args) {
  return this.gbot.AddTask(name, ...args);
};

/**
 * Adds a final task to the pipeline.
 *
 * @param {string} name - The task name.
 * @param {...string} args - Arguments for the task.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.FinalTask("cleanup", "sessionId");
 * if (result === ret.Ok) {
 *   bot.Say("Final task added successfully!");
 * }
 */
Robot.prototype.FinalTask = function (name, ...args) {
  return this.gbot.FinalTask(name, ...args);
};

/**
 * Adds a task to the fail pipeline.
 *
 * @param {string} name - The task name.
 * @param {...string} args - Arguments for the task.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.FailTask("handleError", "error");
 * if (result === ret.Failed) {
 *   bot.Say("Fail task added successfully!", fmt.Warn);
 * }
 */
Robot.prototype.FailTask = function (name, ...args) {
  return this.gbot.FailTask(name, ...args);
};

/**
 * Adds a new job to the pipeline.
 *
 * @param {string} name - The job name.
 * @param {...string} args - Arguments for the job.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.AddJob("sendReport", "reportId");
 * if (result === ret.Ok) {
 *   bot.Say("Job added successfully!");
 * }
 */
Robot.prototype.AddJob = function (name, ...args) {
  return this.gbot.AddJob(name, ...args);
};

/**
 * Adds a new plugin command to the pipeline.
 *
 * @param {string} pluginName - The name of the plugin.
 * @param {string} command - The command to add.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.AddCommand("weatherPlugin", "get todays forecast");
 * if (result === ret.Ok) {
 *   bot.Say("Command added successfully!");
 * }
 */
Robot.prototype.AddCommand = function (pluginName, command) {
  return this.gbot.AddCommand(pluginName, command);
};

/**
 * Adds a plugin command to the final pipeline.
 *
 * @param {string} pluginName - The name of the plugin.
 * @param {string} command - The command to add.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.FinalCommand("weatherPlugin", "finalize forecast");
 * if (result === ret.Ok) {
 *   bot.Say("Final command added successfully!");
 * }
 */
Robot.prototype.FinalCommand = function (pluginName, command) {
  return this.gbot.FinalCommand(pluginName, command);
};

/**
 * Adds a failing command to a plugin.
 *
 * @param {string} pluginName - The name of the plugin.
 * @param {string} command - The command to add.
 * @returns {number} - The retVal return code (ret.*)
 *
 * @example
 * const result = bot.FailCommand("weatherPlugin", "handleError");
 * if (result === ret.Failed) {
 *   bot.Say("Fail command added successfully!", fmt.Warn);
 * }
 */
Robot.prototype.FailCommand = function (pluginName, command) {
  return this.gbot.FailCommand(pluginName, command);
};

// -----------------------------
// Attribute Methods
// -----------------------------
/**
 * Retrieves a bot attribute.
 * @param {string} attr - The attribute name.
 * @returns {{ attribute: string, retVal: ret }} - An object containing the attribute value and return value.
 * @example
 * const result = bot.GetBotAttribute("name");
 * bot.Say("My name is: " + result.attribute);
 */
Robot.prototype.GetBotAttribute = function (attr) {
  return this.gbot.GetBotAttribute(attr);
};

/**
 * Retrieves a user attribute.
 * @param {string} user - The name of the user
 * @param {string} attr - The attribute name.
 * @returns {{ attribute: string, retVal: ret }} - An object containing the attribute value and return value.
 * @example
 * const result = bot.GetUserAttribute("frank", "email");
 * bot.Say("Frank's email is: " + result.attribute);
 */
Robot.prototype.GetUserAttribute = function (user, attr) {
  return this.gbot.GetUserAttribute(user, attr);
};

/**
 * Retrieves a sender attribute.
 * @param {string} attr - The attribute name.
 * @returns {{ attribute: string, retVal: ret }} - An object containing the attribute value and return value.
 * @example
 * const result = bot.GetSenderAttribute("name");
 * bot.Say("Your name is: " + result.attribute);
 */
Robot.prototype.GetSenderAttribute = function (attr) {
  return this.gbot.GetSenderAttribute(attr);
};

// -----------------------------
// Long-Term Memory Methods
// -----------------------------
// In JavaScript, we rely on the underlying Go methods in longterm_memories.go
// but here we show how they'd be used. The usage is purely for doc/SDK convenience.
/**
 * Checks out a datum from long-term memory.
 *
 * @param {string} key - The datum key.
 * @param {boolean} [rw=false] - Whether to request read-write access.
 * @returns {{ retVal: ret, exists: boolean, datum: any, token: string }} 
 *   An object containing:
 *   - `retVal` (number): The return value constant (int-cast of `robot.Ok` or error).
 *   - `exists` (boolean): `true` if the datum was found, otherwise `false`.
 *   - `datum` (any): The checked-out data. If `exists` is false, returns an empty object or null.
 *   - `token` (string): The lock token for read-write operations.
 *
 * @example
 * const result = bot.CheckoutDatum("myKey", true); // read-write
 * if (result.retVal === ret.Ok && result.exists) {
 *   bot.Say("Datum found: " + JSON.stringify(result.datum));
 * } else {
 *   bot.Say("Checkout failed with code: " + ret.string(result.retVal));
 * }
 */
Robot.prototype.CheckoutDatum = function (key, rw) {
  // Pass through to the underlying goja method, which returns an object:
  // { retVal, exists, datum, token }
  return this.gbot.CheckoutDatum(key, rw);
};

/**
 * Updates a previously checked-out datum in long-term memory.
 *
 * @param {{ key: string, token: string, datum: any }} memoryObj 
 *   An object containing:
 *   - `key`: The datum key.
 *   - `token`: The lock token from `CheckoutDatum`.
 *   - `datum`: The updated data to store.
 * @returns {number} 
 *   The return value constant (int-cast of `robot.Ok` or an error code like `robot.DataFormatError`).
 *
 * @example
 * const out = bot.CheckoutDatum("myKey", true);
 * if (out.retVal === ret.Ok && out.exists) {
 *   out.datum.newField = 42;
 *   const rv = bot.UpdateDatum({ key: "myKey", token: out.token, datum: out.datum });
 *   bot.Say("UpdateDatum returned: " + ret.string(rv));
 * }
 */
Robot.prototype.UpdateDatum = function (memoryObj) {
  return this.gbot.UpdateDatum(memoryObj);
};

/**
 * Checks in a previously checked-out datum to long-term memory.
 *
 * @param {{ key: string, token: string }} memoryObj
 *   An object containing:
 *   - `key`: The datum key.
 *   - `token`: The lock token from `CheckoutDatum`.
 * @returns {number}
 *   Always returns `ret.Ok` (int value).
 *
 * @example
 * const out = bot.CheckoutDatum("myKey", true);
 * // ... do something with out.datum ...
 * bot.CheckinDatum({ key: "myKey", token: out.token });
 */
Robot.prototype.CheckinDatum = function (memoryObj) {
  return this.gbot.CheckinDatum(memoryObj);
};

/**
 * Retrieves the current task configuration.
 *
 * @returns {{ config: any, retVal: number }}
 *   An object containing:
 *   - `config` (any): The configuration data if `retVal` is `ret.Ok`; otherwise null.
 *   - `retVal` (number): The return code (e.g., `ret.Ok`, or an error like `ret.InvalidConfigPointer`).
 *
 * @example
 * const tc = bot.GetTaskConfig();
 * if (tc.retVal === ret.Ok) {
 *   bot.Say("Config: " + JSON.stringify(tc.config));
 * } else {
 *   bot.Say("GetTaskConfig error: " + ret.string(tc.retVal));
 * }
 */
Robot.prototype.GetTaskConfig = function () {
  return this.gbot.GetTaskConfig();
};

/**
 * Generates a random integer up to (but not including) n.
 *
 * @param {number} n - The upper bound (exclusive).
 * @returns {number} 
 *   A random integer in the range [0, n-1].
 *
 * @example
 * const rand = bot.RandomInt(10);
 * bot.Say("Random number: " + rand);
 */
Robot.prototype.RandomInt = function (n) {
  return this.gbot.RandomInt(n);
};

/**
 * Selects a random string from the given array of strings.
 *
 * @param {string[]} arr - The array of candidate strings.
 * @returns {string}
 *   A randomly selected string from the array. 
 *   If the array is empty or contains no valid strings, returns "".
 *
 * @example
 * const fruit = bot.RandomString(["apple", "banana", "cherry"]);
 * bot.Say("Random fruit: " + fruit);
 */
Robot.prototype.RandomString = function (arr) {
  return this.gbot.RandomString(arr);
};

/**
 * Pauses execution for the specified number of seconds.
 *
 * @param {number} seconds - The number of seconds to pause.
 * @returns {null}
 *   Always returns `null` in JavaScript (the underlying Go method does not return a ret code).
 *
 * @example
 * bot.Pause(3); // Pauses for 3 seconds
 * // NOTE: There's no retVal for Pause; it simply returns null.
 */
Robot.prototype.Pause = function (seconds) {
  return this.gbot.Pause(seconds);
};

/**
 * Checks if the current user has administrative privileges.
 *
 * @returns {boolean}
 *   `true` if the user is an admin, otherwise `false`.
 *
 * @example
 * const admin = bot.CheckAdmin();
 * if (admin) {
 *   bot.Say("You have admin privileges!");
 * } else {
 *   bot.Say("Access denied.");
 * }
 */
Robot.prototype.CheckAdmin = function () {
  return this.gbot.CheckAdmin();
};

/**
 * Elevates the current user's privileges.
 *
 * @param {boolean} [immediate=false] 
 *   If `true`, forces immediate elevation/2FA prompt. Otherwise normal flow.
 * @returns {boolean}
 *   `true` if elevation succeeded, otherwise `false`.
 *
 * @example
 * const elevated = bot.Elevate(true);
 * if (elevated) {
 *   bot.Say("Elevation succeeded!");
 * } else {
 *   bot.Say("Elevation failed or was not granted.");
 * }
 */
Robot.prototype.Elevate = function (immediate) {
  return this.gbot.Elevate(immediate);
};

/**
 * Logs a message at the specified log level.
 *
 * @param {log} level - The log level constant (matches `log.*`).
 * @param {string} message - The message to log.
 * @returns {number}
 *   The return code: typically `ret.Ok` if logged successfully, or `ret.Fail` if arguments were invalid.
 *
 * @example
 * const rv = bot.Log(log.Info, "This is an informational message.");
 * if (rv === ret.Ok) {
 *   bot.Say("Log succeeded!");
 * } else {
 *   bot.Say("Log failed with code: " + rv);
 * }
 */
Robot.prototype.Log = function (level, message) {
  return this.gbot.Log(level, message);
};

// Export everything so scripts can do:
//   const { robot, ret, task, log, fmt, proto } = require("gopherbot_v1");
module.exports = function () {
  return { Robot, ret, task, log, fmt, proto };
};

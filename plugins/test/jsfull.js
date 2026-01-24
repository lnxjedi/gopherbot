// hello.js
// JavaScript plugin "Hello World" and boilerplate

// Define the default configuration as a YAML string
const defaultConfig = `---
Help:
- Keywords: [ "say", "ask" ]
  Helptext:
  - "(bot), say everything - full test of Say*/Reply*/Send* methods"
- Keywords: [ "config" ]
  Helptext:
  - "(bot), js-config - exercise GetTaskConfig + RandomString"
- Keywords: [ "http" ]
  Helptext:
  - "(bot), js-http - exercise HTTP GET/POST/PUT"
CommandMatchers:
- Regex: (?i:say everything)
  Command: sendmsg
- Regex: (?i:js-config)
  Command: configtest
- Regex: (?i:js-http)
  Command: http
AllowedHiddenCommands:
- sendmsg
Config:
  Openings:
  - "Not completely random 1"
  - "Not completely random 2"
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

    case 'configtest':
      {
        const bot = new Robot();
        const cfg = bot.GetTaskConfig();
        if (cfg.retVal != ret.Ok) {
          bot.Say("No config available");
          return task.Fail;
        }
        const openings = cfg.config.Openings || [];
        bot.Say(bot.RandomString(openings));
        return task.Normal;
      }

    case 'http':
      {
        const bot = new Robot();
        const cfg = bot.GetTaskConfig();
        if (cfg.retVal != ret.Ok) {
          bot.Say("No config available");
          return task.Fail;
        }
        const baseURL = cfg.config.HttpBaseURL;
        if (!baseURL) {
          bot.Say("No http base url configured");
          return task.Fail;
        }
        const http = require("gopherbot_http");
        const client = http.createClient({
          baseURL: baseURL,
          timeoutMs: 5000,
          throwOnHTTPError: true,
        });
        const getRes = client.getJSON("/json/get");
        bot.Say(`HTTP GET ok: ${getRes.method}`);
        const postRes = client.postJSON("/json/post", { value: "alpha" });
        bot.Say(`HTTP POST ok: ${postRes.value}`);
        const putRes = client.putJSON("/json/put", { value: "bravo" });
        bot.Say(`HTTP PUT ok: ${putRes.value}`);
        try {
          client.getJSON("/json/error");
          bot.Say("HTTP ERROR unexpected");
        } catch (err) {
          const status = err && err.status ? err.status : "unknown";
          bot.Say(`HTTP ERROR ok: ${status}`);
        }
        try {
          client.getJSON("/json/slow", { timeoutMs: 50 });
          bot.Say("HTTP TIMEOUT unexpected");
        } catch (err) {
          bot.Say("HTTP TIMEOUT ok");
        }
        return task.Normal;
      }

    default:
      // const botDefault = robot.New();
      // botDefault.Log(log.Error, `JavaScript plugin received unknown command: ${cmd}`);
      return task.Fail;
  }
}

// @ts-ignore - the "process" object is created by goja
handler(process.argv || []); // Call the function with process.argv

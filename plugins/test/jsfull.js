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
- Keywords: [ "subscribe" ]
  Helptext:
  - "(bot), js-subscribe - exercise Subscribe/Unsubscribe"
- Keywords: [ "prompt" ]
  Helptext:
  - "(bot), js-prompts - exercise Prompt* methods (user/channel/thread variants)"
- Keywords: [ "memory" ]
  Helptext:
  - "(bot), js-memory-seed/js-memory-check/js-memory-thread-check - exercise Remember*/Recall context behavior"
  - "(bot), js-memory-datum-seed/js-memory-datum-check/js-memory-datum-checkin - exercise CheckoutDatum/UpdateDatum/CheckinDatum"
- Keywords: [ "identity", "parameter" ]
  Helptext:
  - "(bot), js-identity - exercise Get*Attribute + Set/GetParameter"
  - "(bot), js-parameter-addtask - verify SetParameter value in next AddTask"
CommandMatchers:
- Regex: (?i:say everything)
  Command: sendmsg
- Regex: (?i:js-config)
  Command: configtest
- Regex: (?i:js-http)
  Command: http
- Regex: (?i:js-subscribe)
  Command: subscribe
- Regex: (?i:js-prompts)
  Command: prompts
- Regex: (?i:js-memory-seed)
  Command: memoryseed
- Regex: (?i:js-memory-check)
  Command: memorycheck
- Regex: (?i:js-memory-thread-check)
  Command: memorythreadcheck
- Regex: (?i:js-memory-datum-seed)
  Command: memorydatumseed
- Regex: (?i:js-memory-datum-check)
  Command: memorydatumcheck
- Regex: (?i:js-memory-datum-checkin)
  Command: memorydatumcheckin
- Regex: (?i:js-identity)
  Command: identity
- Regex: (?i:js-parameter-addtask)
  Command: parameteraddtask
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
  const showMemory = (v) => (v && v.length > 0 ? v : "<empty>");

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
    case 'subscribe':
      {
        const bot = new Robot();
        const sub = bot.Subscribe();
        const unsub = bot.Unsubscribe();
        bot.Say(`SUBSCRIBE FLOW: ${sub}/${unsub}`);
        return task.Normal;
      }
    case 'prompts':
      {
        const bot = new Robot();
        const p1 = bot.PromptForReply("SimpleString", "Codename check: pick a mission codename.");
        if (p1.retVal !== ret.Ok) {
          bot.Say(`PROMPT FLOW FAILED 1:${p1.retVal}`);
          return task.Fail;
        }
        const p2 = bot.PromptThreadForReply("SimpleString", "Thread check: pick a favorite snack for launch.");
        if (p2.retVal !== ret.Ok) {
          bot.Say(`PROMPT FLOW FAILED 2:${p2.retVal}`);
          return task.Fail;
        }
        const p3 = bot.PromptUserForReply("SimpleString", bot.user, "DM check: name a secret moon base.");
        if (p3.retVal !== ret.Ok) {
          bot.Say(`PROMPT FLOW FAILED 3:${p3.retVal}`);
          return task.Fail;
        }
        const p4 = bot.PromptUserChannelForReply("SimpleString", bot.user, bot.channel, "Channel check: describe launch weather in two words.");
        if (p4.retVal !== ret.Ok) {
          bot.Say(`PROMPT FLOW FAILED 4:${p4.retVal}`);
          return task.Fail;
        }
        const p5 = bot.PromptUserChannelThreadForReply("SimpleString", bot.user, bot.channel, bot.thread_id, "Thread rally: choose a backup call sign.");
        if (p5.retVal !== ret.Ok) {
          bot.Say(`PROMPT FLOW FAILED 5:${p5.retVal}`);
          return task.Fail;
        }
        bot.Say(`PROMPT FLOW OK: ${p1.reply} | ${p2.reply} | ${p3.reply} | ${p4.reply} | ${p5.reply}`);
        return task.Normal;
      }
    case 'memoryseed':
      {
        const bot = new Robot();
        bot.Remember("launch_snack", "saffron noodles", false);
        bot.Remember("launch_snack", "solar soup", true);
        bot.RememberContext("pad", "orbital-7");
        bot.RememberThread("thread_note", "delta thread", false);
        bot.RememberContextThread("mission", "aurora mission");
        bot.Say("MEMORY SEED: done");
        return task.Normal;
      }
    case 'memorycheck':
      {
        const bot = new Robot();
        const local = bot.Recall("launch_snack", false);
        const shared = bot.Recall("launch_snack", true);
        const ctx = bot.Recall("context:pad", false);
        const thread = bot.Recall("thread_note", false);
        const threadctx = bot.Recall("context:mission", false);
        bot.Say(
          `MEMORY CHECK: local=${showMemory(local)} shared=${showMemory(shared)} ctx=${showMemory(ctx)} thread=${showMemory(thread)} threadctx=${showMemory(threadctx)}`
        );
        return task.Normal;
      }
    case 'memorythreadcheck':
      {
        const bot = new Robot();
        const local = bot.Recall("launch_snack", false);
        const shared = bot.Recall("launch_snack", true);
        const ctx = bot.Recall("context:pad", false);
        const thread = bot.Recall("thread_note", false);
        const threadctx = bot.Recall("context:mission", false);
        bot.Say(
          `MEMORY THREAD CHECK: local=${showMemory(local)} shared=${showMemory(shared)} ctx=${showMemory(ctx)} thread=${showMemory(thread)} threadctx=${showMemory(threadctx)}`
        );
        return task.Normal;
      }
    case 'memorydatumseed':
      {
        const bot = new Robot();
        const out = bot.CheckoutDatum("launch_manifest", true);
        if (out.retVal !== ret.Ok) {
          bot.Say(`MEMORY DATUM SEED FAILED: ${ret.string(out.retVal)}`);
          return task.Fail;
        }
        if (!out.exists || !out.datum || typeof out.datum !== "object" || Array.isArray(out.datum)) {
          out.datum = {};
        }
        out.datum.mission = "opal-orbit";
        out.datum.vehicle = "heron-7";
        out.datum.status = "go";
        const upd = bot.UpdateDatum(out);
        if (upd !== ret.Ok) {
          bot.Say(`MEMORY DATUM SEED FAILED: ${ret.string(upd)}`);
          return task.Fail;
        }
        bot.Say(`MEMORY DATUM SEED: update=${ret.string(upd)}`);
        return task.Normal;
      }
    case 'memorydatumcheck':
      {
        const bot = new Robot();
        const out = bot.CheckoutDatum("launch_manifest", false);
        if (out.retVal !== ret.Ok) {
          bot.Say(`MEMORY DATUM CHECK FAILED: ${ret.string(out.retVal)}`);
          return task.Fail;
        }
        if (!out.exists || !out.datum || typeof out.datum !== "object") {
          bot.Say("MEMORY DATUM CHECK: mission=<empty> vehicle=<empty> status=<empty>");
          return task.Normal;
        }
        const d = out.datum;
        bot.Say(
          `MEMORY DATUM CHECK: mission=${showMemory(d.mission)} vehicle=${showMemory(d.vehicle)} status=${showMemory(d.status)}`
        );
        return task.Normal;
      }
    case 'memorydatumcheckin':
      {
        const bot = new Robot();
        const out = bot.CheckoutDatum("launch_manifest", true);
        if (out.retVal !== ret.Ok) {
          bot.Say(`MEMORY DATUM CHECKIN FAILED: ${ret.string(out.retVal)}`);
          return task.Fail;
        }
        const tokenPresent = !!(out.token && out.token.length > 0);
        const checkin = bot.CheckinDatum(out);
        const checkinRet = checkin === undefined ? ret.Ok : checkin;
        bot.Say(`MEMORY DATUM CHECKIN: exists=${out.exists} token=${tokenPresent} ret=${ret.string(checkinRet)}`);
        return task.Normal;
      }
    case 'identity':
      {
        const bot = new Robot();
        const botName = bot.GetBotAttribute("name");
        const sender = bot.GetSenderAttribute("firstName");
        const otherUser = bot.GetUserAttribute("bob", "firstName");
        const setOK = bot.SetParameter("launch_phase", "phase-amber");
        const phase = bot.GetParameter("definitely_missing_param");
        bot.Say(
          `IDENTITY CHECK: bot=${showMemory(botName.attribute)}/${ret.string(botName.retVal)} sender=${showMemory(sender.attribute)}/${ret.string(sender.retVal)} bob=${showMemory(otherUser.attribute)}/${ret.string(otherUser.retVal)} set=${setOK} param=${showMemory(phase)}`
        );
        return task.Normal;
      }
    case 'parameteraddtask':
      {
        const bot = new Robot();
        const setOK = bot.SetParameter("PIPELINE_SENTINEL", "nebula-42");
        if (!setOK) {
          bot.Say("SETPARAM ADDTASK: set=false");
          return task.Fail;
        }
        const addRet = bot.AddTask("param-show");
        if (addRet !== ret.Ok) {
          bot.Say("SETPARAM ADDTASK: queue=false");
          return task.Fail;
        }
        bot.Say("SETPARAM ADDTASK: queued");
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

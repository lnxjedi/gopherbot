const { Robot, ret, task, log } = require('gopherbot_v1')();

const defaultConfig = `---
Commands:
  - Regex: (?i:js local demo)(?: (.*))?
    Command: demo
  - Regex: (?i:js local prompt)
    Command: prompt
  - Regex: (?i:js local memory)
    Command: memory
  - Regex: (?i:js local config)
    Command: config
`;

function demo(bot) {
  const sender = bot.GetSenderAttribute("fullName").attribute || bot.user;
  const color = bot.GetParameter("CAT_COLOR") || "mystery";
  bot.Say(`JavaScript demo for ${sender}; cat color=${color}.`);
  bot.Reply(`process.argv[3] was ${process.argv[3] || "<none>"}`);
  bot.Fixed().SendChannelMessage(bot.channel, "fixed-width channel message from JavaScript");
  return task.Normal;
}

function prompt(bot) {
  const answer = bot.PromptForReply("SimpleString", "Name of cat?");
  if (answer.retVal !== ret.Ok) {
    bot.Say(`Prompt failed: ${answer.retVal}`);
    return task.Fail;
  }
  bot.Say(`JavaScript heard cat name: ${answer.reply}`);
  return task.Normal;
}

function memory(bot) {
  bot.Remember("last_cat", "Garfield", false);
  bot.Remember("team_cat", "Pixel", true);
  bot.Say(`JavaScript memory local=${bot.Recall("last_cat", false)} shared=${bot.Recall("team_cat", true)}`);

  const profile = bot.CheckoutDatum("cat_profile", true);
  if (profile.retVal === ret.Ok) {
    const datum = profile.exists && profile.datum ? profile.datum : { name: "Garfield", snacks: [] };
    datum.name = "Garfield";
    datum.snacks = datum.snacks || [];
    datum.snacks.push("lasagna");
    profile.datum = datum;
    bot.UpdateDatum(profile);
  }
  return task.Normal;
}

function config(bot) {
  const cfg = bot.GetTaskConfig();
  if (cfg.retVal !== ret.Ok) {
    bot.Say(`JavaScript config unavailable: ${cfg.retVal}`);
    return task.Fail;
  }
  bot.Say(`JavaScript config opening: ${cfg.config.Openings[0]}`);
  bot.SetParameter("JS_LOCAL_DEMO", "ok");
  bot.AddTask("next-task", "from-js");
  return task.Normal;
}

function main(argv) {
  const cmd = argv[2] || "";
  if (cmd === "_configure") {
    return defaultConfig;
  }
  if (cmd === "_init") {
    return task.Normal;
  }

  const bot = new Robot();
  const handlers = { demo, prompt, memory, config };
  if (!handlers[cmd]) {
    bot.Log(log.Error, `unknown JavaScript local command: ${cmd}`);
    return task.Fail;
  }
  return handlers[cmd](bot);
}

main(process.argv || []);

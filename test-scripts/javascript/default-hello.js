const { Robot, task, log } = require('gopherbot_v1')();

const defaultConfig = `---
Commands:
  - Regex: (?i:js default hello)
    Command: hello
`;

function main(argv) {
  const cmd = argv[2] || "";
  if (cmd === "_configure") {
    return defaultConfig;
  }
  if (cmd === "_init") {
    return task.Normal;
  }

  const bot = new Robot();
  if (cmd === "hello") {
    const botName = bot.GetBotAttribute("fullName").attribute || "the robot";
    const sender = bot.GetSenderAttribute("fullName").attribute || bot.user;
    bot.Say(`JavaScript says hello from ${botName} to ${sender} in #${bot.channel}.`);
    bot.Reply("This script only needs the installed default fixture.");
    return task.Normal;
  }

  bot.Log(log.Error, `unknown JavaScript default command: ${cmd}`);
  return task.Fail;
}

main(process.argv || []);

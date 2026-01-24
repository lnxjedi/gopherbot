// chuck.js
// JavaScript version of the Chuck Norris plugin using gopherbot_http.

const defaultConfig = `---
MessageMatchers:
  - Regex: (?i:\\bchuck norris\\b)
    Command: chuck
Config:
  Openings:
  - "Chuck Norris?!?! He's AWESOME!!!"
  - "Oh cool, you like Chuck Norris, too?"
  - "Speaking of Chuck Norris - "
  - "Hey, I know EVERYTHING about Chuck Norris!"
  - "I'm a HUUUUGE Chuck Norris fan!"
  - "Not meaning to eavesdrop or anything, but are we talking about CHUCK NORRIS ?!?"
  - "Oh yeah, Chuck Norris! The man, the myth, the legend."
`;

const { Robot, ret, task, log } = require("gopherbot_v1")();

function handleChuck(bot) {
  const cfg = bot.GetTaskConfig();
  if (cfg.retVal != ret.Ok) {
    bot.Say("Uh-oh, I wasn't able to find any configuration");
    return task.Normal;
  }
  const openings = cfg.config.Openings;
  const opening = bot.RandomString(openings);

  const http = require("gopherbot_http");
  const client = http.createClient({
    baseURL: "https://api.chucknorris.io",
    timeoutMs: 10000,
    throwOnHTTPError: true,
  });

  try {
    const data = client.getJSON("/jokes/random");
    const joke = data && data.value ? data.value : null;
    bot.Say(`${opening} Did you know ...?`);
    bot.Pause(2);
    if (joke) {
      bot.Say(joke);
    } else {
      bot.Say("Chuck Norris is too awesome to describe right now.");
    }
  } catch (err) {
    bot.Log(log.Error, `chuck.js HTTP error: ${err}`);
    bot.Say("I tried to fetch a Chuck Norris joke but something broke.");
  }
}

function handler(argv) {
  const cmd = argv.length > 2 ? argv[2] : "";

  switch (cmd) {
    case "configure":
      return defaultConfig;
    case "chuck":
      handleChuck(new Robot());
      return task.Normal;
    default:
      return task.Fail;
  }
}

// @ts-ignore - the "process" object is created by goja
handler(process.argv || []);

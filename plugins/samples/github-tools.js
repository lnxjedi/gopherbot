const defaultConfig = `---
Commands:
- Regex: '(?i:github[- ]my[- ]review[- ]requests)'
  Command: review-requests
  Usage: "github-my-review-requests"
  Summary: "List open pull requests currently requesting your review."
  Examples:
  - "(alias) github-my-review-requests"
- Regex: '(?i:github[- ]run[- ]workflow ([\\w.-]+\\/[\\w.-]+) ([\\w./-]+) ([\\w./-]+))'
  Command: run-workflow
  Contexts: [ "repository", "workflow", "ref" ]
  Usage: "github-run-workflow <owner/repo> <workflow> <ref>"
  Summary: "Trigger a GitHub Actions workflow dispatch on the given ref."
  Examples:
  - "(alias) github-run-workflow octo-org/octo-repo deploy.yml main"
Config:
  Provider: github
  APIBaseURL: https://api.github.com
  ReviewQuery: "is:open is:pr review-requested:@me archived:false"
  MaxItems: 10
`;

const { Robot, ret, task } = require('gopherbot_v1')();
const http = require("gopherbot_http");

function loadConfig(bot) {
  const cfg = bot.GetTaskConfig();
  if (cfg.retVal === ret.Ok && cfg.config) {
    return cfg.config;
  }
  return {
    Provider: "github",
    APIBaseURL: "https://api.github.com",
    ReviewQuery: "is:open is:pr review-requested:@me archived:false",
    MaxItems: 10,
  };
}

function githubClient(cfg, token) {
  return http.createClient({
    baseURL: cfg.APIBaseURL || "https://api.github.com",
    headers: {
      Authorization: "Bearer " + token,
      Accept: "application/vnd.github+json",
    },
    timeoutMs: 10000,
    throwOnHTTPError: true,
  });
}

function requireToken(bot, provider, user) {
  const tokenResult = bot.GetIdentityCredential(provider, user);
  if (tokenResult.retVal !== ret.Ok || !tokenResult.credential || !tokenResult.credential.value) {
    switch (tokenResult.retVal) {
      case ret.IdentityNotLinked:
        bot.Say("You don't have a linked GitHub account yet. Try `link-github` first.");
        break;
      case ret.IdentityReauthRequired:
        bot.Say("Your linked GitHub account needs to be linked again. Try `link-github`.");
        break;
      default:
        bot.Say("I couldn't get a usable GitHub token right now.");
        break;
    }
    return null;
  }
  return tokenResult.credential.value;
}

function repoLabel(item) {
  if (!item || !item.repository_url) {
    return "unknown-repo";
  }
  const bits = String(item.repository_url).split("/");
  return bits.slice(-2).join("/");
}

function listReviewRequests(bot, cfg) {
  const token = requireToken(bot, cfg.Provider || "github", bot.user);
  if (!token) {
    return task.Fail;
  }
  const client = githubClient(cfg, token);
  const limit = cfg.MaxItems || 10;
  const data = client.getJSON("/search/issues", {
    query: {
      q: cfg.ReviewQuery || "is:open is:pr review-requested:@me archived:false",
      per_page: String(limit),
    },
  });
  const items = Array.isArray(data.items) ? data.items : [];
  if (items.length === 0) {
    bot.Say("No open pull requests are currently requesting your review.");
    return task.Normal;
  }
  const lines = [`Open review requests: ${data.total_count || items.length}`];
  for (let i = 0; i < items.length && i < limit; i++) {
    const item = items[i];
    lines.push(`- ${repoLabel(item)} #${item.number}: ${item.title}`);
  }
  bot.Say(lines.join("\n"));
  return task.Normal;
}

function runWorkflow(bot, cfg, repo, workflow, ref) {
  const token = requireToken(bot, cfg.Provider || "github", bot.user);
  if (!token) {
    return task.Fail;
  }
  const client = githubClient(cfg, token);
  client.request({
    method: "POST",
    path: `/repos/${repo}/actions/workflows/${workflow}/dispatches`,
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ ref: ref }),
    throwOnHTTPError: true,
  });
  bot.Say(`Triggered workflow \`${workflow}\` for \`${repo}\` on ref \`${ref}\`.`);
  return task.Normal;
}

function handler(argv) {
  const cmd = argv.length > 2 ? argv[2] : '';
  switch (cmd) {
    case 'init':
      return task.Normal;
    case 'configure':
      return defaultConfig;
    case 'review-requests':
      return listReviewRequests(new Robot(), loadConfig(new Robot()));
    case 'run-workflow':
      {
        const bot = new Robot();
        const cfg = loadConfig(bot);
        return runWorkflow(bot, cfg, argv[3], argv[4], argv[5]);
      }
    default:
      return task.Fail;
  }
}

handler(process.argv || []);

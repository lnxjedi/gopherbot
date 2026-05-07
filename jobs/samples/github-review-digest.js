const defaultConfig = `---
Config:
  Provider: github
  User: david
  APIBaseURL: https://api.github.com
  ReviewQuery: "is:open is:pr review-requested:@me archived:false"
  MaxItems: 5
`;

const { Robot, ret, task } = require('gopherbot_v1')();
const http = require("http");

function loadConfig(bot) {
  const cfg = bot.GetTaskConfig();
  if (cfg.retVal === ret.Ok && cfg.config) {
    return cfg.config;
  }
  return {
    Provider: "github",
    APIBaseURL: "https://api.github.com",
    ReviewQuery: "is:open is:pr review-requested:@me archived:false",
    MaxItems: 5,
    User: "",
  };
}

function githubRequest(cfg, token, method, path, options) {
  const baseURL = cfg.APIBaseURL || "https://api.github.com";
  const opts = options || {};
  opts.headers = Object.assign({
    Authorization: "Bearer " + token,
    Accept: "application/vnd.github+json",
  }, opts.headers || {});
  opts.timeout = opts.timeout || "10s";
  const response = http.request(method, baseURL + path, opts);
  if (!response.ok) {
    throw new Error(`GitHub API HTTP ${response.statusCode}: ${response.body}`);
  }
  return response;
}

function githubClient(cfg, token) {
  return {
    searchIssues(options) {
      return githubRequest(cfg, token, "GET", "/search/issues", options).json;
    },
  };
}

function repoLabel(item) {
  if (!item || !item.repository_url) {
    return "unknown-repo";
  }
  const bits = String(item.repository_url).split("/");
  return bits.slice(-2).join("/");
}

function handler(argv) {
  const cmd = argv.length > 2 ? argv[2] : '';
  if (cmd === 'init') {
    return task.Normal;
  }
  if (cmd === 'configure') {
    return defaultConfig;
  }

  const bot = new Robot();
  const cfg = loadConfig(bot);
  const targetUser = (cfg.User || "").trim();
  if (!targetUser) {
    bot.Say("github-review-digest requires Config.User to be set.");
    return task.Fail;
  }
  const tokenResult = bot.GetIdentityCredential(cfg.Provider || "github", targetUser);
  if (tokenResult.retVal !== ret.Ok || !tokenResult.credential || !tokenResult.credential.value) {
    bot.Say(`I couldn't get a GitHub token for ${targetUser}.`);
    return task.Fail;
  }
  const client = githubClient(cfg, tokenResult.credential.value);
  const limit = cfg.MaxItems || 5;
  const data = client.searchIssues({
    query: {
      q: cfg.ReviewQuery || "is:open is:pr review-requested:@me archived:false",
      per_page: String(limit),
    },
  });
  const items = Array.isArray(data.items) ? data.items : [];
  if (items.length === 0) {
    bot.SendUserMessage(targetUser, "GitHub review digest: no open review requests right now.");
    return task.Normal;
  }
  const lines = [`GitHub review digest: ${data.total_count || items.length} open review request(s)`];
  for (let i = 0; i < items.length && i < limit; i++) {
    const item = items[i];
    lines.push(`- ${repoLabel(item)} #${item.number}: ${item.title}`);
  }
  bot.SendUserMessage(targetUser, lines.join("\n"));
  return task.Normal;
}

handler(process.argv || []);

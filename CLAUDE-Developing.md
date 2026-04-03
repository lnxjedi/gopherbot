# Developing Gopherbot with Claude Code

Practical guidance for getting the most out of Claude Code on this codebase, especially on a budget.

## Model and Effort Selection

**Default to Sonnet on auto effort.** This handles 80-90% of tasks and is the most quota-efficient choice. Reserve Opus/high-effort for:
- Architecture decisions and design review
- Debugging subtle cross-subsystem issues
- Diagnosing why something failed in a way you don't understand

**Switch back to Sonnet immediately after.** Opus burns quota roughly 5x faster. One Opus conversation doing implementation work can eat a full day's budget.

Use `/model` and `/effort` to switch in-conversation without starting over.

## Prompting for Multi-File Changes

Gopherbot's widest pain point for AI is the **language binding surface** — a single API method touches 11+ files across Go, Lua, JS, Python, Ruby, Bash, and Yaegi. Claude (especially Sonnet) can lose track of secondary update sites.

**Always name the files explicitly.** Instead of:

> Add a FrobnicateWidget method to the Robot API.

Say:

> Add a FrobnicateWidget method to the Robot API. Follow the pattern used by EncryptSecret. See `aidocs/EXTENSION_API.md` "Adding a New Robot API Method" for the full checklist of files to update.

Or even more directly:

> Add FrobnicateWidget to the Robot API. Update all of these files, following the EncryptSecret pattern in each:
> - robot/robot.go, bot/robot.go, bot/http.go
> - modules/lua/bot_api.go, modules/javascript/bot_api.go, modules/gsh/commands.go
> - bot/pipeline_rpc_interpreter.go, modules/yaegi-dynamic-go/yaegi_symbols.go
> - lib/gopherbot_v2.py, lib/gopherbot_v1.rb, lib/gopherbot_v1.sh
> - lib/gopherbot_v1.lua, lib/gopherbot_v1.js

The 5 extra lines in your prompt save multiple correction rounds (and the quota they burn).

## Prompting for Existing Patterns

Claude works best when you point it at a concrete example to follow. Gopherbot has many parallel structures, so this is usually easy:

> Add a new brain provider under brains/redis/. Follow the same structure as brains/dynamodb/.

> Add a test case for FrobnicateWidget in test/js_full_test.go. Follow the EncryptSecret test pattern in the same file.

When the pattern is complex or non-obvious, point at the specific function:

> Add HTTP dispatch for FrobnicateWidget in bot/http.go. Follow the EncryptSecret case in the same switch block.

## Task Sizing

**Smaller tasks burn less quota and produce better results.** Claude Code is interactive — every round-trip carries the full conversation context. A task that takes 15 exchanges costs more (in tokens) than three tasks that each take 5 exchanges.

Break large stories into steps:

1. "Add FrobnicateWidget to the Go interface and engine implementation" (robot/robot.go + bot/robot.go)
2. "Add FrobnicateWidget to the HTTP handler and all interpreter bridges" (bot/http.go + modules/*)
3. "Add FrobnicateWidget to all external language libraries" (lib/*)
4. "Add test coverage for FrobnicateWidget" (test/ + plugins/test/)
5. "Update aidocs/EXTENSION_API.md with FrobnicateWidget docs"

Each step is self-contained and verifiable. If step 2 goes sideways, you haven't wasted the context budget from steps 3-5.

## When to Start a New Conversation

Context grows over the conversation and older messages eventually get compressed. Start fresh when:
- The current task is done and the next one is unrelated
- You've been going back and forth on a problem for 10+ exchanges without resolution
- You're switching subsystems entirely (e.g., from connector work to brain provider work)

Don't try to do a full day's work in one conversation. Three focused 20-minute sessions beat one sprawling 60-minute session.

## Avoiding Quota Waste

**Don't ask Claude to explore the codebase open-endedly.** Instead of "how does the pipeline system work?", read `aidocs/PIPELINE_LIFECYCLE.md` yourself, then ask specific questions.

**Don't use high effort for implementation.** High effort means more thinking tokens per response. It helps with analysis and design; it doesn't make code edits better.

**Don't ask for summaries of what just happened.** Claude will give you a recap after each change by default. If you find this wasteful, say "skip the summary" or "just make the change" in your prompt.

**Don't re-read files Claude already read.** If the conversation is still going and you're working on the same files, Claude remembers what it read. You don't need to say "read bot/http.go again" unless the file actually changed outside the conversation.

## What Claude Code Does Well on This Codebase

- Following an explicit pattern across multiple files when you name them
- Writing integration test cases that follow existing harness conventions
- Updating docs to match code changes
- Refactoring within a single subsystem when the scope is clear
- Explaining unfamiliar code when pointed at specific files/functions

## What to Watch For

- **Missed update sites**: always verify multi-language changes hit every binding. The API method checklist in `aidocs/EXTENSION_API.md` exists for this reason.
- **Invented APIs**: Claude may hallucinate methods or config fields that don't exist. If a suggestion references something you haven't seen before, ask it to grep for it first.
- **Over-engineering**: Claude tends to add error handling, comments, and abstractions beyond what's needed. If a change comes back fancier than expected, push back — "simpler, follow the existing pattern."
- **Stale context on long conversations**: if Claude starts making references to code that looks wrong, it may be working from compressed earlier context. Start a new conversation.

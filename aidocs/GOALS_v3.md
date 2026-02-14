# The Road to Gopherbot v3

This document defines the intent, scope, and evaluation criteria for releasing **Gopherbot v3**. It is written primarily for contributors working on the core codebase (human or AI) and is intended to function as a stable reference point across many development sessions.

The contents here describe *what must be true* for v3 to be considered complete, and *why* those criteria exist. Some sections may be refined over time, but this document has authority over task-level instructions and should be treated as the north star for v3-related work.

## What v3 Represents

Gopherbot v3 is intended to be a significant step toward a more self-contained, internally coherent automation framework. Compared to v2, v3 should reduce environmental fragility, lower the barrier to writing and maintaining extensions, and make the system easier to reason about for both humans and automation.

Put simply: v3 should feel like a system with a clear internal model, not a thin orchestration layer over a pile of external tools.

## Weaknesses of v2 (Diagnostic Context)

The following points describe limitations of the current v2 architecture. They are provided as explanatory context, not as a literal task checklist.

* The core distribution relies too heavily on the presence and behavior of external tooling such as bash, jq, git, and ssh.
* Writing extensions (plugins, tasks, jobs, etc.) often requires depending on external interpreters, which raises the barrier to entry and increases operational fragility.
* The groups system is weak and insufficiently integrated across features.
* The help system is limited and not group-aware.
* The relationship between default configuration and custom configuration is not well documented or well understood.
* Creating new extensions is harder than it should be; the workflow should explicitly support AI-assisted development.
* Bootstrapping new robots is unnecessarily difficult.
* The end-user documentation (maintained in the separate gopherbot-docs repository) is significantly out of date.
* Brain persistence via git is awful/ugly

## Early Enabling Goals

These goals are intended to unblock and accelerate later v3 work. Changes that advance these goals are generally preferred unless a task explicitly states otherwise.
* Create and maintain high-quality AI-oriented context documentation to reduce repeated onboarding costs and improve development velocity.
* Update project dependencies to modern, supported versions.
* Restore and fix the development and test workflow, including making `make test` reliable and meaningful.
* Remove the git-based file brain backup/restore scripts and other related code
* Expand extension-language integration tests (JS/Lua) to cover messaging, config, HTTP, and core Robot API surface.

## Core v3 Goals

### Backward Compatibility
To the greatest extent possible, v3 should be fully backwards-compatible with existing robots created from the original `robot.skel`.

### Reduce Dependence on External Tools and Interpreters

A primary goal of v3 is to minimize reliance on external executables and interpreters in the core distribution.

The engine already includes built-in interpreters for Go, Lua, and JavaScript, along with internal support for git and ssh functionality. Existing robots can already load their repositories without depending on external tooling.

For v3, the intent is that core-distributed plugins, jobs, and related components should preferentially use built-in interpreters rather than shelling out to external tools. This improves portability, predictability, and testability.

The current preferred order for implementing or migrating functionality is:
Lua first for readability and approachability, then Go for performance and safety, then JavaScript where appropriate.

(NOTE: This ordering reflects current judgment and may evolve; the underlying intent—reducing external dependencies—should be treated as stable.)

### Improve the Extension Authoring Experience

Writing new extensions should be straightforward, well-documented, and achievable without deep prior knowledge of the codebase.

The system should explicitly support AI-assisted workflows for creating plugins, tasks, and jobs, including clear contracts, examples, and scaffolding.

Ideally, a new developer, after reading some initial documentation, should be able to ask an AI e.g. "based on the documentation and examples in `writing-extensions/`, please create a new (plugin, job) that does (something)."

Progress notes:
- JS and Lua full integration tests exist and are gated via `RUN_FULL`/`TEST` (see `test/` and `aidocs/TESTING_CURRENT.md`).
- Local HTTP test server exists for deterministic JS/Lua HTTP coverage.
- JS and Lua HTTP helper modules are available for extension authors (see `aidocs/JS_HTTP_API.md`, `aidocs/LUA_HTTP_API.md`).
- Long-term TODO: add a Bash Full integration suite with emphasis on formatting behavior parity (including `-f` fixed-format handling). The Bash library (`lib/gopherbot_v1.sh`) likely needs a cleanup/rewrite.

### Strengthen and Integrate the Groups and Help Systems

The groups system should be more expressive and better integrated across the framework.

Help and discovery mechanisms should be group-aware and provide clear, contextual guidance to users.

* The help sections of plugin configuration should have a new format that generates more easily readable help
* Robot startup should use more than a simple author-provided keyword list; we'll need to come up with a new design
* When users ask for help, they should only be shown commands which they are able to run

### Improve Configuration Clarity and Bootstrapping

The relationship between default configuration and user customization should be clearly defined and documented.

Bootstrapping a new robot should be significantly easier than in v2, with fewer hidden assumptions and clearer initial structure.

The documentation should only instruct the user in how to start a demo robot in a container or empty directory, and the built-in "welcome" plugin should tell the user they can run e.g. ";new robot". That command should start a well-documented process entirely in the terminal window, at the end of which the new robot is stored in a remote git repository, and the user is given clear instructions on what is needed to start their robot running in a container, ec2 instance, etc. (mainly, here are the environment variables you need to define when launching the "gopherbot" executable in production in order to start your robot). The process should end with a reference for where to configure a persistent brain.

### Make Sure Robots have Brains

Since we're getting rid of the default backup/restore to git for the file brain, we need good documentation/process for helping the user establish a persistent brain, maybe via CloudFlare KV.

### Improve Robustness and Security by running Multi-Process
In Gopherbot v3, compiled-in plugins and jobs will continue to run in in-process threads, but extensions loaded from file will run in separate gopherbot child processes. This isolates the main engine from any panics or freezes for most of the work a robot does, and also makes for a cleaner means of implementing privilege separation. It also paves the way for better support of the working directory method(s).

### Make Good Use of AI with Included Components
The current ruby AI implementation should be replaced with a native implementation, making better use of memories. The full-name fallback should be the same; "Floyd, what is the meaning of life?" should go straight to the AI (with the regular, normal system prompt additions). We should update the alias fallback (robot addressed with alias, but no command matched) to be an ai-augmented help - the robot should reply in a thread with something reasonable based on the robot's configuration; for instance "Oops, you typed that in the wrong channel - try that in #devops or #chatops", or "It looks like you're trying to start a remote dev environment, but you didn't supply a valid type ...".

So, AI should be fully integrated if desired, but the AI integration will never actually *do* anything, even with confirmation (at least at this stage) - it'll only ever help the user perform operations. Gopherbot remains a deterministic automation framework - when you ask for a cup of coffee, it just makes the damned coffee the same way every time, without futzing with AI.

### Stretch Goal: Enable a persistent AI engine with support for MCPs
Just imagine: "Astro, the app is really slow - can you see what's going on?" - using RedHat's kubernetes mcp server and maybe Grafana Loki mcp.

## Non-Goals for v3

The following items are explicitly *not* required for v3 and should not block release unless this document is updated to say otherwise.

* Perfect backward compatibility with all v2 robots - when breaking changes are required to meet the above goals, we will document what the user needs to know in an `Upgrading_to_v3.md` document.

## How We Decide v3 Is Ready

Gopherbot v3 is ready to ship when the system demonstrably meets the intent described above, and when a contributor familiar with this document can reasonably argue that v3 is qualitatively different from v2 in terms of internal coherence, portability, and developer experience.

In practice, this means that core functionality no longer depends on fragile external tooling, extension authorship feels intentional rather than accidental, onboarding costs are reduced, and the system’s structure is easier to explain and reason about.

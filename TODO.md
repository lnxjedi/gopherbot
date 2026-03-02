# Upcoming tasks:

### Focused help system
Already documented in GOALS, the help system needs to be more helpful. Needs QA.

### Make built-in interpreters more powerful
Most functionality should be achievable with Lua, JavaScript or Go (yaegi) - certainly all *included* functionality, like protocol setup.

Implement DevOps helpers for JS/Lua (workspace-safe file ops + local exec wrappers, plus tests) — see checklists in `aidocs/JS_METHOD_CHECKLIST.md` and `aidocs/LUA_METHOD_CHECKLIST.md`. These are the basic methods needed by DevOps engineers to do most common automation tasks.

### Improved Slack Support for Fixed and Variable
Using BlockKit and rich_text and rich_text_preformatted should give "perfect" Fixed and Variable support.
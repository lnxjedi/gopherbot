# Language Choices

Gopherbot supports several extension languages, but they are not all equally recommended for new work.

## Recommended order for new v3 work

1. **Lua**
   - built in
   - fast local loop
   - good fit for readable operational automation
2. **Go**
   - strongest type safety
   - best for heavier logic and long-lived maintained plugins
3. **JavaScript**
   - built in
   - useful when your team already thinks in JS

## Still supported

- Bash
- Python
- Ruby

These are valuable for existing robots and for quick integrations, but they depend more on external tooling and are not the main design direction for v3.

## One practical rule

If you can write the extension cleanly in a built-in interpreter or Go, do that first. Reach for external runtimes when they are clearly the right fit, not by habit.

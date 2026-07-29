# macOS Privsep Decision

macOS follows the same process-scoped model as Linux/BSD: the parent returns to
the invoking robot UID and each file-backed child commits once to the robot or
unprivileged UID before extension code.

Do not reintroduce thread-pinned credential switching. Compiled Go remains
trusted in-process code. Privsep is UID-only; inherited groups are not a
boundary, setgid must remain clear, and host privileges should be granted by
UID.

The same setuid tamper checks, install-path reachability requirement, `.env`
mode, startup self-check, and manual validation in `AGENTS.md` apply on macOS.
Platform syscall mechanics belong in `bot/privsep_darwin.go` and its tests.

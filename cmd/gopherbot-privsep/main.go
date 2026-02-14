package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

const defaultTarget = "gopherbot"
const targetMode os.FileMode = 0o4755

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [path-to-gopherbot]\n", filepath.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "Sets ownership to user 'nobody' and enables the setuid bit.")
}

func main() {
	target := defaultTarget
	if len(os.Args) > 2 {
		usage()
		os.Exit(2)
	}
	if len(os.Args) == 2 {
		target = os.Args[1]
	}

	if err := setuidNobody(target); err != nil {
		fmt.Fprintf(os.Stderr, "privsep helper failed: %v\n", err)
		os.Exit(1)
	}
}

func setuidNobody(target string) error {
	if os.Geteuid() != 0 {
		return errors.New("must run as root (or setuid-root)")
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}
	if filepath.Base(absTarget) != "gopherbot" {
		return fmt.Errorf("refusing target %q: helper only supports binaries named 'gopherbot'", absTarget)
	}

	info, err := os.Stat(absTarget)
	if err != nil {
		return fmt.Errorf("stat target: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("target %q is not a regular file", absTarget)
	}

	nobody, err := user.Lookup("nobody")
	if err != nil {
		return fmt.Errorf("looking up user 'nobody': %w", err)
	}
	uid, err := strconv.Atoi(nobody.Uid)
	if err != nil {
		return fmt.Errorf("parsing nobody uid %q: %w", nobody.Uid, err)
	}
	gid, err := strconv.Atoi(nobody.Gid)
	if err != nil {
		return fmt.Errorf("parsing nobody gid %q: %w", nobody.Gid, err)
	}

	if err := os.Chown(absTarget, uid, gid); err != nil {
		return fmt.Errorf("chown target to nobody: %w", err)
	}

	if err := os.Chmod(absTarget, targetMode); err != nil {
		return fmt.Errorf("chmod target setuid: %w", err)
	}

	updated, err := os.Stat(absTarget)
	if err != nil {
		return fmt.Errorf("stat target after update: %w", err)
	}
	st, ok := updated.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("unable to read unix stat fields for verification")
	}
	if int(st.Uid) != uid {
		return fmt.Errorf("verification failed: expected uid %d, got %d", uid, st.Uid)
	}
	if updated.Mode()&os.ModeSetuid == 0 {
		return errors.New("verification failed: setuid bit is not set")
	}
	if updated.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("verification failed: target has group/world writable bits set: %o", updated.Mode().Perm())
	}

	fmt.Printf("Updated %s\n", absTarget)
	fmt.Printf("owner: nobody (%d:%d)\n", uid, gid)
	fmt.Printf("mode: %s\n", updated.Mode())
	return nil
}

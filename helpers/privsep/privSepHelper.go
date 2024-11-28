// privSepHelper.go
//
// A helper program for privilege separation.
// This program should be compiled and setuid root.
//
// Usage:
//   privSepHelper <command> [args...]
//
// Example:
//   privSepHelper my/script.sh "foo" "bar" ""

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// getParentProcessInfo retrieves the Real UID, Effective UID, Saved Set-UID,
// and Real GID of the parent process identified by ppid.
// It reads and parses /proc/[ppid]/status to obtain the necessary information.
func getParentProcessInfo(ppid int) (ruid, euid, rgid int, err error) {
	statusPath := fmt.Sprintf("/proc/%d/status", ppid)
	file, err := os.Open(statusPath)
	if err != nil {
		err = fmt.Errorf("failed to open %s: %v", statusPath, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var uidLine, gidLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Uid:") {
			uidLine = line
		} else if strings.HasPrefix(line, "Gid:") {
			gidLine = line
		}
		if uidLine != "" && gidLine != "" {
			break
		}
	}

	if uidLine == "" || gidLine == "" {
		err = fmt.Errorf("Uid or Gid line not found in %s", statusPath)
		return
	}

	// Parse Uid line
	uidFields := strings.Fields(uidLine)
	if len(uidFields) < 4 {
		err = fmt.Errorf("invalid Uid line in %s: %s", statusPath, uidLine)
		return
	}
	ruid, err = strconv.Atoi(uidFields[1])
	if err != nil {
		err = fmt.Errorf("invalid Real UID in %s: %v", statusPath, err)
		return
	}
	euid, err = strconv.Atoi(uidFields[2])
	if err != nil {
		err = fmt.Errorf("invalid Effective UID in %s: %v", statusPath, err)
		return
	}

	// Parse Gid line
	gidFields := strings.Fields(gidLine)
	if len(gidFields) < 2 {
		err = fmt.Errorf("invalid Gid line in %s: %s", statusPath, gidLine)
		return
	}
	rgid, err = strconv.Atoi(gidFields[1])
	if err != nil {
		err = fmt.Errorf("invalid Real GID in %s: %v", statusPath, err)
		return
	}

	return
}

func main() {
	// Step 1: Check if running with Effective UID root
	euid := syscall.Geteuid()
	if euid != 0 {
		fmt.Fprintf(os.Stderr, "Error: privSepHelper must be run with effective UID root. Current EUID: %d\n", euid)
		os.Exit(1)
	}

	// Step 2: Get Parent Process ID (PPID)
	ppid := os.Getppid()

	// Step 3: Retrieve Parent Process UIDs and GIDs
	ruid, parentEuid, rgid, err := getParentProcessInfo(ppid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving parent process info: %v\n", err)
		os.Exit(1)
	}

	// Step 4: If Parent's EUID == RUID, exit with code 2
	if parentEuid == ruid {
		fmt.Fprintf(os.Stderr, "Error: Parent process (PID %d) has EUID (%d) equal to RUID (%d), which is not allowed.\n", ppid, parentEuid, ruid)
		os.Exit(2)
	}

	// Step 5: Parent's RUID and RGID have been obtained (ruid and rgid)

	// Step 6: Drop Supplemental Groups
	if err := syscall.Setgroups([]int{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error dropping supplemental groups: %v\n", err)
		os.Exit(3)
	}

	// Step 7: Set GID to Parent's RGID Permanently
	if err := syscall.Setregid(rgid, rgid); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting GID to parent's RGID (%d): %v\n", rgid, err)
		os.Exit(4)
	}

	// Step 8: Set UID to Parent's RUID Permanently
	if err := syscall.Setreuid(ruid, ruid); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting UID to parent's RUID (%d): %v\n", ruid, err)
		os.Exit(5)
	}

	// Step 9: Execute the Command with Arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args...]\n", os.Args[0])
		os.Exit(6) // Exit code 6 for incorrect usage
	}

	cmdPath := os.Args[1]
	cmdArgs := os.Args[2:]

	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		// If the command exited with a non-zero exit code, retrieve it
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				fmt.Fprintf(os.Stderr, "Command exited with status %d\n", status.ExitStatus())
				os.Exit(status.ExitStatus())
			}
		}
		// If there was an error starting the command, exit with code 7
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(7)
	}

	// If command runs successfully, exit with its exit code (0)
	os.Exit(0)
}

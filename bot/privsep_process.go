package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	privsepChildRoleEnv      = "GOPHER_PRIVSEP_CHILD_ROLE"
	privsepSelfCheckCommand  = "privsep-self-check"
	privsepRoleNone          = privsepChildRole("")
	privsepRolePrivileged    = privsepChildRole("privileged")
	privsepRoleUnprivileged  = privsepChildRole("unprivileged")
	privsepSelfCheckExitFail = 2
)

type privsepChildRole string

type privsepSupplementaryGroupPolicy struct {
	allowAll bool
	allowed  []int
}

type privsepIdentityReport struct {
	UID    int   `json:"uid"`
	EUID   int   `json:"euid"`
	GID    int   `json:"gid"`
	EGID   int   `json:"egid"`
	Groups []int `json:"groups"`
}

func (r privsepChildRole) valid() bool {
	switch r {
	case privsepRoleNone, privsepRolePrivileged, privsepRoleUnprivileged:
		return true
	default:
		return false
	}
}

func privsepRoleFromString(raw string) (privsepChildRole, error) {
	role := privsepChildRole(strings.TrimSpace(raw))
	if !role.valid() {
		return privsepRoleNone, fmt.Errorf("invalid privsep child role %q", raw)
	}
	return role, nil
}

func privsepRoleEnv(role privsepChildRole) []string {
	if role == privsepRoleNone {
		return nil
	}
	return []string{privsepChildRoleEnv + "=" + string(role)}
}

func privsepRoleForExecution(privileged bool) privsepChildRole {
	if !privSep {
		return privsepRoleNone
	}
	if privileged {
		return privsepRolePrivileged
	}
	return privsepRoleUnprivileged
}

func appendPrivsepRoleEnv(extra []string, role privsepChildRole) []string {
	if role == privsepRoleNone {
		return extra
	}
	return append(extra, privsepRoleEnv(role)...)
}

func commitPrivsepChildFromEnv(required bool) int {
	raw := strings.TrimSpace(os.Getenv(privsepChildRoleEnv))
	if raw == "" {
		if required && privSep {
			fmt.Fprintf(os.Stderr, "Missing %s for privsep child\n", privsepChildRoleEnv)
			return privsepSelfCheckExitFail
		}
		return 0
	}
	role, err := privsepRoleFromString(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return privsepSelfCheckExitFail
	}
	if role == privsepRoleNone {
		if required && privSep {
			fmt.Fprintf(os.Stderr, "Empty privsep role for required child\n")
			return privsepSelfCheckExitFail
		}
		return 0
	}
	if !privSep {
		fmt.Fprintf(os.Stderr, "Privsep role %q requested but privilege separation is not active\n", role)
		return privsepSelfCheckExitFail
	}
	if err := commitPrivsepChildRole(role); err != nil {
		fmt.Fprintf(os.Stderr, "Committing privsep child role %q: %v\n", role, err)
		return privsepSelfCheckExitFail
	}
	return 0
}

func runPrivsepSelfCheck() int {
	report, err := currentPrivsepIdentityReport()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Collecting privsep identity report: %v\n", err)
		return privsepSelfCheckExitFail
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "Encoding privsep identity report: %v\n", err)
		return privsepSelfCheckExitFail
	}
	return 0
}

func validatePrivsepSupplementaryGroupPolicy(policy privsepSupplementaryGroupPolicy, report privsepIdentityReport) error {
	if policy.allowAll {
		return nil
	}
	allowed := make(map[int64]struct{}, len(policy.allowed)+2)
	allowed[int64(unprivGID)] = struct{}{}
	allowed[int64(uint32(unprivGID))] = struct{}{}
	for _, gid := range policy.allowed {
		if gid < 0 {
			return fmt.Errorf("PrivsepAllowedSupplementaryGroups contains negative group ID %d", gid)
		}
		allowed[int64(gid)] = struct{}{}
	}
	var disallowed []string
	for _, gid := range report.Groups {
		if _, ok := allowed[int64(gid)]; ok {
			continue
		}
		disallowed = append(disallowed, strconv.Itoa(gid))
	}
	if len(disallowed) > 0 {
		return fmt.Errorf("privsep unprivileged child retained disallowed supplementary groups: %s", strings.Join(disallowed, ", "))
	}
	return nil
}

func validatePrivsepStartupPolicy(policy privsepSupplementaryGroupPolicy) error {
	if !privSep {
		return nil
	}
	report, err := runPrivsepStartupSelfCheck()
	if err != nil {
		return err
	}
	if report.UID != unprivUID || report.EUID != unprivUID {
		return fmt.Errorf("privsep self-check unprivileged child UID mismatch: uid/euid %d/%d, want %d/%d", report.UID, report.EUID, unprivUID, unprivUID)
	}
	if report.GID != unprivGID || report.EGID != unprivGID {
		return fmt.Errorf("privsep self-check unprivileged child GID mismatch: gid/egid %d/%d, want %d/%d", report.GID, report.EGID, unprivGID, unprivGID)
	}
	if err := validatePrivsepSupplementaryGroupPolicy(policy, report); err != nil {
		return err
	}
	if policy.allowAll && len(report.Groups) > 0 {
		Log(robot.Audit, "PRIVSEP - PrivsepAllowAllSupplementaryGroups enabled; unprivileged child retained groups: %v", report.Groups)
	}
	return nil
}

func runPrivsepStartupSelfCheck() (privsepIdentityReport, error) {
	var report privsepIdentityReport
	cmd := exec.Command(execPath(), privsepSelfCheckCommand)
	env := appendPrivsepRoleEnv(nil, privsepRoleUnprivileged)
	cmd.Env = sanitizedChildEnvironment(env...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return report, fmt.Errorf("privsep self-check failed: %v: %s", err, stderr)
			}
		}
		return report, fmt.Errorf("privsep self-check failed: %v", err)
	}
	if err := json.Unmarshal(out, &report); err != nil {
		return report, fmt.Errorf("decoding privsep self-check report: %v", err)
	}
	return report, nil
}

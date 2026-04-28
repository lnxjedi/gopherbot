package bot

import (
	"strings"
	"testing"
)

func TestValidatePrivsepSupplementaryGroupPolicyAllowsPrimaryGroup(t *testing.T) {
	oldUnprivGID := unprivGID
	t.Cleanup(func() { unprivGID = oldUnprivGID })
	unprivGID = -2

	report := privsepIdentityReport{Groups: []int{int(uint32(unprivGID))}}
	err := validatePrivsepSupplementaryGroupPolicy(privsepSupplementaryGroupPolicy{}, report)
	if err != nil {
		t.Fatalf("validatePrivsepSupplementaryGroupPolicy() error = %v", err)
	}
}

func TestValidatePrivsepSupplementaryGroupPolicyRejectsDisallowedGroups(t *testing.T) {
	oldUnprivGID := unprivGID
	t.Cleanup(func() { unprivGID = oldUnprivGID })
	unprivGID = -2

	report := privsepIdentityReport{Groups: []int{int(uint32(unprivGID)), 20, 701}}
	err := validatePrivsepSupplementaryGroupPolicy(privsepSupplementaryGroupPolicy{allowed: []int{20}}, report)
	if err == nil {
		t.Fatal("validatePrivsepSupplementaryGroupPolicy() error = nil, want disallowed group failure")
	}
	if !strings.Contains(err.Error(), "701") {
		t.Fatalf("validatePrivsepSupplementaryGroupPolicy() error = %q, want disallowed group ID", err.Error())
	}
}

func TestValidatePrivsepSupplementaryGroupPolicyAllowAll(t *testing.T) {
	report := privsepIdentityReport{Groups: []int{20, 701}}
	err := validatePrivsepSupplementaryGroupPolicy(privsepSupplementaryGroupPolicy{allowAll: true}, report)
	if err != nil {
		t.Fatalf("validatePrivsepSupplementaryGroupPolicy() error = %v", err)
	}
}

func TestValidatePrivsepSupplementaryGroupPolicyRejectsNegativeConfig(t *testing.T) {
	report := privsepIdentityReport{Groups: []int{}}
	err := validatePrivsepSupplementaryGroupPolicy(privsepSupplementaryGroupPolicy{allowed: []int{-1}}, report)
	if err == nil {
		t.Fatal("validatePrivsepSupplementaryGroupPolicy() error = nil, want negative group failure")
	}
}

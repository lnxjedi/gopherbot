//go:build test

package main

import "testing"

func TestResolveSuiteSelectorsBySubsystem(t *testing.T) {
	selected, err := resolveSuiteSelectors([]string{"subsystem:pipeline"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) == 0 {
		t.Fatal("subsystem:pipeline selected no suites")
	}
	for _, suite := range selected {
		if !containsAny(suite.Metadata.Subsystems, []string{"pipeline"}) {
			t.Fatalf("%s missing pipeline subsystem: %#v", suite.Name, suite.Metadata)
		}
	}
}

func TestResolveSuiteSelectorsByRuntime(t *testing.T) {
	selected, err := resolveSuiteSelectors([]string{"runtime:lua"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) == 0 {
		t.Fatal("runtime:lua selected no suites")
	}
	for _, suite := range selected {
		if !containsAny(suite.Metadata.Runtimes, []string{"lua"}) {
			t.Fatalf("%s missing lua runtime: %#v", suite.Name, suite.Metadata)
		}
	}
}

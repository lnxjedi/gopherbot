package main

import (
	"net"
	"reflect"
	"strconv"
	"testing"
)

func TestEnsureSSHPortArgsPreservesExplicitFlags(t *testing.T) {
	for _, args := range [][]string{
		{"-level", "info", "-ssh-port", "4222"},
		{"--ssh-port", "4222"},
		{"-ssh-port=4222"},
		{"--ssh-port=0x107e"},
		{"-ssh-port", "4221", "--ssh-port=4222"},
	} {
		t.Run(args[len(args)-1], func(t *testing.T) {
			// An invalid automatic range proves an explicit choice does not probe.
			got, port, err := ensureSSHPortArgs(args, 1, 0)
			if err != nil {
				t.Fatal(err)
			}
			if port != "4222" || !reflect.DeepEqual(got, args) {
				t.Fatalf("args=%v port=%q, want original args and port 4222", got, port)
			}
		})
	}
}

func TestEnsureSSHPortArgsAssignsAvailablePortAndPersistsAcrossRestart(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"-level", "debug", "-ssh-port", "0"}} {
		got, selected, err := ensureSSHPortArgs(args, port, port)
		if err != nil {
			t.Fatal(err)
		}
		want := append(append([]string(nil), args...), "-ssh-port", strconv.Itoa(port))
		if selected != strconv.Itoa(port) || !reflect.DeepEqual(got, want) {
			t.Fatalf("args=%v port=%q, want %v and port %d", got, selected, want, port)
		}
		command := append([]string{"--aidev", "test-token"}, got...)
		command = append(command, "run")
		restarted, restartedPort, err := ensureSSHPortArgs(extractExtraArgs(command), 1, 0)
		if err != nil || restartedPort != selected || !reflect.DeepEqual(restarted, got) {
			t.Fatalf("restart args=%v port=%q err=%v; want %v and %q", restarted, restartedPort, err, got, selected)
		}
	}
}

func TestEnsureSSHPortArgsRejectsUnavailablePortRangeAndInvalidFlags(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if _, _, err := ensureSSHPortArgs(nil, port, port); err == nil {
		t.Fatal("expected error for occupied automatic port range")
	}
	for _, args := range [][]string{{"-ssh-port"}, {"--ssh-port=oops"}, {"-ssh-port", "65536"}} {
		if _, _, err := ensureSSHPortArgs(args, port, port); err == nil {
			t.Fatalf("expected error for invalid args %v", args)
		}
	}
}

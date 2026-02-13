package bot

import (
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestPromptTimeoutForContext(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		task     *Task
		want     time.Duration
	}{
		{
			name:     "default timeout for non-interactive protocol",
			protocol: "slack",
			task:     &Task{taskType: taskGo},
			want:     replyTimeout,
		},
		{
			name:     "default timeout for ssh non-interpreter task",
			protocol: "ssh",
			task:     &Task{taskType: taskExternal, Path: "plugin.py"},
			want:     replyTimeout,
		},
		{
			name:     "extended timeout for ssh compiled Go task",
			protocol: "ssh",
			task:     &Task{taskType: taskGo},
			want:     interactivePromptTimeout,
		},
		{
			name:     "extended timeout for terminal external lua task",
			protocol: "terminal",
			task:     &Task{taskType: taskExternal, Path: "plugin.lua"},
			want:     interactivePromptTimeout,
		},
		{
			name:     "extension check is case-insensitive",
			protocol: "terminal",
			task:     &Task{taskType: taskExternal, Path: "plugin.JS"},
			want:     interactivePromptTimeout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Robot{
				Message: &robot.Message{
					Protocol: getProtocol(tc.protocol),
					Incoming: &robot.ConnectorMessage{Protocol: tc.protocol},
				},
				pipeContext: &pipeContext{currentTask: tc.task},
			}
			got := promptTimeoutForContext(r, tc.task)
			if got != tc.want {
				t.Fatalf("promptTimeoutForContext() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestPromptInternalInterruptedOnShutdownSignal(t *testing.T) {
	prevConnector := interfaces.Connector
	interfaces.Connector = &fakeRuntimeConnector{}
	t.Cleanup(func() {
		interfaces.Connector = prevConnector
	})

	replies.Lock()
	replies.m = make(map[replyMatcher][]replyWaiter)
	replies.Unlock()
	t.Cleanup(func() {
		replies.Lock()
		replies.m = make(map[replyMatcher][]replyWaiter)
		replies.Unlock()
	})

	resetPromptShutdownSignal()
	t.Cleanup(resetPromptShutdownSignal)

	task := &Task{name: "test-prompt", taskType: taskGo}
	r := Robot{
		Message: &robot.Message{
			User:     "alice",
			Channel:  "",
			Protocol: robot.SSH,
			Incoming: &robot.ConnectorMessage{Protocol: "ssh"},
		},
		maps:        &userChanMaps{},
		pipeContext: &pipeContext{currentTask: task},
	}

	type promptResult struct {
		reply string
		ret   robot.RetVal
	}
	resCh := make(chan promptResult, 1)
	go func() {
		rep, ret := r.promptInternal("YesNo", "alice", "", "", "Proceed?")
		resCh <- promptResult{reply: rep, ret: ret}
	}()

	m := replyMatcher{protocol: "ssh", user: "alice", channel: "", thread: ""}
	waitFor(t, "prompt waiter registered", func() bool {
		replies.Lock()
		_, ok := replies.m[m]
		replies.Unlock()
		return ok
	})

	triggerPromptShutdownSignal()

	select {
	case res := <-resCh:
		if res.ret != robot.Interrupted {
			t.Fatalf("promptInternal() ret = %s, want %s", res.ret, robot.Interrupted)
		}
		if res.reply != "" {
			t.Fatalf("promptInternal() reply = %q, want empty", res.reply)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for promptInternal to return after shutdown signal")
	}

	waitFor(t, "prompt waiter removed", func() bool {
		replies.Lock()
		_, ok := replies.m[m]
		replies.Unlock()
		return !ok
	})
}

func TestPromptInternalMatcherIsProtocolScoped(t *testing.T) {
	prevConnector := interfaces.Connector
	interfaces.Connector = &fakeRuntimeConnector{}
	t.Cleanup(func() {
		interfaces.Connector = prevConnector
	})

	replies.Lock()
	replies.m = make(map[replyMatcher][]replyWaiter)
	replies.Unlock()
	t.Cleanup(func() {
		replies.Lock()
		replies.m = make(map[replyMatcher][]replyWaiter)
		replies.Unlock()
	})

	resetPromptShutdownSignal()
	t.Cleanup(resetPromptShutdownSignal)

	task := &Task{name: "test-protocol-scope", taskType: taskGo}
	r := Robot{
		Message: &robot.Message{
			User:     "alice",
			Channel:  "",
			Protocol: robot.SSH,
			Incoming: &robot.ConnectorMessage{Protocol: "ssh"},
		},
		maps:        &userChanMaps{},
		pipeContext: &pipeContext{currentTask: task},
	}

	resCh := make(chan robot.RetVal, 1)
	go func() {
		_, ret := r.promptInternal("YesNo", "alice", "", "", "Proceed?")
		resCh <- ret
	}()

	sshMatcher := replyMatcher{protocol: "ssh", user: "alice", channel: "", thread: ""}
	waitFor(t, "ssh prompt waiter registered", func() bool {
		replies.Lock()
		_, ok := replies.m[sshMatcher]
		replies.Unlock()
		return ok
	})

	replies.Lock()
	_, slackFound := replies.m[replyMatcher{protocol: "slack", user: "alice", channel: "", thread: ""}]
	replies.Unlock()
	if slackFound {
		t.Fatal("reply waiter unexpectedly matched slack protocol key")
	}

	triggerPromptShutdownSignal()
	select {
	case ret := <-resCh:
		if ret != robot.Interrupted {
			t.Fatalf("promptInternal() ret = %s, want %s", ret, robot.Interrupted)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for promptInternal to return after shutdown signal")
	}
}

func TestPromptInternalUsesProtocolScopedUserResolution(t *testing.T) {
	fc := &fakeRuntimeConnector{}
	prevConnector := interfaces.Connector
	interfaces.Connector = fc
	t.Cleanup(func() {
		interfaces.Connector = prevConnector
	})

	replies.Lock()
	replies.m = make(map[replyMatcher][]replyWaiter)
	replies.Unlock()
	t.Cleanup(func() {
		replies.Lock()
		replies.m = make(map[replyMatcher][]replyWaiter)
		replies.Unlock()
	})

	resetPromptShutdownSignal()
	t.Cleanup(resetPromptShutdownSignal)

	task := &Task{name: "test-protocol-user-resolution", taskType: taskGo}
	r := Robot{
		Message: &robot.Message{
			User:     "alice",
			Channel:  "",
			Protocol: robot.Slack,
			Incoming: &robot.ConnectorMessage{Protocol: "slack"},
		},
		maps: &userChanMaps{
			user: map[string]*DirectoryUser{
				"alice": {UserName: "alice"},
			},
			userProto: map[string]map[string]*UserInfo{
				"slack": {
					"alice": {UserName: "alice", UserID: "U123SLACK"},
				},
			},
		},
		pipeContext: &pipeContext{currentTask: task},
	}

	resCh := make(chan robot.RetVal, 1)
	go func() {
		_, ret := r.promptInternal("YesNo", "alice", "", "", "Proceed?")
		resCh <- ret
	}()

	waitFor(t, "prompt sent with protocol-specific user id", func() bool {
		fc.mu.Lock()
		defer fc.mu.Unlock()
		return fc.userCalls > 0 && fc.lastUser == "<U123SLACK>"
	})

	triggerPromptShutdownSignal()
	select {
	case ret := <-resCh:
		if ret != robot.Interrupted {
			t.Fatalf("promptInternal() ret = %s, want %s", ret, robot.Interrupted)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for promptInternal to return after shutdown signal")
	}
}

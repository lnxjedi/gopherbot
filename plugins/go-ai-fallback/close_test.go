package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type closeTestRobot struct {
	robot.Robot
	channel      string
	thread       string
	datums       map[string]json.RawMessage
	debug        string
	unsubscribed bool
	reply        string
}

func (r *closeTestRobot) GetMessage() *robot.Message {
	return &robot.Message{Channel: r.channel}
}

func (r *closeTestRobot) GetParameter(name string) string {
	return map[string]string{
		"GOPHER_PROTOCOL":  "test",
		"GOPHER_CHANNEL":   r.channel,
		"GOPHER_THREAD_ID": r.thread,
		"GOPHER_USER":      "alice",
	}[name]
}

func (r *closeTestRobot) CheckoutDatum(key string, datum interface{}, rw bool) (string, bool, robot.RetVal) {
	data, exists := r.datums[key]
	if exists {
		if err := json.Unmarshal(data, datum); err != nil {
			return "", true, robot.DataFormatError
		}
	}
	return "lock", exists, robot.Ok
}

func (r *closeTestRobot) UpdateDatum(key, token string, datum interface{}) robot.RetVal {
	data, err := json.Marshal(datum)
	if err != nil {
		return robot.DataFormatError
	}
	r.datums[key] = data
	return robot.Ok
}

func (r *closeTestRobot) DeleteDatum(key string) robot.RetVal {
	delete(r.datums, key)
	return robot.Ok
}

func (r *closeTestRobot) Remember(key, value string, shared bool) { r.debug = value }

func (r *closeTestRobot) Unsubscribe() bool {
	r.unsubscribed = true
	return true
}

func (r *closeTestRobot) Say(message string, args ...interface{}) robot.RetVal {
	r.reply = fmt.Sprintf(message, args...)
	return robot.Ok
}

func TestCloseConversationCleanup(t *testing.T) {
	for _, tc := range []struct {
		name    string
		channel string
		thread  string
		state   *conversationState
		cleaned bool
		reply   string
	}{
		{"thread without history", "general", "thread-1", nil, true, "Ok, I'll forget this conversation and unsubscribe this thread."},
		{"thread after failed request", "general", "thread-1", &conversationState{Processed: []string{"test:message-1"}}, true, "Ok, I'll forget this conversation and unsubscribe this thread."},
		{"thread with exchanges", "general", "thread-1", &conversationState{Exchanges: []conversationExchange{{Human: "hello", AI: "hi"}}}, true, "Ok, I'll forget this conversation and unsubscribe this thread."},
		{"thread with summary", "general", "thread-1", &conversationState{Summary: "earlier conversation"}, true, "Ok, I'll forget this conversation and unsubscribe this thread."},
		{"DM with history", "", "", &conversationState{Summary: "earlier conversation"}, true, "Ok, I'll forget this conversation."},
		{"DM without history", "", "", nil, false, "I have no memory of a conversation in progress."},
		{"channel outside thread", "general", "", nil, false, "That command doesn't apply in this context."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &closeTestRobot{channel: tc.channel, thread: tc.thread, datums: make(map[string]json.RawMessage), debug: "true"}
			ctx := makeConversationContext(r)
			if tc.state != nil {
				r.UpdateDatum(ctx.ConversationKey, "", tc.state)
			}
			// Keep an unrelated entry to ensure cleanup is scoped to this conversation.
			idx := conversationIndex{}
			upsertConversationIndexEntry(&idx, ctx.ConversationID, ctx.ConversationKey, nowString())
			upsertConversationIndexEntry(&idx, "unrelated", "other-key", nowString())
			r.UpdateDatum(conversationIndexDatumKey, "", idx)

			if got := PluginHandler(r, "close"); got != robot.Normal {
				t.Fatalf("close returned %v, want Normal", got)
			}
			if want := tc.channel != "" && tc.thread != ""; r.unsubscribed != want {
				t.Errorf("unsubscribed = %v, want %v", r.unsubscribed, want)
			}
			if r.reply != tc.reply {
				t.Errorf("reply = %q, want %q", r.reply, tc.reply)
			}
			if _, exists := r.datums[ctx.ConversationKey]; tc.cleaned && exists {
				t.Error("conversation state was not deleted")
			}
			if tc.cleaned && r.debug != "" {
				t.Error("debug memory was not cleared")
			}
			idx = conversationIndex{}
			if err := json.Unmarshal(r.datums[conversationIndexDatumKey], &idx); err != nil {
				t.Fatal(err)
			}
			if _, exists := idx.Conversations[ctx.ConversationID]; exists == tc.cleaned {
				t.Errorf("conversation index entry exists = %v, cleaned = %v", exists, tc.cleaned)
			}
			if _, exists := idx.Conversations["unrelated"]; !exists {
				t.Error("unrelated conversation index entry was removed")
			}
		})
	}
}

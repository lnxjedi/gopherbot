package googlechat

import (
	"io"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type logOnlyHandler struct {
	logs []string
}

func (h *logOnlyHandler) IncomingMessage(*robot.ConnectorMessage)  {}
func (h *logOnlyHandler) GetProtocolConfig(interface{}) error      { return nil }
func (h *logOnlyHandler) GetBrainConfig(interface{}) error         { return nil }
func (h *logOnlyHandler) GetEventStrings() *[]string               { return nil }
func (h *logOnlyHandler) GetHistoryConfig(interface{}) error       { return nil }
func (h *logOnlyHandler) GetBotInfo() robot.BotInfo                { return robot.BotInfo{} }
func (h *logOnlyHandler) SetBotID(string)                          {}
func (h *logOnlyHandler) SetTerminalWriter(io.Writer)              {}
func (h *logOnlyHandler) SetBotMention(string)                     {}
func (h *logOnlyHandler) GetLogLevel() robot.LogLevel              { return robot.Debug }
func (h *logOnlyHandler) GetInstallPath() string                   { return "" }
func (h *logOnlyHandler) GetConfigPath() string                    { return "" }
func (h *logOnlyHandler) ReadEncryptedFile(string) ([]byte, error) { return nil, nil }
func (h *logOnlyHandler) Log(_ robot.LogLevel, m string, v ...interface{}) {
	h.logs = append(h.logs, m)
}
func (h *logOnlyHandler) GetDirectory(string) error { return nil }
func (h *logOnlyHandler) RaisePriv(string)          {}

func TestNormalizeConfiguredUserMap(t *testing.T) {
	h := &logOnlyHandler{}
	got := normalizeConfiguredUserMap(map[string]string{
		"alice": "users/123",
		"bob":   "456",
		"":      "users/789",
	}, h)
	if got["alice"] != "users/123" {
		t.Fatalf("alice = %q", got["alice"])
	}
	if got["bob"] != "users/456" {
		t.Fatalf("bob = %q", got["bob"])
	}
	if _, ok := got[""]; ok {
		t.Fatal("unexpected empty username entry")
	}
}

func TestNormalizeSubscriptionID(t *testing.T) {
	if got := normalizeSubscriptionID("projects/p/subscriptions/gopherbot-chat-sub"); got != "gopherbot-chat-sub" {
		t.Fatalf("normalizeSubscriptionID() = %q", got)
	}
}

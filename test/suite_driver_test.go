//go:build integration

package tbot_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
	"github.com/lnxjedi/gopherbot/v2/integration/suites"
)

type inProcessSuiteDriver struct {
	t    *testing.T
	conn *testc.TestConnector
}

func newInProcessSuiteDriver(t *testing.T, conn *testc.TestConnector) *inProcessSuiteDriver {
	return &inProcessSuiteDriver{
		t:    t,
		conn: conn,
	}
}

func (d *inProcessSuiteDriver) WaitForIdle(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		WaitForBackgroundInitsForTesting()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (d *inProcessSuiteDriver) DrainEvents(ctx context.Context) ([]Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	ev := GetEvents()
	return append([]Event(nil), (*ev)...), nil
}

func (d *inProcessSuiteDriver) Send(ctx context.Context, msg suites.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	d.conn.SendBotMessage(&testc.TestMessage{
		User:     msg.User,
		Channel:  msg.Channel,
		Message:  msg.Text,
		Threaded: msg.Threaded,
		Hidden:   msg.Hidden,
	})
	return nil
}

func (d *inProcessSuiteDriver) Receive(ctx context.Context, want suites.ExpectedMessage) (suites.Message, error) {
	type receiveResult struct {
		msg *testc.TestMessage
		err error
	}
	rc := make(chan receiveResult, 1)
	go func() {
		msg, err := d.conn.GetBotMessage()
		rc <- receiveResult{msg: msg, err: err}
	}()
	select {
	case <-ctx.Done():
		return suites.Message{}, ctx.Err()
	case res := <-rc:
		if res.err != nil {
			return suites.Message{}, res.err
		}
		if res.msg == nil {
			return suites.Message{}, fmt.Errorf("nil bot message")
		}
		return suites.Message{
			User:     res.msg.User,
			Channel:  res.msg.Channel,
			Text:     res.msg.Message,
			Threaded: res.msg.Threaded,
			Hidden:   res.msg.Hidden,
		}, nil
	}
}

func runRegisteredSuite(t *testing.T, conn *testc.TestConnector, suite suites.Suite) {
	t.Helper()
	failures := suites.RunSuite(context.Background(), newInProcessSuiteDriver(t, conn), suite)
	for _, failure := range failures {
		t.Errorf("%s/%s/%s: %s", failure.Suite, failure.Case, failure.Step, failure.Error)
	}
}

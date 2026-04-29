package suites

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/v2/bot"
)

type TestMessage struct {
	User, Channel, Message string
	Threaded               bool
}

type testItem struct {
	user, channel, message string
	threaded               bool
	replies                []TestMessage
	events                 []bot.Event
	pause                  int
}

const (
	alice = Alice
	bob   = Bob
	carol = Carol
	david = David
	erin  = Erin

	aliceID = AliceID
	bobID   = BobID
	carolID = CarolID
	davidID = DavidID
	erinID  = ErinID

	null     = ""
	general  = General
	random   = Random
	bottest  = BotTest
	deadzone = Deadzone
)

func legacyCases(items []testItem) []Case {
	cases := make([]Case, 0, len(items))
	for _, item := range items {
		cases = append(cases, legacyCase(item, false))
	}
	return cases
}

func legacyRepliesOnlyCases(items []testItem) []Case {
	cases := make([]Case, 0, len(items))
	for _, item := range items {
		cases = append(cases, legacyCase(item, true))
	}
	return cases
}

func legacyCase(item testItem, repliesOnly bool) Case {
	replies := make([]ExpectedMessage, 0, len(item.replies))
	for _, reply := range item.replies {
		replies = append(replies, ExpectedMessage{
			User:        reply.User,
			Channel:     reply.Channel,
			TextPattern: reply.Message,
			Threaded:    reply.Threaded,
		})
	}
	return Case{
		Input: Message{
			User:     item.user,
			Channel:  item.channel,
			Text:     item.message,
			Threaded: item.threaded,
		},
		Replies:     replies,
		Events:      item.events,
		RepliesOnly: repliesOnly,
		Pause:       time.Duration(item.pause) * time.Millisecond,
	}
}

func sendAndReceive(ctx context.Context, d Driver, input Message, want ExpectedMessage) (Message, error) {
	if err := d.WaitForIdle(ctx); err != nil {
		return Message{}, err
	}
	if _, err := d.DrainEvents(ctx); err != nil {
		return Message{}, err
	}
	if err := d.Send(ctx, input); err != nil {
		return Message{}, err
	}
	got, err := d.Receive(ctx, want)
	if err != nil {
		return Message{}, err
	}
	if err := matchMessage(want, got); err != nil {
		return got, err
	}
	return got, nil
}

func receiveAndMatch(ctx context.Context, d Driver, want ExpectedMessage) (Message, error) {
	got, err := d.Receive(ctx, want)
	if err != nil {
		return Message{}, err
	}
	if err := matchMessage(want, got); err != nil {
		return got, err
	}
	return got, nil
}

func checkEvents(ctx context.Context, d Driver, want []bot.Event) error {
	got, err := d.DrainEvents(ctx)
	if err != nil {
		return err
	}
	return matchEvents(want, got)
}

func addFlowFailure(failures *[]Failure, suiteName, caseName, step string, format string, args ...interface{}) {
	*failures = append(*failures, Failure{
		Suite: suiteName,
		Case:  caseName,
		Step:  step,
		Error: fmt.Sprintf(format, args...),
	})
}

func containsAll(text string, needles ...string) error {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return fmt.Errorf("missing %q in %q", needle, text)
		}
	}
	return nil
}

func containsNone(text string, needles ...string) error {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return fmt.Errorf("unexpected %q in %q", needle, text)
		}
	}
	return nil
}

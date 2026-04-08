//go:build integration
// +build integration

package tbot_test

import "testing"

func TestExternalEncryptSecret(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	flow := []testItem{
		{aliceID, general, ";python-encrypt-secret", false, []TestMessage{
			{null, general, "ENCRYPT SECRET: ok", false}}, nil, 0},
		{aliceID, general, ";python-encrypt-secret-unpriv", false, []TestMessage{
			{null, general, "ENCRYPT SECRET: failed", false}}, nil, 0},
		{aliceID, general, ";ruby-encrypt-secret", false, []TestMessage{
			{null, general, "ENCRYPT SECRET: ok", false}}, nil, 0},
		{aliceID, general, ";ruby-encrypt-secret-unpriv", false, []TestMessage{
			{null, general, "ENCRYPT SECRET: failed", false}}, nil, 0},
	}

	for _, step := range flow {
		testcaseRepliesOnly(t, conn, step)
	}

	teardown(t, done, conn)
}

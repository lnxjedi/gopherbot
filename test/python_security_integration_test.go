//go:build integration
// +build integration

package tbot_test

import (
	"testing"
)

func TestPythonSecurity(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest.log", t)

	flow := []testItem{
		{aliceID, general, ";python-sec-open", false, []TestMessage{
			{null, general, "SECURITY CHECK: secopen", false}}, nil, 0},
		{bobID, general, ";python-sec-open", false, []TestMessage{
			{null, general, "SECURITY CHECK: secopen", false}}, nil, 0},
		{aliceID, general, ";python-sec-admincmd", false, []TestMessage{
			{null, general, "SECURITY CHECK: secadmincmd", false}}, nil, 0},
		{bobID, general, ";python-sec-admincmd", false, []TestMessage{
			{null, general, "Sorry, 'pysec/secadmincmd' is only available to bot administrators", false}}, nil, 0},
		{bobID, general, ";python-sec-authz", false, []TestMessage{
			{null, general, "SECURITY CHECK: secauthz", false}}, nil, 0},
		{davidID, general, ";python-sec-authz", false, []TestMessage{
			{null, general, "Sorry, you're not authorized for that command", false}}, nil, 0},
		{bobID, general, ";python-sec-authall", false, []TestMessage{
			{null, general, "SECURITY CHECK: secauthall", false}}, nil, 0},
		{davidID, general, ";python-sec-authall", false, []TestMessage{
			{null, general, "Sorry, you're not authorized for that command", false}}, nil, 0},
		{aliceID, general, ";python-sec-elevated", false, []TestMessage{
			{alice, general, "This command requires.*elevation.*TOTP code.*", false}}, nil, 150},
		{aliceID, general, "123456", false, []TestMessage{
			{null, general, "There were technical issues validating your code.*", false},
			{null, general, "Sorry, elevation failed due to a problem with the elevation service", false}}, nil, 0},
		{aliceID, general, ";python-sec-immediate", false, []TestMessage{
			{alice, general, "This command requires immediate elevation.*TOTP code.*", false}}, nil, 150},
		{aliceID, general, "123456", false, []TestMessage{
			{null, general, "There were technical issues validating your code.*", false},
			{null, general, "Sorry, elevation failed due to a problem with the elevation service", false}}, nil, 0},
		{aliceID, general, "/;python-sec-hidden-ok", false, []TestMessage{
			{null, general, "\\(SECURITY CHECK: sechiddenok\\)", false}}, nil, 0},
		{aliceID, general, "/;python-sec-hidden-denied", false, []TestMessage{
			{alice, general, "\\(?Sorry, 'pysec/sechiddendenied' cannot be run as a hidden command - use the robot's name or alias\\)?", false}}, nil, 0},
		{aliceID, general, ";python-sec-adminonly", false, []TestMessage{
			{null, general, "SECURITY CHECK: secadminonly", false}}, nil, 0},
		{bobID, general, ";python-sec-adminonly", false, []TestMessage{
			{null, general, "No command matched in channel.*", true}}, nil, 0},
		{aliceID, general, ";python-sec-usersonly", false, []TestMessage{
			{null, general, "SECURITY CHECK: secusersonly", false}}, nil, 0},
		{bobID, general, ";python-sec-usersonly", false, []TestMessage{
			{null, general, "No command matched in channel.*", true}}, nil, 0},
		{aliceID, general, ";python-sec-auth-misconfig", false, []TestMessage{
			{null, general, "No command matched in channel.*", true}}, nil, 0},
	}

	for _, step := range flow {
		testcaseRepliesOnly(t, conn, step)
	}

	teardown(t, done, conn)
}

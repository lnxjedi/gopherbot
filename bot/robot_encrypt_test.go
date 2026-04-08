package bot

import (
	"encoding/base64"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func makeEncryptRobot(t *testing.T, privileged bool) (Robot, int) {
	t.Helper()
	tid := getTaskID()
	task := &Task{name: "testplugin", taskType: taskExternal}
	plugin := &Plugin{Task: task}
	w := &worker{pipeContext: &pipeContext{}}
	taskLookup.Lock()
	taskLookup.i[tid] = w
	taskLookup.Unlock()
	r := Robot{
		Message: &robot.Message{
			Incoming: &robot.ConnectorMessage{},
		},
		tid: tid,
		pipeContext: &pipeContext{
			currentTask: plugin,
			privileged:  privileged,
		},
	}
	return r, tid
}

func TestEncryptSecretPrivileged(t *testing.T) {
	cryptKey.Lock()
	oldKey := append([]byte(nil), cryptKey.key...)
	oldInitialized := cryptKey.initialized
	oldInitializing := cryptKey.initializing
	cryptKey.key = []byte("0123456789abcdef0123456789abcdef")
	cryptKey.initialized = true
	cryptKey.initializing = false
	cryptKey.Unlock()
	defer func() {
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.initializing = oldInitializing
		cryptKey.Unlock()
	}()

	r, tid := makeEncryptRobot(t, true)
	defer deregisterWorker(tid)

	const plaintext = "test-secret"
	ciphertext, ret := r.EncryptSecret(plaintext)

	if ret != robot.Ok {
		t.Fatalf("EncryptSecret ret = %v, want Ok", ret)
	}
	if ciphertext == "" {
		t.Fatal("EncryptSecret returned empty ciphertext")
	}
	ct, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("EncryptSecret ciphertext is not valid base64: %v", err)
	}
	cryptKey.RLock()
	key := cryptKey.key
	cryptKey.RUnlock()
	decrypted, err := decrypt(ct, key)
	if err != nil {
		t.Fatalf("decrypt ciphertext failed: %v", err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("round-trip: got %q, want %q", string(decrypted), plaintext)
	}
}

func TestEncryptSecretUnprivileged(t *testing.T) {
	cryptKey.Lock()
	oldKey := append([]byte(nil), cryptKey.key...)
	oldInitialized := cryptKey.initialized
	oldInitializing := cryptKey.initializing
	cryptKey.key = []byte("0123456789abcdef0123456789abcdef")
	cryptKey.initialized = true
	cryptKey.initializing = false
	cryptKey.Unlock()
	defer func() {
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.initializing = oldInitializing
		cryptKey.Unlock()
	}()

	r, tid := makeEncryptRobot(t, false)
	defer deregisterWorker(tid)

	ciphertext, ret := r.EncryptSecret("test-secret")

	if ret != robot.PrivilegeViolation {
		t.Errorf("EncryptSecret ret = %v, want PrivilegeViolation", ret)
	}
	if ciphertext != "" {
		t.Errorf("EncryptSecret ciphertext = %q, want empty on privilege violation", ciphertext)
	}
}

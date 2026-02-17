package bot

import (
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestDatumMethodsAllowColonSubkeysWithNamespaceIsolation(t *testing.T) {
	oldBrain := interfaces.brain
	testBrain := &memBrain{memories: make(map[string]*[]byte)}
	interfaces.brain = testBrain
	defer func() {
		interfaces.brain = oldBrain
	}()

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

	done := make(chan struct{})
	go func() {
		runBrain()
		close(done)
	}()
	defer func() {
		brainQuit()
		<-done
	}()

	makePluginRobot := func(pluginName string) (Robot, int) {
		t.Helper()
		tid := getTaskID()
		task := &Task{name: pluginName, taskType: taskExternal}
		plugin := &Plugin{Task: task}
		w := &worker{}
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
			},
		}
		return r, tid
	}

	listsRobot, listsTID := makePluginRobot("lists")
	defer deregisterWorker(listsTID)
	linksRobot, linksTID := makePluginRobot("links")
	defer deregisterWorker(linksTID)

	datumKey := "list:grocery"

	var listsDatum map[string]string
	lockToken, exists, ret := listsRobot.CheckoutDatum(datumKey, &listsDatum, true)
	if ret != robot.Ok {
		t.Fatalf("lists CheckoutDatum ret = %v, want Ok", ret)
	}
	if exists {
		t.Fatalf("lists CheckoutDatum exists = %t, want false for new key", exists)
	}
	listsDatum = map[string]string{"owner": "lists"}
	if ret := listsRobot.UpdateDatum(datumKey, lockToken, listsDatum); ret != robot.Ok {
		t.Fatalf("lists UpdateDatum ret = %v, want Ok", ret)
	}

	if _, ok := testBrain.memories["lists:"+datumKey]; !ok {
		t.Fatalf("expected stored key %q not found in brain", "lists:"+datumKey)
	}

	var linksDatum map[string]string
	_, exists, ret = linksRobot.CheckoutDatum(datumKey, &linksDatum, false)
	if ret != robot.Ok {
		t.Fatalf("links CheckoutDatum ret = %v, want Ok", ret)
	}
	if exists {
		t.Fatalf("links CheckoutDatum exists = %t, want false before links writes datum", exists)
	}

	lockToken, _, ret = linksRobot.CheckoutDatum(datumKey, &linksDatum, true)
	if ret != robot.Ok {
		t.Fatalf("links CheckoutDatum(rw) ret = %v, want Ok", ret)
	}
	linksDatum = map[string]string{"owner": "links"}
	if ret := linksRobot.UpdateDatum(datumKey, lockToken, linksDatum); ret != robot.Ok {
		t.Fatalf("links UpdateDatum ret = %v, want Ok", ret)
	}

	var listsReadback map[string]string
	_, exists, ret = listsRobot.CheckoutDatum(datumKey, &listsReadback, false)
	if ret != robot.Ok || !exists {
		t.Fatalf("lists CheckoutDatum(readback) ret=%v exists=%t, want Ok/true", ret, exists)
	}
	if got := listsReadback["owner"]; got != "lists" {
		t.Fatalf("lists readback owner = %q, want %q", got, "lists")
	}

	if ret := linksRobot.DeleteDatum(datumKey); ret != robot.Ok {
		t.Fatalf("links DeleteDatum ret = %v, want Ok", ret)
	}
	if ret := linksRobot.DeleteDatum(datumKey); ret != robot.Ok {
		t.Fatalf("links DeleteDatum(second call) ret = %v, want Ok", ret)
	}

	_, exists, ret = listsRobot.CheckoutDatum(datumKey, &listsReadback, false)
	if ret != robot.Ok || !exists {
		t.Fatalf("lists CheckoutDatum(after links delete) ret=%v exists=%t, want Ok/true", ret, exists)
	}
}

func TestCheckinDatumAllowsColonSubkeys(t *testing.T) {
	oldBrain := interfaces.brain
	interfaces.brain = &memBrain{memories: make(map[string]*[]byte)}
	defer func() {
		interfaces.brain = oldBrain
	}()

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

	done := make(chan struct{})
	go func() {
		runBrain()
		close(done)
	}()
	defer func() {
		brainQuit()
		<-done
	}()

	tid := getTaskID()
	task := &Task{name: "lists", taskType: taskExternal}
	plugin := &Plugin{Task: task}
	w := &worker{}
	taskLookup.Lock()
	taskLookup.i[tid] = w
	taskLookup.Unlock()
	defer deregisterWorker(tid)

	r := Robot{
		Message: &robot.Message{
			Incoming: &robot.ConnectorMessage{},
		},
		tid: tid,
		pipeContext: &pipeContext{
			currentTask: plugin,
		},
	}

	var datum map[string]string
	lockToken, _, ret := r.CheckoutDatum("list:grocery", &datum, true)
	if ret != robot.Ok {
		t.Fatalf("CheckoutDatum ret = %v, want Ok", ret)
	}
	r.CheckinDatum("list:grocery", lockToken)

	doneCheckout := make(chan robot.RetVal, 1)
	go func() {
		var d map[string]string
		_, _, rv := r.CheckoutDatum("list:grocery", &d, true)
		doneCheckout <- rv
	}()

	select {
	case rv := <-doneCheckout:
		if rv != robot.Ok {
			t.Fatalf("CheckoutDatum after CheckinDatum ret = %v, want Ok", rv)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("CheckoutDatum blocked after CheckinDatum for colon key")
	}
}

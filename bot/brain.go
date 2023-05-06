package bot

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// Map of registered brains
var brains = make(map[string]func(robot.Handler) robot.SimpleBrain)

// Set on start-up
var encryptBrain bool

// For aes brain encryption
var cryptKey = struct {
	key                       []byte
	initializing, initialized bool
	sync.RWMutex
}{}

// Definitions of bot keys and prefixes

// The "real" key to en-/de-crypt memories;
// the user-supplied key unlocks this, allowing
// the user to re-key if they change how the key
// is supplied.
const botEncryptionKey = "bot:encryptionKey"
const encryptedKeyFile = "binary-encrypted-key"

// People generally expect the robot to remember things longer.
const channelMemoryDuration = 7 * time.Minute   // In the main channel, conversation context/topics tend to change often
const threadMemoryDuration = 7 * time.Hour * 24 // In a thread, the conversation context/topic tends to last days

type memState int

const (
	newMemory memState = iota
	seen
	available
)

type memstatus struct {
	state   memState
	token   string // whoever has this token owns the lock for this memory
	waiters []checkOutRequest
}

var brainChanEvents = make(chan interface{})

type checkOutRequest struct {
	key   string
	rw    bool
	reply chan checkOutReply
}

type checkOutReply struct {
	token  string
	bytes  *[]byte
	exists bool
	RetVal robot.RetVal
}

type checkInRequest struct {
	key   string
	token string
}

type updateRequest struct {
	key   string
	token string
	datum *[]byte
	reply chan robot.RetVal
}

type pauseRequest struct {
	resume chan struct{}
	wid    int
}

type quitRequest struct {
	reply chan struct{}
}

// how often does the robot cycle through memories and update state?
// a value of time.Second means a lock will last between 1 and 2 seconds
const memCycle = time.Second

func replyToWaiter(m *memstatus) {
	creq := m.waiters[0]
	m.waiters = m.waiters[1:]
	lt, d, e, r := getDatum(creq.key, true)
	m.state = newMemory
	m.token = lt
	creq.reply <- checkOutReply{lt, d, e, r}
}

// brain locking for backups
// the brain shouldn't be big, and this pauses all activity for
// a maximum of lockMax seconds
const lockMax = 28

var brainLocks = struct {
	locks map[int]chan struct{}
	sync.Mutex
}{
	make(map[int]chan struct{}),
	sync.Mutex{},
}

// runBrain is the select loop that serializes access to brain
// functions and insures consistency.
func runBrain() {
	raiseThreadPriv("runBrain loop")
	// map key to status
	memories := make(map[string]*memstatus)
	brainTicker := time.NewTicker(memCycle)
loop:
	for {
		select {
		case evt := <-brainChanEvents:
			switch evt.(type) {
			case pauseRequest:
				pb := evt.(pauseRequest)
				Log(robot.Debug, "Brain pause requested by worker %d", pb.wid)
				select {
				case <-pb.resume:
					continue
				case <-time.After(lockMax * time.Second):
					Log(robot.Warn, "Brain pause timed out after %d seconds for worker %d", lockMax, pb.wid)
					brainLocks.Lock()
					delete(brainLocks.locks, pb.wid)
					brainLocks.Unlock()
					continue
				}
			case checkOutRequest:
				creq := evt.(checkOutRequest)
				memStat, exists := memories[creq.key]
				if !exists {
					lt, d, e, r := getDatum(creq.key, creq.rw)
					if r != robot.Ok {
						creq.reply <- checkOutReply{lt, d, e, r}
						continue
					}
					if creq.rw {
						m := &memstatus{
							newMemory,
							lt,
							make([]checkOutRequest, 0, 2),
						}
						memories[creq.key] = m
					}
					creq.reply <- checkOutReply{lt, d, e, r}
					continue
				}
				if !creq.rw {
					lt, d, e, r := getDatum(creq.key, creq.rw)
					creq.reply <- checkOutReply{lt, d, e, r}
					continue
				} // read-write request below
				// if state is available, there are no waiters
				if memStat.state == available {
					lt, d, e, r := getDatum(creq.key, creq.rw)
					memStat.state = newMemory
					memStat.token = lt // this memory has a new owner now
					memories[creq.key] = memStat
					creq.reply <- checkOutReply{lt, d, e, r}
				} else {
					memStat.waiters = append(memStat.waiters, creq)
					memories[creq.key] = memStat
				}
			case checkInRequest:
				ci := evt.(checkInRequest)
				m, ok := memories[ci.key]
				if !ok {
					continue
				}
				// memory expired and somebody else owns it
				if ci.token != m.token {
					continue
				}
				if len(m.waiters) > 0 {
					replyToWaiter(m)
					continue
				}
				delete(memories, ci.key)
			case updateRequest:
				ur := evt.(updateRequest)
				m, ok := memories[ur.key]
				if !ok {
					ur.reply <- robot.DatumNotFound
					continue
				}
				if ur.token != m.token {
					ur.reply <- robot.DatumLockExpired
					continue
				}
				ur.reply <- storeDatum(ur.key, ur.datum)
				if len(m.waiters) > 0 {
					replyToWaiter(m)
					continue
				}
				delete(memories, ur.key)
			case quitRequest:
				qr := evt.(quitRequest)
				qr.reply <- struct{}{}
				break loop
			}
		case <-brainTicker.C:
			now := time.Now()
			// Expire thread subscriptions - see thread_subscriptions.go
			isDirty := expireSubscriptions(now)
			if isDirty {
				go saveSubscriptions()
			}
			ephemeralMemories.Lock()
			for context, memory := range ephemeralMemories.m {
				if len(context.thread) > 0 {
					if now.Sub(memory.Timestamp) > threadMemoryDuration {
						delete(ephemeralMemories.m, context)
						ephemeralMemories.dirty = true
					}
				} else {
					if now.Sub(memory.Timestamp) > channelMemoryDuration {
						delete(ephemeralMemories.m, context)
						ephemeralMemories.dirty = true
					}
				}
			}
			isDirty = ephemeralMemories.dirty
			ephemeralMemories.Unlock()
			if isDirty {
				go saveEphemeralMemories()
			}
			for _, m := range memories {
				switch m.state {
				case newMemory:
					m.state = seen
				case seen:
					if len(m.waiters) > 0 {
						replyToWaiter(m)
						continue
					}
					m.state = available
				}
			}
		}
	}
}

func brainQuit() {
	reply := make(chan struct{})
	brainChanEvents <- quitRequest{reply}
	Log(robot.Debug, "Brain exiting on quit")
	<-reply
}

const keyRegex = `[\w:]+` // keys can ony be word chars + separator (:)
var keyRe = regexp.MustCompile(keyRegex)

// checkout returns the []byte from the brain, with a lock token granting
// ownership for a limited time
func checkout(d string, rw bool) (string, *[]byte, bool, robot.RetVal) {
	if !keyRe.MatchString(d) {
		Log(robot.Error, "Invalid memory key, ':' disallowed: %s", d)
		return "", nil, false, robot.InvalidDatumKey
	}
	reply := make(chan checkOutReply)
	brainChanEvents <- checkOutRequest{d, rw, reply}
	rep := <-reply
	Log(robot.Trace, "Brain datum checkout for %s, rw: %t - token: %s, exists: %t, ret: %d",
		d, rw, rep.token, rep.exists, rep.RetVal)
	return rep.token, rep.bytes, rep.exists, rep.RetVal
}

// update sends updated []byte to the brain while holding the lock, or discards
// the data and returns an error.
func update(d, lt string, datum *[]byte) (ret robot.RetVal) {
	if lt == "" {
		return robot.Ok
	}
	reply := make(chan robot.RetVal)
	Log(robot.Trace, "Updating datum %s, token: %s", d, lt)
	brainChanEvents <- updateRequest{d, lt, datum, reply}
	return <-reply
}

// checkinDatum is the internal version of CheckinDatum that uses the key as-is
func checkinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	Log(robot.Trace, "Checking in datum %s, token: %s", key, locktoken)
	brainChanEvents <- checkInRequest{key, locktoken}
}

// pauseBrain pauses the brain for backups, passing a resume channel
func pauseBrain(wid int, resume chan struct{}) {
	brainChanEvents <- pauseRequest{resume, wid}
}

// checkoutDatum is the robot internal version of CheckoutDatum that uses
// the provided key as-is.
func checkoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret robot.RetVal) {
	var dbytes *[]byte
	locktoken, dbytes, exists, ret = checkout(key, rw)
	if exists { // exists = true implies no error
		err := json.Unmarshal(*dbytes, datum)
		if err != nil {
			Log(robot.Error, "Unmarshalling datum %s: %v", key, err)
			exists = false
			ret = robot.DataFormatError
		}
	}
	return
}

// updateDatum is the internal version of UpdateDatum that uses the key as-is
func updateDatum(key, locktoken string, datum interface{}) (ret robot.RetVal) {
	dbytes, err := json.Marshal(datum)
	if err != nil {
		Log(robot.Error, "Marshalling datum %s: %v", key, err)
		return robot.DataFormatError
	}
	return update(key, locktoken, &dbytes)
}

func (w *worker) getNameSpace(t interface{}) string {
	task, plugin, _ := getTask(t)
	// A configured NameSpace always takes precedence
	if len(task.NameSpace) > 0 {
		return task.NameSpace
	}
	// Plugins never inherit the pipeline namespace,
	// because they implement authorizers and elevators.
	if plugin != nil {
		return task.name
	}
	w.Lock()
	defer w.Unlock()
	// Inherit namespace from the pipeline
	if len(w.nameSpace) > 0 {
		return w.nameSpace
	}
	return task.name
}

// CheckoutDatum gets a datum from the robot's brain and unmarshals it into
// a struct. If rw is set, the datum is checked out read-write and a non-empty
// lock token is returned that expires after lockTimeout (250ms). The bool
// return indicates whether the datum exists. Datum must be a pointer to a
// var.
func (r Robot) CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret robot.RetVal) {
	if strings.ContainsRune(key, ':') {
		ret = robot.InvalidDatumKey
		Log(robot.Error, "Invalid memory key, ':' disallowed: %s", key)
		return
	}
	w := getLockedWorker(r.tid)
	w.Unlock()
	ns := w.getNameSpace(r.currentTask)
	if len(r.nsExtension) > 0 {
		key = ns + ":" + r.nsExtension + ":" + key
	} else {
		key = ns + ":" + key
	}
	return checkoutDatum(key, datum, rw)
}

// CheckinDatum unlocks a datum without updating it, it always succeeds
func (r Robot) CheckinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	if strings.ContainsRune(key, ':') {
		return
	}
	w := getLockedWorker(r.tid)
	w.Unlock()
	ns := w.getNameSpace(r.currentTask)
	if len(r.nsExtension) > 0 {
		key = ns + ":" + r.nsExtension + ":" + key
	} else {
		key = ns + ":" + key
	}
	checkinDatum(key, locktoken)
}

// UpdateDatum tries to update a piece of data in the robot's brain, providing
// a struct to marshall and a (hopefully good) lock token. If err != nil, the
// update failed.
func (r Robot) UpdateDatum(key, locktoken string, datum interface{}) (ret robot.RetVal) {
	if strings.ContainsRune(key, ':') {
		Log(robot.Error, "Invalid memory key, ':' disallowed: %s", key)
		return robot.InvalidDatumKey
	}
	w := getLockedWorker(r.tid)
	w.Unlock()
	ns := w.getNameSpace(r.currentTask)
	if len(r.nsExtension) > 0 {
		key = ns + ":" + r.nsExtension + ":" + key
	} else {
		key = ns + ":" + key
	}
	return updateDatum(key, locktoken, datum)
}

// Remember adds a ephemeral memory (with no backing store) to the robot's
// brain. This is used internally for resolving the meaning of "it", but can
// be used by plugins to remember other contextual facts. Since memories are
// indexed by user and channel, but not plugin, these facts can be referenced
// between plugins. This functionality is considered EXPERIMENTAL.
func (r Robot) Remember(key, value string, shared bool) {
	timestamp := time.Now()
	memory := ephemeralMemory{value, timestamp}
	context := r.makeMemoryContext(key, false, shared)
	Log(robot.Trace, "storing ephemeral memory \"%s\" -> \"%s\"", key, value)
	ephemeralMemories.Lock()
	ephemeralMemories.m[context] = memory
	if len(context.thread) > 0 {
		ephemeralMemories.dirty = true
	}
	ephemeralMemories.Unlock()
}

// RememberThread is identical to Remember, except that it forces the memory
// to associate with the thread.
func (r Robot) RememberThread(key, value string, shared bool) {
	timestamp := time.Now()
	memory := ephemeralMemory{value, timestamp}
	context := r.makeMemoryContext(key, true, shared)
	Log(robot.Trace, "storing ephemeral memory \"%s\" -> \"%s\"", key, value)
	ephemeralMemories.Lock()
	ephemeralMemories.m[context] = memory
	ephemeralMemories.dirty = true
	ephemeralMemories.Unlock()
}

// RememberContext is a convenience function that stores a context reference in
// short term memories. e.g. RememberContext("server", "web1.my.dom") means that
// next time the user uses "it" in the context of a "server", the robot will
// substitute "web1.my.dom".
func (r Robot) RememberContext(context, value string) {
	r.Remember("context:"+context, value, false)
}

// RememberContextThread is identical to RememberContext, except that the memory
// is forced to associate with the thread.
func (r Robot) RememberContextThread(context, value string) {
	r.RememberThread("context:"+context, value, false)
}

// Recall recalls a short term memory, or the empty string if it doesn't exist.
// Note that there are no RecallThread methods - Recall is always in the current
// context.
func (r Robot) Recall(key string, shared bool) string {
	context := r.makeMemoryContext(key, false, shared)
	ephemeralMemories.Lock()
	memory, ok := ephemeralMemories.m[context]
	ephemeralMemories.Unlock()
	Log(robot.Trace, "recalling ephemeral memory \"%s\" -> \"%s\"", key, memory.Memory)
	if !ok {
		return ""
	}
	return memory.Memory
}

// RegisterSimpleBrain allows brain implementations to register a function with a named
// brain type that returns an SimpleBrain interface.
// This can only be called from a brain provider's init() function(s). Pass in a Logger
// so the brain can log it's own error messages if needed.
func RegisterSimpleBrain(name string, provider func(robot.Handler) robot.SimpleBrain) {
	if stopRegistrations {
		return
	}
	if brains[name] != nil {
		log.Fatal("Attempted registration of duplicate brain provider name:", name)
	}
	brains[name] = provider
}

// When EncryptBrain is true, the brain needs to be initialized.
// NOTE: All locking is done with the cryptKey mutex, bypassing
// the brain loop.
func initializeEncryptionFromBrain(key string) bool {
	kbytes := []byte(key)
	if len(kbytes) < 32 {
		Log(robot.Error, "Failed to initialize brain, provided encryption key < 32 bytes")
		return false
	}
	kbytes = kbytes[0:32]
	cryptKey.Lock()
	if cryptKey.initialized || cryptKey.initializing {
		i := cryptKey.initializing
		cryptKey.Unlock()
		return i
	}
	cryptKey.key = kbytes
	cryptKey.initializing = true
	var err error
	cryptKey.Unlock()
	// retrieve the 'real' key
	_, rk, exists, ret := getDatum(botEncryptionKey, true)
	if ret != robot.Ok {
		cryptKey.Lock()
		cryptKey.initializing = false
		cryptKey.Unlock()
		Log(robot.Error, "Retrieving botEncryptionKey from brain: %s", ret)
		return false
	}
	if exists {
		cryptKey.Lock()
		cryptKey.key = *rk
		cryptKey.initialized = true
		cryptKey.initializing = false
		cryptKey.Unlock()
		return true
	}
	sb := make([]byte, 32)
	_, err = rand.Read(sb)
	if err != nil {
		Log(robot.Error, "Generating new random encryption key: %v", err)
		cryptKey.initializing = false
		return false
	}
	h := handler{}
	if err := h.GetDirectory(configPath); err == nil {
		var bek []byte
		var err error
		if bek, err = encrypt(sb, kbytes); err != nil {
			Log(robot.Fatal, "Encrypting new random key")
		}
		var bekbuff bytes.Buffer
		encoder := base64.NewEncoder(base64.StdEncoding, &bekbuff)
		encoder.Write(bek)
		bekbuff.Write([]byte("\n"))
		if err := os.WriteFile(filepath.Join(configPath, encryptedKeyFile), bekbuff.Bytes(), os.FileMode(0600)); err != nil {
			Log(robot.Fatal, "Writing new random key: %v", err)
		}
	} else {
		Log(robot.Fatal, "Getting custom directory: %v", err)
	}
	cryptKey.Lock()
	cryptKey.key = sb
	cryptKey.initialized = true
	cryptKey.initializing = false
	cryptKey.Unlock()
	return true
}

// getDatum retrieves a blob of bytes from the brain provider and optionally
// decrypts it
func getDatum(dkey string, rw bool) (token string, databytes *[]byte, exists bool, ret robot.RetVal) {
	var decrypted []byte

	if !keyRe.MatchString(dkey) {
		Log(robot.Error, "Invalid memory key, ':' disallowed: %s", dkey)
		return "", nil, false, robot.InvalidDatumKey
	}
	brain := interfaces.brain
	if brain == nil {
		Log(robot.Error, "Brain function called with no brain configured")
		return "", nil, false, robot.BrainFailed
	}
	if rw { // checked out read/write, generate a lock token
		ltb := make([]byte, 8)
		rand.Read(ltb)
		token = fmt.Sprintf("%x", ltb)
	} else {
		token = ""
	}
	var err error
	var db *[]byte
	db, exists, err = brain.Retrieve(dkey)
	if err != nil {
		return "", nil, false, robot.BrainFailed
	}
	if !exists {
		return token, nil, false, robot.Ok
	}
	if encryptBrain {
		cryptKey.RLock()
		initialized := cryptKey.initialized
		initializing := cryptKey.initializing
		key := cryptKey.key
		cryptKey.RUnlock()
		if initializing {
			if dkey != botEncryptionKey {
				Log(robot.Warn, "Retrieve called with uninitialized brain for '%s'", dkey)
				return "", nil, false, robot.BrainFailed
			}
			decrypted, err = decrypt(*db, key)
			if err != nil {
				Log(robot.Error, "Failed to decrypt the encryption key, bad key provided?: %v", err)
				return "", nil, false, robot.BrainFailed
			}
			db = &decrypted
			return token, db, true, robot.Ok
		}
		if initialized {
			decrypted, err = decrypt(*db, key)
			if err != nil {
				// This should only ever happen with the CLI, but could corrupt
				// the binary key.
				if dkey != botEncryptionKey {
					Log(robot.Warn, "Decryption failed for '%s', assuming unencrypted and converting to encrypted", dkey)
					// Calling storeDatum writes to storage without invalidating the lock token
					storeDatum(dkey, db)
				}
			} else {
				db = &decrypted
			}
			return token, db, true, robot.Ok
		}
		Log(robot.Warn, "Retrieve called on uninitialized brain for '%s'", dkey)
		return "", nil, false, robot.BrainFailed
	}
	return token, db, true, robot.Ok
}

// storeDatum takes a blob of bytes and optionally encrypts it before sending it
// to the brain provider
func storeDatum(dkey string, datum *[]byte) robot.RetVal {
	brain := interfaces.brain
	if brain == nil {
		Log(robot.Error, "Brain function called with no brain configured")
		return robot.BrainFailed
	}
	if encryptBrain {
		cryptKey.RLock()
		initialized := cryptKey.initialized
		initializing := cryptKey.initializing
		key := cryptKey.key
		cryptKey.RUnlock()
		if !initialized {
			// When re-keying, we store the 'real' key while uninitialized with a new key
			if !(initializing && dkey == botEncryptionKey) {
				Log(robot.Error, "storeDatum called for '%s' with encryptBrain true, but encryption not initialized", key)
				return robot.BrainFailed
			}
		}
		encrypted, err := encrypt(*datum, key)
		if err != nil {
			Log(robot.Error, "Failed encrypting '%s': %v", dkey, err)
			return robot.BrainFailed
		}
		datum = &encrypted
	}
	err := brain.Store(dkey, datum)
	if err != nil {
		Log(robot.Error, "Storing datum %s: %v", dkey, err)
		return robot.BrainFailed
	}
	return robot.Ok
}

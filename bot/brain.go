package bot

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// Map of registered brains
var brains = make(map[string]func(robot.Handler) robot.SimpleBrain)

// short-term memories, mostly what "it" is
type shortTermMemory struct {
	memory    string
	timestamp time.Time
}

type memoryContext struct {
	key, user, channel string
}

var shortTermMemories = struct {
	m map[memoryContext]shortTermMemory
	sync.Mutex
}{}

// Set on start-up
var encryptBrain bool

// For aes brain encryption
var cryptKey = struct {
	key                       []byte
	initializing, initialized bool
	sync.RWMutex
}{}

// For stored secrets
type brainParams struct {
	retrieved        bool
	TaskParams       map[string]map[string][]byte
	RepositoryParams map[string]map[string][]byte
}

// Definitions of bot keys and prefixes

// The "real" key to en-/de-crypt memories;
// the user-supplied key unlocks this, allowing
// the user to re-key if they change how the key
// is supplied.
const botEncryptionKey = "bot:encryptionKey"

const paramKey = "bot:parameters"
const secretKey = "bot:secrets"

const histPrefix = "bot:histories:"

const shortTermDuration = 7 * time.Minute

type brainOpType int

const (
	checkOutBytes brainOpType = iota
	checkInBytes
	updateBytes
	quit
)

type brainOp struct {
	opType brainOpType
	opData interface{}
}

type checkOutRequest struct {
	key   string
	rw    bool
	reply chan checkOutReply
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

type checkOutReply struct {
	token  string
	bytes  *[]byte
	exists bool
	RetVal robot.RetVal
}

type quitRequest struct {
	reply chan struct{}
}

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

var brainChanEvents = make(chan brainOp)

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

// When EncryptBrain is true, the brain needs to be initialized.
// NOTE: All locking is done with the cryptKey mutex, bypassing
// the brain loop.
func initializeEncryptionFromBrain(key string) bool {
	kbytes := []byte(key)
	if len(kbytes) < 32 {
		Log(robot.Error, "Failed to initialize brain, provided brain key < 32 bytes")
		return false
	}
	cryptKey.Lock()
	if cryptKey.initialized || cryptKey.initializing {
		i := cryptKey.initializing
		cryptKey.Unlock()
		return i
	}
	cryptKey.key = kbytes[0:32]
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
		Log(robot.Error, "Generating new random brain key: %v", err)
		cryptKey.initializing = false
		return false
	}
	ret = storeDatum(botEncryptionKey, &sb)
	cryptKey.Lock()
	if ret != robot.Ok {
		Log(robot.Error, "Storing brain key, failed to initialize")
		cryptKey.initializing = false
		cryptKey.Unlock()
		return false
	}
	cryptKey.key = sb
	cryptKey.initialized = true
	cryptKey.initializing = false
	cryptKey.Unlock()
	return true
}

// Most likely used when switching from configured to interactively-provided
// encryption key
func reKey(newkey string) bool {
	// NOTE: this function should temporarily set initialized = false
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
	brain := botCfg.brain
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
	db, exists, err = botCfg.brain.Retrieve(dkey)
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
				Log(robot.Error, "Failed to decrypt the brain key, bad key provided?: %v", err)
				return "", nil, false, robot.BrainFailed
			}
			db = &decrypted
			return token, db, true, robot.Ok
		}
		if initialized {
			decrypted, err = decrypt(*db, key)
			if err != nil {
				Log(robot.Warn, "Decryption failed for '%s', assuming unencrypted and converting to encrypted", dkey)
				// Calling storeDatum writes to storage without invalidating the lock token
				storeDatum(dkey, db)
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
	brain := botCfg.brain
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
				Log(robot.Error, "storeDatum called for '%s' with encryptBrain true, but brain not initialized", key)
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
	err := botCfg.brain.Store(dkey, datum)
	if err != nil {
		Log(robot.Error, "Storing datum %s: %v", dkey, err)
		return robot.BrainFailed
	}
	return robot.Ok
}

var brLock sync.RWMutex

// runBrain is the select loop that serializes access to brain
// functions and insures consistency.
func runBrain() {
	raiseThreadPriv("runBrain loop")
	shortTermMemories.Lock()
	shortTermMemories.m = make(map[memoryContext]shortTermMemory)
	shortTermMemories.Unlock()
	// map key to status
	memories := make(map[string]*memstatus)
	processMemories := time.Tick(memCycle)
loop:
	for {
		select {
		case evt := <-brainChanEvents:
			switch evt.opType {
			case checkOutBytes:
				creq := evt.opData.(checkOutRequest)
				memStat, exists := memories[creq.key]
				if !exists {
					lt, d, e, r := getDatum(creq.key, creq.rw)
					if r != robot.Ok {
						creq.reply <- checkOutReply{lt, d, e, r}
						break
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
					break
				}
				if !creq.rw {
					lt, d, e, r := getDatum(creq.key, creq.rw)
					creq.reply <- checkOutReply{lt, d, e, r}
					break
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
			case checkInBytes:
				ci := evt.opData.(checkInRequest)
				m, ok := memories[ci.key]
				if !ok {
					break
				}
				// memory expired and somebody else owns it
				if ci.token != m.token {
					break
				}
				if len(m.waiters) > 0 {
					replyToWaiter(m)
					break
				}
				delete(memories, ci.key)
			case updateBytes:
				ur := evt.opData.(updateRequest)
				m, ok := memories[ur.key]
				if !ok {
					ur.reply <- robot.DatumNotFound
					break
				}
				if ur.token != m.token {
					ur.reply <- robot.DatumLockExpired
					break
				}
				ur.reply <- storeDatum(ur.key, ur.datum)
				if len(m.waiters) > 0 {
					replyToWaiter(m)
					break
				}
				delete(memories, ur.key)
			case quit:
				qr := evt.opData.(quitRequest)
				qr.reply <- struct{}{}
				break loop
			}
		case <-processMemories:
			now := time.Now()
			shortTermMemories.Lock()
			for k, v := range shortTermMemories.m {
				if now.Sub(v.timestamp) > shortTermDuration {
					delete(shortTermMemories.m, k)
				}
			}
			shortTermMemories.Unlock()
			for _, m := range memories {
				switch m.state {
				case newMemory:
					m.state = seen
				case seen:
					if len(m.waiters) > 0 {
						replyToWaiter(m)
						break
					}
					m.state = available
				}
			}
		}
	}
}

func brainQuit() {
	reply := make(chan struct{})
	brainChanEvents <- brainOp{quit, quitRequest{reply}}
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
	creq := checkOutRequest{d, rw, reply}
	brainChanEvents <- brainOp{checkOutBytes, creq}
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
	ur := updateRequest{d, lt, datum, reply}
	Log(robot.Trace, "Updating datum %s, token: %s", d, lt)
	brainChanEvents <- brainOp{updateBytes, ur}
	return <-reply
}

// checkinDatum is the internal version of CheckinDatum that uses the key as-is
func checkinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	Log(robot.Trace, "Checking in datum %s, token: %s", key, locktoken)
	ci := checkInRequest{key, locktoken}
	brainChanEvents <- brainOp{checkInBytes, ci}
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

// CheckoutDatum gets a datum from the robot's brain and unmarshals it into
// a struct. If rw is set, the datum is checked out read-write and a non-empty
// lock token is returned that expires after lockTimeout (250ms). The bool
// return indicates whether the datum exists.
func (r Robot) CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret robot.RetVal) {
	if strings.ContainsRune(key, ':') {
		ret = robot.InvalidDatumKey
		Log(robot.Error, "Invalid memory key, ':' disallowed: %s", key)
		return
	}
	c := r.getContext()
	task, _, _ := getTask(c.currentTask)
	if len(c.nsExtension) > 0 {
		key = task.NameSpace + ":" + c.nsExtension + ":" + key
	} else {
		key = task.NameSpace + ":" + key
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
	c := r.getContext()
	task, _, _ := getTask(c.currentTask)
	if len(c.nsExtension) > 0 {
		key = task.NameSpace + ":" + c.nsExtension + ":" + key
	} else {
		key = task.NameSpace + ":" + key
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
	c := r.getContext()
	task, _, _ := getTask(c.currentTask)
	if len(c.nsExtension) > 0 {
		key = task.NameSpace + ":" + c.nsExtension + ":" + key
	} else {
		key = task.NameSpace + ":" + key
	}
	return updateDatum(key, locktoken, datum)
}

// Remember adds a short-term memory (with no backing store) to the robot's
// brain. This is used internally for resolving the meaning of "it", but can
// be used by plugins to remember other contextual facts. Since memories are
// indexed by user and channel, but not plugin, these facts can be referenced
// between plugins. This functionality is considered EXPERIMENTAL.
func (r Robot) Remember(key, value string) {
	timestamp := time.Now()
	memory := shortTermMemory{value, timestamp}
	context := memoryContext{key, r.User, r.Channel}
	Log(robot.Trace, "SHORTMEM: Storing short-term memory \"%s\" -> \"%s\"", key, value)
	shortTermMemories.Lock()
	shortTermMemories.m[context] = memory
	shortTermMemories.Unlock()
}

// RememberContext is a convenience function that stores a context reference in
// short term memories. e.g. RememberContext("server", "web1.my.dom") means that
// next time the user uses "it" in the context of a "server", the robot will
// substitute "web1.my.dom".
func (r Robot) RememberContext(context, value string) {
	r.Remember("context:"+context, value)
}

// Recall recalls a short term memory, or the empty string if it doesn't exist
func (r Robot) Recall(key string) string {
	context := memoryContext{key, r.User, r.Channel}
	shortTermMemories.Lock()
	memory, ok := shortTermMemories.m[context]
	shortTermMemories.Unlock()
	Log(robot.Trace, "SHORTMEM: Recalling short-term memory \"%s\" -> \"%s\"", key, memory.memory)
	if !ok {
		return ""
	}
	return memory.memory
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

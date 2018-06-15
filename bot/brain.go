package bot

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

// SimpleBrain is the simple interface for a configured brain, where the robot
// handles all locking issues.
type SimpleBrain interface {
	// Store stores a blob of data with a string key, returns error
	// if there's a problem storing the datum.
	Store(key string, blob *[]byte) error
	// Retrieve returns a blob of data (probably JSON) given a string key,
	// and exists=true if the data blob was found, or error if the brain
	// malfunctions.
	Retrieve(key string) (blob *[]byte, exists bool, err error)
}

// Map of registered brains
var brains = make(map[string]func(Handler, *log.Logger) SimpleBrain)

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
}{
	nil,
	sync.Mutex{},
}

const paramPrefix = "bot:parameters:"

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
	reply chan RetVal
}

type checkOutReply struct {
	token  string
	bytes  *[]byte
	exists bool
	retval RetVal
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

func getDatum(dkey string, rw bool) (token string, databytes *[]byte, exists bool, ret RetVal) {
	if !keyRe.MatchString(dkey) {
		err := fmt.Errorf("Invalid key supplied to checkout: %s", dkey)
		Log(Error, err)
		return "", nil, false, InvalidDatumKey
	}
	robot.RLock()
	brain := robot.brain
	robot.RUnlock()
	if brain == nil {
		Log(Error, "Brain function called with no brain configured")
		return "", nil, false, BrainFailed
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
	db, exists, err = robot.brain.Retrieve(dkey)
	if err != nil {
		return "", nil, false, BrainFailed
	}
	return token, db, exists, Ok
}

func storeDatum(key string, datum *[]byte) RetVal {
	robot.RLock()
	brain := robot.brain
	robot.RUnlock()
	if brain == nil {
		Log(Error, "Brain function called with no brain configured")
		return BrainFailed
	}
	err := robot.brain.Store(key, datum)
	if err != nil {
		Log(Error, fmt.Sprintf("Storing datum %s: %v", key, err))
		return BrainFailed
	}
	return Ok
}

var brLock sync.RWMutex

// runBrain is the select loop that serializes access to brain
// functions and insures consistency.
func runBrain() {
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
					if r != Ok {
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
					ur.reply <- DatumNotFound
					break
				}
				if ur.token != m.token {
					ur.reply <- DatumLockExpired
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
	Log(Debug, "Brain exiting on quit")
	<-reply
}

const keyRegex = `[\w:]+` // keys can ony be word chars + separator (:)
var keyRe = regexp.MustCompile(keyRegex)

// checkout returns the []byte from the brain, with a lock token granting
// ownership for a limited time
func checkout(d string, rw bool) (string, *[]byte, bool, RetVal) {
	if !keyRe.MatchString(d) {
		err := fmt.Errorf("Invalid key supplied to checkout: %s", d)
		Log(Error, err)
		return "", nil, false, InvalidDatumKey
	}
	reply := make(chan checkOutReply)
	creq := checkOutRequest{d, rw, reply}
	brainChanEvents <- brainOp{checkOutBytes, creq}
	rep := <-reply
	Log(Trace, fmt.Sprintf("Brain datum checkout for %s, rw: %t - token: %s, exists: %t, ret: %d",
		d, rw, rep.token, rep.exists, rep.retval))
	return rep.token, rep.bytes, rep.exists, rep.retval
}

// update sends updated []byte to the brain while holding the lock, or discards
// the data and returns an error.
func update(d, lt string, datum *[]byte) (ret RetVal) {
	if lt == "" {
		return Ok
	}
	reply := make(chan RetVal)
	ur := updateRequest{d, lt, datum, reply}
	Log(Trace, fmt.Sprintf("Updating datum %s, token: %s", d, lt))
	brainChanEvents <- brainOp{updateBytes, ur}
	return <-reply
}

// checkinDatum is the internal version of CheckinDatum that uses the key as-is
func checkinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	Log(Trace, fmt.Sprintf("Checking in datum %s, token: %s", key, locktoken))
	ci := checkInRequest{key, locktoken}
	brainChanEvents <- brainOp{checkInBytes, ci}
}

// checkoutDatum is the robot internal version of CheckoutDatum that uses
// the provided key as-is.
func checkoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret RetVal) {
	var dbytes *[]byte
	locktoken, dbytes, exists, ret = checkout(key, rw)
	if exists { // exists = true implies no error
		err := json.Unmarshal(*dbytes, datum)
		if err != nil {
			Log(Error, fmt.Errorf("Unmarshalling datum %s: %v", key, err))
			exists = false
			ret = DataFormatError
		}
	}
	return
}

// updateDatum is the internal version of UpdateDatum that uses the key as-is
func updateDatum(key, locktoken string, datum interface{}) (ret RetVal) {
	dbytes, err := json.Marshal(datum)
	if err != nil {
		Log(Error, fmt.Sprintf("Unmarshalling datum %s: %v", key, err))
		return DataFormatError
	}
	return update(key, locktoken, &dbytes)
}

// CheckoutDatum gets a datum from the robot's brain and unmarshals it into
// a struct. If rw is set, the datum is checked out read-write and a non-empty
// lock token is returned that expires after lockTimeout (250ms). The bool
// return indicates whether the datum exists.
func (r *Robot) CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret RetVal) {
	c := r.getContext()
	key = c.NameSpace + ":" + key
	return checkoutDatum(key, datum, rw)
}

// CheckinDatum unlocks a datum without updating it, it always succeeds
func (r *Robot) CheckinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	c := r.getContext()
	key = c.NameSpace + ":" + key
	checkinDatum(key, locktoken)
}

// UpdateDatum tries to update a piece of data in the robot's brain, providing
// a struct to marshall and a (hopefully good) lock token. If err != nil, the
// update failed.
func (r *Robot) UpdateDatum(key, locktoken string, datum interface{}) (ret RetVal) {
	c := r.getContext()
	key = c.NameSpace + ":" + key
	return updateDatum(key, locktoken, datum)
}

// Remember adds a short-term memory (with no backing store) to the robot's
// brain. This is used internally for resolving the meaning of "it", but can
// be used by plugins to remember other contextual facts. Since memories are
// indexed by user and channel, but not plugin, these facts can be referenced
// between plugins. This functionality is considered EXPERIMENTAL.
func (r *Robot) Remember(key, value string) {
	timestamp := time.Now()
	memory := shortTermMemory{value, timestamp}
	context := memoryContext{key, r.User, r.Channel}
	Log(Trace, fmt.Sprintf("SHORTMEM: Storing short-term memory \"%s\" -> \"%s\"", key, value))
	shortTermMemories.Lock()
	shortTermMemories.m[context] = memory
	shortTermMemories.Unlock()
}

// RememberContext is a convenience function that stores a context reference in
// short term memories. e.g. RememberContext("server", "web1.my.dom") means that
// next time the user uses "it" in the context of a "server", the robot will
// substitute "web1.my.dom".
func (r *Robot) RememberContext(context, value string) {
	r.Remember("context:"+context, value)
}

// Recall recalls a short term memory, or the empty string if it doesn't exist
func (r *Robot) Recall(key string) string {
	context := memoryContext{key, r.User, r.Channel}
	shortTermMemories.Lock()
	memory, ok := shortTermMemories.m[context]
	shortTermMemories.Unlock()
	Log(Trace, fmt.Sprintf("SHORTMEM: Recalling short-term memory \"%s\" -> \"%s\"", key, memory.memory))
	if !ok {
		return ""
	}
	return memory.memory
}

// RegisterSimpleBrain allows brain implementations to register a function with a named
// brain type that returns an SimpleBrain interface.
// This can only be called from a brain provider's init() function(s). Pass in a Logger
// so the brain can log it's own error messages if needed.
func RegisterSimpleBrain(name string, provider func(Handler, *log.Logger) SimpleBrain) {
	if stopRegistrations {
		return
	}
	if brains[name] != nil {
		log.Fatal("Attempted registration of duplicate brain provider name:", name)
	}
	brains[name] = provider
}

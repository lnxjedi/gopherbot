package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

// Map of registered brains
var brains = make(map[string]func(Handler, *log.Logger) SimpleBrain)

// short-term memories, mostly what "it" is
type shortTermMemory struct {
	memory  string
	learned time.Time
}

type memoryContext struct {
	key, user, channel string
}

var shortTermMemories = make(map[memoryContext]shortTermMemory)
var shortLock sync.Mutex

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
	cr := m.waiters[0]
	m.waiters = m.waiters[1:]
	lt, d, e, r := getDatum(cr.key, true)
	m.state = newMemory
	m.token = lt
	cr.reply <- checkOutReply{lt, d, e, r}
}

func getDatum(dkey string, rw bool) (token string, databytes *[]byte, exists bool, ret RetVal) {
	if !keyRe.MatchString(dkey) {
		err := fmt.Errorf("Invalid key supplied to checkout: %s", dkey)
		Log(Error, err)
		return "", nil, false, InvalidDatumKey
	}
	b.lock.RLock()
	brain := b.brain
	b.lock.RUnlock()
	if brain == nil {
		Log(Error, "Brain function called with no brain configured")
		return "", nil, false, BrainFailed
	}
	if rw { // checked out read/write, generate a lock token
		ltb := make([]byte, 8)
		random.Read(ltb)
		token = fmt.Sprintf("%x", ltb)
	} else {
		token = ""
	}
	var err error
	var db []byte
	db, exists, err = b.brain.Retrieve(dkey)
	if err != nil {
		return "", nil, false, BrainFailed
	}
	return token, &db, exists, Ok
}

func storeDatum(key string, datum *[]byte) RetVal {
	b.lock.RLock()
	brain := b.brain
	b.lock.RUnlock()
	if brain == nil {
		Log(Error, "Brain function called with no brain configured")
		return BrainFailed
	}
	err := b.brain.Store(key, *datum)
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
	// map key to status
	memories := make(map[string]*memstatus)
	processMemories := time.Tick(memCycle)
loop:
	for {
		select {
		case evt := <-brainChanEvents:
			switch evt.opType {
			case checkOutBytes:
				cr := evt.opData.(checkOutRequest)
				memStat, exists := memories[cr.key]
				if !exists {
					lt, d, e, r := getDatum(cr.key, cr.rw)
					if r != Ok {
						cr.reply <- checkOutReply{lt, d, e, r}
						break
					}
					if cr.rw {
						m := &memstatus{
							newMemory,
							lt,
							make([]checkOutRequest, 0, 2),
						}
						memories[cr.key] = m
					}
					cr.reply <- checkOutReply{lt, d, e, r}
					break
				}
				if !cr.rw {
					lt, d, e, r := getDatum(cr.key, cr.rw)
					cr.reply <- checkOutReply{lt, d, e, r}
					break
				} // read-write request below
				// if state is available, there are no waiters
				if memStat.state == available {
					lt, d, e, r := getDatum(cr.key, cr.rw)
					memStat.state = newMemory
					memStat.token = lt // this memory has a new owner now
					memories[cr.key] = memStat
					cr.reply <- checkOutReply{lt, d, e, r}
				} else {
					memStat.waiters = append(memStat.waiters, cr)
					memories[cr.key] = memStat
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
			shortLock.Lock()
			for k, v := range shortTermMemories {
				if now.Sub(v.learned) > shortTermDuration {
					delete(shortTermMemories, k)
				}
			}
			shortLock.Unlock()
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
	cr := checkOutRequest{d, rw, reply}
	brainChanEvents <- brainOp{checkOutBytes, cr}
	r := <-reply
	Log(Trace, fmt.Sprintf("Brain datum checkout for %s, rw: %t - token: %s, exists: %t, ret: %d",
		d, rw, r.token, r.exists, r.retval))
	return r.token, r.bytes, r.exists, r.retval
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
	plugMapLock.Lock()
	pluginName := plugIDNameMap[r.pluginID]
	plugMapLock.Unlock()
	key = pluginName + ":" + key
	return checkoutDatum(key, datum, rw)
}

// CheckinDatum unlocks a datum without updating it, it always succeeds
func (r *Robot) CheckinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	plugMapLock.Lock()
	pluginName := plugIDNameMap[r.pluginID]
	plugMapLock.Unlock()
	key = pluginName + ":" + key
	checkinDatum(key, locktoken)
}

// UpdateDatum tries to update a piece of data in the robot's brain, providing
// a struct to marshall and a (hopefully good) lock token. If err != nil, the
// update failed.
func (r *Robot) UpdateDatum(key, locktoken string, datum interface{}) (ret RetVal) {
	plugMapLock.Lock()
	pluginName := plugIDNameMap[r.pluginID]
	plugMapLock.Unlock()
	key = pluginName + ":" + key
	return updateDatum(key, locktoken, datum)
}

// Remember adds a short-term memory (with no backing store) to the robot's
// brain. This is used internally for resolving the meaning of "it", but can
// be used by plugins to remember other contextual facts. Since memories are
// indexed by user and channel, but not plugin, these facts can be referenced
// between plugins. This functionality is considered EXPERIMENTAL.
func (r *Robot) Remember(key, value string) {
	learned := time.Now()
	memory := shortTermMemory{value, learned}
	context := memoryContext{key, r.User, r.Channel}
	shortLock.Lock()
	shortTermMemories[context] = memory
	shortLock.Unlock()
}

// RememberNoun is a convenience function that stores a noun in short term
// memories. e.g. RememberNoun("server", "web1.my.dom") means that next time
// the user uses "it" in the context of a "server", the robot will substitute
// "web1.my.dom".
func (r *Robot) RememberNoun(noun, value string) {
	r.Remember("noun:"+noun, value)
}

// Recall recalls a short term memory, or the empty string if it doesn't exist
func (r *Robot) Recall(key string) string {
	context := memoryContext{key, r.User, r.Channel}
	shortLock.Lock()
	memory, ok := shortTermMemories[context]
	shortLock.Unlock()
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

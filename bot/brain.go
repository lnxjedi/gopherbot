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

func storeDatum(key string, datum *[]byte) (ret RetVal) {
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
		ret = BrainFailed
	}
	return Ok
}

var brainRunning bool
var brLock sync.RWMutex

// runBrain is the select loop that serializes access to brain
// functions and insures consistency.
func runBrain() {
	brainRunning = true
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
				m := memories[ci.key]
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
				m := memories[ur.key]
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
				break loop
			}
		case <-processMemories:
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
	brLock.Lock()
	brainRunning = false
	brLock.Unlock()
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
	brainChanEvents <- brainOp{updateBytes, ur}
	return <-reply
}

// checkinDatum is the internal version of CheckinDatum that uses the key as-is
func checkinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
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
	b.lock.RLock()
	pluginName := plugins[plugIDmap[r.pluginID]].name
	b.lock.RUnlock()
	key = pluginName + ":" + key
	return checkoutDatum(key, datum, rw)
}

// CheckinDatum unlocks a datum without updating it, it always succeeds
func (r *Robot) CheckinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	b.lock.RLock()
	pluginName := plugins[plugIDmap[r.pluginID]].name
	b.lock.RUnlock()
	key = pluginName + ":" + key
	checkinDatum(key, locktoken)
}

// UpdateDatum tries to update a piece of data in the robot's brain, providing
// a struct to marshall and a (hopefully good) lock token. If err != nil, the
// update failed.
func (r *Robot) UpdateDatum(key, locktoken string, datum interface{}) (ret RetVal) {
	b.lock.RLock()
	pluginName := plugins[plugIDmap[r.pluginID]].name
	b.lock.RUnlock()
	key = pluginName + ":" + key
	return updateDatum(key, locktoken, datum)
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

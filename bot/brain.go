package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

// Maximum amount of time that a plugin can hold a lock on a datum.
// If the plugin tries to UpdateDatum() after the timeout expires, it'll
// get an error. Even an external plugin should be able to manage
// 200 ms.
const lockTimeout = 200 * time.Millisecond

// Map of registered brains
var brains = make(map[string]func(Handler, *log.Logger) SimpleBrain)

// Global lock protecting the map of datum locks, individual datam,
// and datumLock.checkouts.
var dataLock sync.Mutex

// lock struct that protects a checked out datum
type datumLock struct {
	checkouts  int // the number of threads checking out this datum
	sync.Mutex     // the lock that protects the datum
}

// A global map of all synchronized items - we'll add the "glist"
var data = make(map[string]*datumLock)

// The lock is good as long a the token exists; token gets deleted
// on expiration.
var lockTokens = make(map[string]bool)
var ltLock sync.Mutex

const keyRegex = `[\w:]+` // keys can ony be word chars + separator (:)
var keyRe = regexp.MustCompile(keyRegex)

// checkout returns the []byte from the brain, with a lock that expires
// after lockTimeout. It returns a lock token, a pointer to the raw []byte
// data, true if the key exists, and a RetVal.
func checkout(d string, rw bool) (string, *[]byte, bool, RetVal) {
	if !keyRe.MatchString(d) {
		err := fmt.Errorf("Invalid key supplied to checkout: %s", d)
		Log(Error, err)
		return "", nil, false, InvalidDatumKey
	}
	var dl *datumLock
	dataLock.Lock() // wait for access to the global list
	dl, ok := data[d]
	if ok {
		dl.checkouts++
	} else {
		dl = &datumLock{checkouts: 1}
		data[d] = dl
	}
	dataLock.Unlock()
	var lt string
	if rw { // checked out read/write, generate a lock token
		ltb := make([]byte, 8)
		random.Read(ltb)
		lt = fmt.Sprintf("%x", ltb)
		ltLock.Lock()
		lockTokens[lt] = true
		ltLock.Unlock()
	} else {
		lt = ""
	}
	dl.Lock() // block until we get the lock for the datum
	// Retrieve the datum from the brain before starting the timer
	datum, exists, err := b.brain.Retrieve(d)
	if err != nil {
		dl.Unlock()
		return "", nil, false, BrainFailed
	}
	if rw {
		// once rw lock is acquired, spin off lock expiration thread
		go func(lt string, dl *datumLock) {
			time.Sleep(lockTimeout)
			ltLock.Lock()
			if _, ok := lockTokens[lt]; ok {
				Log(Error, fmt.Sprintf("Lock token %s expired, releasing lock", lt))
				delete(lockTokens, lt)
				dataLock.Lock()
				dl.Unlock()
				dl.checkouts--
				if dl.checkouts == 0 { // nobody was waiting
					delete(data, d)
				}
				dataLock.Unlock()
			}
			ltLock.Unlock()
		}(lt, dl)
	} else { // read-only copy checked out
		dataLock.Lock()
		dl.Unlock()
		dl.checkouts--
		if dl.checkouts == 0 {
			delete(data, d)
		}
		dataLock.Unlock()
	}
	return lt, &datum, exists, Ok
}

// update sends updated []byte to the brain while holding the lock, or discards
// the data and returns an error.
func update(d, lt string, datum *[]byte) (ret RetVal) {
	dataLock.Lock() // acquire the global lock
	dl, ok := data[d]
	if !ok {
		Log(Error, fmt.Sprintf("Update called on non-existent datum: %s", d))
		return DatumNotFound
	}
	ltLock.Lock() // we hope to get this lock before the timeout thread does
	if _, ok := lockTokens[lt]; ok {
		err := b.brain.Store(d, *datum)
		dl.Unlock() // unlock after we've updated, successful or not
		if err != nil {
			Log(Error, fmt.Sprintf("Storing datum %s: %v", d, err))
			ret = BrainFailed
		}
		delete(lockTokens, lt)
		dl.checkouts--
		if dl.checkouts == 0 {
			delete(data, d)
		}
		// when !ok, the lock token is expired and the dl is already unlocked
	} else {
		ret = DatumLockExpired
	}
	ltLock.Unlock()
	// Up to now has been 'instant' (no blocking) since the global lock was acquired
	dataLock.Unlock()
	return
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

// checkinDatum is the internal version of CheckinDatum that uses the key as-is
func checkinDatum(key, locktoken string) {
	if locktoken == "" {
		return
	}
	ltLock.Lock()
	if _, ok := lockTokens[locktoken]; ok { // see if the lock was even held
		delete(lockTokens, locktoken)
		dataLock.Lock()
		dl, ok := data[key]
		if ok {
			dl.Unlock()
			dl.checkouts--
			if dl.checkouts == 0 {
				delete(data, key)
			}
		}
		dataLock.Unlock()
	}
	ltLock.Unlock()
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

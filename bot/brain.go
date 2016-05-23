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
var brains map[string]func(l Logger, conf json.RawMessage) SimpleBrain = make(map[string]func(Logger, json.RawMessage) SimpleBrain)

// Global lock protecting the map of datum locks, individual datam,
// and datumLock.checkouts.
var dataLock sync.Mutex

// lock struct that protects a checked out datum
type datumLock struct {
	checkouts  int // the number of threads checking out this datum
	sync.Mutex     // the lock that protects the datum
}

// A global map of all synchronized items - we'll add the "glist"
var data map[string]*datumLock = make(map[string]*datumLock)

// The lock is good as long a the token exists; token gets deleted
// on expiration.
var lockTokens map[string]bool = make(map[string]bool)
var ltLock sync.Mutex

const keyRegex = `[\w:]+` // keys can ony be word chars + separator (:)
var keyRe = regexp.MustCompile(keyRegex)

// checkout returns the []byte from the brain, with a lock that expires
// after lockTimeout. It returns a lock token, a pointer to the raw []byte
// data, and true if the key exists.
func (r *robot) checkout(d string, rw bool) (string, *[]byte, bool, error) {
	if !keyRe.MatchString(d) {
		err := fmt.Errorf("Invalid key supplied to checkout: %s", d)
		r.Log(Error, err)
		return "", nil, false, err
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
	datum, exists := r.brain.Retrieve(d)
	if rw {
		// once rw lock is acquired, spin off lock expiration thread
		go func(lt string, dl *datumLock) {
			time.Sleep(lockTimeout)
			ltLock.Lock()
			if _, ok := lockTokens[lt]; ok {
				r.Log(Error, "Lock token %s expired, releasing lock", lt)
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
	return lt, &datum, exists, nil
}

// update sends updated []byte to the brain while holding the lock, or discards
// the data and returns an error.
func (r *robot) update(d, lt string, datum *[]byte) error {
	dataLock.Lock() // acquire the global lock
	dl, ok := data[d]
	if !ok {
		r.Log(Error, "Update called on non-existent datum: %s", d)
		return fmt.Errorf("Update called on non-existent datum")
	}
	ltLock.Lock() // we hope to get this lock before the timeout thread does
	updated := false
	var err error
	if _, ok := lockTokens[lt]; ok {
		updated = true
		err = r.brain.Store(d, *datum)
		dl.Unlock() // unlock after we've updated
		delete(lockTokens, lt)
		dl.checkouts--
		if dl.checkouts == 0 {
			delete(data, d)
		}
	} // when !ok, the dl is already unlocked
	ltLock.Unlock()
	// Up to now has been 'instant' (no blocking) since the global lock was acquired
	dataLock.Unlock()
	if updated {
		return err
	} else {
		return fmt.Errorf("Lock %s expired", lt)
	}
}

// CheckoutDatum gets a datum from the robot's brain and unmarshals it into
// a struct. If rw is set, the datum is checked out read-write and a non-empty
// lock token is returned that expires after lockTimeout (250ms). The bool
// return indicates whether the datum exists.
func (r *Robot) CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, err error) {
	b := r.robot
	b.lock.RLock()
	pluginName := b.plugins[b.plugIDmap[r.pluginID]].Name
	b.lock.RUnlock()
	key = pluginName + ":" + key
	return r.checkoutDatum(key, datum, rw)
}

// checkoutDatum is the robot internal version of CheckoutDatum that uses
// the provided key as-is.
func (r *Robot) checkoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, err error) {
	var dbytes *[]byte
	locktoken, dbytes, exists, err = r.checkout(key, rw)
	if err != nil {
		r.Log(Error, fmt.Errorf("Reading key %s from brain:", key, err))
		return "", false, err
	}
	if exists {
		err = json.Unmarshal(*dbytes, datum)
		if err != nil {
			r.Log(Error, fmt.Errorf("Unmarshalling datum %s:", key, err))
			return "", false, err
		}
	}
	return locktoken, exists, nil
}

// Checkin unlocks a datum without updating it
func (r *Robot) Checkin(key, locktoken string) {
	b := r.robot
	b.lock.RLock()
	pluginName := b.plugins[b.plugIDmap[r.pluginID]].Name
	b.lock.RUnlock()
	key = pluginName + ":" + key
	r.checkin(key, locktoken)
}

// checkin is the internal version of Checkin that uses the key as-is
func (r *Robot) checkin(key, locktoken string) {
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

// UpdateDatum tries to update a piece of data in the robot's brain, providing
// a struct to marshall and a (hopefully good) lock token. If err != nil, the
// update failed.
func (r *Robot) UpdateDatum(key, locktoken string, datum interface{}) error {
	b := r.robot
	b.lock.RLock()
	pluginName := b.plugins[b.plugIDmap[r.pluginID]].Name
	b.lock.RUnlock()
	key = pluginName + ":" + key
	return r.updateDatum(key, locktoken, datum)
}

// updateDatum is the internal version of UpdateDatum that uses the key as-is
func (r *Robot) updateDatum(key, locktoken string, datum interface{}) error {
	dbytes, err := json.Marshal(datum)
	if err != nil {
		errmsg := fmt.Errorf("Unmarshalling datum: %v", err)
		r.Log(Error, errmsg)
		return errmsg
	}
	b := r.robot
	err = b.update(key, locktoken, &dbytes)
	if err != nil {
		b.Log(Error, fmt.Errorf("Updating datum %s: %v", key, err))
	}
	return err
}

// RegisterBrain allows brain implementations to register a function with a named
// brain type that returns an XXXBrain interface (currently only SimpleBrain).
// When the bot initializes, it will look for a function registered under the configured
// "Brain" in gopherbot.json, then pass in rawJSON config and get back an interface.
// This can only be called from a brain provider's init() function(s). Pass in a Logger
// so the brain can log error messages if needed.
func RegisterBrain(name string, provider func(Logger, json.RawMessage) SimpleBrain) {
	if stopRegistrations {
		return
	}
	if brains[name] != nil {
		log.Fatal("Attempted registration of duplicate brain provider name:", name)
	}
	brains[name] = provider
}

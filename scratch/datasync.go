package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

var random *rand.Rand

// A grocery list whose access needs to synchronized
var glist []string = []string{"bananas"}

// A list of items to add to the grocery list
var alist []string = []string{
	"cereal",
	"bricks",
	"tiles",
	"lollipops",
	"gum",
	"drinks",
	"beer",
	"tortillas",
	"lemons",
	"pears",
	"fritos",
	"cake",
	"broccoli",
	"cheezits",
	"carrots",
	"milk",
	"cheese",
	"bread",
	"oranges",
	"avocados",
	"asparagus",
	"pencils",
	"journal",
	"laptop computer",
	"oatmeal",
	"chips",
	"dips",
	"chains",
	"whips",
	"whales",
	"harpoons",
}

const lockTimeout = time.Second

// The struct for sync'ing access to a list
type datumLock struct {
	checkouts int
	sync.Mutex
}

// A global map of all synchronized items - we'll add the "glist"
var data map[string]*datumLock = make(map[string]*datumLock)

// Global lock protecting the map of datum locks, individual datam,
// and datumLock.checkouts. For this example the only datam is the
// global glist.
var dataLock sync.Mutex

var lockTokens map[string]bool = make(map[string]bool)
var ltLock sync.Mutex

// checkout blocks until the thread can get write access, then
// grants exclusive rights to the datum for 1 sec.
func checkout(d string) ([]string, string) {
	dataLock.Lock() // wait for access to the global list
	var dl *datumLock
	dl, ok := data[d]
	if ok {
		dl.checkouts++
	} else {
		log.Printf("Creating new datumLock: %s\n", d)
		dl = &datumLock{checkouts: 1}
		data[d] = dl
	}
	dataLock.Unlock()
	ltb := make([]byte, 8)
	random.Read(ltb)
	lt := fmt.Sprintf("%x", ltb)
	ltLock.Lock()
	lockTokens[lt] = true
	ltLock.Unlock()
	dl.Lock() // block until we get the lock
	// now this lock token can use the lock for 1 second
	go func(lt string, dl *datumLock) {
		time.Sleep(lockTimeout)
		ltLock.Lock()
		if _, ok := lockTokens[lt]; ok {
			log.Printf("Lock token %s expired, releasing lock for next waiting thread", lt)
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
	return glist, lt
}

// update updates the global list if the lockToken is still good,
// then decrements the checkouts counter and deletes the item from
// the global list if 0.
func update(d string, g []string, lt string) error {
	updated := false
	dataLock.Lock() // acquire the global lock
	dl, ok := data[d]
	if !ok {
		return fmt.Errorf("Update called on non-existent datum")
	}
	ltLock.Lock()
	if _, ok := lockTokens[lt]; ok {
		updated = true
		glist = g   // update the list
		dl.Unlock() // unlock after we've updated
		delete(lockTokens, lt)
		dl.checkouts--
		if dl.checkouts == 0 {
			log.Printf("Deleting datum %s\n", d)
			delete(data, d)
		}
	} // when !ok, the dl is already unlocked
	ltLock.Unlock()
	// Up to now has been 'instant' (no blocking) since the global lock was acquired
	dataLock.Unlock()
	if updated {
		return nil
	} else {
		return fmt.Errorf("Lock %s expired", lt)
	}
}

func main() {
	// Seed the pseudo-random number generator, for plugin IDs, RandomString, etc.
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	var wg sync.WaitGroup
	fmt.Printf("List at start: %v\n", glist)
	for i := 0; i < len(alist); i++ {
		wg.Add(1)
		go func(i int) {
			log.Printf("Thread #%d waiting for the lock", i)
			gl, lt := checkout("grocerylist") // get a 1 second exclusive lock on grocerylist
			log.Printf("Thread #%d acquired the lock with token %s", i, lt)
			sleep := i/2*100 + 50
			if i%2 == 0 {
				time.Sleep(time.Duration(sleep) * time.Millisecond)
			} else {
				sleep = 0
			}
			gl = append(gl, alist[i])
			err := update("grocerylist", gl, lt)
			if err != nil {
				log.Printf("*** ERROR *** Thread #%d error adding %s to grocery list after sleeping %d milliseconds: %v\n", i, alist[i], sleep, err)
			} else {
				log.Printf("Thread #%d successfully added %s to the grocery list after sleeping %d milliseconds, exiting", i, alist[i], sleep)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Println()
	fmt.Printf("List at end: %v, tried to add %d items, total %d\n", glist, len(alist), len(glist))
}

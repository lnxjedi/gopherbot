package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

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

/*var alist []string = []string{
	"cereal",
	"carrots",
	"milk",
	"cheese",
	"bread",
	"oranges",
	"avocados",
	"pencils",
	"journal",
	"laptop computer",
	"harpoons",
}*/

const lockTimeout = time.Second

// The struct for sync'ing access to a list
type datumLock struct {
	checkouts int
	sync.Mutex
}

var currentToken = 0

// The lock item given on checkout
type lockToken struct {
	tNum   int
	isGood chan bool
}

// A global map of all synchronized items - we'll add the "glist"
var data map[string]*datumLock = make(map[string]*datumLock)

// Global lock protecting the map of datum locks, individual datam,
// and datumLock.checkouts. For this example the only datam is the
// global glist.
var dataLock sync.Mutex

// checkout blocks until the thread can get write access, then
// grants exclusive rights to the datum for 1 sec.
func checkout(d string) ([]string, lockToken) {
	dataLock.Lock() // wait for access to the global list
	var dl *datumLock
	dl, ok := data[d]
	if ok {
		dl.checkouts++
	} else {
		log.Printf("Creating new datum: %s\n", d)
		dl = &datumLock{checkouts: 1}
		data[d] = dl
	}
	var lt lockToken
	lt.tNum = currentToken
	currentToken++
	lt.isGood = make(chan bool)
	dataLock.Unlock()
	dl.Lock() // block until we get the lock
	// now this lock token can use the lock for 1 second
	go func(lt lockToken, dl *datumLock) {
		expired := false
		select {
		case lt.isGood <- true: // in this case, the datum has been checked in/updated, and will unlock dl when it finishes
		case <-time.After(lockTimeout):
			expired = true
			log.Println("Lock expired, releasing lock for next waiting thread")
			dl.Unlock() // the lock expired, another thread can get the lock a few lines above
		}
		if expired {
			lt.isGood <- false // the lock is expired, but we block here until the thread eventually tries to update
		}
	}(lt, dl)
	return glist, lt
}

// update updates the global list if the lockToken is still good,
// then decrements the checkouts counter and deletes the item from
// the global list if 0.
func update(d string, g []string, lt lockToken) error {
	dataLock.Lock() // acquire the global lock
	updated := false
	dl, ok := data[d]
	if !ok {
		return fmt.Errorf("Update called on non-existent datum")
	}
	ok = <-lt.isGood // happens instantly with true or false
	if ok {          // fast enough, we can update the list!
		updated = true
		glist = g   // update the list
		dl.Unlock() // unlock after we've updated
	} // when !ok, the dl is already unlocked
	// Up to now has been 'instant' (no blocking) since the global lock was acquired
	dl.checkouts--
	if dl.checkouts == 0 {
		log.Printf("Deleting datum %s\n", d)
		delete(data, d)
	}
	dataLock.Unlock()
	if updated {
		return nil
	} else {
		return fmt.Errorf("Lock expired")
	}
}

func main() {
	var wg sync.WaitGroup
	fmt.Printf("List at start: %v\n", glist)
	for i := 0; i < len(alist); i++ {
		wg.Add(1)
		go func(i int) {
			log.Printf("Thread #%d waiting for the lock", i)
			gl, lt := checkout("grocerylist") // get a 1 second exclusive lock on grocerylist
			log.Printf("Thread #%d acquired the lock", i)
			sleep := i / 2 * 100
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

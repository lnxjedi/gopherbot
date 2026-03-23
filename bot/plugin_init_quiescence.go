package bot

import "sync"

type pluginInitBatch struct {
	done chan struct{}
	wg   sync.WaitGroup
}

func newPluginInitBatch() *pluginInitBatch {
	return &pluginInitBatch{
		done: make(chan struct{}),
	}
}

func (b *pluginInitBatch) add() {
	b.wg.Add(1)
}

func (b *pluginInitBatch) complete() {
	b.wg.Done()
}

func (b *pluginInitBatch) seal() {
	go func() {
		b.wg.Wait()
		close(b.done)
	}()
}

func (b *pluginInitBatch) wait() {
	if b == nil {
		return
	}
	<-b.done
}

var pluginInitState = struct {
	sync.Mutex
	batch *pluginInitBatch
}{
	batch: func() *pluginInitBatch {
		b := newPluginInitBatch()
		b.seal()
		return b
	}(),
}

func setCurrentPluginInitBatch(batch *pluginInitBatch) {
	if batch == nil {
		batch = newPluginInitBatch()
		batch.seal()
	}
	pluginInitState.Lock()
	pluginInitState.batch = batch
	pluginInitState.Unlock()
}

func waitForPluginInitQuiescence() {
	pluginInitState.Lock()
	batch := pluginInitState.batch
	pluginInitState.Unlock()
	batch.wait()
}

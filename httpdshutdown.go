package httpdshutdown

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

type ShutdownHook func() error

type Watcher struct {
	connsWG       *sync.WaitGroup
	shutdownHooks []ShutdownHook
	timeoutMS     int
}

// NewWatcher construct a Watcher with a timeout and an optional set of shutdown hooks
// to be called at the time of shutdown.
func NewWatcher(t int, hooks ...ShutdownHook) (*Watcher, error) {
	if t < 0 {
		return nil, errors.New("timeout must be a positive number")
	}
	w := new(Watcher)
	w.timeoutMS = t
	w.connsWG = new(sync.WaitGroup)
	w.shutdownHooks = make([]ShutdownHook, len(hooks))
	copy(w.shutdownHooks, hooks)
	return w, nil
}

// RecordConnState counts open and closed connections.
func (w *Watcher) RecordConnState(newState http.ConnState) {
	if w == nil {
		// we panic here instead of returning nil as the calling context does not
		// do any error checking
		panic("RecordConnState: receiver is nil")
	}
	switch newState {
	case http.StateNew:
		w.connsWG.Add(1)
	case http.StateClosed, http.StateHijacked:
		w.connsWG.Done()
	}
}

// RunHooks executes registered hooks, each of which blocks.
func (w *Watcher) RunHooks() error {
	if w == nil {
		return errors.New("RunHooks: receiver is nil")
	}
	for _, f := range w.shutdownHooks {
		err := f()
		if err != nil {
			log.Printf("shutdown hook err: %v\n", err.Error())
		}
	}
	return nil
}

// OnStop will be called by a daemon's signal handler when it is time to shutdown. If there
// are any shutdown handlers, they will be called. The timeout set on the watcher will
// be honored.
func (w *Watcher) OnStop() error {
	if w == nil {
		return errors.New("OnStop: receiver is nil")
	}
	waitChan := make(chan bool, 1)
	go func() {
		w.connsWG.Wait()
		waitChan <- true
	}()
	select {
	case <-waitChan:
		log.Printf("OnStop: conns completed, graceful exit possible; running any hooks.")
		_ = w.RunHooks()
		return nil
	case <-time.After(time.Duration(w.timeoutMS) * time.Millisecond):
		log.Printf("OnStop: shutdown timed out, running any hooks.")
		_ = w.RunHooks()
		return errors.New("OnStop: shutdown timed out.")
	}
}

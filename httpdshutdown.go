// Package httpdshutdown implements some convenience functions for cleanly shutting down
// an http daemon.
package httpdshutdown

import (
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"
)

// ShutdownHook is the type callers will implement in their own daemon shutdown handlers.
type ShutdownHook func() error

type Watcher struct {
	connsWG       *sync.WaitGroup // Allows us to wait for conns to complete.
	shutdownHooks []ShutdownHook  // Run these when daemon is done or timed out.
	timeoutMS     int             // Grace period for daemon shutdown.
	accepting     bool            // Should the daemon accept conns?
	acceptingMut  *sync.RWMutex   // Protect access to accepting state.
}

// NewWatcher construct a Watcher with a timeout and an optional set of shutdown hooks
// to be called at the time of shutdown.
func NewWatcher(timeoutMS int, hooks ...ShutdownHook) (*Watcher, error) {
	if timeoutMS < 0 {
		return nil, errors.New("timeout must be a positive number")
	}
	w := new(Watcher)
	w.timeoutMS = timeoutMS
	w.connsWG = new(sync.WaitGroup)
	w.shutdownHooks = make([]ShutdownHook, len(hooks))
	copy(w.shutdownHooks, hooks)
	w.accepting = false
	w.acceptingMut = new(sync.RWMutex)
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

// Accepting sets the Watcher's accepting state to true or false.
func (w *Watcher) Accepting(acceptingState bool) error {
	if w == nil {
		return errors.New("Accepting: receiver is nil")
	}
	w.acceptingMut.Lock()
	w.accepting = acceptingState
	w.acceptingMut.Unlock()
	return nil
}

// IsAccepting tells the caller if the daemon will accept conns or not.
func (w *Watcher) IsAccepting() (bool, error) {
	if w == nil {
		return false, errors.New("IsAccepting: receiver is nil")
	}
	w.acceptingMut.RLock()
	acceptingState := w.accepting
	w.acceptingMut.RUnlock()
	return acceptingState, nil
}

// OnStop will be called by a daemon's signal handler when it is time to shutdown. If there
// are any shutdown handlers, they will be called. The timeout set on the watcher will
// be honored.
func (w *Watcher) OnStop() error {
	if w == nil {
		return errors.New("OnStop: receiver is nil")
	}
	_ = w.Accepting(false) // No longer accepting conns.
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

// SigHandle is an example of a typical signal handler that will attempt a graceful shutdown
// for a set of known signals.
func SigHandle(c <-chan os.Signal, w *Watcher) {
	if w == nil {
		// panic since this will typically be launched as a goroutine.
		panic("SigHandler: Watcher is nil")
	}
	for sig := range c {
		if sig == syscall.SIGTERM || sig == syscall.SIGQUIT || sig == syscall.SIGHUP {
			log.Printf("*** caught signal %v, stop\n", sig)
			stopErr := w.OnStop()
			if stopErr != nil {
				log.Printf("OnStop err: %s", stopErr.Error())
				log.Printf("control has failed to shut down gracefully\n")
				os.Exit(1)
			}
			log.Printf("control has shut down gracefully\n")
			os.Exit(0)
		} else if sig == syscall.SIGINT {
			log.Printf("*** caught signal %v, PANIC stop\n", sig)
			panic("panic exit")
		} else {
			// uncomment this if you want to see uncaught signals
			// log.Printf("**** caught unchecked signal %v\n", sig)
		}
	}
}

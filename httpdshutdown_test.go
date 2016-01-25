package httpdshutdown

import (
	"fmt"
	"net/http"
	"testing"
)

func TestNil(t *testing.T) {
	var w *Watcher
	w = nil
	err := w.Accepting(true)
	if err == nil {
		t.Errorf("TestNil: should have error")
	}
	_,err = w.IsAccepting()
	if err == nil {
		t.Errorf("TestNil: should have error")
	}
	err = w.OnStop()
	if err == nil {
		t.Errorf("TestNil: should have error")
	}
}

func TestBadTimeout(t *testing.T) {
	_,w_err := NewWatcher(-1)
	if w_err == nil {
		t.Errorf("TestBadTimeout: should have error")
	}
}

func TestValid(t *testing.T) {
	w,w_err := NewWatcher(3000)	
	if w == nil || w_err != nil {
		t.Errorf("TestValid: should not be nil")
	}
	err := w.Accepting(true)
	if err != nil {
		t.Errorf("TestValid: should not have error")
	}
	accepting,err_a := w.IsAccepting()
	if err_a != nil {
		t.Errorf("TestValid: should not have error")
	}
	if accepting != true {
		t.Errorf("TestValid: should be true")
	}
	err = w.OnStop()
	if err != nil {
		t.Errorf("TestValid: should not have error")
	}
}

func sampleShutdownHook() error {
	fmt.Println("shutdown hook called")
	return nil
}

func TestStop(t *testing.T) {
	w,w_err := NewWatcher(3000,sampleShutdownHook)
	if w == nil || w_err != nil {
		t.Errorf("TestStop: should not be nil")
	}
	err := w.Accepting(true)
	if err != nil {
		t.Errorf("TestStop: should not have error")
	}
	w.RecordConnState(http.StateNew)
	err = w.OnStop()
	if err == nil {
		t.Errorf("TestStop: should have error from 1 second timeout to force stop")
	}
	w.RecordConnState(http.StateClosed)
	err = w.OnStop()
	if err != nil {
		t.Errorf("TestStop: should not have an error")
	}
	w.RecordConnState(http.StateNew)
	w.RecordConnState(http.StateHijacked)
	err = w.OnStop()
	if err != nil {
		t.Errorf("TestStop: should not have an error")
	}
}

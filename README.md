[![License BSD](https://img.shields.io/badge/License-BSD-blue.svg)](http://opensource.org/licenses/BSD-3-Clause)
[![Go Report Card](https://goreportcard.com/badge/github.com/bradclawsie/httpdshutdown)](https://goreportcard.com/report/github.com/bradclawsie/httpdshutdown)
[![GoDoc](https://godoc.org/github.com/bradclawsie/httpshutdown?status.svg)](http://godoc.org/github.com/bradclawsie/httpdshutdown)

# httpdshutdown
Count connections to permit daemons to stop safely, and run hooks at
shutdown time.

# NOTE
As of early 2017, the Go standard library includes some of this
functionality in the standard `net/http` library. You might want to check that
first. 

# EXAMPLE

```
package main

import (
    "log"
	"net"
	"net/http"
	"github.com/bradclawsie/httpdshutdown"
	"os"
	"os/signal"
	"time"
)

func sampleShutdownHook1() error {
	log.Println("shutdown hook 1 called")
	return nil
}

func sampleShutdownHook2() error {
	log.Println("shutdown hook 2 called")
	return nil
}

func main() {
	log.Printf("launching with pid:%d\n", os.Getpid())
	watcher, watcher_err := httpdshutdown.NewWatcher(2000, sampleShutdownHook1, sampleShutdownHook2)
	if watcher == nil || watcher_err != nil {
		panic("could not construct watcher")
	}

	// Launch the signal handler and exit logic in a goroutine since the http daemon
	// issued later will run in the foreground.
	go func() {
		sigs := make(chan os.Signal, 1)
		exitcode := make(chan int, 1)
		signal.Notify(sigs)
		go watcher.SigHandle(sigs, exitcode)
		code := <-exitcode
		log.Printf("exit with code:%d", code)
		os.Exit(code)
	}()

	srv := &http.Server{
		Addr: ":8080",
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		ConnState: func(conn net.Conn, newState http.ConnState) {
			log.Printf("(1) NEW CONN STATE:%v\n", newState)
			watcher.RecordConnState(newState)
			return
		},
	}

	log.Fatal(srv.ListenAndServe())
}

```

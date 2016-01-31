/*

This example illustrates how to start a new watcher, run its signal handler, and allow
it to watch the connection state for a daemon

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

*/
package main

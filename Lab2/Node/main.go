package main

import (
	"bufio"
	"fmt"
	"math/big"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
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

func HandleConnection(listener net.Listener, node *Node) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept failed:", err.Error())
			continue
		}
		go jsonrpc.ServeConn(conn)
	}
}

func testBetween() {
	// Test between in big.Int
	start := big.NewInt(26)
	elt := big.NewInt(56)
	end := big.NewInt(26)
	fmt.Println(between(start, elt, end, true))
}

func main() {
	// testBetween()
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

		IPAddr := fmt.Sprintf("%s:%d", Arguments.Address, Arguments.Port)
		tcpAddr, err := net.ResolveTCPAddr("tcp4", IPAddr)
		if err != nil {
			fmt.Println("ResolveTCPAddr failed:", err.Error())
			os.Exit(1)
		}
		rpc.Register(node)
		// Listen to the address

		// cert, _ := tls.LoadX509KeyPair("../chord.crt", "../chord.key")
		// config := &tls.Config{
		// 	Certificates: []tls.Certificate{cert},
		// }
		// listener, err := tls.Listen("tcp", tcpAddr.String(), config)

		listener, err := net.Listen("tcp", tcpAddr.String())
		if err != nil {
			fmt.Println("ListenTCP failed:", err.Error())
			os.Exit(1)
		}
		fmt.Println("Local node listening on ", tcpAddr)
		// Use a separate goroutine to accept connection
		go HandleConnection(listener, node)

		if valid == 0 {
			// Join exsiting chord
			RemoteAddr := fmt.Sprintf("%s:%d", Arguments.JoinAddress, Arguments.JoinPort)

			// Connect to the remote node
			fmt.Println("Connecting to the remote node..." + RemoteAddr)
			err := node.joinChord(NodeAddress(RemoteAddr))
			if err != nil {
				fmt.Println("Join RPC call failed")
				os.Exit(1)
			} else {
				fmt.Println("Join RPC call success")
			}
		} else if valid == 1 {
			// Create new chord
			node.createChord()
			// Combine address and port, convert port to string
		}

		// Start periodic tasks
		se_stab := ScheduledExecutor{delay: time.Duration(Arguments.Stabilize) * time.Millisecond, quit: make(chan int)}
		se_stab.Start(func() {
			node.stablize()
		})

		se_ff := ScheduledExecutor{delay: time.Duration(Arguments.FixFingers) * time.Millisecond, quit: make(chan int)}
		se_ff.Start(func() {
			node.fixFingers()
		})

		se_cp := ScheduledExecutor{delay: time.Duration(Arguments.CheckPred) * time.Millisecond, quit: make(chan int)}
		se_cp.Start(func() {
			node.checkPredecessor()
		})

		// Get user input for printing states
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Enter command: ")
			command, _ := reader.ReadString('\n')
			command = strings.TrimSpace(command)
			command = strings.ToUpper(command)
			if command == "PRINTSTATE" || command == "PS" {
				node.printState()
				getLocalAddress()
			} else if command == "LOOKUP" || command == "L" {
				fmt.Println("Please enter the key you want to lookup")
				key, _ := reader.ReadString('\n')
				key = strings.TrimSpace(key)
				fmt.Println(key)
				resultAddr, err := clientLookUp(key, node)
				if err != nil {
					fmt.Print(err)
				} else {
					fmt.Println("The address of the key is ", resultAddr)

				}
				// Check if the key is stored in the node
				checkFileExistRPCReply := CheckFileExistRPCReply{}
				err = ChordCall(resultAddr, "Node.CheckFileExistRPC", key, &checkFileExistRPCReply)
				if err != nil {
					fmt.Println("Check file exist RPC call failed")
				} else {
					if checkFileExistRPCReply.Exist {
						// Get the address of the node that stores the file
						var getNameRPCReply GetNameRPCReply
						err = ChordCall(resultAddr, "Node.GetNameRPC", "", &getNameRPCReply)
						if err != nil {
							fmt.Println("Get name RPC call failed")
						} else {
							fmt.Println("The file is stored at ", getNameRPCReply.Name)
						}
					} else {
						fmt.Println("The file is not stored in the node")
					}
				}
			} else if command == "STOREFILE" || command == "S" {
				fmt.Println("Please enter the file name you want to store")
				fileName, _ := reader.ReadString('\n')
				fileName = strings.TrimSpace(fileName)
				err := clientStoreFile(fileName, node)
				if err != nil {
					fmt.Print(err)
				} else {
					fmt.Println("Store file success")
				}
			} else if command == "QUIT" || command == "Q" {
				// Quit the program
				// Assign a value to quit channel to stop periodic tasks
				se_stab.quit <- 1
				se_ff.quit <- 1
				se_cp.quit <- 1
				os.Exit(0)
			} else if command == "GET" || command == "G" {
				// Get file from the network
				fmt.Println("Please enter the file name you want to get")
				fileName, _ := reader.ReadString('\n')
				fileName = strings.TrimSpace(fileName)
				err := clientGetFile(fileName, node)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Get file success")
				}
			} else {
				fmt.Println("Invalid command")
			}
		}
	}

}

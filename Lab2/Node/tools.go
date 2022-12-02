package main

import (
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/rpc/jsonrpc"
	"os"
	"regexp"
)

/*------------------------------------------------------------*/
/*                  Comm Interface By: Alexwell               */
/*------------------------------------------------------------*/

/*
* @description: Communication interface between nodes
* @param: 		targetNode: the address of the node to be connected
* @param: 		method: the name of the method to be called, e.g. "Node.FindSuccessorRPC".
*						method need to be registered in the RPC server, and have Golang compliant RPC method style
* @param:		request: the request to be sent
* @param:		reply: the reply to be received
* @return:		error: the error returned by the RPC call
 */
/*
type RPCServive interface{

	node.FindSuccessorRPC(requestID *big.Int, reply *FindSuccessorRPCReply) error
}
*/
func ChordCall(targetNode NodeAddress, method string, request interface{}, reply interface{}) error {
	// fmt.Println("Dial to ", targetNode)
	client, err := jsonrpc.Dial("tcp", string(targetNode))
	if err != nil {
		fmt.Println("Dial Error: ", err)
		return err
	}
	defer client.Close()
	err = client.Call(method, request, reply)
	if err != nil {
		fmt.Println("Call Error:", err)
		return err
	}
	return nil
}

/*------------------------------------------------------------*/
/*                     Tool Functions Below                   */
/*------------------------------------------------------------*/

type Arguments struct {
	// Read command line arguments
	Address     NodeAddress // Current node address
	Port        int         // Current node port
	JoinAddress NodeAddress // Joining node address
	JoinPort    int         // Joining node port
	Stabilize   int         // The time in milliseconds between invocations of stabilize.
	FixFingers  int         // The time in milliseconds between invocations of fix_fingers.
	CheckPred   int         // The time in milliseconds between invocations of check_predecessor.
	Successors  int
	ClientName  string
}

func getCmdArgs() Arguments {
	// Read command line arguments
	var a string  // Current node address
	var p int     // Current node port
	var ja string // Joining node address
	var jp int    // Joining node port
	var ts int    // The time in milliseconds between invocations of stabilize.
	var tff int   // The time in milliseconds between invocations of fix_fingers.
	var tcp int   // The time in milliseconds between invocations of check_predecessor.
	var r int     // The number of successors to maintain.
	var i string  // Client name

	// Parse command line arguments
	flag.StringVar(&a, "a", "localhost", "Current node address")
	flag.IntVar(&p, "p", 8000, "Current node port")
	flag.StringVar(&ja, "ja", "Unspecified", "Joining node address")
	flag.IntVar(&jp, "jp", 8000, "Joining node port")
	flag.IntVar(&ts, "ts", 1000, "The time in milliseconds between invocations of stabilize.")
	flag.IntVar(&tff, "tff", 1000, "The time in milliseconds between invocations of fix_fingers.")
	flag.IntVar(&tcp, "tcp", 1000, "The time in milliseconds between invocations of check_predecessor.")
	flag.IntVar(&r, "r", 3, "The number of successors to maintain.")
	flag.StringVar(&i, "i", "Default", "Client ID")
	flag.Parse()

	// Return command line arguments
	return Arguments{
		Address:     NodeAddress(a),
		Port:        p,
		JoinAddress: NodeAddress(ja),
		JoinPort:    jp,
		Stabilize:   ts,
		FixFingers:  tff,
		CheckPred:   tcp,
		Successors:  r,
		ClientName:  i,
	}
}

func CheckArgsValid(args Arguments) int {
	// Check if Ip address is valid or not
	if net.ParseIP(string(args.Address)) == nil && args.Address != "localhost" {
		fmt.Println("IP address is invalid")
		return -1
	}
	// Check if port is valid
	if args.Port < 1024 || args.Port > 65535 {
		fmt.Println("Port number is invalid")
		return -1
	}

	// Check if durations are valid
	if args.Stabilize < 1 || args.Stabilize > 60000 {
		fmt.Println("Stabilize time is invalid")
		return -1
	}
	if args.FixFingers < 1 || args.FixFingers > 60000 {
		fmt.Println("FixFingers time is invalid")
		return -1
	}
	if args.CheckPred < 1 || args.CheckPred > 60000 {
		fmt.Println("CheckPred time is invalid")
		return -1
	}

	// Check if number of successors is valid
	if args.Successors < 1 || args.Successors > 32 {
		fmt.Println("Successors number is invalid")
		return -1
	}

	// Check if client name is s a valid string matching the regular expression [0-9a-fA-F]{40}
	if args.ClientName != "Default" {
		matched, err := regexp.MatchString("[0-9a-fA-F]*", args.ClientName)
		if err != nil || !matched {
			fmt.Println("Client Name is invalid")
			return -1
		}
	}

	// Check if joining address and port is valid or not
	if args.JoinAddress != "Unspecified" {
		// Addr is specified, check if addr & port are valid
		if net.ParseIP(string(args.JoinAddress)) != nil || args.JoinAddress == "localhost" {
			// Check if join port is valid
			if args.JoinPort < 1024 || args.JoinPort > 65535 {
				fmt.Println("Join port number is invalid")
				return -1
			}
			// Join the chord
			return 0
		} else {
			fmt.Println("Joining address is invalid")
			return -1
		}
	} else {
		// Join address is not specified, create a new chord ring
		// ignroe jp input
		return 1
	}
}

// func call(address string, method string, request interface{}, reply interface{}) error{
// 	return rpc.NewClientWithCodec(rpc.NewClientCodec(
// }

func strHash(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

func between(start, elt, end *big.Int, inclusive bool) bool {
	if end.Cmp(start) > 0 {
		return (start.Cmp(elt) < 0 && elt.Cmp(end) < 0) || (inclusive && elt.Cmp(end) == 0)
	} else {
		return start.Cmp(elt) < 0 || elt.Cmp(end) < 0 || (inclusive && elt.Cmp(end) == 0)
	}
}

func clientLookUp(key string, node *Node) (NodeAddress, error) {
	// Find the successor of key
	// Return the successor's address and port
	newKey := strHash(key)
	addr := find(newKey, node.Address)
	if addr == "-1" {
		return "", errors.New("cannot find the store position of the key")
	} else {
		return addr, nil
	}
}

// File structure
type FileRPC struct {
	Id      *big.Int
	Name    string
	Content []byte
}

func clientStoreFile(fileName string, node *Node) error {
	// Store the file in the node
	// Return the address and port of the node that stores the file
	addr, err := clientLookUp(fileName, node)
	if err != nil {
		return err
	} else {
		fmt.Println("The file is stored in node: ", addr)
	}
	// Open file and pack into fileRPC
	filepath := "../file_upload/" + fileName
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Cannot open the file")
		return err
	}
	defer file.Close()
	// Init new file struct and put content into it
	newFile := FileRPC{}
	newFile.Name = fileName
	newFile.Content, err = ioutil.ReadAll(file)
	newFile.Id = strHash(fileName)
	if err != nil {
		return err
	} else {
		reply := new(StoreFileRPCReply)
		err = ChordCall(addr, "Node.StoreFileRPC", newFile, &reply)
		if reply.Err != nil && err != nil {
			return errors.New("cannot store the file")
		}
	}
	return nil
}

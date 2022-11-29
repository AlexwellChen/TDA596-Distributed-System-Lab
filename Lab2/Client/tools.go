package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"flag"
	"fmt"
	"math/big"
	"net/rpc"
	"time"
)

/*------------------------------------------------------------*/
/*                    Node Defination Below                   */
/*------------------------------------------------------------*/

// Main function + Node defination :Qi

var fingerTableSize = 161 // Use 1-160 Todo: 真的需要160的finger table吗？

type Key string // For file

type NodeAddress string // For node

// FileAddress: [K]13 store in [N]14

type Node struct {
	// Node attributes
	Name       string   // Name: IP:Port or User specified Name. Exp: [N]14
	Identifier *big.Int // Hash(Address) -> Chord space Identifier

	// For Chord search
	Address     NodeAddress // Address should be "IP:Port"
	FingerTable []NodeAddress

	// For Chord stabilization
	Predecessor NodeAddress
	Successors  []NodeAddress // Multiple successors to handle first succesor node failures

	// For Chord data encryption
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey

	Bucket map[Key]string // Hash Key -> File name value store
	/* Exp:
	     ------------Store File-------------
	     	Hash(Hello.txt) -> 123
			Bucket[123] = Hello.txt

	     -------------Read File-------------
	     	FileName = Bucket[123]
			ReadFile(FileName) -> Hello World
	*/
}

func (node *Node) generateRSAKey(bits int) {
	// GenerateKey函数使用随机数据生成器random生成一对具有指定字位数的RSA密钥
	// Reader是一个全局、共享的密码用强随机数生成器
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(err)
	}
	node.PrivateKey = privateKey
	node.PublicKey = &privateKey.PublicKey
}

func NewNode(args Arguments) *Node {
	// Create a new node
	node := &Node{}
	node.Address = args.Address
	node.Name = args.ClientName
	node.Identifier = strHash(string(node.Address))
	node.FingerTable = make([]NodeAddress, fingerTableSize)
	node.Predecessor = ""
	node.Successors = make([]NodeAddress, args.Successors)
	node.Bucket = make(map[Key]string)
	node.generateRSAKey(2048)
	node.initFingerTable()
	node.initSuccessors()
	return node
}

func (node *Node) initFingerTable() {
	// Initialize finger table
	for i := 0; i < fingerTableSize; i++ {
		node.FingerTable[i] = node.Address
	}
}

func (node *Node) initSuccessors() {
	// Initialize successors
	successorsSize := len(node.Successors)
	for i := 0; i < successorsSize; i++ {
		node.Successors[i] = node.Address
	}
}

func (node *Node) joinChord() {
	// Todo: Join the Chord ring
}

func (node *Node) createChord() {
	// Todo: Create a Chord ring
}

func (node *Node) leaveChord() {
	// Todo: Leave the Chord ring
	// For failure handling, backup the data in the bucket to the successor (Bonus)
}

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
func ChordCall(targetNode NodeAddress, method string, request interface{}, reply interface{}) error {
	client, err := rpc.DialHTTP("tcp", string(targetNode))
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.Call(method, request, &reply)
	return err
}

/*------------------------------------------------------------
                Stabilizing Functions Below	By:wang
--------------------------------------------------------------*/

func (node *Node) Stabilize() {
	// Todo: Stabilize the Chord ring
	// next stores the index of the next finger to fix.

}

// check whether predecessor has failed
func (node *Node) CheckPredecessor() error {
	pred := node.Predecessor
	if pred != "" {
		//check connection
		client, err := rpc.DialHTTP("tcp", string(pred))
		//if connection failed, set predecessor to nil
		if err != nil {
			fmt.Printf("Predecessor %v has failed", pred)
			node.Predecessor = ""
		} else {
			client.Close()
		}
	}
	return nil
}

// Notify tells the node at 'address' that it might be our predecessor
func (node *Node) Notify(address string) error {
	//if (predecessor is nil or n' ∈ (predecessor, n))
	if node.Predecessor == "" ||
		between(strHash(string(node.Predecessor)),
			strHash(address), strHash(string(node.Address)), false) {
		//predecessor = n'
		node.Predecessor = NodeAddress(address)
	}
	return nil
}

func (node *Node) FixFingers() {
	//Todo: refreshes finger table entries
}

/*------------------------------------------------------------*/
/*                  Routing Functions By: Alexwell            */
/*------------------------------------------------------------*/

type FindSuccessorRPCReply struct {
	found            bool
	SuccessorAddress NodeAddress
}

/*
* @description: RPC method Packaging for FindSuccessor, running on remote node
* @param: 		requestID: the client ID to be searched
* @return: 		found: whether the key is found
* 				successor: the successor of the key
 */
func (node *Node) FindSuccessorRPC(requestID string, reply *FindSuccessorRPCReply) error {
	fmt.Println("-------------- Invoke FindSuccessor_RPC function ------------")
	reply.found, reply.SuccessorAddress = node.findSuccessor(requestID)
	return nil
}

// Local use function
func (node *Node) findSuccessor(requestID string) (bool, NodeAddress) {
	fmt.Println("*************** Invoke findSuccessor function ***************")
	if between(node.Identifier, strHash(requestID), strHash(string(node.Successors[0])), true) {
		return true, node.Successors[0]
	} else {
		return false, node.closePrecedingNode(requestID)
	}
}

// Local use function
func (node *Node) closePrecedingNode(requestID string) NodeAddress {
	fmt.Println("************ Invoke closePrecedingNode function ************")
	fingerTableSize := len(node.FingerTable)
	for i := fingerTableSize - 1; i >= 1; i-- {
		if between(node.Identifier, strHash(string(node.FingerTable[i])), strHash(requestID), true) {
			return node.FingerTable[i]
		}
	}
	return node.Successors[0]
}

// Local use function
func find(id string, startNode NodeAddress) NodeAddress {
	fmt.Println("****************** Invoke find function *********************")
	found := false
	nextNode := startNode
	i := 0
	maxSteps := 10
	for !found && i < maxSteps {
		// Todo: Send request to nextNode, execute FindSuccessor(id), and return the result
		// found, nextNode = nextNode.FindSuccessor(id)
		result := FindSuccessorRPCReply{}
		err := ChordCall(nextNode, "Node.FindSuccessorRPC", id, &result)
		if err != nil {
			fmt.Println(err)
			break
		}
		found = result.found
		nextNode = result.SuccessorAddress
		i++
	}
	if found {
		fmt.Println("Find Success in ", i, " steps.")
		return nextNode
	} else {
		fmt.Println("Find Failed, please try again.")
		return "-1"
	}
}

/*------------------------------------------------------------*/
/*                     Tool Functions Below                   */
/*------------------------------------------------------------*/

type Arguments struct {
	// Read command line arguments
	Address     NodeAddress   // Current node address
	Port        int           // Current node port
	JoinAddress NodeAddress   // Joining node address
	JoinPort    int           // Joining node port
	Stabilize   time.Duration // The time in milliseconds between invocations of stabilize.
	FixFingers  time.Duration // The time in milliseconds between invocations of fix_fingers.
	CheckPred   time.Duration // The time in milliseconds between invocations of check_predecessor.
	Successors  int
	ClientName  string
}

func getCmdArgs() Arguments {
	// Read command line arguments
	var a string          // Current node address
	var p int             // Current node port
	var ja string         // Joining node address
	var jp int            // Joining node port
	var ts time.Duration  // The time in milliseconds between invocations of stabilize.
	var ttf time.Duration // The time in milliseconds between invocations of fix_fingers.
	var tcp time.Duration // The time in milliseconds between invocations of check_predecessor.
	var r int             // The number of successors to maintain.
	var i string          // Client name

	// Parse command line arguments
	flag.StringVar(&a, "a", "localhost", "Current node address")
	flag.IntVar(&p, "p", 8000, "Current node port")
	flag.StringVar(&ja, "ja", "Unspecified", "Joining node address")
	flag.IntVar(&jp, "jp", 8000, "Joining node port")
	flag.DurationVar(&ts, "ts", 1000, "The time in milliseconds between invocations of stabilize.")
	flag.DurationVar(&ttf, "ttf", 1000, "The time in milliseconds between invocations of fix_fingers.")
	flag.DurationVar(&tcp, "tcp", 1000, "The time in milliseconds between invocations of check_predecessor.")
	flag.IntVar(&r, "r", 3, "The number of successors to maintain.")
	flag.StringVar(&i, "i", "Unspecified", "Client name")
	flag.Parse()

	// Return command line arguments
	return Arguments{
		Address:     NodeAddress(a),
		Port:        p,
		JoinAddress: NodeAddress(ja),
		JoinPort:    jp,
		Stabilize:   ts,
		FixFingers:  ttf,
		CheckPred:   tcp,
		Successors:  r,
		ClientName:  i,
	}
}

func checkArgsValid(args Arguments) {
	// Todo: Check if the command line arguments are valid

}

// func call(address string, method string, request interface{}, reply interface{}) error{
// 	return rpc.NewClientWithCodec(jsonrpc.NewClientCodec(
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

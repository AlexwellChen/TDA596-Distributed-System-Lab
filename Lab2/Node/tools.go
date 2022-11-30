package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/rpc"
	"regexp"
)

/*------------------------------------------------------------*/
/*                    Node Defination Below                   */
/*------------------------------------------------------------*/

// Main function + Node defination :Qi

// Test with 10 nodes on Chord ring, finger table size should larger than 5
var fingerTableSize = 6 // Each finger table i contains the id of (n + 2^i) mod (2^m)th node. Use [0, 5] as i and space would be [(n+1)%64, (n+32)%64]
var m = 6               // Chord space has 2^6 = 64 identifiers

// 2^m
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(m)), nil)

type Key string // For file

type NodeAddress string // For node

// FileAddress: [K]13 store in [N]14

// fingerEntry represents a single finger table entry
type fingerEntry struct {
	Id      []byte      // ID hash of (n + 2^i) mod (2^m)
	Address NodeAddress // RemoteAddress
}

type Node struct {
	// Node attributes
	Name       string   // Name: IP:Port or User specified Name. Exp: [N]14
	Identifier *big.Int // Hash(Address) -> Chord space Identifier

	// For Chord search
	Address     NodeAddress // Address should be "IP:Port"
	FingerTable []fingerEntry
	next        int // next stores the index of the next finger to fix. [1,m]

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
	node.FingerTable = make([]fingerEntry, fingerTableSize)
	node.next = 0
	node.Predecessor = ""
	node.Successors = make([]NodeAddress, args.Successors)
	node.Bucket = make(map[Key]string)
	node.generateRSAKey(2048)
	node.initFingerTable()
	node.initSuccessors()
	return node
}

/*
* @description: fingerEntry.Id could be seen as the Chord ring address
* 	            fingerEntry.Address is the real ip address of the file exist node or the node itself
 */
func (node *Node) initFingerTable() {
	// Initialize finger table
	for i := 0; i < fingerTableSize; i++ {
		// Caculate the id of the ith finger
		// id = (n + 2^i) mod (2^m)
		id := new(big.Int).Add(node.Identifier, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(i)), nil))
		id.Mod(id, hashMod)
		node.FingerTable[i].Id = id.Bytes()

		// Address is the acutal ip address of the nodes on Chord ring
		node.FingerTable[i].Address = node.Address
	}
}

func (node *Node) initSuccessors() {
	// Initialize successors
	successorsSize := len(node.Successors)
	for i := 0; i < successorsSize; i++ {
		node.Successors[i] = ""
	}
}

func (node *Node) joinChord(joinNode NodeAddress) {
	// Todo: Join the Chord ring
	// Find the successor of the node's identifier
	// Set the node's predecessor to nil and successors to the exits node
	// joinNode is the successor of current node, which is node.Successors[0]
	// current node will be the predecessor of joinNode
	node.Predecessor = ""
	node.Successors[0] = joinNode

	// Fine other successors, use FindSuccessor RPC
	for i := 1; i < len(node.Successors); i++ {
		var reply FindSuccessorRPCReply
		err := ChordCall(node.Successors[i-1], "Node.FindSuccessorRPC", node.Identifier, &reply)
		if err != nil {
			fmt.Println("Error: ", err)
			break
		}
		if reply.found {
			node.Successors[i] = reply.SuccessorAddress
		} else {
			fmt.Println("Find ", i, "th successor failed")
			break
		}
	}

	// Notify the successor[0] that we are its predecessor
	reply := false
	err := ChordCall(node.Successors[0], "Node.SetPredecessorRPC", node.Address, &reply)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	if reply {
		fmt.Println("Set predecessor success")
	} else {
		fmt.Println("Set predecessor failed")
	}
}

func (node *Node) setPredecessor(predecessorAddress NodeAddress) *bool {
	node.Predecessor = predecessorAddress
	flag := true
	return &flag
}

func (node *Node) SetPredecessorRPC(predecessorAddress NodeAddress, reply *bool) error {
	fmt.Println("-------------- Invoke SetPredecessorRPC function ------------")
	reply = node.setPredecessor(predecessorAddress)
	return nil
}

func (node *Node) createChord() {
	// Create a new Chord ring
	// Set the node's predecessor to nil and successors to itself
	node.Predecessor = ""
	node.Successors[0] = node.Address
}

func (node *Node) leaveChord() {
	// Todo: Leave the Chord ring
	// For failure handling, backup the data in the bucket to the successor (Bonus)
}

func (node *Node) printState() {
	// Print current node state
	fmt.Println("-------------- Current Node State ------------")
	fmt.Println("Node Name: ", node.Name)
	fmt.Println("Node Address: ", node.Address)
	fmt.Println("Node Identifier: ", node.Identifier)
	fmt.Println("Node Predecessor: ", node.Predecessor)
	fmt.Println("Node Successors: ")
	for i := 0; i < len(node.Successors); i++ {
		fmt.Println("Successor ", i, " address: ", node.Successors[i])
	}
	fmt.Println("Node Finger Table: ")
	for i := 0; i < fingerTableSize; i++ {
		enrty := node.FingerTable[i]
		id := new(big.Int).SetBytes(enrty.Id)
		address := enrty.Address
		fmt.Println("Finger ", i, " id: ", id, ", address: ", address)
	}
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
/*
type RPCServive interface{

	node.FindSuccessorRPC(requestID *big.Int, reply *FindSuccessorRPCReply) error
}
*/
func ChordCall(targetNode NodeAddress, method string, request interface{}, reply interface{}) error {
	client, err := rpc.Dial("tcp", string(targetNode))
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.Call(method, request, &reply)
	return err
}

/*
------------------------------------------------------------

	Stabilizing Functions Below	By:wang

--------------------------------------------------------------
*/
// get node's predecessor
func (node *Node) GetPredecessor(none *struct{}, predecessor *NodeAddress) error {
	fmt.Println("-------------- Invoke GetPredecessor function --------------")
	*predecessor = node.Predecessor
	if node.Predecessor == "" {
		return errors.New("predecessor is empty")
	} else {
		return nil
	}
}

// get node's successorList
func (node *Node) GetSuccessorList(none *struct{}, successorList *[]NodeAddress) error {
	fmt.Println("-------------- Invoke GetSuccessorList function -------------")
	*successorList = node.Successors[:]
	return nil
}

// verifies n’s immediate
func (node *Node) stabilize() error {
	//Todo: search paper 看看是每个successor都要修改prodecessor还是只修改第一个successor
	//Todo: search paper 看看是不是要fix successorList
	fmt.Println("***************** Invoke stabilize function *****************")
	var successors []NodeAddress
	err := ChordCall(node.Successors[0], "Node.GetSuccessorList", struct{}{}, &successors)
	if err == nil {
		for i := 0; i < len(successors); i++ {
			node.Successors[i+1] = successors[i]
		}
	} else {
		fmt.Println("GetSuccessorList failed")
		if node.Successors[0] == "" {
			fmt.Println("Node Successor[0] is empty -> use self as successor")
			node.Successors[0] = node.Address
		} else {
			for i := 0; i < len(node.Successors); i++ {
				if i == len(node.Successors)-1 {
					node.Successors[i] = ""
				} else {
					node.Successors[i] = successors[i+1]
				}
			}
		}
	}
	var predecessor NodeAddress = ""
	err = ChordCall(node.Successors[0], "Node.GetPredecessor", struct{}{}, &predecessor)
	if err == nil {
		if predecessor != "" && between(strHash(string(node.Address)),
			strHash(string(predecessor)), strHash(string(node.Successors[0])), false) {
			node.Successors[0] = predecessor
		}
	}

	err = ChordCall(node.Successors[0], "Node.Notify", node.Address, &struct{}{})
	if err != nil {
		fmt.Println("Notify failed")
	}
	return nil

}

// check whether predecessor has failed
func (node *Node) checkPredecessor() error {
	fmt.Println("************* Invoke checkPredecessor function **************")
	pred := node.Predecessor
	if pred != "" {
		//check connection
		client, err := rpc.Dial("tcp", string(pred))
		//if connection failed, set predecessor to nil
		if err != nil {
			fmt.Printf("Predecessor %s has failed", string(pred))
			node.Predecessor = ""
		} else {
			client.Close()
		}
	}
	return nil
}

// 'address' thinks it might be our predecessor
func (node *Node) Notify(address NodeAddress) error {
	fmt.Println("------------------- Invoke Notify function ------------------")
	//if (predecessor is nil or n' ∈ (predecessor, n))
	if node.Predecessor == "" ||
		between(strHash(string(node.Predecessor)),
			strHash(string(address)), strHash(string(node.Address)), false) {
		//predecessor = n'
		node.Predecessor = address
	}
	return nil
}

// calculate (n + 2^(k-1) ) mod 2^m
func (node *Node) fingerEntry(fingerentry int) *big.Int {
	//Todo: check if use len(node.Address) or fingerTableSize
	fmt.Println("************** Invoke fingerEntry function ******************")
	// 2^m ? use len(node.Address)
	//var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(len(node.FingerTable)-1)), nil)
	// id = n (node n identifier)
	id := node.Identifier
	two := big.NewInt(2)
	//exponent := big.NewInt(int64(len(node.FingerTable)) - 1) // fingerentry -1?
	exponent := big.NewInt(int64(fingerentry - 1))
	//2^(k-1)
	two.Exp(two, exponent, nil)
	// n + 2^(k-1)
	id.Add(id, two)
	// (n + 2^(k-1) ) mod 2^m , 1 <= k <= m
	return id.Mod(id, hashMod)
}

// refreshes finger table entries, next stores the index of the next finger to fix
func (node *Node) fixFingers() error {
	//Todo: search paper check node.next 在到了m的时候是不是要从1开始还是0，以及初始化
	fmt.Println("*************** Invoke findSuccessor function ***************")
	for {
		node.next = node.next + 1
		//use 1-160, m = 160 next > m = next > fingerTableSize-1
		if node.next > fingerTableSize-1 {
			node.next = 1
		}
		id := node.fingerEntry(node.next)
		//find successor of id
		_, addr := node.findSuccessor(id)
		if addr != "" && addr != node.FingerTable[node.next].Address {
			node.FingerTable[node.next].Address = addr
			node.FingerTable[node.next].Id = id.Bytes()
		}
		/* 		result := FindSuccessorRPCReply{}
		   		err := ChordCall(node.Address, "Node.FindSuccessorRPC", id, &result)
		   		if err != nil {
		   			fmt.Println(err)
		   			break
		   		}
		   		nextNode := result.SuccessorAddress */

		//update fingerEntry(next)
		for {
			node.next = node.next + 1
			if node.next > fingerTableSize-1 {
				node.next = 0
				return nil
			}

			if between(strHash(string(node.Address)), id, strHash(string(addr)), false) && addr != "" {
				node.FingerTable[node.next].Id = id.Bytes()
				node.FingerTable[node.next].Address = NodeAddress(addr)
			} else {
				node.next--
				return nil
			}
		}
	}
}

/*------------------------------------------------------------*/
/*                  Routing Functions By: Alexwell            */
/*------------------------------------------------------------*/

type FindSuccessorRPCReply struct {
	found            bool
	SuccessorAddress NodeAddress
}

/*
* @description: RPC method Packaging for findSuccessor, running on remote node
* @param: 		requestID: the client address or file name to be searched
* @return: 		found: whether the key is found
* 				successor: the successor of the key
 */
func (node *Node) FindSuccessorRPC(requestID *big.Int, reply *FindSuccessorRPCReply) error {
	fmt.Println("-------------- Invoke FindSuccessor_RPC function ------------")
	reply.found, reply.SuccessorAddress = node.findSuccessor(requestID)
	return nil
}

// Local use function
func (node *Node) findSuccessor(requestID *big.Int) (bool, NodeAddress) {
	fmt.Println("*************** Invoke findSuccessor function ***************")
	if between(node.Identifier, requestID, strHash(string(node.Successors[0])), true) {
		return true, node.Successors[0]
	} else {
		return false, node.closePrecedingNode(requestID)
	}
}

// Local use function
func (node *Node) closePrecedingNode(requestID *big.Int) NodeAddress {
	fmt.Println("************ Invoke closePrecedingNode function ************")
	fingerTableSize := len(node.FingerTable)
	for i := fingerTableSize - 1; i >= 1; i-- {
		if between(node.Identifier, strHash(string(node.FingerTable[i].Address)), requestID, true) {
			return node.FingerTable[i].Address
		}
	}
	return node.Successors[0]
}

// Local use function
// Lookup
func find(id *big.Int, startNode NodeAddress) NodeAddress {
	fmt.Println("****************** Invoke find function *********************")
	found := false
	nextNode := startNode
	i := 0
	maxSteps := 10 // 2^maxSteps
	for !found && i < maxSteps {
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
	flag.StringVar(&i, "i", "Default", "Client Name")
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

	// Check if client ID is s a valid string matching the regular expression [0-9a-fA-F]{40}
	if args.ClientName != "Default" {
		matched, err := regexp.MatchString("[0-9a-fA-F]*", args.ClientName)
		if err != nil || !matched {
			fmt.Println("Client ID is invalid")
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

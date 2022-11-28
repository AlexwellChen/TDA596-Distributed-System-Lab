package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"flag"
	"math/big"
	"time"
)

/*------------------------------------------------------------*/
/*                    Node Defination Below                   */
/*------------------------------------------------------------*/

var successorsSize = 3

var fingerTableSize = 160 // Todo: 真的需要这么大的finger table吗？

type Key string

type NodeAddress string

type Node struct {
	// For Chord search
	Address     NodeAddress
	FingerTable []NodeAddress

	// For Chord stabilization
	Predecessor NodeAddress
	Successors  []NodeAddress // Multiple successors to handle first succesor node failures

	// For Chord data encryption
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey

	Bucket map[Key]string // Hash Key -> File value store
}

func (node *Node) GenerateRSAKey(bits int) {
	// GenerateKey函数使用随机数据生成器random生成一对具有指定字位数的RSA密钥
	// Reader是一个全局、共享的密码用强随机数生成器
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(err)
	}
	node.PrivateKey = privateKey
	node.PublicKey = &privateKey.PublicKey
}

func (node *Node) Init(args Arguments) {
	// Initialize node
	node.Address = args.Address
	node.FingerTable = make([]NodeAddress, fingerTableSize)
	node.Predecessor = ""
	node.Successors = make([]NodeAddress, successorsSize)
	node.Bucket = make(map[Key]string)
	node.GenerateRSAKey(2048)
}

func (node *Node) InitFingerTable() {
	// Initialize finger table
	for i := 0; i < fingerTableSize; i++ {
		node.FingerTable[i] = node.Address
	}
}

func (node *Node) InitSuccessors() {
	// Initialize successors
	for i := 0; i < successorsSize; i++ {
		node.Successors[i] = node.Address
	}
}

func (node *Node) Stabilize() {
	// Todo: Stabilize the Chord ring

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
	ClientID    string
}

func Hash(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

func GetCmdArgs() Arguments {
	// Read command line arguments
	var a string          // Current node address
	var p int             // Current node port
	var ja string         // Joining node address
	var jp int            // Joining node port
	var ts time.Duration  // The time in milliseconds between invocations of stabilize.
	var ttf time.Duration // The time in milliseconds between invocations of fix_fingers.
	var tcp time.Duration // The time in milliseconds between invocations of check_predecessor.
	var r int             // The number of successors to maintain.
	var i string          // Client ID

	// Parse command line arguments
	flag.StringVar(&a, "a", "localhost", "Current node address")
	flag.IntVar(&p, "p", 8000, "Current node port")
	flag.StringVar(&ja, "ja", "Unspecified", "Joining node address")
	flag.IntVar(&jp, "jp", 8000, "Joining node port")
	flag.DurationVar(&ts, "ts", 1000, "The time in milliseconds between invocations of stabilize.")
	flag.DurationVar(&ttf, "ttf", 1000, "The time in milliseconds between invocations of fix_fingers.")
	flag.DurationVar(&tcp, "tcp", 1000, "The time in milliseconds between invocations of check_predecessor.")
	flag.IntVar(&r, "r", 3, "The number of successors to maintain.")
	flag.StringVar(&i, "i", "Unspecified", "Client ID")

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
		ClientID:    i,
	}
}

func CheckArgsValid(args Arguments) {
	// Todo: Check if the command line arguments are valid

}

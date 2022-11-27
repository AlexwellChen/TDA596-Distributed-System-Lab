package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"flag"
	"math/big"
	"time"
)

var successorsSize = 5

type Key string

type NodeAddress string

type Node struct {
	Address     NodeAddress
	FingerTable []NodeAddress
	Predecessor NodeAddress
	Successors  []NodeAddress
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey

	Bucket map[Key]string
}

type Arguments struct {
	// Read command line arguments
	Address     NodeAddress // Current node address
	Port        int         // Current node port
	JoinAddress NodeAddress // Joining node address
	JoinPort    int         // Joining node port
	Stabilize   time.Duration
	FixFingers  time.Duration
	CheckPred   time.Duration
	Successors  int
	ClientID    string
}

func (node *Node) GenerateRSAKey(bits int) {
	//GenerateKey函数使用随机数据生成器random生成一对具有指定字位数的RSA密钥
	//Reader是一个全局、共享的密码用强随机数生成器
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(err)
	}
	node.PrivateKey = privateKey
	node.PublicKey = &privateKey.PublicKey
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

func (node *Node) Init(args Arguments) {
	// Initialize node
	node.Address = args.Address
	node.FingerTable = make([]NodeAddress, 160)
	node.Predecessor = ""
	node.Successors = make([]NodeAddress, successorsSize)
	node.Bucket = make(map[Key]string)
	node.GenerateRSAKey(2048)
}

func hash(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

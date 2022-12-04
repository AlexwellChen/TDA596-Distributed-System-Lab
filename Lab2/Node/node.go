package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
)

/*------------------------------------------------------------*/
/*                    Node Defination Below                   */
/*------------------------------------------------------------*/

// Main function + Node defination :Qi

// Test with 10 nodes on Chord ring, finger table size should larger than 5
var fingerTableSize = 6 // Each finger table i contains the id of (n + 2^i) mod (2^m)th node. Use [1, 6] as i and space would be [(n+1)%64, (n+32)%64]
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
	next        int // next stores the index of the next finger to fix. [0,m-1]

	// For Chord stabilization
	Predecessor NodeAddress
	Successors  []NodeAddress // Multiple successors to handle first succesor node failures

	// For Chord data encryption
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	// Create bucket in form of map
	Bucket map[*big.Int]string
	Backup map[*big.Int]string
	// Bucket map[string]string // Hash Key -> File name value store
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

	// Store private key in Node folder
	priDerText := x509.MarshalPKCS1PrivateKey(privateKey)
	block := pem.Block{
		Type: node.Name + "-private Key",

		Headers: nil,

		Bytes: priDerText,
	}
	node_files_folder := "../files/" + node.Name
	privateHandler, err := os.Create(node_files_folder + "/private.pem")
	if err != nil {
		panic(err)
	}
	defer privateHandler.Close()
	pem.Encode(privateHandler, &block)

	// Store public key in Node folder
	pubDerText, err := x509.MarshalPKIXPublicKey(node.PublicKey)
	if err != nil {
		panic(err)
	}
	block = pem.Block{
		Type: node.Name + "-public Key",

		Headers: nil,

		Bytes: pubDerText,
	}
	publicHandler, err := os.Create(node_files_folder + "/public.pem")
	if err != nil {
		panic(err)
	}
	defer publicHandler.Close()
	pem.Encode(publicHandler, &block)
}

func NewNode(args Arguments) *Node {
	// Create a new node
	node := &Node{}
	node.Address = NodeAddress(fmt.Sprintf("%s:%d", args.Address, args.Port))
	if args.ClientName == "Default" {
		node.Name = string(node.Address)
	} else {
		node.Name = args.ClientName
	}
	node.Identifier = strHash(string(node.Name))
	node.Identifier.Mod(node.Identifier, hashMod)
	if node.Identifier.Cmp(big.NewInt(0)) == 0 {
		// Identifier should not be 0, exit os
		fmt.Println("Node identifier should not be 0, try another name")
		os.Exit(1)
	}
	node.FingerTable = make([]fingerEntry, fingerTableSize+1)
	node.Bucket = make(map[*big.Int]string)
	node.Backup = make(map[*big.Int]string)
	node.next = 0 // start from -1, then use fixFingers() to add 1 -> 0 max: m-1
	node.Predecessor = ""
	node.Successors = make([]NodeAddress, args.Successors)
	node.initFingerTable()
	node.initSuccessors()
	// Create Node folder in upper directory
	if _, err := os.Stat("../files/" + node.Name); os.IsNotExist(err) {
		err := os.Mkdir("../files/"+node.Name, 0777)
		if err != nil {
			fmt.Println("Create Node folder failed")
		} else {
			// Create file_upload folder in Node folder
			if _, err := os.Stat("../files/" + node.Name + "/file_upload"); os.IsNotExist(err) {
				os.Mkdir("../files/"+node.Name+"/file_upload", 0777)
			} else {
				fmt.Println("file_upload folder already exist")
			}
			// Create file_download folder in Node folder
			if _, err := os.Stat("../files/" + node.Name + "/file_download"); os.IsNotExist(err) {
				os.Mkdir("../files/"+node.Name+"/file_download", 0777)
			} else {
				fmt.Println("file_download folder already exist")
			}
			// Create chord_storage folder in Node folder
			if _, err := os.Stat("../files/" + node.Name + "/chord_storage"); os.IsNotExist(err) {
				os.Mkdir("../files/"+node.Name+"/chord_storage", 0777)
			} else {
				fmt.Println("chord_storage folder already exist")
			}
		}
		node.generateRSAKey(2048)
	} else {
		fmt.Println("Node folder already exist")
		// Init bucket
		// Read all files in chord_storage folder
		files, err := ioutil.ReadDir("../files/" + node.Name + "/chord_storage")
		if err != nil {
			fmt.Println("Read chord_storage folder failed")
		}
		for _, file := range files {
			// Store file name in bucket
			fileName := file.Name()
			fileHash := strHash(fileName)
			fileHash.Mod(fileHash, hashMod)
			node.Bucket[fileHash] = fileName
		}
		// Init private key
		privateHandler, err := os.Open("../files/" + node.Name + "/private.pem")
		if err != nil {
			panic(err)
		}
		defer privateHandler.Close()
		privateKeyBuffer, err := ioutil.ReadAll(privateHandler)
		priBlock, _ := pem.Decode(privateKeyBuffer)
		privateKey, err := x509.ParsePKCS1PrivateKey(priBlock.Bytes)
		if err != nil {
			panic(err)
		}
		node.PrivateKey = privateKey
		node.PublicKey = &node.PrivateKey.PublicKey
	}

	return node
}

/*
* @description: fingerEntry.Id could be seen as the Chord ring address
* 	            fingerEntry.Address is the real ip address of the file exist node or the node itself
 */
func (node *Node) initFingerTable() {
	// Initialize finger table
	node.FingerTable[0].Id = node.Identifier.Bytes()
	node.FingerTable[0].Address = node.Address
	fmt.Println("fingerTable[0] = ", node.FingerTable[0].Id, node.FingerTable[0].Address)
	for i := 1; i < fingerTableSize+1; i++ {
		// Caculate the id of the ith finger
		// id = (n + 2^i-1) mod (2^m)
		id := new(big.Int).Add(node.Identifier, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(i)-1), nil))
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

func (node *Node) joinChord(joinNode NodeAddress) error {
	// Todo: Join the Chord ring
	// Find the successor of the node's identifier
	// Set the node's predecessor to nil and successors to the exits node
	// joinNode is the successor of current node, which is node.Successors[0]
	// current node will be the predecessor of joinNode
	node.Predecessor = ""
	fmt.Printf("Node %s join the Chord ring: %s ", node.Name, joinNode)

	//  Join node is in charge of looking for the successor of the node's identifier
	// 1. Call the joinNode's findSuccessor() to find the successor of the node's identifier
	var reply FindSuccessorRPCReply
	err := ChordCall(joinNode, "Node.FindSuccessorRPC", node.Identifier, &reply)
	fmt.Println("Successor: ", reply.SuccessorAddress)
	node.Successors[0] = reply.SuccessorAddress
	if err != nil {
		return err
	}
	// 2. Call the successor's notify() to notify the successor that the node is its predecessor
	err = ChordCall(node.Successors[0], "Node.NotifyRPC", node.Address, &reply)
	if err != nil {
		return err
	}
	return nil
}

func (node *Node) createChord() {
	// Create a new Chord ring
	// Set the node's predecessor to nil and successors to itself
	node.Predecessor = ""
	// All successors are itself when create a new Chord ring
	for i := 0; i < len(node.Successors); i++ {
		node.Successors[i] = node.Address
	}
}

func (node *Node) leaveChord() {
	// Todo: What fault tolerance should be considered? unexpected node failure or user exit?
}

func (node *Node) printState() {
	// Print current node state
	fmt.Println("-------------- Current Node State ------------")
	fmt.Println("Node Name: ", node.Name)
	fmt.Println("Node Address: ", node.Address)
	fmt.Println("Node Identifier: ", new(big.Int).SetBytes(node.Identifier.Bytes()))
	fmt.Println("Node Predecessor: ", node.Predecessor)
	fmt.Println("Node Successors: ")
	for i := 0; i < len(node.Successors); i++ {
		fmt.Println("Successor ", i, " address: ", node.Successors[i])
	}
	fmt.Println("Node Finger Table: ")
	for i := 0; i < fingerTableSize+1; i++ {
		enrty := node.FingerTable[i]
		id := new(big.Int).SetBytes(enrty.Id)
		address := enrty.Address
		fmt.Println("Finger ", i, " id: ", id, ", address: ", address)
	}
	fmt.Println("Node Bucket: ")
	for k, v := range node.Bucket {
		fmt.Println("Key: ", k, ", Value: ", v)
	}
	fmt.Println("Node Backup:")
	for k, v := range node.Backup {
		fmt.Println("Key: ", k, ", Value: ", v)
	}

}

/*------------------------------------------------------------*/
/*                    RPC functions Below                     */
/*------------------------------------------------------------*/

type SetPredecessorRPCReply struct {
	Success bool
}

func (node *Node) setPredecessor(predecessorAddress NodeAddress) bool {
	node.Predecessor = predecessorAddress
	flag := true
	return flag
}

// TODO: warning here:argument reply is overwritten before first use
func (node *Node) SetPredecessorRPC(predecessorAddress NodeAddress, reply *SetPredecessorRPCReply) error {
	fmt.Println("-------------- Invoke SetPredecessorRPC function ------------")
	reply.Success = node.setPredecessor(predecessorAddress)
	if reply.Success {
		fmt.Println("Set predecessor success")
	} else {
		fmt.Println("Set predecessor failed")
		return errors.New("set predecessor failed")
	}
	return nil
}

func (node *Node) storeChordFile(f FileRPC, backup bool) bool {
	// Store the file in the bucket
	// Return true if success, false if failed
	// Append the file to the bucket
	f.Id.Mod(f.Id, hashMod)
	if backup {
		node.Backup[f.Id] = f.Name
	} else {
		node.Bucket[f.Id] = f.Name
		fmt.Println("Bucket: ", node.Bucket)
	}
	currentNodeFileDownloadPath := "../files/" + node.Name + "/chord_storage/"
	filepath := currentNodeFileDownloadPath + f.Name
	// Create the file on file path and store content
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Println("Create file failed")
		return false
	}
	defer file.Close()
	_, err = file.Write(f.Content)
	if err != nil {
		fmt.Println("Write file failed")
		return false
	}
	// Store the file in the file download folder
	return true
}

func (node *Node) storeLocalFile(f FileRPC) bool {
	// Store the file in the bucket
	// Return true if success, false if failed
	// Append the file to the bucket
	f.Id.Mod(f.Id, hashMod)
	currentNodeFileDownloadPath := "../files/" + node.Name + "/file_download/"
	filepath := currentNodeFileDownloadPath + f.Name
	// Create the file on file path and store content
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Println("Create file failed")
		return false
	}
	defer file.Close()
	_, err = file.Write(f.Content)
	if err != nil {
		fmt.Println("Write file failed")
		return false
	}
	// Store the file in the file download folder
	return true
}

type StoreFileRPCReply struct {
	Success bool
	Err     error
	Backup  bool
}

func (node *Node) StoreFileRPC(f FileRPC, reply *StoreFileRPCReply) error {
	fmt.Println("-------------- Invoke StoreFileRPC function ------------")
	reply.Success = node.storeChordFile(f, reply.Backup)
	if !reply.Success {
		reply.Err = errors.New("store file failed")
	} else {
		reply.Err = nil
	}
	return nil
}

func (node *Node) GetFileRPC(f FileRPC, reply *FileRPC) error {
	fmt.Println("-------------- Invoke GetFileRPC function ------------")
	// Get the file from the bucket
	// Return the file if success, return error if failed
	f.Id.Mod(f.Id, hashMod)
	fmt.Println("Get file id: ", f.Id)
	var fileName string
	var ok bool
	// iterate the bucket to find the file
	for key, value := range node.Bucket {
		if key.Cmp(f.Id) == 0 {
			fileName = value
			ok = true
			break
		}
	}
	fmt.Println("Get file status: ", fileName, " ", ok)
	if !ok {
		// Print bucket
		fmt.Println("Bucket: ", node.Bucket)
		return errors.New("file not found")
	}

	// Read the file from the file download folder
	currentNodeFileDownloadPath := "../files/" + node.Name + "/chord_storage/"
	filepath := currentNodeFileDownloadPath + fileName
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	// Return the file
	reply.Id = f.Id
	reply.Name = fileName
	reply.Content = fileContent
	return nil
}

func (node *Node) encryptFile(content []byte) []byte {
	// Encrypt the file
	// Return the encrypted file
	// Encrypt the file content
	publicKey := node.PublicKey
	encryptedContent, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, content)
	if err != nil {
		fmt.Println("Encrypt file failed")
		return nil
	}
	return encryptedContent
}

func (node *Node) decryptFile(content []byte) []byte {
	// Decrypt the file
	// Return the decrypted file
	// Decrypt the file content
	privateKey := node.PrivateKey
	decryptedContent, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, content)
	if err != nil {
		fmt.Println("Decrypt file failed")
		return nil
	}
	return decryptedContent
}

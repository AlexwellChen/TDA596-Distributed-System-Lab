package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/rpc/jsonrpc"
	"os"
	"strings"
)

/*
------------------------------------------------------------

	Stabilizing Functions Below	By:wang

--------------------------------------------------------------
*/

// verifies n’s immediate
func (node *Node) stablize() error {
	// fmt.Println("***************** Invoke stablize function *****************")

	// First request the successor list of your successor[0]
	var getSuccessorListRPCReply GetSuccessorListRPCReply
	err := ChordCall(node.Successors[0], "Node.GetSuccessorListRPC", struct{}{}, &getSuccessorListRPCReply)
	successors := getSuccessorListRPCReply.SuccessorList
	if err == nil {
		for i := 0; i < len(successors)-1; i++ {
			node.Successors[i+1] = successors[i]
		}
	} else {
		fmt.Println("GetSuccessorList failed")
		if node.Successors[0] == "" {
			// No successor, use self as successor
			fmt.Println("Node Successor[0] is empty -> use self as successor")
			node.Successors[0] = node.Address
		} else {
			// Successor[0] might be dead, remove it from the list, and shift the list
			for i := 0; i < len(node.Successors); i++ {
				if i == len(node.Successors)-1 {
					node.Successors[i] = ""
				} else {
					node.Successors[i] = node.Successors[i+1]
				}
			}
		}
	}

	var getPredecessorRPCReply GetPredecessorRPCReply
	err = ChordCall(node.Successors[0], "Node.GetPredecessorRPC", struct{}{}, &getPredecessorRPCReply)
	if err == nil {
		// Get successor's name
		var successorName string
		var getSuccessorNameRPCReply GetNameRPCReply
		err = ChordCall(node.Successors[0], "Node.GetNameRPC", "", &getSuccessorNameRPCReply)
		if err != nil {
			fmt.Println("Get successor[0] name failed")
			return err
		}
		successorName = getSuccessorNameRPCReply.Name

		// Get predecessor's name
		predecessorAddr := getPredecessorRPCReply.PredecessorAddress
		var getNameReply GetNameRPCReply
		err = ChordCall(predecessorAddr, "Node.GetNameRPC", "", &getNameReply)
		if err != nil {
			fmt.Println("Get predecessor name failed: ", err)
			return err
		}
		predecessorName := getNameReply.Name
		nodeId := strHash(string(node.Name))
		nodeId.Mod(nodeId, hashMod)
		predecessorId := strHash(string(predecessorName))
		predecessorId.Mod(predecessorId, hashMod)
		successorId := strHash(string(successorName))
		successorId.Mod(successorId, hashMod)
		if predecessorAddr != "" && between(nodeId,
			predecessorId, successorId, false) {
			node.Successors[0] = predecessorAddr
		}
	}
	ChordCall(node.Successors[0], "Node.NotifyRPC", node.Address, &NotifyRPCReply{})

	// fmt.Println("------------DO COPY NODE BUCKET TO SUCCESSOR[0]------------")
	// First empty successor's backup
	deleteSuccessorBackupRPCReply := DeleteSuccessorBackupRPCReply{}
	err = ChordCall(node.Successors[0], "Node.DeleteSuccessorBackupRPC", struct{}{}, &deleteSuccessorBackupRPCReply)
	if err != nil {
		fmt.Println("empty successor backup failed")
		return err
	}

	// If only one node in the network, do not copy backup
	if node.Successors[0] == node.Address {
		return nil
	}
	lastValue := ""
	// Iterate through node's bucket, copy file to successor[0]'s backup
	for k, v := range node.Bucket {
		newFile := FileRPC{}
		newFile.Name = v
		newFile.Id = k
		// check loop
		if v == lastValue {
			break
		}
		if v != "" {
			lastValue = v
			filepath := "../files/" + node.Name + "/chord_storage/" + v
			file, err := os.Open(filepath)
			if err != nil {
				fmt.Println("Copy to backup: open file failed: ", err)
				return err
			}
			defer file.Close()
			newFile.Content, err = ioutil.ReadAll(file)
			if err != nil {
				fmt.Println("Copy to backup: read file failed: ", err)
				return err
			} else {
				reply := new(SuccessorStoreFileRPCReply)
				err = ChordCall(node.Successors[0], "Node.SuccessorStoreFileRPC", newFile, &reply)
				if reply.Err != nil && err != nil {
					fmt.Println("Copy to backup: store file failed: ", reply.Err, " and ", err)
				}
			}
		}
	}

	// Clean redundant files
	node.cleanRedundantFile()
	return nil
}

// check whether predecessor has failed
func (node *Node) checkPredecessor() error {
	// fmt.Println("************* Invoke checkPredecessor function **************")
	pred := node.Predecessor
	if pred != "" {
		//check connection
		ip := strings.Split(string(pred), ":")[0]
		port := strings.Split(string(pred), ":")[1]

		ip = NAT(ip) // Transalate the internal ip address to public ip address (if external ip is used)

		predAddr := ip + ":" + port
		_, err := jsonrpc.Dial("tcp", predAddr)
		if err != nil {
			fmt.Printf("Predecessor %s has failed\n", string(pred))
			node.Predecessor = ""
			// fmt.Println("------------DO COPY BUCKUP TO BUCKET------------")
			for k, v := range node.Backup {
				if v != "" {
					node.Bucket[k] = v
				}
			}

		}
	}
	return nil
}

// calculate (n + 2^(k-1) ) mod 2^m
func (node *Node) fingerEntry(fingerentry int) *big.Int {
	//fmt.Println("************** Invoke fingerEntry function ******************")
	// id = n (node n identifier)
	id := node.Identifier
	two := big.NewInt(2)
	// 2^(k-1) here use [0,m-1], so k-1 = fingerentry
	exponent := big.NewInt(int64(fingerentry) - 1)
	//2^(k-1)
	fingerEntry := new(big.Int).Exp(two, exponent, nil)
	// n + 2^(k-1)
	sum := new(big.Int).Add(id, fingerEntry)
	// (n + 2^(k-1) ) mod 2^m , 1 <= k <= m
	return new(big.Int).Mod(sum, hashMod)
}

// refreshes finger table entries, next stores the index of the next finger to fix
func (node *Node) fixFingers() error {
	// fmt.Println("*************** Invoke fixfinger function ***************")
	// Lock node.next

	// node.mutex.Lock()
	node.next = node.next + 1
	// node.mutex.Unlock()

	if node.next > fingerTableSize {
		// node.mutex.Lock()
		node.next = 1
		// node.mutex.Unlock()

	}
	id := node.fingerEntry(node.next)
	//find successor of id
	result := FindSuccessorRPCReply{}
	err := ChordCall(node.Address, "Node.FindSuccessorRPC", id, &result)
	if result.Found == false {
		fmt.Println("FindSuccessorRPC failed:", result)
	}
	if err != nil {
		fmt.Println("Find successor failed")
		return err
	}
	// Get successor's name
	var getSuccessorNameRPCReply GetNameRPCReply
	err = ChordCall(result.SuccessorAddress, "Node.GetNameRPC", "", &getSuccessorNameRPCReply)
	if err != nil {
		fmt.Println("node.Next: ", node.next)
		fmt.Println("Fix finger get successor name failed")
		return err
	}
	node.FingerTable[node.next].Id = id.Bytes()
	if node.FingerTable[node.next].Address != result.SuccessorAddress && result.SuccessorAddress != "" {
		fmt.Println("FingerTable[", node.next, "] = ", getSuccessorNameRPCReply.Name)
		node.FingerTable[node.next].Address = result.SuccessorAddress
	}
	//optimization, update other finger table entries use the first successor
	for {
		// node.mutex.Lock()
		node.next = node.next + 1
		// node.mutex.Unlock()

		if node.next > fingerTableSize {
			// we have updated all entries, set to 0
			// node.mutex.Lock()
			node.next = 0
			// node.mutex.Unlock()
			return nil
		}
		id := node.fingerEntry(node.next)
		var getSuccessorNameRPCReply GetNameRPCReply
		err := ChordCall(result.SuccessorAddress, "Node.GetNameRPC", "", &getSuccessorNameRPCReply)
		if err != nil {
			fmt.Println("Get successor name failed")
			return err
		}
		successorName := getSuccessorNameRPCReply.Name
		successorId := strHash(string(successorName))
		successorId.Mod(successorId, hashMod)
		if between(node.Identifier, id, successorId, false) && result.SuccessorAddress != "" {
			if node.FingerTable[node.next].Address != result.SuccessorAddress && result.SuccessorAddress != "" {
				node.FingerTable[node.next].Id = id.Bytes()
				node.FingerTable[node.next].Address = result.SuccessorAddress
				fmt.Println("FingerTable[", node.next, "] = ", result.SuccessorAddress)
			}
		} else {
			// node.mutex.Lock()
			node.next--
			// node.mutex.Unlock()
			return nil
		}
	}
}

/*------------------------------------------------------------*/
/*                    RPC functions Below                     */
/*------------------------------------------------------------*/

// -------------------------- NotifyRPC ----------------------------
type NotifyRPCReply struct {
	Success bool
	Err     error
}

// 'address' thinks it might be our predecessor
func (node *Node) notify(address NodeAddress) (bool, error) {
	// fmt.Println("***************** Invoke notify function ********************")
	// if (predecessor is nil or n' ∈ (predecessor, n))
	// Get predecessor name
	if node.Predecessor != "" {
		predcessorName := ""
		var getPredecessorNameRPCReply GetNameRPCReply
		err := ChordCall(node.Predecessor, "Node.GetNameRPC", "", &getPredecessorNameRPCReply)
		if err != nil {
			fmt.Println("Get predecessor name failed: ", err)
			return false, err
		}

		// Get address name
		addressName := ""
		var getAddressNameRPCReply GetNameRPCReply
		err = ChordCall(address, "Node.GetNameRPC", "", &getAddressNameRPCReply)
		if err != nil {
			fmt.Println("Get address name failed: ", err)
			return false, err
		}

		predcessorName = getPredecessorNameRPCReply.Name
		predcessorId := strHash(predcessorName)
		predcessorId.Mod(predcessorId, hashMod)

		addressName = getAddressNameRPCReply.Name
		addressId := strHash(addressName)
		addressId.Mod(addressId, hashMod)

		nodeId := node.Identifier
		if between(predcessorId, addressId, nodeId, false) {
			//predecessor = n'
			node.Predecessor = address
			fmt.Println(node.Name, "'s Predecessor is set to", address)
			return true, nil
		} else {
			return false, nil
		}
	} else {
		node.Predecessor = address
		fmt.Println(node.Name, "'s Predecessor is set to", address)
		return true, nil
	}

}

func (node *Node) moveFiles(addr NodeAddress) {
	// Parse local bucket
	// Get address name
	addressName := ""
	var getAddressNameRPCReply GetNameRPCReply
	err := ChordCall(addr, "Node.GetNameRPC", "", &getAddressNameRPCReply)
	if err != nil {
		fmt.Println("Get address name failed: ", err)
		return
	}
	addressName = getAddressNameRPCReply.Name
	addressId := strHash(addressName)
	addressId.Mod(addressId, hashMod)

	// Iterate through local bucket
	for key, element := range node.Bucket {
		fileId := key
		fileName := element
		filepath := "../files/" + node.Name + "/chord_storage/" + fileName
		file, err := os.Open(filepath)
		if err != nil {
			fmt.Println("Cannot open the file")
		}
		defer file.Close()
		// Init new file struct and put content into it
		newFile := FileRPC{}
		newFile.Name = fileName
		newFile.Content, err = ioutil.ReadAll(file)
		if err != nil {
			fmt.Println("Cannot read the file")
		}
		newFile.Id = key
		if between(fileId, addressId, node.Identifier, true) && fileId.Cmp(node.Identifier) != 0 || fileId.Cmp(addressId) == 0 { // if file shouldn't be in this node or file should be in addressId node
			//move file to new node
			var moveFileRPCReply StoreFileRPCReply
			moveFileRPCReply.Backup = false
			// Move local file to new predecessor using storeFile function
			err := ChordCall(addr, "Node.StoreFileRPC", newFile, &moveFileRPCReply)
			if err != nil {
				fmt.Println("Move file failed: ", err)
			}
			//delete file from local bucket
			fmt.Println("FileId: ", fileId)
			fmt.Println("addressId: ", addressId)
			fmt.Println("Address is: ", addr)
			fmt.Println("node.Identifier: ", node.Identifier)
			delete(node.Bucket, key)
			// delete file from local directory
			err = os.Remove(filepath)
			if err != nil {
				fmt.Println("Cannot delete the file")
			}
		}
	}
}

func (node *Node) NotifyRPC(address NodeAddress, reply *NotifyRPCReply) error {
	// fmt.Println("---------------- Invoke NotifyRPC function ------------------")
	if node.Successors[0] != node.Address {
		node.moveFiles(address)
	}
	reply.Success, reply.Err = node.notify(address)
	return nil
}

// -------------------------- GetSuccessorListRPC ----------------------------
type GetSuccessorListRPCReply struct {
	SuccessorList []NodeAddress
}

// get node's successorList
func (node *Node) getSuccessorList() []NodeAddress {
	// fmt.Println("************* Invoke getSuccessorList function **************")
	return node.Successors[:]
}

func (node *Node) GetSuccessorListRPC(none *struct{}, reply *GetSuccessorListRPCReply) error {
	// fmt.Println("------------ Invoke GetSuccessorListRPC function ------------")
	reply.SuccessorList = node.getSuccessorList()
	return nil
}

type GetPredecessorRPCReply struct {
	PredecessorAddress NodeAddress
}

// get node's predecessor
func (node *Node) getPredecessor() NodeAddress {
	// fmt.Println("************** Invoke getPredecessor function ***************")
	return node.Predecessor
}
func (node *Node) GetPredecessorRPC(none *struct{}, reply *GetPredecessorRPCReply) error {
	// fmt.Println("------------- Invoke GetPredecessorRPC function -------------")
	reply.PredecessorAddress = node.getPredecessor()
	if reply.PredecessorAddress == "" {
		return errors.New("predecessor is empty")
	} else {
		return nil
	}
}

type DeleteSuccessorBackupRPCReply struct {
	Success bool
}

func (node *Node) deleteSuccessorBackup() bool {
	// Iterate through successor's backup and delete all files
	for key, _ := range node.Backup {

		// filepath := "../files/" + node.Name + "/chord_storage/" + fileName
		// err := os.Remove(filepath)
		// if err != nil {
		// 	fmt.Println("Cannot delete file: ", fileName)
		// }
		delete(node.Backup, key)
	}
	return true
}

func (node *Node) DeleteSuccessorBackupRPC(none *struct{}, reply *DeleteSuccessorBackupRPCReply) error {
	// fmt.Println("------------- Invoke DeleteSuccessorBackupRPC function -------------")
	reply.Success = node.deleteSuccessorBackup()
	return nil
}

func (node *Node) successorStoreFile(f FileRPC) bool {
	//fmt.Println("************** Invoke successorStoreFile function ***************")
	// Store file in successor's backup
	f.Id.Mod(f.Id, hashMod)
	node.Backup[f.Id] = f.Name
	// Write file to local
	filepath := "../files/" + node.Name + "/chord_storage/" + f.Name
	file, err := os.Create(filepath)
	if err != nil {
		fmt.Println("Cannot create backup file")
	}
	defer file.Close()
	_, err = file.Write(f.Content)
	if err != nil {
		fmt.Println("Cannot write backup file")
	}
	// fmt.Println("Stab Backup: ", node.Backup)
	return true
}

type SuccessorStoreFileRPCReply struct {
	Success bool
	Err     error
}

func (node *Node) SuccessorStoreFileRPC(f FileRPC, reply *SuccessorStoreFileRPCReply) error {
	// fmt.Println("------------- Invoke SuccessorStoreFileRPC function -------------")
	reply.Success = node.successorStoreFile(f)
	if !reply.Success {
		reply.Err = errors.New("store file failed")
	} else {
		reply.Err = nil
	}
	return nil
}

func (node *Node) cleanRedundantFile() {
	// Read local chord_storage directory
	files, err := ioutil.ReadDir("../files/" + node.Name + "/chord_storage")
	if err != nil {
		fmt.Println("Cannot read chord_storage directory")
	}
	// Iterate through local chord_storage directory
	for _, file := range files {
		// Get file name
		fileName := file.Name()
		// Get file id
		fileId := sha1.New()
		fileId.Write([]byte(fileName))
		key := new(big.Int)
		key.SetBytes(fileId.Sum(nil))
		key.Mod(key, hashMod)
		// Check if file is in local bucket and local backup
		inBucket := false
		inBackup := false
		for k, _ := range node.Bucket {
			if k.Cmp(key) == 0 {
				inBucket = true
			}
		}
		for k, _ := range node.Backup {
			if k.Cmp(key) == 0 {
				inBackup = true
			}
		}
		if !inBucket && !inBackup {
			// Delete file from local chord_storage directory
			filepath := "../files/" + node.Name + "/chord_storage/" + fileName
			err = os.Remove(filepath)
			if err != nil {
				fmt.Println("Cannot delete file: ", fileName)
			}

		}
	}
}

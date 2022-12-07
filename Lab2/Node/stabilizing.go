package main

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	//"net/rpc"
	"io/ioutil"
	"net/rpc/jsonrpc"
	"os"
)

/*
------------------------------------------------------------

	Stabilizing Functions Below	By:wang

--------------------------------------------------------------
*/

// verifies n’s immediate
func (node *Node) stablize() error {
	//??Truncate the list if needed so it is not too long
	//??(measuring it against the maximum length discussed above).
	// fmt.Println("***************** Invoke stablize function *****************")
	//node.Successors[0] = node.getSuccessor()
	var getSuccessorListRPCReply GetSuccessorListRPCReply
	//fmt.Println("node.Successors: ", node.Successors[0])
	err := ChordCall(node.Successors[0], "Node.GetSuccessorListRPC", struct{}{}, &getSuccessorListRPCReply)
	successors := getSuccessorListRPCReply.SuccessorList
	if err == nil {
		for i := 0; i < len(successors)-1; i++ {
			//fmt.Printf("node successors[%d]: successors[%d]: %s\n", i+1, i, successors[i])
			node.Successors[i+1] = successors[i]
			//Todo: check if need to do a loop chordCall for all successors[0]
		}
	} else {
		fmt.Println("GetSuccessorList failed")
		// If there is no such element (the list is empty)
		// set your successor to your own address.
		if node.Successors[0] == "" {
			fmt.Println("Node Successor[0] is empty -> use self as successor")
			node.Successors[0] = node.Address
		} else {
			// Chop the first element off your successors list
			// and set your successor to the next element in the list.
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
			/* fmt.Println(strHash(string(node.Name)),"and",
				strHash(string(predecessorName)), "and",strHash(string(successorName)))
				fmt.Println(node.Identifier)
				fmt.Println(strHash(string(node.Name)).Cmp(strHash(string(predecessorName))))
				fmt.Println(strHash(string(predecessorName)).Cmp(strHash(string(successorName))))
			fmt.Printf("Predecessor %s is between %s and %s\n", predecessorAddr, node.Name, successorName)
			fmt.Printf("Set successor[0] to %s\n", predecessorAddr) */
			node.Successors[0] = predecessorAddr
		}
	}
	// fmt.Println("------------DO COPY NODE BUCKET TO SUCCESSOR[0]------------")
	deleteSuccessorBackupRPCReply := DeleteSuccessorBackupRPCReply{}
	err = ChordCall(node.Successors[0], "Node.DeleteSuccessorBackupRPC", struct{}{}, &deleteSuccessorBackupRPCReply)
	if err != nil {
		fmt.Println("empty successor backup failed")
		return err
	}
	lastValue := ""
	for k, v := range node.Bucket {
		newFile := FileRPC{}
		newFile.Name = v
		newFile.Id = k
		//fmt.Println("lastValue: ", lastValue)
		//fmt.Println("newFile.Name: ", newFile.Name)
		// check loop
		if v == lastValue {
			//fmt.Println("Loop detected, break")
			break
		}
		if v != "" {
			//fmt.Println("lastValue and v: ", lastValue, "and", v)
			lastValue = v
			filepath := "../files/" + node.Name + "/chord_storage/" + v
			file, err := os.Open(filepath)
			if err != nil {
				fmt.Println("Copy to backup: open file failed: ", err)
				return err
			}
			defer file.Close()
			newFile.Content, err = ioutil.ReadAll(file) // check if need?
			if err != nil {
				fmt.Println("Copy to backup: read file failed: ", err)
				return err
			} else {
				//TODO: check if need encrypt??
				reply := new(SuccessorStoreFileRPCReply)
				err = ChordCall(node.Successors[0], "Node.SuccessorStoreFileRPC", newFile, &reply)
				if reply.Err != nil && err != nil {
					fmt.Println("Copy to backup: store file failed: ", reply.Err, " and ", err)
				}
			}
		}
	}
	var fakeReply NotifyRPCReply
	ChordCall(node.Successors[0], "Node.NotifyRPC", node.Address, &fakeReply)
	/* 	if !fakeReply.Success {
	   		// fmt.Println("Notify failed: ", fakeReply.err)
	   	} else {
	   		// fmt.Println("Notify success")
	   	} */
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
		if ip == getLocalAddress() {
			ip = "localhost"
		}
		/*
		* NAT: ip is internal ip, need to be changed to external ip
		 */
		// wwq's NAT
		if ip == "172.31.21.112" {
			ip = "3.89.241.69"
		}

		// cfz's NAT
		if ip == "192.168.31.236" {
			ip = "95.80.36.91"
		}

		predAddr := ip + ":" + port
		_, err := jsonrpc.Dial("tcp", predAddr)
		//_, err := jsonrpc.Dial("tcp", string(pred))
		//if connection failed, set predecessor to nil
		if err != nil {
			fmt.Printf("Predecessor %s has failed\n", string(pred))
			// Retry 3 times
			success := false
			for i := 0; i < 3; i++ {
				_, err = jsonrpc.Dial("tcp", predAddr)
				if err != nil {
					fmt.Println("Retry ", i+1, " times")
					time.Sleep(1 * time.Second)
				} else {
					success = true
					break
				}
			}
			if !success {
				node.Predecessor = ""
				// fmt.Println("------------DO COPY BUCKUP TO BUCKET------------")
				for k, v := range node.Backup {
					if v != "" {
						node.Bucket[k] = v
					}
				}
			}

		}
		//defer client.Close()
	}
	return nil
}

// calculate (n + 2^(k-1) ) mod 2^m
func (node *Node) fingerEntry(fingerentry int) *big.Int {
	//fmt.Println("************** Invoke fingerEntry function ******************")
	//var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(len(node.FingerTable)-1)), nil)
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
	var mutex sync.Mutex
	mutex.Lock()
	node.next = node.next + 1
	// Unlock node.next
	mutex.Unlock()

	if node.next > fingerTableSize {
		mutex.Lock()
		node.next = 1
		mutex.Unlock()
	}
	id := node.fingerEntry(node.next)
	//find successor of id
	result := FindSuccessorRPCReply{}
	err := ChordCall(node.Address, "Node.FindSuccessorRPC", id, &result)
	if err != nil {
		fmt.Println("Find successor failed")
		return err
	}
	// fmt.Println("FindSuccessorRPC recieve result: ", result)
	//update fingertable(next)
	/* 	if result.found {
		node.FingerTable[node.next].Address = result.SuccessorAddress
		node.FingerTable[node.next].Id = id.Bytes()
	} */
	// // Get successor's name
	var getSuccessorNameRPCReply GetNameRPCReply
	err = ChordCall(result.SuccessorAddress, "Node.GetNameRPC", "", &getSuccessorNameRPCReply)
	if err != nil {
		fmt.Println("node.Next: ", node.next)
		fmt.Println("Fix finger get successor name failed")
		return err
	}
	//fmt.Println("FingerTable[", node.next, "] = ", getSuccessorNameRPCReply.Name)
	node.FingerTable[node.next].Id = id.Bytes()
	if node.FingerTable[node.next].Address != result.SuccessorAddress && result.SuccessorAddress != "" {
		fmt.Println("FingerTable[", node.next, "] = ", getSuccessorNameRPCReply.Name)
		node.FingerTable[node.next].Address = result.SuccessorAddress
	}
	/* 		_, addr := node.findSuccessor(id)
	   		if addr != "" && addr != node.FingerTable[node.next].Address {
	   			node.FingerTable[node.next].Address = addr
	   			node.FingerTable[node.next].Id = id.Bytes()
	   		} */
	//optimization, update other finger table entries use the first successor
	for {
		mutex.Lock()
		node.next = node.next + 1
		mutex.Unlock()
		if node.next > fingerTableSize {
			// we have updated all entries, set to 0
			mutex.Lock()
			node.next = 0
			mutex.Unlock()
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
			mutex.Lock()
			node.next--
			mutex.Unlock()
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
	//if (predecessor is nil or n' ∈ (predecessor, n))
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
		// fmt.Println("predcessorId: ", predcessorId, "addressId: ", addressId, "nodeId: ", nodeId)
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

// TODO: Add return for moveFiles function
func (node *Node) moveFiles(addr NodeAddress) {
	// Parse local bucket
	addressId := strHash(string(addr))
	addressId.Mod(addressId, hashMod)
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
		if between(fileId, addressId, node.Identifier, false) {
			//move file to new node
			var moveFileRPCReply StoreFileRPCReply
			// Move local file to new predecessor using storeFile function
			err := ChordCall(addr, "Node.StoreFileRPC", newFile, &moveFileRPCReply)
			if err != nil {
				fmt.Println("Move file failed: ", err)
			}
			//delete file from local bucket
			delete(node.Bucket, key)
			// delete file from local directory
			err = os.Remove(filepath)
			if err != nil {
				fmt.Println("Cannot delete the file")
			}
		}
	}
}

// TODO: check if the notifyrpc function is correct
func (node *Node) NotifyRPC(address NodeAddress, reply *NotifyRPCReply) error {
	// fmt.Println("---------------- Invoke NotifyRPC function ------------------")
	node.moveFiles(address)
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
	// fmt.Println("************** Invoke deleteSuccessorBackup function ***************")
	// // Delete backup files first
	// backupPath := "../files/" + node.Name + "/chord_storage/"
	// // Read file names from node.Backup
	// for _, fileName := range node.Backup {
	// 	filepath := backupPath + fileName
	// 	err := os.Remove(filepath)
	// 	if err != nil {
	// 		fmt.Println("Cannot delete file: ", fileName)
	// 	}
	// }
	// Clear node.Backup
	node.Backup = make(map[*big.Int]string)
	//fmt.Println("Backup is deleted : ", node.Backup)

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
	//fmt.Println("File", f.Name, "is stored in", node.Name, "'s backup")
	fmt.Println("Stab Backup: ", node.Backup)
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

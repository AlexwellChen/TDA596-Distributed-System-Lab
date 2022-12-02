package main

import (
	"errors"
	"fmt"
	"math/big"
	"net/rpc"
)

/*
------------------------------------------------------------

	Stabilizing Functions Below	By:wang

--------------------------------------------------------------
*/

// verifies n’s immediate
func (node *Node) stabilize() error {
	//Todo: search paper 看看是每个successor都要修改prodecessor还是只修改第一个successor
	//Todo: search paper 看看是不是要fix successorList
	//??Truncate the list if needed so it is not too long
	//??(measuring it against the maximum length discussed above).
	// fmt.Println("***************** Invoke stabilize function *****************")
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
	var getPredecessorRPCRepy GetPredecessorRPCRepy
	err = ChordCall(node.Successors[0], "Node.GetPredecessorRPC", struct{}{}, &getPredecessorRPCRepy)
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
		predecessorAddr := getPredecessorRPCRepy.PredecessorAddress
		var getNameReply GetNameRPCReply
		err = ChordCall(predecessorAddr, "Node.GetNameRPC", "", &getNameReply)
		if err != nil {
			fmt.Println("Get predecessor name failed: ", err)
			return err
		}
		predecessorName := getNameReply.Name
		nodeId:= strHash(string(node.Name))
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
	var fakeReply NotifyRPCReply
	err = ChordCall(node.Successors[0], "Node.NotifyRPC", node.Address, &fakeReply)
	if !fakeReply.Success {
		// fmt.Println("Notify failed: ", fakeReply.err)
	} else {
		// fmt.Println("Notify success")
	}
	return nil
}

// check whether predecessor has failed
func (node *Node) checkPredecessor() error {
	// fmt.Println("************* Invoke checkPredecessor function **************")
	pred := node.Predecessor
	if pred != "" {
		//check connection
		client, err := rpc.Dial("tcp", string(pred))
		//if connection failed, set predecessor to nil
		if err != nil {
			fmt.Printf("Predecessor %s has failed\n", string(pred))
			node.Predecessor = ""
			client.Close()
		} else {
			client.Close()
		}
	}
	return nil
}

// calculate (n + 2^(k-1) ) mod 2^m
func (node *Node) fingerEntry(fingerentry int) *big.Int {
	//Todo: check if use len(node.Address) or fingerTableSize
	// fmt.Println("************** Invoke fingerEntry function ******************")
	// 2^m ? use len(node.Address)
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
	//Todo: search paper check node.next 在到了m的时候是不是要从1开始还是0，以及初始化
	// fmt.Println("*************** Invoke findSuccessor function ***************")
	node.next = node.next + 1
	//use 0 to m-1, init next = -1, then use next+1 to 0
	if node.next > fingerTableSize  {
		node.next = 1
	}
	id := node.fingerEntry(node.next)
	//find successor of id
	result := FindSuccessorRPCReply{}
	err := ChordCall(node.Address, "Node.FindSuccessorRPC", id, &result)
	if err != nil {
		fmt.Println("Find successor failed")
		return err
	}
	//update fingertable(next)
	/* 	if result.found {
		node.FingerTable[node.next].Address = result.SuccessorAddress
		node.FingerTable[node.next].Id = id.Bytes()
	} */
	// // Get successor's name
	var getSuccessorNameRPCReply GetNameRPCReply
	err = ChordCall(result.SuccessorAddress, "Node.GetNameRPC", "", &getSuccessorNameRPCReply)
	if err != nil {
		fmt.Println("Get successor name failed")
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
		node.next = node.next + 1
		if node.next > fingerTableSize {
			// we have updated all entries, set to -1
			node.next = 0
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
			node.next--
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
	err     error
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

// TODO: check if the notifyrpc function is correct
func (node *Node) NotifyRPC(address NodeAddress, reply *NotifyRPCReply) error {
	// fmt.Println("---------------- Invoke NotifyRPC function ------------------")
	reply.Success, reply.err = node.notify(address)
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

type GetPredecessorRPCRepy struct {
	PredecessorAddress NodeAddress
}

// get node's predecessor
func (node *Node) getPredecessor() NodeAddress {
	// fmt.Println("************** Invoke getPredecessor function ***************")
	return node.Predecessor
}
func (node *Node) GetPredecessorRPC(none *struct{}, reply *GetPredecessorRPCRepy) error {
	// fmt.Println("------------- Invoke GetPredecessorRPC function -------------")
	reply.PredecessorAddress = node.getPredecessor()
	if reply.PredecessorAddress == "" {
		return errors.New("predecessor is empty")
	} else {
		return nil
	}
}

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
	fmt.Println("***************** Invoke stabilize function *****************")
	//node.Successors[0] = node.getSuccessor()
	var getSuccessorListRPCReply GetSuccessorListRPCReply
	err := ChordCall(node.Successors[0], "Node.GetSuccessorListRPC", struct{}{}, &getSuccessorListRPCReply)
	successors := getSuccessorListRPCReply.SuccessorList
	if err == nil {
		for i := 0; i < len(successors)-1; i++ {
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
		successorName := ""
		err = ChordCall(node.Successors[0], "Node.GetNameRPC", "", &successorName)
		if err != nil {
			fmt.Println("Get successor[0] name failed")
			return err
		}
		predecessorAddr := getPredecessorRPCRepy.PredecessorAddress
		var getNameReply GetNameRPCReply
		err = ChordCall(predecessorAddr, "Node.GetNameRPC", "", &getNameReply)
		if err != nil {
			fmt.Println("Get predecessor name failed")
			return err
		}
		predecessorName := getNameReply.Name
		if predecessorAddr != "" && between(strHash(string(node.Name)),
			strHash(string(predecessorName)), strHash(string(successorName)), false) {
			node.Successors[0] = predecessorAddr
		}
	}
	var fakeReply NotifyRPCReply
	err = ChordCall(node.Successors[0], "Node.NotifyRPC", node.Address, &fakeReply)
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
		client, err := rpc.DialHTTP("tcp", string(pred))
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

// calculate (n + 2^(k-1) ) mod 2^m
func (node *Node) fingerEntry(fingerentry int) *big.Int {
	//Todo: check if use len(node.Address) or fingerTableSize
	fmt.Println("************** Invoke fingerEntry function ******************")
	// 2^m ? use len(node.Address)
	//var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(len(node.FingerTable)-1)), nil)
	// id = n (node n identifier)
	id := node.Identifier
	two := big.NewInt(2)
	// 2^(k-1) here use [0,m-1], so k-1 = fingerentry
	exponent := big.NewInt(int64(fingerentry))
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
	fmt.Println("*************** Invoke findSuccessor function ***************")
	node.next = node.next + 1
	//use 0 to m-1, init next = -1, then use next+1 to 0
	if node.next > fingerTableSize-1 {
		node.next = 0
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
	node.FingerTable[node.next].Address = result.SuccessorAddress
	node.FingerTable[node.next].Id = id.Bytes()
	/* 		_, addr := node.findSuccessor(id)
	   		if addr != "" && addr != node.FingerTable[node.next].Address {
	   			node.FingerTable[node.next].Address = addr
	   			node.FingerTable[node.next].Id = id.Bytes()
	   		} */
	//optimization, update other finger table entries use the first successor
	for {
		node.next = node.next + 1
		if node.next > fingerTableSize-1 {
			node.next = 0
			return nil
		}
		id := node.fingerEntry(node.next)
		var getNameRPCReply GetNameRPCReply
		err := ChordCall(result.SuccessorAddress, "Node.GetNameRPC", "", &getNameRPCReply)
		if err != nil {
			fmt.Println("Get successor name failed")
			return err
		}
		if between(strHash(string(node.Name)), id, strHash(string(getNameRPCReply.Name)), false) && result.SuccessorAddress != "" {
			node.FingerTable[node.next].Id = id.Bytes()
			node.FingerTable[node.next].Address = result.SuccessorAddress
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
}

// 'address' thinks it might be our predecessor
func (node *Node) notify(address NodeAddress) *bool {
	fmt.Println("***************** Invoke notify function ********************")
	//if (predecessor is nil or n' ∈ (predecessor, n))
	predcessorName := ""
	err := ChordCall(node.Predecessor, "Node.GetNameRPC", "", &predcessorName)
	if err != nil {
		fmt.Println("Get predecessor name failed")
		return nil
	}
	addressName := ""
	err = ChordCall(address, "Node.GetNameRPC", "", &addressName)
	if err != nil {
		fmt.Println("Get address name failed")
		return nil
	}
	if node.Predecessor == "" ||
		between(strHash(string(predcessorName)),
			strHash(string(addressName)), strHash(string(node.Name)), false) {
		//predecessor = n'
		node.Predecessor = address
	}
	flag := true
	return &flag
}

// TODO: check if the notifyrpc function is correct
func (node *Node) NotifyRPC(address *NodeAddress, reply *NotifyRPCReply) error {
	fmt.Println("---------------- Invoke NotifyRPC function ------------------")
	reply.Success = *node.notify(*address)
	return nil
}

// -------------------------- GetSuccessorListRPC ----------------------------
type GetSuccessorListRPCReply struct {
	SuccessorList []NodeAddress
}

// get node's successorList
func (node *Node) getSuccessorList() []NodeAddress {
	fmt.Println("************* Invoke getSuccessorList function **************")
	return node.Successors[:]
}

func (node *Node) GetSuccessorListRPC(none *struct{}, reply *GetSuccessorListRPCReply) error {
	fmt.Println("------------ Invoke GetSuccessorListRPC function ------------")
	reply.SuccessorList = node.getSuccessorList()
	return nil
}

// -------------------------- GetPredecessorRPC ----------------------------
type GetPredecessorRPCRepy struct {
	PredecessorAddress NodeAddress
}

// get node's predecessor
func (node *Node) getPredecessor() NodeAddress {
	fmt.Println("************** Invoke getPredecessor function ***************")
	return node.Predecessor
}
func (node *Node) GetPredecessorRPC(none *struct{}, reply *GetPredecessorRPCRepy) error {
	fmt.Println("------------- Invoke GetPredecessorRPC function -------------")
	reply.PredecessorAddress = node.getPredecessor()
	if reply.PredecessorAddress == "" {
		return errors.New("predecessor is empty")
	} else {
		return nil
	}
}

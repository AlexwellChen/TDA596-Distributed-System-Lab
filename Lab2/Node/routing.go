package main

import (
	"fmt"
	"math/big"
)

/*------------------------------------------------------------*/
/*                  Routing Functions By: Alexwell            */
/*------------------------------------------------------------*/

// Local use function
func (node *Node) closePrecedingNode(requestID *big.Int) NodeAddress {
	// fmt.Println("************ Invoke closePrecedingNode function ************")
	fingerTableSize := len(node.FingerTable)
	for i := fingerTableSize - 1; i >= 1; i-- {
		var reply GetNameRPCReply
		ChordCall(node.FingerTable[i].Address, "Node.GetNameRPC", "", &reply)
		if between(node.Identifier, strHash(string(reply.Name)), requestID, true) {
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
/*                    RPC functions Below                     */
/*------------------------------------------------------------*/

// -------------------------- FindSuccessorRPCReply ----------------------------------//
type FindSuccessorRPCReply struct {
	found            bool
	SuccessorAddress NodeAddress
}

// Local use function
func (node *Node) findSuccessor(requestID *big.Int) (bool, NodeAddress) {
	// fmt.Println("*************** Invoke findSuccessor function ***************")
	successorName := ""
	var getNameRPCReply GetNameRPCReply
	ChordCall(node.Successors[0], "Node.GetNameRPC", "", &getNameRPCReply)
	successorName = getNameRPCReply.Name
	if between(node.Identifier, requestID, strHash(string(successorName)), true) {
		// fmt.Println("Successor is: ", node.Successors[0])
		return true, node.Successors[0]
	} else {
		successorAddr := node.closePrecedingNode(requestID)
		// fmt.Println("Close Preceding Node is: ", successorAddr)
		return false, successorAddr
	}
}

/*
* @description: RPC method Packaging for findSuccessor, running on remote node
* @param: 		requestID: the client address or file name to be searched
* @return: 		found: whether the key is found
* 				successor: the successor of the key
 */
func (node *Node) FindSuccessorRPC(requestID *big.Int, reply *FindSuccessorRPCReply) error {
	// fmt.Println("-------------- Invoke FindSuccessorRPC function ------------")
	reply.found, reply.SuccessorAddress = node.findSuccessor(requestID)
	return nil
}

// -------------------------- GetNameRPC ----------------------------------//
type GetNameRPCReply struct {
	Name string
}

// Get target node name/id

func (node *Node) getName() string {
	return node.Name
}

/*
* @description: RPC method Packaging for getName, running on remote node
* @param: 		fakeRequest: not used
* @return: 		reply: the name of the node, use for hash(name) to get the node id
 */
func (node *Node) GetNameRPC(fakeRequest string, reply *GetNameRPCReply) error {
	reply.Name = node.getName()
	return nil
}

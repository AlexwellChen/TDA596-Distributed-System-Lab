package main

import (
	"fmt"
	"math/big"
)

/*------------------------------------------------------------*/
/*                  Routing Functions By: Alexwell            */
/*------------------------------------------------------------*/

// Local use functionFindSuccessorRPC
func (node *Node) closePrecedingNode(requestID *big.Int) NodeAddress {
	// fmt.Println("************ Invoke closePrecedingNode function ************")
	fingerTableSize := len(node.FingerTable)
	for i := fingerTableSize - 1; i >= 1; i-- {
		var reply GetNameRPCReply
		ChordCall(node.FingerTable[i].Address, "Node.GetNameRPC", "", &reply)
		fingerId := strHash(reply.Name)
		fingerId.Mod(fingerId, hashMod)
		if between(node.Identifier, fingerId, requestID, false) {
			return node.FingerTable[i].Address
		}
	}
	return node.Successors[0]
}

// Local use function
// Lookup
func find(id *big.Int, startNode NodeAddress) NodeAddress {
	fmt.Println("****************** Invoke find function *********************")
	fmt.Println("The id to be found is: ", id.Mod(id, hashMod))
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
		fmt.Println("The result of find is: ", result)
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
func (node *Node) findSuccessor(requestID *big.Int) (res FindSuccessorRPCReply) {
	// fmt.Println("*************** Invoke findSuccessor function ***************")
	successorName := ""
	var getNameRPCReply GetNameRPCReply
	ChordCall(node.Successors[0], "Node.GetNameRPC", "", &getNameRPCReply)
	successorName = getNameRPCReply.Name
	successorId := strHash(successorName)
	successorId.Mod(successorId, hashMod)
	requestID.Mod(requestID, hashMod)

	if between(node.Identifier, requestID, successorId, true) {
		if requestID.String() == "29" {
			fmt.Println("Between range is ", node.Identifier, requestID, successorId)
			fmt.Println("Successor is: ", node.Successors[0])
		}
		res = FindSuccessorRPCReply{true, node.Successors[0]}
	} else {
		successorAddr := node.closePrecedingNode(requestID)
		// fmt.Println("Close Preceding Node is: ", successorAddr)
		// Get the successor of the close preceding node
		var findSuccessorRPCReply FindSuccessorRPCReply
		ChordCall(successorAddr, "Node.FindSuccessorRPC", requestID, &findSuccessorRPCReply)
		res = FindSuccessorRPCReply{findSuccessorRPCReply.found, findSuccessorRPCReply.SuccessorAddress}

	}
	return res
}

/*
* @description: RPC method Packaging for findSuccessor, running on remote node
* @param: 		requestID: the client address or file name to be searched
* @return: 		found: whether the key is found
* 				successor: the successor of the key
 */
func (node *Node) FindSuccessorRPC(requestID *big.Int, reply *FindSuccessorRPCReply) error {
	// fmt.Println("-------------- Invoke FindSuccessorRPC function ------------")
	*reply = node.findSuccessor(requestID)
	if !reply.found {
		// found == false means we are using closePrecedingNode to find the successor
		// fmt.Println("find fail in FindSuccessorRPC")
	}
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

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
		err := ChordCall(node.FingerTable[i].Address, "Node.GetNameRPC", "", &reply)
		if err != nil {
			fmt.Println("Error in closePrecedingNode function: ", err)
			continue
		}
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
			fmt.Println("Error in find function: ", err)
		}
		found = result.Found
		// fmt.Println("The result of find is: ", result)
		// found = result.found
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
	Found            bool
	SuccessorAddress NodeAddress
}

// Local use function
func (node *Node) FindSuccessorRPC(requestID *big.Int, reply *FindSuccessorRPCReply) error {
	// fmt.Println("*************** Invoke findSuccessor function ***************")
	successorName := ""
	var getNameRPCReply GetNameRPCReply
	err := ChordCall(node.Successors[0], "Node.GetNameRPC", "", &getNameRPCReply)
	if err != nil {
		fmt.Println("Error in findSuccessorRPC: ", err)
		reply.Found = false
		reply.SuccessorAddress = "Error in findSuccessorRPC at " + node.Successors[0]
		return nil
	}
	successorName = getNameRPCReply.Name
	successorId := strHash(successorName)
	successorId.Mod(successorId, hashMod)
	requestID.Mod(requestID, hashMod)

	if between(node.Identifier, requestID, successorId, true) {
		// if requestID.String() == "29" {
		// 	fmt.Println("Between rangeis ", node.Identifier, requestID, successorId)
		// 	fmt.Println("Successor is: ", node.Successors[0])
		// }
		reply.Found = true
		reply.SuccessorAddress = node.Successors[0]
		// return &res
	} else {

		successorAddr := node.closePrecedingNode(requestID)
		// if requestID.String() == "15" {
		// 	fmt.Println("Find closest preceding node at ", node.Address, " for ", requestID, " is ", successorAddr, "")
		// }
		// Get the successor of the close preceding node
		var findSuccessorRPCReply FindSuccessorRPCReply
		err := ChordCall(successorAddr, "Node.FindSuccessorRPC", requestID, &findSuccessorRPCReply)
		if err != nil {
			fmt.Println("Error in findSuccessorRPC: ", err)
			reply.Found = false
			reply.SuccessorAddress = "Error in findSuccessorRPC at " + successorAddr
		} else {
			reply.Found = true
			reply.SuccessorAddress = findSuccessorRPCReply.SuccessorAddress
		}
		// return &res
	}
	return nil
}

/*
* @description: RPC method Packaging for findSuccessor, running on remote node
* @param: 		requestID: the client address or file name to be searched
* @return: 		found: whether the key is found
* 				successor: the successor of the key
 */

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

func NAT(addr string) string {
	/*
	* NAT: ip is internal ip, need to be changed to external ip
	 */
	new_addr := addr
	getLocalAddress_res := getLocalAddress()
	// fmt.Println("getLocalAddress_res: ", getLocalAddress_res)
	// fmt.Println("Input addr: ", addr)
	if addr == getLocalAddress_res {
		new_addr = "localhost"
	}

	// wwq's NAT
	if addr == "172.31.21.112" {
		new_addr = "54.145.27.145"
	}

	// cfz's NAT
	if addr == "192.168.31.236" {
		new_addr = "95.80.36.91"
	}

	// jetson's NAT
	if addr == "192.168.31.153" {
		new_addr = "95.80.36.91"
	}
	if addr == "172.25.90.182" {
		new_addr = "50.93.222.140"
	}	

	return new_addr
}

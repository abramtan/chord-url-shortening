package node

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/rpc"
	"strconv"
)

type Hash uint64 //[32]byte

type IPAddress string

/*
Function to generate the hash of the the input IP address
*/
func (ip IPAddress) GenerateHash() Hash { // TODO: EST_NO_OF_MACHINES should be the max num of machines our chord can take
	EST_NO_OF_MACHINES := 11
	MAX_RING_SIZE := int64(math.Pow(2, float64(EST_NO_OF_MACHINES)))

	data := []byte(ip)
	id := sha256.Sum256(data)
	unmoddedID := new(big.Int).SetBytes(id[:8])
	modValue := new(big.Int).SetInt64(MAX_RING_SIZE)
	moddedID := new(big.Int).Mod(unmoddedID, modValue)
	return Hash(moddedID.Int64())
}

type Node struct {
	// id          uint64
	ipAddress   IPAddress
	fingerTable []IPAddress
	successor   IPAddress
	predecessor IPAddress
	// fixFingerNext int
}

func (n *Node) GetIPAddress() IPAddress {
	return n.ipAddress
}
func (n *Node) GetFingerTable() *[]IPAddress {
	return &n.fingerTable
}

var nodeCount int

func InitNode(nodeAr *[]*Node) (Node, []*Node) {
	nodeCount++
	port := strconv.Itoa(nodeCount*1111 + nodeCount - 1)
	helperIp := "0.0.0.0"
	helperPort := "1111"

	var addr = "0.0.0.0" + ":" + port

	// Create new Node object for yourself
	node := Node{
		// id:          IPAddress(addr).GenerateHash(M),
		ipAddress:   IPAddress(addr),
		fingerTable: make([]IPAddress, 0),
	}

	fmt.Println("My IP Address is", string(node.ipAddress))
	// fmt.Println("My id is", node.id)

	// Bind yourself to a port and listen to it
	tcpAddr, errtc := net.ResolveTCPAddr("tcp", string(node.ipAddress))
	if errtc != nil {
		fmt.Println("Error resolving TCP address", errtc)
	}
	inbound, errin := net.ListenTCP("tcp", tcpAddr)
	if errin != nil {
		fmt.Println("Could not listen to TCP address", errin)
	}

	// Register RPC methods and accept incoming requests
	rpc.Register(&node)
	fmt.Printf("Node is running at IP address: %s\n", tcpAddr.String())
	go rpc.Accept(inbound)

	// node at which the node is joining
	// helperIp = helperIp[:len(helperIp)-1]
	fmt.Println(nodeAr)

	/*
		When a node first joins, it checks if it is the first node, then creates a new
		chord network, or joins an existing chord network accordingly.
	*/
	if len(*nodeAr) == 0 { // I am the only node in this network
		*nodeAr = append(*nodeAr, &node)
		node.CreateNetwork()
	} else {
		*nodeAr = append(*nodeAr, &node)
		node.JoinNetwork(IPAddress(helperIp + ":" + helperPort))
	}

	return node, *nodeAr
}

func (n *Node) Run() {
	for {
		// receive msg to search for shortID
		break
	}
}

/*
Node utility function to call RPC given a request message, and a destination IP address.
*/
func (node *Node) CallRPC(msg RMsg, IP string) RMsg {
	fmt.Printf("Nodeid: %v IP: %s is sending message %v to IP: %s\n", msg.OutgoingIP, msg.IncomingIP, msg.MsgType, IP)
	clnt, err := rpc.Dial("tcp", IP)
	reply := RMsg{}
	if err != nil {
		// fmt.Printf(msg.msgType)
		fmt.Printf("Nodeid: %v IP: %s received reply %v from IP: %s\n", msg.OutgoingIP, msg.IncomingIP, msg.MsgType, IP)
		reply.MsgType = EMPTY
		return reply
	}
	err = clnt.Call("Node.HandleIncomingMessage", &msg, &reply)
	if err != nil {
		fmt.Printf("Error calling RPC\n")
		fmt.Printf("Nodeid: %d IP: %s received reply %v from IP: %s\n", msg.OutgoingIP, msg.IncomingIP, msg.MsgType, IP)
		reply.MsgType = EMPTY
		return reply
	}
	fmt.Printf("Received reply from %s\n", IP)
	return reply
}

/*
NODE Function to handle incoming RPC Calls and where to route them
*/
func (node *Node) HandleIncomingMessage(msg *RMsg, reply *RMsg) error {
	fmt.Println("msg", msg, "where")
	fmt.Println("Message of type", msg.MsgType, "received.")
	switch msg.MsgType {
	case JOIN:
		fmt.Println("Received JOIN message")
		reply.MsgType = ACK
	}
	return nil
}

func (n *Node) CreateNetwork() {
	// following pseudo code
	n.predecessor = IPAddress("")
	n.successor = n.ipAddress // itself
	fmt.Println("Succesfully Created Network", n)
}

// TODO: Update to use "FIND_SUCCESSOR"
func (n *Node) JoinNetwork(joiningIP IPAddress) {
	// following pseudo code
	n.predecessor = IPAddress("")

	joinMsg := RMsg{
		MsgType:    JOIN,
		OutgoingIP: n.ipAddress,
		IncomingIP: joiningIP,
	}

	reply := n.CallRPC(joinMsg, string(joiningIP))
	n.successor = joiningIP // itself

	fmt.Println("Succesfully Joined Network", n, reply)
}

func (id Hash) inBetween(start IPAddress, until IPAddress, includingUntil bool) bool {
	startID := start.GenerateHash()
	untilID := until.GenerateHash()

	if startID == untilID {
		return true
	} else if startID < untilID {
		if includingUntil {
			return startID < id && id <= untilID
		} else {
			return startID < id && id < untilID
		}
	} else {
		if includingUntil {
			return untilID <= id || id < startID
		} else {
			return untilID < id || id < startID
		}
	}
}

func (n *Node) FindSuccessor(targetIPAddress IPAddress) IPAddress {
	if targetIPAddress.GenerateHash().inBetween(n.ipAddress, n.successor, true) {
		return n.successor
	} else {
		otherNodeIP := n.ClosestPrecedingNode(10, targetIPAddress)
		// TODO: call FindSuccessor on otherNodeIP
		fmt.Println(otherNodeIP)
	}
	return IPAddress("")
}

func (n *Node) ClosestPrecedingNode(numberOfMachines int, targetIPAddress IPAddress) IPAddress {
	for i := numberOfMachines; i > 0; i-- {
		if n.fingerTable[i].GenerateHash().inBetween(n.ipAddress, targetIPAddress, false) {
			return n.fingerTable[i]
		}
	}
	return n.ipAddress
}

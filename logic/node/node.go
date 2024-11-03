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

type ShortURL string

type LongURL string

/*
Function to generate the hash of the the input IP address
*/
func (ip IPAddress) GenerateHash() Hash { // TODO: EST_NO_OF_MACHINES should be the max num of machines our chord can take
	EST_NO_OF_MACHINES := 10 // This is actually not m
	MAX_RING_SIZE := int64(math.Pow(2, float64(EST_NO_OF_MACHINES)))

	data := []byte(ip)
	id := sha256.Sum256(data)
	unmoddedID := new(big.Int).SetBytes(id[:8])
	modValue := new(big.Int).SetInt64(MAX_RING_SIZE)
	moddedID := new(big.Int).Mod(unmoddedID, modValue)
	return Hash(moddedID.Int64())
}

type Entry struct {
	shortURL ShortURL
	longURL  LongURL
}

type Node struct {
	// id          uint64
	ipAddress   IPAddress
	fingerTable []IPAddress
	successor   IPAddress
	predecessor IPAddress
	// fixFingerNext int
	urlMap map[ShortURL]LongURL
}

func (n *Node) SetSuccessor(ipAddress IPAddress) {
	n.successor = ipAddress
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
		urlMap:      make(map[ShortURL]LongURL),
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

	// Register new server (because we are running)
	server := rpc.NewServer()
	// Register RPC methods and accept incoming requests
	server.Register(&node)
	fmt.Printf("Node is running at IP address: %s\n", tcpAddr.String())
	fmt.Println("nodeIP", node.ipAddress)
	go server.Accept(inbound)

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
	// for {
	if n.ipAddress == "0.0.0.0:10007" {
		// act as if it gets url
		// put in url and retrieve it
		entries := []Entry{
			{shortURL: "short1", longURL: "http://example.com/long1"},
			{shortURL: "short2", longURL: "http://example.com/long2"},
			{shortURL: "short3", longURL: "http://example.com/long3"},
		}

		for _, entry := range entries {
			fmt.Println(entry.shortURL, "SHORT HASH", IPAddress(entry.shortURL).GenerateHash())
			fmt.Println(entry.shortURL, n.ipAddress, "RUN NODE")
			short := IPAddress(entry.shortURL)
			successor := n.FindSuccessor(short)
			fmt.Println(entry.shortURL, successor, "FOUND SUCCESSOR")
		}
	}
	// }
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
	fmt.Println("msg", msg, "node ip", node.ipAddress)
	fmt.Println("Message of type", msg.MsgType, "received.")
	switch msg.MsgType {
	case JOIN:
		fmt.Println("Received JOIN message")
		reply.MsgType = ACK
	case FIND_SUCCESSOR:
		fmt.Println("Received FIND SUCCESSOR message")
		fmt.Println(msg.Payload[0])
		successor := node.FindSuccessor(msg.Payload[0]) // first value should be the target IP Address
		reply.Payload = []IPAddress{successor}
	case STORE_URL:
		shortURL := ShortURL(msg.Payload[0])
		longURL := LongURL(msg.Payload[1])
		node.urlMap[shortURL] = longURL
		reply.MsgType = ACK
	case RETRIEVE_URL:
		shortURL := ShortURL(msg.Payload[0])
		if longURL, found := node.urlMap[shortURL]; found {
			reply.QueryResponse = []string{string(longURL)}
		}
		reply.MsgType = ACK
	}
	return nil // nil means no error, else will return reply
}

func (n *Node) CreateNetwork() {
	// following pseudo code
	n.predecessor = IPAddress("")
	n.successor = n.ipAddress // itself
	fmt.Println("Succesfully Created Network", n)
}

func (n *Node) JoinNetwork(joiningIP IPAddress) {
	// following pseudo code
	n.predecessor = IPAddress("")

	findSuccessorMsg := RMsg{
		MsgType:    FIND_SUCCESSOR,
		OutgoingIP: n.ipAddress,
		IncomingIP: joiningIP,
		Payload:    []IPAddress{n.ipAddress},
	}

	reply := n.CallRPC(findSuccessorMsg, string(joiningIP))
	n.successor = reply.Payload[0]

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
	}

	otherNodeIP := n.ClosestPrecedingNode(len(n.fingerTable), targetIPAddress)
	fmt.Println(otherNodeIP, "alkfjlsakdflkasjdfkl")
	if otherNodeIP == n.ipAddress {
		fmt.Println("equivalent")
		return n.successor
	}

	findSuccMsg := RMsg{
		MsgType:    FIND_SUCCESSOR,
		OutgoingIP: n.ipAddress,
		IncomingIP: otherNodeIP,
		Payload:    []IPAddress{targetIPAddress},
	}

	reply := n.CallRPC(findSuccMsg, string(otherNodeIP))
	fmt.Println(reply)
	return reply.Payload[0]
}

// This function isnt being called at the moment
// TODO : not implemented correctly
func (n *Node) ClosestPrecedingNode(numberOfMachines int, targetIPAddress IPAddress) IPAddress {
	for i := numberOfMachines - 1; i >= 0; i-- {
		if n.fingerTable[i].GenerateHash().inBetween(n.ipAddress, targetIPAddress, false) {
			return n.fingerTable[i]
		}
	}
	return n.ipAddress
}

func (n *Node) GenerateShortURL(longURL LongURL) ShortURL {
	hash := sha256.Sum256([]byte(longURL))
	// 6-byte short URL for simplicity
	// TODO: short url cannot just be hashed but should be a shorter url?
	short := fmt.Sprintf("%x", hash[:6]) 
	return ShortURL(short)
}

func (n *Node) StoreURL(shortURL ShortURL, longURL LongURL) {
	targetNodeIP := n.FindSuccessor(IPAddress(shortURL))
	if targetNodeIP == n.ipAddress {
		n.urlMap[shortURL] = longURL
		fmt.Printf("Stored URL: %s -> %s on Node %s\n", shortURL, longURL, n.ipAddress)
	} else {
		storeMsg := RMsg{
			MsgType:    STORE_URL,
			OutgoingIP: n.ipAddress,
			IncomingIP: targetNodeIP,
			Payload:    []IPAddress{IPAddress(shortURL), IPAddress(longURL)},
		}
		fmt.Printf("Sending STORE_URL message to Node %s\n", targetNodeIP)
		n.CallRPC(storeMsg, string(targetNodeIP))
	}
}

func (n *Node) RetrieveURL(shortURL ShortURL) (LongURL, bool) {
	targetNodeIP := n.FindSuccessor(IPAddress(shortURL))
	if targetNodeIP == n.ipAddress {
		longURL, found := n.urlMap[shortURL]
		if found {
			fmt.Printf("Retrieved URL: %s -> %s from Node %s\n", shortURL, longURL, n.ipAddress)
		}
		return longURL, found
	} else {
		retrieveMsg := RMsg{
			MsgType:    RETRIEVE_URL,
			OutgoingIP: n.ipAddress,
			IncomingIP: targetNodeIP,
			Payload:    []IPAddress{IPAddress(shortURL)},
		}
		reply := n.CallRPC(retrieveMsg, string(targetNodeIP))
		if len(reply.QueryResponse) > 0 {
			return LongURL(reply.QueryResponse[0]), true
		}
		return "", false
	}
}

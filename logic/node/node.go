package node

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/rpc"
	"strconv"
	"sync"
)

type Hash uint64 //[32]byte

type IPAddress string

type ShortURL string

type LongURL string

// Constants
const (
	M = 10
	// CACHE_SIZE         = 5
	// REPLICATION_FACTOR = 2
)

/*
Function to generate the hash of the the input IP address
*/
func (ip IPAddress) GenerateHash() Hash { // TODO: EST_NO_OF_MACHINES should be the max num of machines our chord can take
	// EST_NO_OF_MACHINES := 10 // This is actually not m

	MAX_RING_SIZE := int64(math.Pow(2, float64(M)))

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
	fixFingerNext int
	ipAddress     IPAddress
	fingerTable   []IPAddress
	successor     IPAddress
	predecessor   IPAddress
	urlMap        map[ShortURL]LongURL
	mu            sync.Mutex
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

func InitNode(nodeAr *[]*Node) *Node {
	nodeCount++
	port := strconv.Itoa(nodeCount*1111 + nodeCount - 1)

	// send to the same node each time
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

	// Bind yourself to a port and listen to it
	tcpAddr, errtc := net.ResolveTCPAddr("tcp", string(node.ipAddress))
	if errtc != nil {
		fmt.Println("Error resolving TCP address", errtc)
	}
	inbound, errin := net.ListenTCP("tcp", tcpAddr)
	if errin != nil {
		fmt.Println("Could not listen to TCP address", errin)
	}

	// Register new server (because we are running goroutines)
	server := rpc.NewServer()
	// Register RPC methods and accept incoming requests
	server.Register(&node)
	fmt.Printf("Node is running at IP address: %s\n", tcpAddr.String())
	fmt.Println("nodeIP", node.ipAddress)
	go server.Accept(inbound)

	/* Joining at the port 0.0.0.0:1111 */
	if helperPort == port { // I am the joining node
		// *nodeAr = append(*nodeAr, &node)
		node.CreateNetwork()
	} else {
		// *nodeAr = append(*nodeAr, &node)
		node.JoinNetwork(IPAddress(helperIp + ":" + helperPort))
	}

	go node.Maintain()

	return &node
}

func (n *Node) fixFingers() {
	n.fixFingerNext++
	if n.fixFingerNext > M-1 { // because we are 0-indexed
		n.fixFingerNext = 0
	}

	convToHash := float64(n.ipAddress.GenerateHash()) + math.Pow(2, float64(n.fixFingerNext))
	// ensure it doesn't exceed the ring
	convToHash = math.Mod(float64(convToHash), math.Pow(2, M))
	n.fingerTable[n.fixFingerNext] = n.FindSuccessor(Hash(convToHash))
}

func (n *Node) Maintain() {
	for {

	}
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
			successor := n.FindSuccessor(short.GenerateHash())
			fmt.Println(entry.shortURL, successor, "FOUND SUCCESSOR")
		}
	}
	// }
}

/*
Node utility function to call RPC given a request message, and a destination IP address.
*/
func (node *Node) CallRPC(msg RMsg, IP string) RMsg {
	fmt.Printf("Nodeid: %v IP: %s is sending message %v to IP: %s\n", msg.SenderIP, msg.RecieverIP, msg.MsgType, IP)
	clnt, err := rpc.Dial("tcp", IP)
	reply := RMsg{}
	if err != nil {
		// fmt.Printf(msg.msgType)
		fmt.Printf("Nodeid: %v IP: %s received reply %v from IP: %s\n", msg.SenderIP, msg.RecieverIP, msg.MsgType, IP)
		reply.MsgType = EMPTY
		return reply
	}
	err = clnt.Call("Node.HandleIncomingMessage", &msg, &reply)
	if err != nil {
		fmt.Printf("Error calling RPC\n")
		fmt.Printf("Nodeid: %d IP: %s received reply %v from IP: %s\n", msg.SenderIP, msg.RecieverIP, msg.MsgType, IP)
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
		fmt.Println(msg.TargetIP[0])
		successor := node.FindSuccessor(msg.TargetIP[0].GenerateHash()) // first value should be the target IP Address
		reply.TargetIP = []IPAddress{successor}
	case STORE_URL:
		fmt.Println("Received STORE_URL message")
		entry := msg.StoreEntry[0]
		shortURL := entry.shortURL
		longURL := entry.longURL
		defer node.mu.Unlock()
		node.mu.Lock()
		node.urlMap[shortURL] = longURL
		reply.MsgType = ACK
	case RETRIEVE_URL:
		fmt.Println("Received RETRIEVE_URL message")
		shortURL := ShortURL(msg.TargetIP[0])
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
		SenderIP:   n.ipAddress,
		RecieverIP: joiningIP,
		TargetHash: []Hash{n.ipAddress.GenerateHash()},
	}

	reply := n.CallRPC(findSuccessorMsg, string(joiningIP))
	n.successor = reply.TargetIP[0]

	fmt.Println("Succesfully Joined Network", n, reply)
}

func (id Hash) inBetween(start Hash, until Hash, includingUntil bool) bool {
	if start == until {
		return true
	} else if start < until {
		if includingUntil {
			return start < id && id <= until
		} else {
			return start < id && id < until
		}
	} else {
		if includingUntil {
			return until <= id || id < start
		} else {
			return until < id || id < start
		}
	}
}

func (n *Node) FindSuccessor(targetID Hash) IPAddress {
	sizeOfRing := math.Pow(2, M)
	if targetID > Hash(sizeOfRing) {
		panic("Bigger than Ring")
	}

	if targetID.inBetween(n.ipAddress.GenerateHash(), n.successor.GenerateHash(), true) {
		return n.successor
	}

	otherNodeIP := n.ClosestPrecedingNode(len(n.fingerTable), targetID)
	if otherNodeIP == n.ipAddress {
		return n.successor
	}

	findSuccMsg := RMsg{
		MsgType:    FIND_SUCCESSOR,
		SenderIP:   n.ipAddress,
		RecieverIP: otherNodeIP,
		TargetHash: []Hash{targetID},
	}

	reply := n.CallRPC(findSuccMsg, string(otherNodeIP))
	fmt.Println(reply)
	return reply.TargetIP[0]
}

// TODO : not implemented correctly
func (n *Node) ClosestPrecedingNode(numberOfMachines int, targetID Hash) IPAddress {
	for i := numberOfMachines - 1; i >= 0; i-- {
		if n.fingerTable[i].GenerateHash().inBetween(n.ipAddress.GenerateHash(), targetID, false) {
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
	targetNodeIP := n.FindSuccessor(IPAddress(shortURL).GenerateHash())
	if targetNodeIP == n.ipAddress {
		n.urlMap[shortURL] = longURL
		fmt.Printf("Stored URL: %s -> %s on Node %s\n", shortURL, longURL, n.ipAddress)
	} else {
		storeMsg := RMsg{
			MsgType:    STORE_URL,
			SenderIP:   n.ipAddress,
			RecieverIP: targetNodeIP,
			StoreEntry: []Entry{{shortURL, longURL}},
		}
		fmt.Printf("Sending STORE_URL message to Node %s\n", targetNodeIP)
		n.CallRPC(storeMsg, string(targetNodeIP))
	}
}

func (n *Node) RetrieveURL(shortURL ShortURL) (LongURL, bool) {
	targetNodeIP := n.FindSuccessor(IPAddress(shortURL).GenerateHash())
	if targetNodeIP == n.ipAddress {
		longURL, found := n.urlMap[shortURL]
		if found {
			fmt.Printf("Retrieved URL: %s -> %s from Node %s\n", shortURL, longURL, n.ipAddress)
		}
		return longURL, found
	} else {
		retrieveMsg := RMsg{
			MsgType:    RETRIEVE_URL,
			SenderIP:   n.ipAddress,
			RecieverIP: targetNodeIP,
			TargetHash: []Hash{IPAddress(shortURL).GenerateHash()},
		}
		reply := n.CallRPC(retrieveMsg, string(targetNodeIP))
		if len(reply.QueryResponse) > 0 {
			return LongURL(reply.QueryResponse[0]), true
		}
		return "", false
	}
}

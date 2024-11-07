package node

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"

	// "math/rand/v2"
	"errors"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"time"
)

type Hash uint64 //[32]byte

type HashableString string

type ShortURL string

type LongURL string

func nilLongURL() LongURL {
	return LongURL("")
}

func (u LongURL) isNil() bool {
	return u == nilLongURL()
}

// Constants
const (
	M = 10
	// CACHE_SIZE         = 5
	// REPLICATION_FACTOR = 2
)

func nilHashableString() HashableString {
	return HashableString("")
}

func (ip HashableString) isNil() bool {
	return ip == nilHashableString()
}

/*
Function to generate the hash of the the input IP address
*/
func (ip HashableString) GenerateHash() Hash { // TODO: EST_NO_OF_MACHINES should be the max num of machines our chord can take
	if ip.isNil() {
		panic("Tried to call GenerateHash() on nil HashableString")
	}

	MAX_RING_SIZE := int64(math.Pow(2, float64(M)))

	data := []byte(ip)
	id := sha256.Sum256(data)
	unmoddedID := new(big.Int).SetBytes(id[:8])
	modValue := new(big.Int).SetInt64(MAX_RING_SIZE)
	moddedID := new(big.Int).Mod(unmoddedID, modValue)
	return Hash(moddedID.Int64())
}

type Entry struct {
	ShortURL ShortURL
	LongURL  LongURL
}

type Node struct {
	// id          uint64
	fixFingerNext int
	ipAddress     HashableString
	fingerTable   []HashableString
	successor     HashableString
	Predecessor   HashableString
	urlMap        map[ShortURL]LongURL
	mu            sync.Mutex
}

func (n *Node) SetSuccessor(ipAddress HashableString) {
	n.successor = ipAddress
}
func (n *Node) GetIPAddress() HashableString {
	return n.ipAddress
}
func (n *Node) GetFingerTable() *[]HashableString {
	return &n.fingerTable
}

var nodeCount int

func InitClient() *Node {
	// port := strconv.Itoa(nodeCount*1111 + nodeCount - 1)

	var addr = "0.0.0.0" + ":" + "1110"

	// Create new Node object for client
	node := Node{
		ipAddress: HashableString(addr),
	}

	fmt.Println("My Client IP Address is", string(node.ipAddress))

	// Bind yourself to a port and listen to it
	tcpAddr, errtc := net.ResolveTCPAddr("tcp", string(node.ipAddress))
	if errtc != nil {
		fmt.Println("Error resolving Client TCP address", errtc)
	}
	inbound, errin := net.ListenTCP("tcp", tcpAddr)
	if errin != nil {
		fmt.Println("Could not listen to Client TCP address", errin)
	}

	// Register new server (because we are running goroutines)
	server := rpc.NewServer()
	// Register RPC methods and accept incoming requests
	server.Register(&node)
	fmt.Printf("Client node is running at IP address: %s\n", tcpAddr.String())
	go server.Accept(inbound)

	return &node
}

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
		ipAddress:   HashableString(addr),
		fingerTable: make([]HashableString, M), // this is the length of the finger table 2**M
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
	go server.Accept(inbound)

	/* Joining at the port 0.0.0.0:1111 */
	// TODO : Handle joining at random nodes
	if helperPort == port { // I am the joining node
		*nodeAr = append(*nodeAr, &node)
		node.CreateNetwork()
	} else {
		*nodeAr = append(*nodeAr, &node)
		node.JoinNetwork(HashableString(helperIp + ":" + helperPort))
	}

	// go node.Maintain()

	return &node
}

func (n *Node) stabilise() {

	getPredecessorMsg := RMsg{
		MsgType:    GET_PREDECESSOR,
		SenderIP:   n.ipAddress,
		RecieverIP: n.successor,
	}

	reply := n.CallRPC(getPredecessorMsg, string(n.successor))
	succPred := reply.TargetIP // x == my successor's predecessor
	if succPred.isNil() {
		fmt.Println("IS NIL?", succPred)
	} else {
		if succPred.GenerateHash().inBetween(n.ipAddress.GenerateHash(), n.successor.GenerateHash(), false) {
			fmt.Println("SETTING SUCCESSOR", n.ipAddress, n.successor, succPred)
			n.successor = succPred
		}
	}

	// send a notify message to the successor to tell it to run the notify() function
	notifyMsg := RMsg{
		MsgType:    NOTIFY,
		SenderIP:   n.ipAddress,
		RecieverIP: n.successor,
	}
	// we don't use the reply
	reply2 := n.CallRPC(notifyMsg, string(n.successor))
	if reply2.MsgType == ACK {
		fmt.Println("Recv ACK for Notify Msg from", n.ipAddress)
	}
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
		n.fixFingers()
		n.stabilise()
		time.Sleep(1 * time.Millisecond)
	}
}

func (n *Node) Run() {
	// for {
	if n.ipAddress == "0.0.0.0:10007" {
		// act as if it gets url
		// put in url and retrieve it
		entries := []Entry{
			{ShortURL: "short1", LongURL: "http://example.com/long1"},
			{ShortURL: "short2", LongURL: "http://example.com/long2"},
			{ShortURL: "short3", LongURL: "http://example.com/long3"},
		}

		for _, entry := range entries {
			fmt.Println(entry.ShortURL, "SHORT HASH", HashableString(entry.ShortURL).GenerateHash())
			fmt.Println(entry.ShortURL, n.ipAddress, "RUN NODE")
			short := HashableString(entry.ShortURL)
			successor := n.FindSuccessor(short.GenerateHash())
			fmt.Println(entry.ShortURL, successor, "FOUND SUCCESSOR")
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
		// fmt.Println("Error calling RPC", err)
		fmt.Printf("Nodeid: %s IP: %s received reply %v from IP: %s\n", msg.SenderIP, msg.RecieverIP, msg.MsgType, IP)
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
	// fmt.Println("msg", msg, "node ip", node.ipAddress)
	fmt.Println("Message of type", msg, "received.")
	switch msg.MsgType {
	case JOIN:
		fmt.Println("Received JOIN message")
		reply.MsgType = ACK
	case FIND_SUCCESSOR:
		fmt.Println("Received FIND SUCCESSOR message")
		fmt.Println(msg.TargetHash)
		successor := node.FindSuccessor(msg.TargetHash) // first value should be the target IP Address
		reply.TargetIP = successor
	case CLIENT_STORE_URL:
		fmt.Println("Received CLIENT_STORE_URL message")
		entry := msg.StoreEntry
		ip, err := node.StoreURL(entry)
		reply.TargetIP = ip
		reply.MsgType = ACK
		if err != nil {
			fmt.Println(err.Error())
			reply.MsgType = EMPTY
		}
	case CLIENT_RETRIEVE_URL:
		fmt.Println("Received CLIENT_RETRIEVE_URL message")
		ShortURL := msg.RetrieveEntry.ShortURL
		LongURL, found := node.RetrieveURL(ShortURL)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: nilLongURL()}
		}
		reply.MsgType = ACK
	case STORE_URL:
		// Error checking in case it's the shortURL hash is not actually for this node?
		entry := msg.StoreEntry
		defer node.mu.Unlock()
		node.mu.Lock()
		node.urlMap[entry.ShortURL] = entry.LongURL
		fmt.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, node.ipAddress)
		// send appropiate reply back to the initial node that the client contacted
		reply.TargetIP = node.ipAddress
		reply.MsgType = ACK
	case RETRIEVE_URL:
		fmt.Println("Received RETRIEVE_URL message")
		ShortURL := msg.RetrieveEntry.ShortURL
		LongURL, found := node.urlMap[ShortURL]
		fmt.Println("AFTER RECV, RETRIEVED LONGURL", LongURL)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: nilLongURL()}
		}
		reply.MsgType = ACK
	case GET_PREDECESSOR:
		fmt.Println("Received GET PRED message", node.Predecessor)
		reply.TargetIP = node.Predecessor
	case NOTIFY:
		fmt.Println("Received NOTIFY message")
		nPrime := msg.SenderIP
		node.Notify(nPrime)
		reply.MsgType = ACK
	case EMPTY:
		panic("ERROR, EMPTY MESSAGE")
	}
	return nil // nil means no error, else will return reply
}

func (node *Node) Notify(nPrime HashableString) {
	if node.Predecessor.isNil() {
		node.Predecessor = nPrime
	}

	// handle the exception case of nPrime == node.Pred || node.ipAddress

	if nPrime.GenerateHash().inBetween(
		node.Predecessor.GenerateHash(),
		node.ipAddress.GenerateHash(),
		false,
	) {
		node.Predecessor = nPrime
	}
}

func (n *Node) CreateNetwork() {
	// following pseudo code
	n.Predecessor = nilHashableString()
	n.successor = n.ipAddress // itself
	fmt.Println("Succesfully Created Network", n)
}

func (n *Node) JoinNetwork(joiningIP HashableString) {
	// following pseudo code
	n.Predecessor = nilHashableString()

	findSuccessorMsg := RMsg{
		MsgType:    FIND_SUCCESSOR,
		SenderIP:   n.ipAddress,
		RecieverIP: joiningIP,
		TargetHash: n.ipAddress.GenerateHash(),
	}

	reply := n.CallRPC(findSuccessorMsg, string(joiningIP))
	n.successor = reply.TargetIP

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
			return start < id || id <= until
		} else {
			return start < id || id < until
		}
	}
}

func (n *Node) FindSuccessor(targetID Hash) HashableString {
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
		TargetHash: targetID,
	}

	reply := n.CallRPC(findSuccMsg, string(otherNodeIP))
	if reply.TargetIP.isNil() {
		panic(fmt.Sprintf("%+v\n", reply))
	}
	return reply.TargetIP
}

// TODO : not implemented correctly
func (n *Node) ClosestPrecedingNode(numberOfMachines int, targetID Hash) HashableString {
	for i := numberOfMachines - 1; i >= 0; i-- {
		if n.fingerTable[i] != HashableString("") { // extra check to make sure our finger table is not empty
			if n.fingerTable[i].GenerateHash().inBetween(n.ipAddress.GenerateHash(), targetID, false) {
				return n.fingerTable[i]
			}
		}
	}
	return n.ipAddress
}

func (n *Node) GenerateShortURL(LongURL LongURL) ShortURL {
	hash := sha256.Sum256([]byte(LongURL))
	// 6-byte short URL for simplicity
	// TODO: short url cannot just be hashed but should be a shorter url?
	short := fmt.Sprintf("%x", hash[:6])
	return ShortURL(short)
}

func (n *Node) ClientSendStoreURL(longUrl string, shortUrl string, nodeAr []*Node) HashableString {
	longURL := LongURL(longUrl)
	shortURL := ShortURL(shortUrl)

	// currently hardcoded the list of nodes that the client can call
	callNode := nodeAr[rand.IntN(len(nodeAr)-1)] // THIS IS NOT AVAILABLE

	// clientIP := node.HashableString("clientIP")
	clientStoreMsg := RMsg{
		MsgType:    CLIENT_STORE_URL,
		SenderIP:   n.GetIPAddress(),
		RecieverIP: callNode.GetIPAddress(),
		StoreEntry: Entry{ShortURL: shortURL, LongURL: longURL},
	}

	fmt.Printf("Client sending CLIENT_STORE_URL message to Node %s\n", callNode.GetIPAddress())
	// for checking purposes
	reply := n.CallRPC(clientStoreMsg, string(callNode.GetIPAddress()))
	fmt.Println("NODE :", reply.TargetIP, "successfully stored shortURL.")
	return reply.TargetIP
}

func (n *Node) ClientRetrieveURL(shortUrl string, nodeAr []*Node) (Entry, bool) {
	// longURL := LongURL(longUrl)
	shortURL := ShortURL(shortUrl)

	// currently hardcoded the list of nodes that the client can call
	callNode := nodeAr[rand.IntN(len(nodeAr)-1)] // THIS IS NOT AVAILABLE IRL

	// clientIP := node.HashableString("clientIP")
	clientRetrieveMsg := RMsg{
		MsgType:       CLIENT_RETRIEVE_URL,
		SenderIP:      n.GetIPAddress(),
		RecieverIP:    callNode.GetIPAddress(),
		RetrieveEntry: Entry{ShortURL: shortURL, LongURL: nilLongURL()},
	}

	fmt.Printf("Client sending CLIENT_RETRIEVE_URL message to Node %s\n", callNode.GetIPAddress())
	// for checking purposes
	reply := n.CallRPC(clientRetrieveMsg, string(callNode.GetIPAddress()))
	return reply.RetrieveEntry, !reply.RetrieveEntry.LongURL.isNil()
}

func (n *Node) StoreURL(entry Entry) (HashableString, error) {
	// this handles the correct node to send the entry to
	targetNodeIP := n.FindSuccessor(HashableString(entry.ShortURL).GenerateHash())

	if targetNodeIP == n.ipAddress {
		defer n.mu.Unlock()
		n.mu.Lock()
		n.urlMap[entry.ShortURL] = entry.LongURL
		fmt.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, n.ipAddress)
		return n.ipAddress, nil
	} else {
		storeMsg := RMsg{
			MsgType:    STORE_URL,
			SenderIP:   n.ipAddress,
			RecieverIP: targetNodeIP,
			StoreEntry: entry,
		}
		fmt.Printf("Sending STORE_URL message to Node %s\n", targetNodeIP)
		reply := n.CallRPC(storeMsg, string(targetNodeIP))
		if reply.MsgType == ACK {
			return reply.TargetIP, nil
		}
	}
	return nilHashableString(), errors.New("no valid IP address found for storing")
}

func (n *Node) RetrieveURL(ShortURL ShortURL) (LongURL, bool) {
	targetNodeIP := n.FindSuccessor(HashableString(ShortURL).GenerateHash())
	if targetNodeIP == n.ipAddress {
		LongURL, found := n.urlMap[ShortURL]
		return LongURL, found
	} else {
		retrieveMsg := RMsg{
			MsgType:       RETRIEVE_URL,
			SenderIP:      n.ipAddress,
			RecieverIP:    targetNodeIP,
			RetrieveEntry: Entry{ShortURL: ShortURL, LongURL: nilLongURL()},
		}
		reply := n.CallRPC(retrieveMsg, string(targetNodeIP))

		LongURL := reply.RetrieveEntry.LongURL
		fmt.Println("LONG URL RETRIEVED", LongURL)
		if !LongURL.isNil() {
			return LongURL, true
		} else {
			return nilLongURL(), false
		}
	}
}

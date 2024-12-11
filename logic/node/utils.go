package node

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand/v2"
	"net"
	"net/rpc"
	"sync"
)

// Message types.
const (
	PING                       = "ping"                       // Used to check predecessor.
	ACK                        = "ack"                        // Used for general acknowledgements.
	FIND_SUCCESSOR             = "find_successor"             // Used to find successor.
	FIND_SUCCESSOR_ADD         = "find_successor_add"         // used to check hop count
	CLOSEST_PRECEDING_NODE     = "closest_preceding_node"     // Used to find the closest preceding node, given a successor id.
	GET_PREDECESSOR            = "get_predecessor"            // Used to get the predecessor of some node.
	CREATE_SUCCESSOR_LIST      = "create_successor_list"      // Used in RPC call to get node.Successor
	GET_SUCCESSOR_LIST         = "get_successor_list"         // get the successor's successor list to maintain list
	NOTIFY                     = "notify"                     // Used to notify a node about a new predecessor.
	NOTIFY_ACK                 = "notify_ack"                 // Used to ack a "notify", and possibly send the entries that the new predecessor needs to take on.
	EMPTY                      = "empty"                      // Placeholder or undefined message type or errenous communications.
	JOIN                       = "join"                       // testing the join function
	STORE_URL                  = "store_url"                  // Used to store a url in the node.
	RETRIEVE_URL               = "retrieve_url"               // Used to retrieve a url from the node.
	CLIENT_STORE_URL           = "client_store_url"           // Client tells node to store a single short/long URL pair
	CLIENT_RETRIEVE_URL        = "client_retrieve_url"        // Client tells node to retrieve a single short/long URL pair
	SEND_REPLICA_DATA          = "send_replica_data"          // used to send node data to successors
	NOTIFY_SUCCESSOR_LEAVING   = "notify_successor_leaving"   // Voluntary leaving - telling the successor
	NOTIFY_PREDECESSOR_LEAVING = "notify_predecessor_leaving" // Voluntary leaving - telling the predecessor
)

const (
	M        = 10
	NUMNODES = 20
	REPLICAS = 5
)

type URLData struct {
	LongURL   LongURL
	Timestamp int64
}

type RMsg struct {
	MsgType        string
	SenderIP       HashableString       // Sender IP
	RecieverIP     HashableString       // Receiver IP
	TargetHash     Hash                 // Hash Value of the value to be found (shortURL or IP Address )
	TargetIP       HashableString       // IP of the Found Node
	StoreEntry     Entry                // for passing the short/long URL pair to be stored for a ShortURL request
	RetrieveEntry  Entry                // for passing the retrieved longURL for a RetrieveURL request
	HopCount       int                  // For succList
	SuccList       []HashableString     // for successors
	ReplicaData    map[ShortURL]URLData // for replicas
	Keys           map[ShortURL]URLData // For transferring keys when voluntatily leaving
	NewPredecessor HashableString       // Informing successor of its new predecessor
	LastNode       HashableString       // Last node in the successor list of the node leaving
	Timestamp      int64
	cacheString    string
	CheckFlow      []HashableString
}

type Node struct {
	Mu            sync.RWMutex
	ipAddress     HashableString
	fixFingerNext int
	fingerTable   []HashableString
	successor     HashableString
	predecessor   HashableString
	UrlMap        URLMap
	SuccList      []HashableString
	FailFlag      bool
	StoreHop      bool
}

type URLMap struct {
	Mu     sync.Mutex
	UrlMap map[HashableString]map[ShortURL]URLData
}

type Entry struct {
	ShortURL  ShortURL
	LongURL   LongURL
	Timestamp int64
}

// func (m *URLMap) copy() map[HashableString]map[ShortURL]URLData {
// 	m.Mu.Lock()
// 	defer m.M
// 	res := make(map[HashableString]map[ShortURL]URLData)
// 	for k,v
// }

func (m *URLMap) copyChild(idx HashableString) (map[ShortURL]URLData, bool) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	newMap := make(map[ShortURL]URLData)
	target, found := m.UrlMap[idx]
	if found {
		for k, v := range target {
			newMap[k] = v
		}
		return newMap, found
	}
	return nil, found
}

func (m *URLMap) copyGrandchild(idx HashableString, childIdx ShortURL) (URLData, bool) {
	res, found := m.copyChild(idx)
	if found {
		res2, found2 := res[childIdx]
		return res2, found2
	} else {
		return URLData{}, found
	}
}

func (m *URLMap) copyGrandchildWithoutFoundCheck(idx HashableString, childIdx ShortURL) URLData {
	res, _ := m.copyGrandchild(idx, childIdx)
	return res
}

func (m *URLMap) copyChildWithoutFoundCheck(idx HashableString) map[ShortURL]URLData {
	res, _ := m.copyChild(idx)
	return res
}

func (m *URLMap) update(idx HashableString, entry map[ShortURL]URLData) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.UrlMap[idx] = entry
}

func (m *URLMap) updateChild(idx HashableString, childIdx ShortURL, entry URLData) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.UrlMap[idx][childIdx] = entry
}

func (m *URLMap) delete(idx HashableString) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	delete(m.UrlMap, idx)
}

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

// UTILITY FUNCTIONS - Node

/*
Node utility function to call RPC given a request message, and a destination IP address.
*/
func (node *Node) CallRPC(msg RMsg, IP string) (RMsg, error) {
	log.Printf("Nodeid: %v IP: %s is sending message %v to IP: %s\n", msg.SenderIP, msg.RecieverIP, msg.MsgType, IP)

	clnt, err := rpc.Dial("tcp", IP)
	if err != nil {
		log.Printf("Failed to connect to %s: %v\n", IP, err)
		// Return an empty RMsg and the error
		return RMsg{}, fmt.Errorf("failed to connect to %s: %w", IP, err)
	}
	defer clnt.Close()

	reply := RMsg{}
	err = clnt.Call("Node.HandleIncomingMessage", &msg, &reply)
	if err != nil {
		log.Printf("RPC call to %s failed: %v\n", IP, err)
		// Return an empty RMsg and the error
		return RMsg{}, fmt.Errorf("RPC call to %s failed: %w", IP, err)
	}

	log.Printf("Nodeid: %s IP: %s received reply %v from IP: %s\n", msg.SenderIP, msg.RecieverIP, msg.MsgType, IP)
	return reply, nil
}

func (n *Node) GenerateShortURL(LongURL LongURL) ShortURL {
	hash := sha256.Sum256([]byte(LongURL))
	// 6-byte short URL for simplicity
	// TODO: short url cannot just be hashed but should be a shorter url?
	short := fmt.Sprintf("%x", hash[:6])
	// log.Println(short)
	return ShortURL(short)
}

func (n *Node) SetSuccessor(ipAddress HashableString) {
	n.successor = ipAddress
}

func (n *Node) GetIPAddress() HashableString {
	return n.ipAddress
}

func (n *Node) GetFixFingerCount() int {
	return n.fixFingerNext
}

func (n *Node) GetFingerTable() *[]HashableString {
	return &n.fingerTable
}

func (n *Node) GetSuccessor() HashableString {
	return n.successor
}

func (n *Node) GetPredecessor() HashableString {
	return n.predecessor
}

func (n *Node) GetURLMap() *URLMap {
	return &n.UrlMap
}

func (n *Node) GetSuccList() []HashableString {
	return n.SuccList
}

// UTILITY FUNCTIONS - HashableString

func nilHashableString() HashableString {
	return HashableString("")
}

func (ip HashableString) isNil() bool {
	return ip == nilHashableString()
}

// Function to generate Hash of Input String
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

// UTILITY FUNCTIONS - Hash

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

// UTILITY FUNCTIONS - ClientNode

func InitClient() *Node {
	var addr = "0.0.0.0" + ":" + "1110"

	// Create new Node object for client
	node := Node{
		ipAddress: HashableString(addr),
	}

	log.Println("My Client IP Address is", string(node.ipAddress))

	// Bind yourself to a port and listen to it
	tcpAddr, errtc := net.ResolveTCPAddr("tcp", string(node.ipAddress))
	if errtc != nil {
		log.Println("Error resolving Client TCP address", errtc)
	}
	inbound, errin := net.ListenTCP("tcp", tcpAddr)
	if errin != nil {
		log.Println("Could not listen to Client TCP address", errin)
	}

	// Register new server (because we are running goroutines)
	server := rpc.NewServer()
	// Register RPC methods and accept incoming requests
	server.Register(&node)
	log.Printf("Client node is running at IP address: %s\n", tcpAddr.String())
	go server.Accept(inbound)

	return &node
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
		HopCount:   0,
		CheckFlow:  make([]HashableString, 0),
	}

	log.Printf("Client sending CLIENT_STORE_URL message to Node %s\n", callNode.GetIPAddress())
	// for checking purposes
	reply, err := n.CallRPC(clientStoreMsg, string(callNode.GetIPAddress()))
	if err != nil {
		log.Println("Error in ClientSendStoreURL", err)
	}
	log.Println("NODE :", reply.TargetIP, "successfully stored shortURL.")
	fmt.Println("Store Hop Count:", reply.HopCount, "---", "Store Hop Flow:", reply.CheckFlow)
	return reply.TargetIP
}

func (n *Node) ClientRetrieveURL(shortUrl string, nodeAr []*Node, cacheBool string) (Entry, bool) {
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
		cacheString:   cacheBool,
	}

	log.Printf("Client sending CLIENT_RETRIEVE_URL message to Node %s\n", callNode.GetIPAddress())
	// for checking purposes
	reply, err := n.CallRPC(clientRetrieveMsg, string(callNode.GetIPAddress()))
	if err != nil {
		log.Println("Error in ClientRetrieveURL", err)
	}
	fmt.Println("Retrieve Hop Count:", reply.HopCount)
	return reply.RetrieveEntry, !reply.RetrieveEntry.LongURL.isNil()
}

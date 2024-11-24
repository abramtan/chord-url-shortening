package node

import (
	"errors"
	"log"
	"math"
	"net"
	"net/rpc"
	"strconv"
	"time"
)

var nodeCount int

/*
NODE Function to handle incoming RPC Calls and where to route them
*/
func (node *Node) HandleIncomingMessage(msg *RMsg, reply *RMsg) error {
	// log.Println("msg", msg, "node ip", node.ipAddress)
	log.Println("Message of type", msg, "received.")
	switch msg.MsgType {
	case JOIN:
		log.Println("Received JOIN message")
		reply.MsgType = ACK
	case FIND_SUCCESSOR:
		log.Println("Received FIND SUCCESSOR message")
		log.Println(msg.TargetHash)
		successor := node.FindSuccessor(msg.TargetHash) // first value should be the target IP Address
		reply.TargetIP = successor
	case CLIENT_STORE_URL:
		log.Println("Received CLIENT_STORE_URL message")
		entry := msg.StoreEntry
		ip, err := node.StoreURL(entry)
		reply.TargetIP = ip
		reply.MsgType = ACK
		if err != nil {
			log.Println(err.Error())
			reply.MsgType = EMPTY
		}
	case CLIENT_RETRIEVE_URL:
		log.Println("Received CLIENT_RETRIEVE_URL message")
		ShortURL := msg.RetrieveEntry.ShortURL
		LongURL, found := node.RetrieveURL(ShortURL)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: nilLongURL()}
		}
		reply.MsgType = ACK
	case STORE_URL:
		// TODO: Error checking in case it's the shortURL hash is not actually for this node?
		entry := msg.StoreEntry
		defer node.mu.Unlock()
		node.mu.Lock()
		node.UrlMap[entry.ShortURL] = entry.LongURL
		log.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, node.ipAddress)
		// send appropiate reply back to the initial node that the client contacted
		reply.TargetIP = node.ipAddress
		reply.MsgType = ACK
	case RETRIEVE_URL:
		log.Println("Received RETRIEVE_URL message")
		ShortURL := msg.RetrieveEntry.ShortURL
		LongURL, found := node.UrlMap[ShortURL]
		log.Println("AFTER RECV, RETRIEVED LONGURL", LongURL)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: nilLongURL()}
		}
		reply.MsgType = ACK
	case GET_PREDECESSOR:
		log.Println("Received GET PRED message", node.predecessor)
		reply.TargetIP = node.predecessor
	case NOTIFY:
		log.Println("Received NOTIFY message")
		nPrime := msg.SenderIP
		node.Notify(nPrime)
		reply.MsgType = ACK
	case PING:
		log.Println("Received PING message")
		reply.MsgType = ACK
	case CREATE_SUCCESSOR_LIST:
		node.appendSuccList(msg.HopCount, msg.SuccList)
	case GET_SUCCESSOR_LIST:
		reply.SuccList = node.SuccList
	case SEND_REPLICA_DATA:
		log.Println("Recieved Node Data")
		node.StoreReplica(msg.ReplicaData)
		reply.MsgType = ACK
	case EMPTY:
		panic("ERROR, EMPTY MESSAGE")
	}
	return nil // nil means no error, else will return reply
}

// TODO : to implement this at the end of stablise()
// TODO : Synch SuccessorLists? (via taking successors.succlist and then prepending n.successor)
// TODO : implement checkSuccessorAlive in stablise ==> need to update the successor + succList
func (n *Node) InitSuccList() ([]HashableString, error) {

	initSuccessorListMsg := RMsg{
		MsgType:    CREATE_SUCCESSOR_LIST,
		SenderIP:   n.ipAddress,
		RecieverIP: n.successor,
		HopCount:   REPLICAS,
		SuccList:   make([]HashableString, 0),
	}

	reply := n.CallRPC(initSuccessorListMsg, string(n.successor))
	if len(reply.SuccList) == 0 {
		return []HashableString{}, errors.New("successor list empty")
	}
	n.mu.Lock()
	n.SuccList = reply.SuccList
	n.mu.Unlock()
	return reply.SuccList, nil
}

func (n *Node) appendSuccList(hopCount int, succList []HashableString) ([]HashableString, error) {
	// this method is recursive (it can be done iteratively)
	hopCount--
	if hopCount == 0 {
		if len(succList) == 0 {
			return []HashableString{}, errors.New("SuccList Empty after reaching count 0")
		}
		succList = append(succList, n.ipAddress)
		return succList, nil
	} else {
		// append itself to successorList
		succList = append(succList, n.ipAddress)

		forwardSuccessorListMsg := RMsg{
			MsgType:    CREATE_SUCCESSOR_LIST,
			SenderIP:   n.ipAddress,
			RecieverIP: n.successor,
			HopCount:   hopCount,
			SuccList:   succList,
		}

		reply := n.CallRPC(forwardSuccessorListMsg, string(n.successor))
		return reply.SuccList, nil
	}
}

func (n *Node) MaintainSuccList() {
	// ping successor
	// if active, take successor.succList[:-1] and then prepend n.successor ==> make this the succList

	// step 1 : check first successor in successor list
	// successor := n.successor
	getSuccList := RMsg{
		MsgType:    GET_SUCCESSOR_LIST,
		SenderIP:   n.ipAddress,
		RecieverIP: n.successor,
	}

	reply := n.CallRPC(getSuccList, string(n.successor))
	successorSuccList := reply.SuccList
	n.mu.Lock()
	n.SuccList = append([]HashableString{n.successor}, successorSuccList[:len(successorSuccList)-1]...) // exclude last element
	currMap := n.SuccList
	replicaData := n.UrlMap
	n.mu.Unlock()

	// after maintain, use SUCCLIST to send updated ver of own data
	for _, succIP := range currMap {
		sendSuccData := RMsg{
			MsgType:     SEND_REPLICA_DATA,
			SenderIP:    n.ipAddress,
			RecieverIP:  succIP,
			ReplicaData: replicaData,
		}

		n.CallRPC(sendSuccData, string(succIP))
	}
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
		UrlMap:      make(map[ShortURL]LongURL),
		SuccList:    make([]HashableString, REPLICAS),
	}

	log.Println("My IP Address is", string(node.ipAddress))

	// Bind yourself to a port and listen to it
	tcpAddr, errtc := net.ResolveTCPAddr("tcp", string(node.ipAddress))
	if errtc != nil {
		log.Println("Error resolving TCP address", errtc)
	}
	inbound, errin := net.ListenTCP("tcp", tcpAddr)
	if errin != nil {
		log.Println("Could not listen to TCP address", errin)
	}

	// Register new server (because we are running goroutines)
	server := rpc.NewServer()
	// Register RPC methods and accept incoming requests
	server.Register(&node)
	log.Printf("Node is running at IP address: %s\n", tcpAddr.String())
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
		log.Println("IS NIL?", succPred)
	} else {
		if succPred.GenerateHash().inBetween(n.ipAddress.GenerateHash(), n.successor.GenerateHash(), false) {
			log.Println("SETTING SUCCESSOR", n.ipAddress, n.successor, succPred)
			n.successor = succPred
		}
	}

	// send a notify message to the successor to tell it to run the notify() function
	notifyMsg := RMsg{
		MsgType:    NOTIFY,
		SenderIP:   n.ipAddress,
		RecieverIP: n.successor,
	}

	reply2 := n.CallRPC(notifyMsg, string(n.successor))
	if reply2.MsgType == ACK {
		log.Println("Recv ACK for Notify Msg from", n.ipAddress)
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

func (n *Node) checkPredecessor() {
	pingMsg := RMsg{
		MsgType:    PING,
		SenderIP:   n.ipAddress,
		RecieverIP: n.predecessor,
	}

	// RPC call to predecessor
	reply := n.CallRPC(pingMsg, string(n.predecessor))
	// No response from predecessor, set predecessor to nil
	if reply.MsgType == EMPTY {
		n.predecessor = nilHashableString()
	}
}

func (n *Node) Maintain() {
	for {
		n.fixFingers()
		n.stabilise()
		n.checkPredecessor()
		n.MaintainSuccList()
		time.Sleep(2 * time.Millisecond)
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
			log.Println(entry.ShortURL, "SHORT HASH", HashableString(entry.ShortURL).GenerateHash())
			log.Println(entry.ShortURL, n.ipAddress, "RUN NODE")
			short := HashableString(entry.ShortURL)
			successor := n.FindSuccessor(short.GenerateHash())
			log.Println(entry.ShortURL, successor, "FOUND SUCCESSOR")
		}
	}
	// }
}

func (node *Node) StoreReplica(incomingData map[ShortURL]LongURL) {

	// if incomingData != nil {
	// 	node.mu.Lock()
	// 	for short, long := range incomingData {
	// 		node.UrlMap[short] = long
	// 	}
	// 	node.mu.Unlock()
	// }
	log.Println("ADDING REPLICA DATA")
}

func (node *Node) Notify(nPrime HashableString) {
	if node.predecessor.isNil() {
		node.predecessor = nPrime
	}

	if nPrime.GenerateHash().inBetween(
		node.predecessor.GenerateHash(),
		node.ipAddress.GenerateHash(),
		false,
	) {
		node.predecessor = nPrime
	}
}

func (n *Node) CreateNetwork() {
	n.predecessor = nilHashableString()
	n.successor = n.ipAddress // itself
	log.Println("Succesfully Created Network", n)
}

func (n *Node) JoinNetwork(joiningIP HashableString) {
	n.predecessor = nilHashableString()

	findSuccessorMsg := RMsg{
		MsgType:    FIND_SUCCESSOR,
		SenderIP:   n.ipAddress,
		RecieverIP: joiningIP,
		TargetHash: n.ipAddress.GenerateHash(),
	}

	reply := n.CallRPC(findSuccessorMsg, string(joiningIP))
	n.successor = reply.TargetIP

	log.Println("Succesfully Joined Network", n, reply)
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
		// panic(log.Printf("%+v\n", reply))
		log.Panicln(reply)
	}
	return reply.TargetIP
}

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

func (n *Node) StoreURL(entry Entry) (HashableString, error) {
	// this handles the correct node to send the entry to
	targetNodeIP := n.FindSuccessor(HashableString(entry.ShortURL).GenerateHash())

	if targetNodeIP == n.ipAddress {
		defer n.mu.Unlock()
		n.mu.Lock()
		n.UrlMap[entry.ShortURL] = entry.LongURL
		log.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, n.ipAddress)
		return n.ipAddress, nil
	} else {
		storeMsg := RMsg{
			MsgType:    STORE_URL,
			SenderIP:   n.ipAddress,
			RecieverIP: targetNodeIP,
			StoreEntry: entry,
		}
		log.Printf("Sending STORE_URL message to Node %s\n", targetNodeIP)
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
		LongURL, found := n.UrlMap[ShortURL]
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
		log.Println("LONG URL RETRIEVED", LongURL)
		if !LongURL.isNil() {
			return LongURL, true
		} else {
			return nilLongURL(), false
		}
	}
}

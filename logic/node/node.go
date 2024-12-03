package node

import (
	"errors"
	"fmt"
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
		fmt.Printf("Received CLIENT_STORE_URL message")
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
		node.mu.Lock()
		_, mapFound := node.UrlMap[node.ipAddress]
		if mapFound {
			node.UrlMap[node.ipAddress][entry.ShortURL] = URLData{LongURL: entry.LongURL}
		} else {
			node.UrlMap[node.ipAddress] = make(map[ShortURL]URLData)
			node.UrlMap[node.ipAddress][entry.ShortURL] = URLData{LongURL: entry.LongURL}
		}
		log.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, node.ipAddress)
		// send appropiate reply back to the initial node that the client contacted
		node.mu.Unlock()
		reply.TargetIP = node.ipAddress
		reply.MsgType = ACK
	case RETRIEVE_URL:
		log.Println("Received RETRIEVE_URL message")
		ShortURL := msg.RetrieveEntry.ShortURL
		URLDataFound, found := node.UrlMap[node.ipAddress][ShortURL]
		log.Println("AFTER RECV, RETRIEVED LONGURL", URLDataFound)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: URLDataFound.LongURL}
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
		node.StoreReplica(msg)
		reply.MsgType = ACK
	case NOTIFY_SUCCESSOR_LEAVING:
		log.Print("Received voluntarily leaving message (successor)")
		node.voluntaryLeavingSuccessor(msg.Keys, msg.NewPredecessor)
		reply.MsgType = ACK
	case NOTIFY_PREDECESSOR_LEAVING:
		log.Print("Received voluntarily leaving message (predecessor)")
		node.voluntaryLeavingPredecessor(msg.SenderIP, msg.LastNode)
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

	reply, err := n.CallRPC(initSuccessorListMsg, string(n.successor))
	if len(reply.SuccList) == 0 || err != nil {
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

		reply, err := n.CallRPC(forwardSuccessorListMsg, string(n.successor))
		if err != nil {
			return []HashableString{}, err
		}
		return reply.SuccList, nil
	}
}

func (n *Node) MaintainSuccList() {
	getSuccList := RMsg{
		MsgType:    GET_SUCCESSOR_LIST,
		SenderIP:   n.ipAddress,
		RecieverIP: n.successor,
	}

	reply, err := n.CallRPC(getSuccList, string(n.successor))
	if err != nil || len(reply.SuccList) == 0 {
		log.Printf("WARN: Successor %s is unresponsive or returned an empty successor list.\n", n.successor)

		defer n.mu.Unlock()
		n.mu.Lock()
		if len(n.SuccList) > 1 {
			n.successor = n.SuccList[1]
			log.Printf("INFO: Promoting %s to primary successor.\n", n.successor)
		} else {
			log.Println("ERROR: No valid successors available.")
			// n.mu.Unlock()
			return
		}
		// n.mu.Unlock()
		return
	}

	successorSuccList := reply.SuccList
	n.mu.Lock()
	n.SuccList = append([]HashableString{n.successor}, successorSuccList[:len(successorSuccList)-1]...) // Exclude the last element
	currMap := n.SuccList
	replicaData := n.UrlMap[n.ipAddress]
	n.mu.Unlock()

	for _, succIP := range currMap {
		sendSuccData := RMsg{
			MsgType:     SEND_REPLICA_DATA,
			SenderIP:    n.ipAddress,
			RecieverIP:  succIP,
			ReplicaData: replicaData,
			Timestamp:   time.Now(),
		}
		log.Printf("INFO: sendSuccData: ", sendSuccData)

		// Ensure replica data is sent successfully
		reply, err := n.CallRPC(sendSuccData, string(succIP))
		if err != nil {
			log.Printf("WARN: Failed to send replica data to %s: %v\n", succIP, err)
			log.Printf(reply.MsgType)
		}
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
		UrlMap:      make(map[HashableString]map[ShortURL]URLData),
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

	reply, err := n.CallRPC(getPredecessorMsg, string(n.successor))
	if err != nil {
		log.Println("ERROR: Could not get predecessor", err)
		return
	}
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

	reply2, _ := n.CallRPC(notifyMsg, string(n.successor))
	if reply2.MsgType == ACK {
		log.Println("Recv ACK for Notify Msg from", n.ipAddress)
	}
}

func (n *Node) fixFingers() {
	n.mu.Lock()
	defer n.mu.Unlock()
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
	reply, err := n.CallRPC(pingMsg, string(n.predecessor))
	// No response from predecessor, set predecessor to nil
	if reply.MsgType == EMPTY || err != nil {
		n.predecessor = nilHashableString()
	}
}

func (n *Node) Maintain() {
	for {
		n.fixFingers()
		n.stabilise()
		n.checkPredecessor()
		n.MaintainSuccList()
		time.Sleep(5 * time.Millisecond)
	}
}

// Node voluntary leaving
func (n *Node) Leave() {
	// Transfer keys to successor and inform successor of its new predecessor
	voluntaryLeaveSuccessorMsg := RMsg{
		MsgType:        NOTIFY_SUCCESSOR_LEAVING,
		SenderIP:       n.ipAddress,
		RecieverIP:     n.successor,
		Keys:           n.UrlMap[n.ipAddress],
		NewPredecessor: n.predecessor,
	}
	successorReply, err := n.CallRPC(voluntaryLeaveSuccessorMsg, string(n.successor)) // RPC call to successor
	if err != nil {
		log.Println("Error in Voluntary leave when sending successor", err)
	}
	if successorReply.MsgType == EMPTY {
		fmt.Printf("Failed to inform successor %s I am leaving\n", n.successor)
	} else {
		fmt.Printf("Successfully informed successor %s I am leaving\n", n.successor)
	}

	// Telling predecessor the last node in its successor list
	voluntaryLeavePredecessorMsg := RMsg{
		MsgType:    NOTIFY_PREDECESSOR_LEAVING,
		SenderIP:   n.ipAddress,
		RecieverIP: n.predecessor,
		LastNode:   n.SuccList[len(n.SuccList)-1],
	}
	predecessorReply, err := n.CallRPC(voluntaryLeavePredecessorMsg, string(n.predecessor)) // RPC call to predecessor
	if err != nil {
		log.Println("Error in Voluntary leave when sending predecessor", err)
	}
	if predecessorReply.MsgType == EMPTY {
		fmt.Printf("Failed to inform predecessor %s I am leaving\n", n.predecessor)
	} else {
		fmt.Printf("Successfully informed predecessor %s I am leaving\n", n.predecessor)
	}
}

// Informing successor of voluntary leaving
func (n *Node) voluntaryLeavingSuccessor(keys map[ShortURL]URLData, newPredecessor HashableString) {
	n.mu.Lock()
	fmt.Printf("Message received, original map is %s, predecessor is %s\n", n.UrlMap, n.predecessor)
	for k, v := range keys {
		n.UrlMap[n.ipAddress][k] = v
	}
	n.predecessor = newPredecessor
	fmt.Printf("Update complete, new map is %s, new predecessor is %s\n", n.UrlMap, n.predecessor)
	n.mu.Unlock()
}

// Informing predecessor of voluntary leaving
func (n *Node) voluntaryLeavingPredecessor(sender HashableString, lastNode HashableString) {
	n.mu.Lock()
	fmt.Printf("Message received, original successor list is %s\n", n.SuccList)
	newSuccList := []HashableString{}
	for _, succ := range n.SuccList {
		if succ != sender {
			newSuccList = append(newSuccList, succ)
		}
	}
	newSuccList = append(newSuccList, lastNode)
	n.SuccList = newSuccList
	fmt.Printf("Update complete, new successor list is %s\n", n.SuccList)
	n.mu.Unlock()
}

// func (n *Node) Run() {
// 	// for {
// 	if n.ipAddress == "0.0.0.0:10007" {
// 		// act as if it gets url
// 		// put in url and retrieve it
// 		entries := []Entry{
// 			{ShortURL: "short1", LongURL: "http://example.com/long1"},
// 			{ShortURL: "short2", LongURL: "http://example.com/long2"},
// 			{ShortURL: "short3", LongURL: "http://example.com/long3"},
// 		}

// 		for _, entry := range entries {
// 			log.Println(entry.ShortURL, "SHORT HASH", HashableString(entry.ShortURL).GenerateHash())
// 			log.Println(entry.ShortURL, n.ipAddress, "RUN NODE")
// 			short := HashableString(entry.ShortURL)
// 			successor := n.FindSuccessor(short.GenerateHash())
// 			log.Println(entry.ShortURL, successor, "FOUND SUCCESSOR")
// 		}
// 	}
// 	// }
// }

func (node *Node) StoreReplica(replicaMsg *RMsg) {
	node.mu.Lock()
	defer node.mu.Unlock()
	senderNode := replicaMsg.SenderIP
	log.Printf("storeReplica being called")
	timestamp := time.Now()

	replica, replicaFound := node.UrlMap[senderNode]
	if !replicaFound {
		node.UrlMap[senderNode] = make(map[ShortURL]URLData)
		// node.UrlMap[senderNode] = replicaMsg.ReplicaData
	}

	if replica == nil {
		node.UrlMap[senderNode] = make(map[ShortURL]URLData)
	}

	for short, URLDataFound := range replicaMsg.ReplicaData {
		node.UrlMap[senderNode][short] = URLData{
			LongURL:   URLDataFound.LongURL,
			Timestamp: timestamp,
		}
	}

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

	reply, err := n.CallRPC(findSuccessorMsg, string(joiningIP))
	if err != nil {
		log.Println("ERROR: Could not find successor", err)
		return
	}
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

	reply, err := n.CallRPC(findSuccMsg, string(otherNodeIP))
	if reply.TargetIP.isNil() || err != nil {
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
	fmt.Printf("Storing URL: %s -> %s\n", entry.ShortURL, entry.LongURL)
	targetNodeIP := n.FindSuccessor(HashableString(entry.ShortURL).GenerateHash())

	cacheHash := HashableString("CACHE")
	n.mu.Lock()
	if _, cacheExists := n.UrlMap[cacheHash]; !cacheExists {
		n.UrlMap[cacheHash] = make(map[ShortURL]URLData)
	}
	n.mu.Unlock()
	if targetNodeIP == n.ipAddress {
		// defer
		n.mu.Lock()
		if _, found := n.UrlMap[n.ipAddress]; !found { // if not found, make map
			n.UrlMap[n.ipAddress] = make(map[ShortURL]URLData)
		}
		n.UrlMap[n.ipAddress][entry.ShortURL] = URLData{LongURL: entry.LongURL, Timestamp: time.Now()}
		n.UrlMap[cacheHash][entry.ShortURL] = URLData{
			LongURL:   entry.LongURL,
			Timestamp: time.Now(),
		}

		n.mu.Unlock()
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
		reply, err := n.CallRPC(storeMsg, string(targetNodeIP))
		if err != nil {
			return nilHashableString(), err
		}
		if reply.MsgType == ACK {
			ackTimestamp := time.Now() // Use the current timestamp for cache
			n.mu.Lock()
			n.UrlMap[cacheHash][entry.ShortURL] = URLData{
				LongURL:   entry.LongURL,
				Timestamp: ackTimestamp,
			}
			n.mu.Unlock()

			// fmt.Printf("Updated Cache: %s -> %s (Timestamp: %v)", entry.ShortURL, entry.LongURL, ackTimestamp)
			log.Printf("Updated Cache: %s -> %s (Timestamp: %v)", entry.ShortURL, entry.LongURL, ackTimestamp)
			return reply.TargetIP, nil
		}
	}
	return nilHashableString(), errors.New("no valid IP address found for storing")
}

func (n *Node) RetrieveURL(shortUrl ShortURL) (LongURL, bool) {
	log.Printf("Inside RetrieveURL for ShortURL: %s", shortUrl)

	// Find the primary node responsible for the ShortURL
	targetNodeIP := n.FindSuccessor(HashableString(shortUrl).GenerateHash())

	// Attempt retrieval from the primary node
	if targetNodeIP == n.ipAddress {
		if URLDataFound, found := n.UrlMap[n.ipAddress][shortUrl]; found {
			return URLDataFound.LongURL, true
		}
		return nilLongURL(), false
	}

	// this means need to call the correct node

	// qn why is it storing in its own node

	cacheHash := HashableString("CACHE")
	// Step 1 : check if cache exists
	_, cacheExists := n.UrlMap[cacheHash]

	if !cacheExists { // make
		n.UrlMap[cacheHash] = make(map[ShortURL]URLData)
	}

	localEntry, exists := n.UrlMap[cacheHash][shortUrl]
	localTimestamp := time.Time{}
	if exists {
		localTimestamp = localEntry.Timestamp
	}

	// send retrieve url message to targetNode
	retrieveMsg := RMsg{
		MsgType:       RETRIEVE_URL,
		SenderIP:      n.ipAddress,
		RecieverIP:    targetNodeIP,
		RetrieveEntry: Entry{ShortURL: shortUrl, LongURL: nilLongURL(), Timestamp: localTimestamp},
	}

	reply, err := n.CallRPC(retrieveMsg, string(targetNodeIP))

	// if node has no error in it's reply, check the timestamp
	if err == nil {
		retrievedURL := reply.RetrieveEntry.LongURL
		retrievedTimestamp := reply.RetrieveEntry.Timestamp
		log.Printf("Retrieved URL: %s with Timestamp: %v", retrievedURL, retrievedTimestamp)

		if !retrievedURL.isNil() {
			// Conflict resolution
			if !exists || retrievedTimestamp.After(localEntry.Timestamp) {
				n.UrlMap[cacheHash][shortUrl] = URLData{
					LongURL:   retrievedURL,
					Timestamp: retrievedTimestamp,
				}
				log.Printf("Conflict resolved: Updated local data for %s with newer data.", shortUrl)
			}
			return retrievedURL, true
		}
		return nilLongURL(), false
	}

	// Primary node failed; query replicas
	log.Printf("Primary node failed. Querying replicas for %s", shortUrl)

	// Retrieve the successor list via RPC
	succListMsg := RMsg{
		MsgType:    GET_SUCCESSOR_LIST,
		SenderIP:   n.ipAddress,
		RecieverIP: targetNodeIP,
	}
	succListReply, err := n.CallRPC(succListMsg, string(targetNodeIP))
	if err != nil {
		log.Printf("Failed to get successor list from %s: %v", targetNodeIP, err)
		return nilLongURL(), false
	}

	// go through the successor list of the target node
	for _, successorIP := range succListReply.SuccList {
		if successorIP == targetNodeIP {
			continue
		}
		retrieveMsg.RecieverIP = successorIP
		reply, err := n.CallRPC(retrieveMsg, string(successorIP))
		if err != nil {
			log.Printf("Replica node %s failed: %v", successorIP, err)
			continue
		}

		retrievedURL := reply.RetrieveEntry.LongURL
		retrievedTimestamp := reply.RetrieveEntry.Timestamp
		if !retrievedURL.isNil() {
			// Conflict resolution
			if !exists || retrievedTimestamp.After(localEntry.Timestamp) {
				n.UrlMap[cacheHash][shortUrl] = URLData{
					LongURL:   retrievedURL,
					Timestamp: retrievedTimestamp,
				}
				log.Printf("Conflict resolved: Updated local data for %s with newer data from replica.", shortUrl)
			}
			return retrievedURL, true
		}
	}

	// If no replica has the data, return failure
	log.Printf("Failed to retrieve %s from all replicas.", shortUrl)
	return nilLongURL(), false
}

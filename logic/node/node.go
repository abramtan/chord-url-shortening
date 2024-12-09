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
		node.Mu.Lock()
		ShortURL := msg.RetrieveEntry.ShortURL
		node.Mu.Unlock()
		LongURL, found := node.RetrieveURL(ShortURL)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: nilLongURL()}
		}
		reply.MsgType = ACK
	case STORE_URL:
		// TODO: Error checking in case it's the shortURL hash is not actually for this node?
		node.Mu.Lock()
		entry := msg.StoreEntry
		_, mapFound := node.UrlMap[node.ipAddress]
		if mapFound {
			node.UrlMap[node.ipAddress][entry.ShortURL] = URLData{LongURL: entry.LongURL, Timestamp: time.Now().Unix()}
		} else {
			node.UrlMap[node.ipAddress] = make(map[ShortURL]URLData)
			node.UrlMap[node.ipAddress][entry.ShortURL] = URLData{LongURL: entry.LongURL, Timestamp: time.Now().Unix()}
		}
		log.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, node.ipAddress)
		// send appropiate reply back to the initial node that the client contacted
		node.Mu.Unlock()
		reply.TargetIP = node.GetIPAddress()
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
		node.Mu.Lock()
		pred := node.predecessor
		node.Mu.Unlock()
		log.Println("Received GET PRED message", pred)
		reply.TargetIP = pred
	case NOTIFY:
		log.Println("Received NOTIFY message")
		nPrime := msg.SenderIP
		node.Notify(nPrime)
		reply.MsgType = ACK
    case NOTIFY_ACK:
		log.Println("Received NOTIFY_ACK message")
        node.receiveShiftKeys(msg.Keys)
        reply.MsgType = ACK
	case PING:
		log.Println("Received PING message")
		reply.MsgType = ACK
	case CREATE_SUCCESSOR_LIST:
		node.appendSuccList(msg.HopCount, msg.SuccList)
	case GET_SUCCESSOR_LIST:
		node.Mu.Lock()
		reply.SuccList = node.SuccList
		node.Mu.Unlock()
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

	n.Mu.Lock()
	successor := n.successor
	n.Mu.Unlock()

	initSuccessorListMsg := RMsg{
		MsgType:    CREATE_SUCCESSOR_LIST,
		SenderIP:   n.ipAddress,
		RecieverIP: successor,
		HopCount:   REPLICAS,
		SuccList:   make([]HashableString, 0),
	}

	reply, err := n.CallRPC(initSuccessorListMsg, string(successor))
	if len(reply.SuccList) == 0 || err != nil {
		return []HashableString{}, errors.New("successor list empty")
	}
	n.Mu.Lock()
	n.SuccList = reply.SuccList
	n.Mu.Unlock()
	return reply.SuccList, nil
}

func (n *Node) appendSuccList(hopCount int, succList []HashableString) ([]HashableString, error) {
	// this method is recursive (it can be done iteratively)
	hopCount--
	n.Mu.Lock()
	ip := n.ipAddress
	succ := n.successor
	n.Mu.Unlock()

	if hopCount == 0 {
		if len(succList) == 0 {
			return []HashableString{}, errors.New("SuccList Empty after reaching count 0")
		}
		succList = append(succList, ip)
		return succList, nil
	} else {
		// append itself to successorList

		succList = append(succList, ip)

		forwardSuccessorListMsg := RMsg{
			MsgType:    CREATE_SUCCESSOR_LIST,
			SenderIP:   ip,
			RecieverIP: succ,
			HopCount:   hopCount,
			SuccList:   succList,
		}

		reply, err := n.CallRPC(forwardSuccessorListMsg, string(succ))
		if err != nil {
			return []HashableString{}, err
		}
		return reply.SuccList, nil
	}
}

func (n *Node) MaintainSuccList() {
	n.Mu.Lock()
	senderIP := n.ipAddress
	receiverIP := n.successor
	n.Mu.Unlock()

	getSuccList := RMsg{
		MsgType:    GET_SUCCESSOR_LIST,
		SenderIP:   senderIP,
		RecieverIP: receiverIP,
	}

	reply, err := n.CallRPC(getSuccList, string(receiverIP))
	if err != nil || len(reply.SuccList) == 0 {
		log.Printf("WARN: Successor %s is unresponsive or returned an empty successor list.\n", receiverIP)

		defer n.Mu.Unlock()
		n.Mu.Lock()
		if len(n.SuccList) > 1 {
			n.successor = n.SuccList[1]
			log.Printf("INFO: Promoting %s to primary successor.\n", n.successor)
		} else {
			log.Println("ERROR: No valid successors available.")
			return
		}
		return
	}

	n.Mu.Lock()
	successorSuccList := reply.SuccList
	n.SuccList = append([]HashableString{n.successor}, successorSuccList[:len(successorSuccList)-1]...) // Exclude the last element
	currMap := n.SuccList
	replicaData := n.UrlMap[n.ipAddress]
	n.Mu.Unlock()

	for _, succIP := range currMap {
		sendSuccData := RMsg{
			MsgType:     SEND_REPLICA_DATA,
			SenderIP:    n.ipAddress,
			RecieverIP:  succIP,
			ReplicaData: replicaData,
			Timestamp:   time.Now().Unix(),
		}
		log.Println("INFO: sendSuccData: ", sendSuccData)

		// Ensure replica data is sent successfully
		reply, err := n.CallRPC(sendSuccData, string(succIP))
		if err != nil {
			log.Printf("WARN: Failed to send replica data to %s: %v\n", succIP, err)
			log.Printf(reply.MsgType)
		}
	}
}

func (n *Node) customAccept(server *rpc.Server, lis net.Listener) {
	for {
		conn, err := lis.Accept() // lis.Accept() does not send anything back to client, so this is ok

		if err != nil {
			continue
		}

		// Reject connections based on custom logic
		if n.FailFlag {
			// fmt.Println(n.ipAddress, "reject call from", conn)
			conn.Close()
			continue
		}

		// Serve the accepted connection
		go server.ServeConn(conn)
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
		FailFlag:    false,
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
	// go server.Accept(inbound)
	go node.customAccept(server, inbound)

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
	// go through the successor list of itself
	for _, successorIP := range n.SuccList {

		getPredecessorMsg := RMsg{
			MsgType:    GET_PREDECESSOR,
			SenderIP:   n.ipAddress,
			RecieverIP: successorIP,
		}

		reply, err := n.CallRPC(getPredecessorMsg, string(successorIP))
		if err != nil || reply.TargetIP.isNil() { // err or empty senderIP
			log.Println("ERROR: Could not get predecessor", err, reply, successorIP)
			// continue // go to the next successor node
		} else {
			log.Println("INFO: Predecessor", reply)
		}

		// if no error and has senderIP
		succPred := reply.TargetIP // x == my successor's predecessor
		if succPred.isNil() {
			log.Println("IS NIL?", succPred)
		} else {
			if succPred.GenerateHash().inBetween(n.ipAddress.GenerateHash(), successorIP.GenerateHash(), false) {
				log.Println("SETTING SUCCESSOR", n.ipAddress, successorIP, succPred)
				n.Mu.Lock()
				n.successor = succPred
				n.Mu.Unlock()
			}
		}

		// send a notify message to the successor to tell it to run the notify() function
		notifyMsg := RMsg{
			MsgType:    NOTIFY,
			SenderIP:   n.ipAddress,
			RecieverIP: n.successor,
		}

		// TODO: handle keys that exist between me and my current successor
		reply2, reply2err := n.CallRPC(notifyMsg, string(n.successor))
		if reply2.MsgType == ACK && reply2err == nil {
			log.Println("Recv ACK for Notify Msg from", n.ipAddress, reply2)
			break // success so no we dont need continue the for loop
		} else {
			log.Println("FAILED for Notify Msg from", n.ipAddress, reply2)
			// continue // try next node cause the successor has failed
		}
	}
}

func (n *Node) receiveShiftKeys(transferKeys map[ShortURL]URLData) {
    n.Mu.Lock()
    n.UrlMap[n.ipAddress] = transferKeys
    n.Mu.Unlock() 
}

func (n *Node) fixFingers() {
	n.Mu.Lock()
	n.fixFingerNext++
	n.Mu.Unlock()
	if n.fixFingerNext > M-1 { // because we are 0-indexed
		n.fixFingerNext = 0
	}

	convToHash := float64(n.ipAddress.GenerateHash()) + math.Pow(2, float64(n.fixFingerNext))
	// ensure it doesn't exceed the ring
	convToHash = math.Mod(float64(convToHash), math.Pow(2, M))
	successor := n.FindSuccessor(Hash(convToHash))

	n.Mu.Lock()
	n.fingerTable[n.fixFingerNext] = successor
	n.Mu.Unlock()
}

func (n *Node) checkPredecessor() {
	n.Mu.Lock()
	ip := n.ipAddress
	pred := n.predecessor
	n.Mu.Unlock()

	pingMsg := RMsg{
		MsgType:    PING,
		SenderIP:   ip,
		RecieverIP: pred,
	}

	// RPC call to predecessor
	reply, err := n.CallRPC(pingMsg, string(pred))
	// No response from predecessor, set predecessor to nil
	if reply.MsgType == EMPTY || err != nil {
		n.Mu.Lock()
		n.predecessor = nilHashableString()
		n.Mu.Unlock()
	}
}

func (n *Node) Maintain() {
	for {
		if !n.FailFlag {
			n.fixFingers()
			n.stabilise()
			n.checkPredecessor()
			n.MaintainSuccList()
		}
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
	n.Mu.Lock()
	fmt.Printf("Message received, original map is %v, predecessor is %s\n", n.UrlMap, n.predecessor)
	for k, v := range keys {
		n.UrlMap[n.ipAddress][k] = v
	}
	n.predecessor = newPredecessor
	fmt.Printf("Update complete, new map is %v, new predecessor is %s\n", n.UrlMap, n.predecessor)
	n.Mu.Unlock()
}

// Informing predecessor of voluntary leaving
func (n *Node) voluntaryLeavingPredecessor(sender HashableString, lastNode HashableString) {
	n.Mu.Lock()
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
	n.Mu.Unlock()
}

func (node *Node) StoreReplica(replicaMsg *RMsg) {
	node.Mu.Lock()
	defer node.Mu.Unlock()
	senderNode := replicaMsg.SenderIP
	log.Printf("storeReplica being called")
	timestamp := time.Now().Unix()

	// replica, replicaFound := node.UrlMap[senderNode]
	node.UrlMap[senderNode] = make(map[ShortURL]URLData)

	// if !replicaFound {
	// 	node.UrlMap[senderNode] = make(map[ShortURL]URLData)
	// 	// node.UrlMap[senderNode] = replicaMsg.ReplicaData
	// }

	// if replica == nil {
	// 	node.UrlMap[senderNode] = make(map[ShortURL]URLData)
	// }

	// node.UrlMap[senderNode] = replicaMsg.ReplicaData

	for short, URLDataFound := range replicaMsg.ReplicaData {
		node.UrlMap[senderNode][short] = URLData{
			LongURL:   URLDataFound.LongURL,
			Timestamp: timestamp,
		}
	}

	log.Println("ADDING REPLICA DATA")
}

func (node *Node) Notify(nPrime HashableString) {
	node.Mu.Lock()
	update := false
	if node.predecessor.isNil() {
		node.predecessor = nPrime
		update = true
	}

	if nPrime.GenerateHash().inBetween(
		node.predecessor.GenerateHash(),
		node.ipAddress.GenerateHash(),
		false,
	) {
		node.predecessor = nPrime
		update = true
	}

    node.Mu.Unlock()
	if update {
        // if node.pred updates send keys
        transferKey := make(map[ShortURL]URLData, 0)
        keepKey := make(map[ShortURL]URLData, 0)
		// check which exists between itself and pred
		for short, long := range node.UrlMap[node.ipAddress] {
			log.Println(short, long)
			if !HashableString(short).GenerateHash().inBetween(node.ipAddress.GenerateHash(), node.predecessor.GenerateHash(), false) { // the last bool param should not matter, since it is the exact id of the new joining node
				transferKey[short] = long
			} else {
				keepKey[short] = long
			}
		}

        transferKeyMsg := RMsg{
            MsgType: NOTIFY_ACK,
            SenderIP: node.ipAddress,
            RecieverIP: nPrime,
            Keys: transferKey,
        }

        reply, err := node.CallRPC(transferKeyMsg, string(nPrime))
        if err != nil {
            log.Printf("Something went wrong during notify shift keys")
        }

        if reply.MsgType == ACK {
            node.Mu.Lock()
            node.UrlMap[node.ipAddress] = keepKey
            node.Mu.Unlock()
        }
	}

	// node.Mu.Unlock()
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

	n.Mu.Lock()
	ip := n.ipAddress
	succ := n.successor
	n.Mu.Unlock()

	if targetID.inBetween(ip.GenerateHash(), succ.GenerateHash(), true) {
		return succ
	}

	otherNodeIP := n.ClosestPrecedingNode(targetID)
	if otherNodeIP == ip {
		return succ
	}

	findSuccMsg := RMsg{
		MsgType:    FIND_SUCCESSOR,
		SenderIP:   ip,
		RecieverIP: otherNodeIP,
		TargetHash: targetID,
	}

	reply, err := n.CallRPC(findSuccMsg, string(otherNodeIP))
	if reply.TargetIP.isNil() || err != nil {
		// panic(log.Printf("%+v\n", reply))
		// log.Panicln(reply)
		for idx, elem := range n.fingerTable {
			if elem == otherNodeIP {
				n.fingerTable[idx] = HashableString("")
			}
		}
		return n.FindSuccessor(targetID)
	}
	return reply.TargetIP
}

func (n *Node) ClosestPrecedingNode(targetID Hash) HashableString {
	defer n.Mu.Unlock()
	n.Mu.Lock()
	for _, finger := range n.fingerTable {
		if finger != HashableString("") { // extra check to make sure our finger table is not empty
			if finger.GenerateHash().inBetween(n.ipAddress.GenerateHash(), targetID, false) {
				return finger
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
	n.Mu.Lock()
	if _, cacheExists := n.UrlMap[cacheHash]; !cacheExists {
		n.UrlMap[cacheHash] = make(map[ShortURL]URLData)
	}
	n.Mu.Unlock()
	if targetNodeIP == n.ipAddress {
		// defer
		n.Mu.Lock()
		if _, found := n.UrlMap[n.ipAddress]; !found { // if not found, make map
			n.UrlMap[n.ipAddress] = make(map[ShortURL]URLData)
		}
		n.UrlMap[n.ipAddress][entry.ShortURL] = URLData{LongURL: entry.LongURL, Timestamp: time.Now().Unix()}
		n.UrlMap[cacheHash][entry.ShortURL] = URLData{
			LongURL:   entry.LongURL,
			Timestamp: time.Now().Unix(),
		}

		n.Mu.Unlock()
		log.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, n.ipAddress)
		return n.GetIPAddress(), nil
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
			ackTimestamp := time.Now().Unix() // Use the current timestamp for cache
			n.Mu.Lock()
			n.UrlMap[cacheHash][entry.ShortURL] = URLData{
				LongURL:   entry.LongURL,
				Timestamp: ackTimestamp,
			}
			n.Mu.Unlock()

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

	n.Mu.Lock()
	cacheHash := HashableString("CACHE")
	// Step 1 : check if cache exists
	_, cacheExists := n.UrlMap[cacheHash]

	if !cacheExists { // make
		n.UrlMap[cacheHash] = make(map[ShortURL]URLData)
	}

	localEntry, exists := n.UrlMap[cacheHash][shortUrl]
	localTimestamp := time.Time{}.Unix()
	if exists {
		localTimestamp = localEntry.Timestamp
	}
	n.Mu.Unlock()

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
			if retrievedTimestamp > 0 && (!exists || retrievedTimestamp > (localEntry.Timestamp)) {
				n.UrlMap[cacheHash][shortUrl] = URLData{
					LongURL:   retrievedURL,
					Timestamp: time.Now().Unix(),
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
			if retrievedTimestamp > 0 && (!exists || retrievedTimestamp > (localEntry.Timestamp)) {
				n.UrlMap[cacheHash][shortUrl] = URLData{
					LongURL:   retrievedURL,
					Timestamp: time.Now().Unix(),
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

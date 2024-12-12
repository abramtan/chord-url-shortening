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
	case FIND_SUCCESSOR_ADD:
		log.Println("Received FIND SUCCESSOR message")
		log.Println(msg.TargetHash)
		successor, curr_hc, currFlow := node.FindSuccessorAddCount(msg.TargetHash, msg.HopCount, msg.CheckFlow) // first value should be the target IP Address
		reply.TargetIP = successor
		reply.HopCount = curr_hc
		reply.CheckFlow = currFlow
	case CLIENT_STORE_URL:
		log.Println("Received CLIENT_STORE_URL message")
		entry := msg.StoreEntry
		ip, currentHC, currFlow, err := node.StoreURL(entry, msg.HopCount, msg.CheckFlow)
		reply.TargetIP = ip
		reply.MsgType = ACK
		reply.HopCount = currentHC
		reply.CheckFlow = currFlow
		if err != nil {
			log.Println(err.Error())
			reply.MsgType = EMPTY
		}
	case CLIENT_RETRIEVE_URL:
		log.Println("Received CLIENT_RETRIEVE_URL message")
		node.Mu.Lock()
		ShortURL := msg.RetrieveEntry.ShortURL
		node.Mu.Unlock()
		LongURL, currentHC, currFlow, found := node.RetrieveURL(ShortURL, msg.HopCount, msg.CheckFlow, msg.CacheString)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: nilLongURL()}
		}
		reply.HopCount = currentHC
		reply.MsgType = ACK
		reply.CheckFlow = currFlow
	case STORE_URL:
		// TODO: Error checking in case it's the shortURL hash is not actually for this node?
		// node.Mu.Lock()
		entry := msg.StoreEntry
		_, mapFound := node.UrlMap.copyChild(node.ipAddress) //node.UrlMap[node.ipAddress]
		if mapFound {
			node.UrlMap.updateChild(node.ipAddress, entry.ShortURL, URLData{LongURL: entry.LongURL, Timestamp: time.Now().Unix()})
		} else {
			node.UrlMap.update(node.ipAddress, make(map[ShortURL]URLData))
			node.UrlMap.updateChild(node.ipAddress, entry.ShortURL, URLData{LongURL: entry.LongURL, Timestamp: time.Now().Unix()})
		}
		log.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, node.ipAddress)
		// send appropiate reply back to the initial node that the client contacted
		reply.CheckFlow = append(msg.CheckFlow, node.GetIPAddress())
		reply.HopCount = msg.HopCount + 1
		reply.TargetIP = node.GetIPAddress()
		reply.MsgType = ACK
	case RETRIEVE_URL:
		log.Println("Received RETRIEVE_URL message")
		ShortURL := msg.RetrieveEntry.ShortURL
		URLDataFound, found := node.UrlMap.copyGrandchild(node.ipAddress, ShortURL)
		log.Println("AFTER RECV, RETRIEVED LONGURL", URLDataFound)
		if found {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: URLDataFound.LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: ShortURL, LongURL: nilLongURL()}
		}
		reply.HopCount = msg.HopCount + 1
		reply.CheckFlow = append(msg.CheckFlow, node.GetIPAddress())
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
		// node.Mu.Lock()
		node.StoreReplica(msg)
		// node.Mu.Unlock()
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
	replicaData := n.UrlMap.copyChildWithoutFoundCheck(n.ipAddress)
	// replicaDataCopy := make(map[ShortURL]URLData)
	// for k,v := range(replicaData) {
	//     replicaDataCopy[k] = v
	// }
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
			conn.Close()
			continue
		}

		// Serve the accepted connection
		go server.ServeConn(conn)
	}
}

func InitNode(nodeAr *[]*Node) *Node {
	nodeCount++
	port := strconv.Itoa((nodeCount - 1) * 10 + 1111)

	// send to the same node each time
	helperIp := "0.0.0.0"
	helperPort := "1111"

	var addr = "0.0.0.0" + ":" + port

	// Create new Node object for yourself
	node := Node{
		// id:          IPAddress(addr).GenerateHash(M),
		ipAddress:   HashableString(addr),
		fingerTable: make([]HashableString, M),                                     // this is the length of the finger table 2**M
		UrlMap:      URLMap{UrlMap: make(map[HashableString]map[ShortURL]URLData)}, // TODO: mutex?
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
	// n.Mu.Lock()
	n.UrlMap.update(n.ipAddress, transferKeys)
	// n.Mu.Unlock()
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
	successor, _, _ := n.FindSuccessorAddCount(Hash(convToHash), 0, make([]HashableString, 0))

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

		ownEntries := n.UrlMap.copyChildWithoutFoundCheck(n.ipAddress)
		entriesFromPred := n.UrlMap.copyChildWithoutFoundCheck(pred)
		combinedEntries := make(map[ShortURL]URLData)
		for k, v := range ownEntries {
			combinedEntries[k] = v
		}
		for k, v := range entriesFromPred {
			combinedEntries[k] = v
		}
		n.UrlMap.update(n.ipAddress, combinedEntries)
		n.UrlMap.delete(pred)
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
	n.Mu.Lock()
	n.FailFlag = true
	n.Mu.Unlock()

	voluntaryLeaveSuccessorMsg := RMsg{
		MsgType:        NOTIFY_SUCCESSOR_LEAVING,
		SenderIP:       n.ipAddress,
		RecieverIP:     n.successor,
		Keys:           n.UrlMap.copyChildWithoutFoundCheck(n.ipAddress),
		NewPredecessor: n.predecessor,
	}
	successorReply, err := n.CallRPC(voluntaryLeaveSuccessorMsg, string(n.successor)) // RPC call to successor
	if err != nil {
		log.Println("Error in Voluntary leave when sending successor", err)
	}
	if successorReply.MsgType == EMPTY {
		log.Printf("Failed to inform successor %s I am leaving\n", n.successor)
	} else {
		log.Printf("Successfully informed successor %s I am leaving\n", n.successor)
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
		log.Printf("Failed to inform predecessor %s I am leaving\n", n.predecessor)
	} else {
		log.Printf("Successfully informed predecessor %s I am leaving\n", n.predecessor)
	}
}

// Informing successor of voluntary leaving
func (n *Node) voluntaryLeavingSuccessor(keys map[ShortURL]URLData, newPredecessor HashableString) {
	n.Mu.Lock()
	InfoLog.Printf("Message received, original map is %v, predecessor is %s\n", n.UrlMap.UrlMap, n.predecessor)
	for k, v := range keys {
		n.UrlMap.updateChild(n.ipAddress, k, v)
	}
	n.UrlMap.delete(n.predecessor)
	n.predecessor = newPredecessor
	InfoLog.Printf("Update complete, new map is %v, new predecessor is %s\n", n.UrlMap.UrlMap, n.predecessor)
	n.Mu.Unlock()
}

// Informing predecessor of voluntary leaving
func (n *Node) voluntaryLeavingPredecessor(sender HashableString, lastNode HashableString) {
	n.Mu.Lock()
	InfoLog.Printf("Message received, original successor list is %s\n", n.SuccList)
	newSuccList := []HashableString{}
	for _, succ := range n.SuccList {
		if succ != sender {
			newSuccList = append(newSuccList, succ)
		}
	}
	newSuccList = append(newSuccList, lastNode)
	n.SuccList = newSuccList
	InfoLog.Printf("Update complete, new successor list is %s\n", n.SuccList)
	n.Mu.Unlock()
}

func (node *Node) StoreReplica(replicaMsg *RMsg) {
	node.Mu.Lock()
	defer node.Mu.Unlock()
	senderNode := replicaMsg.SenderIP
	log.Printf("storeReplica being called")
	timestamp := time.Now().Unix()

	// replica, replicaFound := node.UrlMap[senderNode]
	node.UrlMap.update(senderNode, make(map[ShortURL]URLData))

	// if !replicaFound {
	// 	node.UrlMap[senderNode] = make(map[ShortURL]URLData)
	// 	// node.UrlMap[senderNode] = replicaMsg.ReplicaData
	// }

	// if replica == nil {
	// 	node.UrlMap[senderNode] = make(map[ShortURL]URLData)
	// }

	// node.UrlMap[senderNode] = replicaMsg.ReplicaData

	for short, URLDataFound := range replicaMsg.ReplicaData {
		node.UrlMap.updateChild(senderNode, short, URLData{
			LongURL:   URLDataFound.LongURL,
			Timestamp: timestamp,
		})
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
		// node.Mu.Lock()
		currMap := node.UrlMap.copyChildWithoutFoundCheck(node.ipAddress)
		// node.Mu.Unlock()
		for short, long := range currMap {
			log.Println(short, long)
			if HashableString(short).GenerateHash().inBetween(node.predecessor.GenerateHash(), node.GetIPAddress().GenerateHash(), false) { // the last bool param should not matter, since it is the exact id of the new joining node
				keepKey[short] = long
			} else {
				transferKey[short] = long
			}
		}

		transferKeyMsg := RMsg{
			MsgType:    NOTIFY_ACK,
			SenderIP:   node.ipAddress,
			RecieverIP: nPrime,
			Keys:       transferKey,
		}

		reply, err := node.CallRPC(transferKeyMsg, string(nPrime))
		if err != nil {
			log.Printf("Something went wrong during notify shift keys")
		}

		if reply.MsgType == ACK {
			// node.Mu.Lock()
			node.UrlMap.update(node.ipAddress, keepKey)
			// node.Mu.Unlock()
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
		MsgType:    FIND_SUCCESSOR_ADD,
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

// func (n *Node) FindSuccessor(targetID Hash) HashableString {
// 	sizeOfRing := math.Pow(2, M)
// 	if targetID > Hash(sizeOfRing) {
// 		panic("Bigger than Ring")
// 	}

// 	n.Mu.Lock()
// 	ip := n.ipAddress
// 	succ := n.successor
// 	n.Mu.Unlock()

// 	if targetID.inBetween(ip.GenerateHash(), succ.GenerateHash(), true) {
// 		return succ
// 	}

// 	otherNodeIP := n.ClosestPrecedingNode(targetID)
// 	if otherNodeIP == ip {
// 		return succ
// 	}

// 	findSuccMsg := RMsg{
// 		MsgType:    FIND_SUCCESSOR,
// 		SenderIP:   ip,
// 		RecieverIP: otherNodeIP,
// 		TargetHash: targetID,
// 	}

// 	reply, err := n.CallRPC(findSuccMsg, string(otherNodeIP))
// 	if reply.TargetIP.isNil() || err != nil {
// 		// panic(log.Printf("%+v\n", reply))
// 		// log.Panicln(reply)
// 		for idx, elem := range n.fingerTable {
// 			if elem == otherNodeIP {
// 				n.fingerTable[idx] = HashableString("")
// 			}
// 		}
// 		return n.FindSuccessor(targetID)
// 	}
// 	return reply.TargetIP
// }

func (n *Node) FindSuccessorAddCount(targetID Hash, hc int, currFlow []HashableString) (HashableString, int, []HashableString) {
	sizeOfRing := math.Pow(2, M)
	if targetID > Hash(sizeOfRing) {
		panic("Bigger than Ring")
	}

	// increase Hop Count
	hc++
	currFlow = append(currFlow, n.GetIPAddress())

	n.Mu.Lock()
	ip := n.ipAddress
	succ := n.successor
	n.Mu.Unlock()

	if targetID.inBetween(ip.GenerateHash(), succ.GenerateHash(), true) {
		return succ, hc, currFlow
	}

	otherNodeIP := n.ClosestPrecedingNode(targetID)
	if otherNodeIP == ip {
		return succ, hc, currFlow
	}

	findSuccMsg := RMsg{
		MsgType:    FIND_SUCCESSOR_ADD,
		SenderIP:   ip,
		RecieverIP: otherNodeIP,
		TargetHash: targetID,
		HopCount:   hc,
		CheckFlow:  currFlow,
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
		currFlow = append(currFlow, otherNodeIP)
		return n.FindSuccessorAddCount(targetID, hc+1, currFlow) // add one for the failed RPC call
	}
	return reply.TargetIP, reply.HopCount, reply.CheckFlow
}

func (n *Node) ClosestPrecedingNode(targetID Hash) HashableString {
	defer n.Mu.Unlock()
	n.Mu.Lock()
	for i := len(n.fingerTable) - 1; i >= 0; i-- {
		finger := n.fingerTable[i]
		if finger != HashableString("") { // extra check to make sure our finger table is not empty
			if finger.GenerateHash().inBetween(n.ipAddress.GenerateHash(), targetID, false) {
				return finger
			}
		}
	}
	return n.ipAddress
}

func (n *Node) StoreURL(entry Entry, hc int, currFlow []HashableString) (HashableString, int, []HashableString, error) {
	// this handles the correct node to send the entry to
	targetNodeIP, curr_hc, storeCurrFlow := n.FindSuccessorAddCount(HashableString(entry.ShortURL).GenerateHash(), hc, currFlow)

	cacheHash := HashableString("CACHE")
	cache, cacheExists := n.UrlMap.copyChild(cacheHash)
	if !cacheExists {
		n.UrlMap.update(cacheHash, make(map[ShortURL]URLData))
		cache = make(map[ShortURL]URLData)
	}
	if len(cache) >= 10 { 
		n.UrlMap.removeOldestChild(cacheHash)
	}
	
	if targetNodeIP == n.ipAddress {
		if _, found := n.UrlMap.copyChild(n.ipAddress); !found { // if not found, make map
			n.UrlMap.update(n.ipAddress, make(map[ShortURL]URLData))
		}
		n.UrlMap.updateChild(n.ipAddress, entry.ShortURL, URLData{LongURL: entry.LongURL, Timestamp: time.Now().Unix()})
		n.UrlMap.updateChild(cacheHash, entry.ShortURL, URLData{
			LongURL:   entry.LongURL,
			Timestamp: time.Now().Unix(),
		})
		InfoLog.Printf("Stored URL: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, n.ipAddress)
		InfoLog.Printf("Stored URL in cache: %s -> %s on Node %s\n", entry.ShortURL, entry.LongURL, n.ipAddress)
		return n.GetIPAddress(), curr_hc, storeCurrFlow, nil
	} else {
		storeMsg := RMsg{
			MsgType:    STORE_URL,
			SenderIP:   n.ipAddress,
			RecieverIP: targetNodeIP,
			StoreEntry: entry,
			HopCount:   curr_hc,
			CheckFlow:  storeCurrFlow,
		}
		InfoLog.Printf("Sending STORE_URL message to Node %s\n", targetNodeIP)
		reply, err := n.CallRPC(storeMsg, string(targetNodeIP))
		if err != nil {
			return nilHashableString(), curr_hc, storeCurrFlow, err
		}
		if reply.MsgType == ACK {
			ackTimestamp := time.Now().Unix() // Use the current timestamp for cache
			n.UrlMap.updateChild(cacheHash, entry.ShortURL, URLData{
				LongURL:   entry.LongURL,
				Timestamp: ackTimestamp,
			})

			InfoLog.Printf("Updated Cache on Node %s: %s -> %s (Timestamp: %v)", n.GetIPAddress(), entry.ShortURL, entry.LongURL, ackTimestamp)
			return reply.TargetIP, reply.HopCount, reply.CheckFlow, nil
		}
	}
	return nilHashableString(), curr_hc, storeCurrFlow, errors.New("no valid IP address found for storing")
}

func (n *Node) RetrieveURL(shortUrl ShortURL, hc int, currFlow []HashableString, cacheString string) (LongURL, int, []HashableString, bool) {
	log.Printf("Inside RetrieveURL for ShortURL: %s\n", shortUrl)

	cacheHash := HashableString("CACHE")
	n.Mu.Lock()

	// Step 1 : check if cache exists
	cache, cacheExists := n.UrlMap.copyChild(cacheHash)
	if !cacheExists { // make
		n.UrlMap.update(cacheHash, make(map[ShortURL]URLData))
		cache = make(map[ShortURL]URLData)
	}
	if len(cache) >= 10 { 
		n.UrlMap.removeOldestChild(cacheHash)
	}
	localEntry, exists := n.UrlMap.copyGrandchild(cacheHash, shortUrl)
	localTimestamp := time.Time{}.Unix()
	if exists {
		localTimestamp = localEntry.Timestamp
	}
	n.Mu.Unlock()

	// If mode is "cache" and the URL exists in the cache, return it
	if cacheString == "cache" {
		if exists {
			hc++
			currFlow = append(currFlow, n.GetIPAddress())
			// InfoLog.Printf("Cache hit for ShortURL: %s", shortUrl)
			return localEntry.LongURL, hc, currFlow, true
		}
	}
	// Mode is "nocache" or cache miss: retrieve data from the primary node
	InfoLog.Printf("Retrieving URL from primary node for ShortURL: %s", shortUrl)

	// Find the primary node responsible for the ShortURL
	targetNodeIP, curr_hc, storeCurrFlow := n.FindSuccessorAddCount(HashableString(shortUrl).GenerateHash(), hc, currFlow)

	// Attempt retrieval from the primary node
	if targetNodeIP == n.ipAddress {
		URLDataFound, found := n.UrlMap.copyGrandchild(n.ipAddress, shortUrl)
		if found {
			return URLDataFound.LongURL, curr_hc, storeCurrFlow, true
		}
		return nilLongURL(), curr_hc, storeCurrFlow, false
	}

	// send retrieve url message to targetNode
	retrieveMsg := RMsg{
		MsgType:       RETRIEVE_URL,
		SenderIP:      n.ipAddress,
		RecieverIP:    targetNodeIP,
		RetrieveEntry: Entry{ShortURL: shortUrl, LongURL: nilLongURL(), Timestamp: localTimestamp},
		HopCount:      curr_hc,
		CheckFlow:     storeCurrFlow,
	}

	reply, err := n.CallRPC(retrieveMsg, string(targetNodeIP))
	curr_hc = reply.HopCount // Update Hopcount accorind to Retrieve URL hop count
	storeCurrFlow = reply.CheckFlow

	// if node has no error in it's reply, check the timestamp
	if err == nil {
		retrievedURL := reply.RetrieveEntry.LongURL
		retrievedTimestamp := reply.RetrieveEntry.Timestamp
		InfoLog.Printf("Retrieved URL: %s with Timestamp: %v through RPC", retrievedURL, retrievedTimestamp)

		if !retrievedURL.isNil() {
			if retrievedTimestamp > 0 || (!exists || retrievedTimestamp > (localEntry.Timestamp)) {
				n.UrlMap.updateChild(cacheHash, shortUrl, URLData{
					LongURL:   retrievedURL,
					Timestamp: time.Now().Unix(),
				})
				InfoLog.Printf("Primary Node %s Hit : Conflict resolved/updated cache: Updated local data for %s with newer data.", n.GetIPAddress(), shortUrl)
			}
			return retrievedURL, curr_hc, storeCurrFlow, true
		}
		return nilLongURL(), curr_hc, storeCurrFlow, false
	}

	// Primary node failed; query replicas
	InfoLog.Printf("Primary node failed. Querying replicas for %s from successor list", shortUrl)

	// Retrieve the successor list via RPC
	succListMsg := RMsg{
		MsgType:    GET_SUCCESSOR_LIST,
		SenderIP:   n.ipAddress,
		RecieverIP: targetNodeIP,
	}
	succListReply, err := n.CallRPC(succListMsg, string(targetNodeIP))
	if err != nil {
		log.Printf("Failed to get successor list from %s: %v", targetNodeIP, err)
		return nilLongURL(), curr_hc, storeCurrFlow, false
	}

	// go through the successor list of the target node
	for _, successorIP := range succListReply.SuccList {
		if successorIP == targetNodeIP {
			continue
		}
		retrieveMsg.RecieverIP = successorIP
		reply, err := n.CallRPC(retrieveMsg, string(successorIP))
		curr_hc = reply.HopCount // Update Hopcount accorind to Retrieve URL hop count
		storeCurrFlow = reply.CheckFlow
		if err != nil {
			InfoLog.Printf("Replica node %s failed: %v", successorIP, err)
			continue
		}

		retrievedURL := reply.RetrieveEntry.LongURL
		retrievedTimestamp := reply.RetrieveEntry.Timestamp
		if !retrievedURL.isNil() {
			InfoLog.Printf("Retrieved URL: %s with Timestamp: %v from node %s through replica succ list RPC", retrievedURL, retrievedTimestamp, reply.SenderIP)
			// Conflict resolution
			if retrievedTimestamp > 0 || (!exists || retrievedTimestamp > (localEntry.Timestamp)) {
				n.UrlMap.updateChild(cacheHash, shortUrl, URLData{
					LongURL:   retrievedURL,
					Timestamp: time.Now().Unix(),
				})
				InfoLog.Printf("Conflict resolved/updated cache: Updated local data for %s with newer data from replica.", shortUrl)
			}
			return retrievedURL, curr_hc, storeCurrFlow, true
		}
	}

	// If no replica has the data, return failure
	log.Printf("Failed to retrieve %s from all replicas.", shortUrl)
	return nilLongURL(), curr_hc, storeCurrFlow, false
}

func (m *URLMap) removeOldestChild(idx HashableString) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	target, found := m.UrlMap[idx]
	if !found {
		return
	}

	var oldestKey ShortURL
	var oldestTime int64 = time.Now().Unix()

	// loop through to find the oldest timestamp key
	for k, v := range target {
		if v.Timestamp < oldestTime {
			oldestTime = v.Timestamp
			oldestKey = k
		}
	}

	// delete key
	delete(target, oldestKey)
	m.UrlMap[idx] = target
}
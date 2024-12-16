package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Configuration and Constants
// ============================================================================
const (
	M        = 10
	REPLICAS = 3
	EMPTY    = "empty"
	ACK      = "ack"

	JOIN                       = "join"
	FIND_SUCCESSOR             = "find_successor"
	PING                       = "ping"
	STORE_URL                  = "store_url"
	RETRIEVE_URL               = "retrieve_url"
	GET_PREDECESSOR            = "get_predecessor"
	NOTIFY                     = "notify"
	NOTIFY_ACK                 = "notify_ack"
	CREATE_SUCCESSOR_LIST      = "create_successor_list"
	GET_SUCCESSOR_LIST         = "get_successor_list"
	SEND_REPLICA_DATA          = "send_replica_data"
	NOTIFY_SUCCESSOR_LEAVING   = "notify_successor_leaving"
	NOTIFY_PREDECESSOR_LEAVING = "notify_predecessor_leaving"
	CLIENT_STORE_URL           = "client_store_url"
	CLIENT_RETRIEVE_URL        = "client_retrieve_url"

	MAX_BOOTSTRAP_RETRIES    = 5
	BOOTSTRAP_RETRY_INTERVAL = 2 * time.Second
)

// ============================================================================
// Types
// ============================================================================
type Hash uint64
type HashableString string
type ShortURL string
type LongURL string

type URLData struct {
	LongURL   LongURL
	Timestamp int64
}

type Entry struct {
	ShortURL  ShortURL
	LongURL   LongURL
	Timestamp int64
}

type RMsg struct {
	MsgType        string
	SenderIP       HashableString
	RecieverIP     HashableString
	TargetHash     Hash
	TargetIP       HashableString
	StoreEntry     Entry
	RetrieveEntry  Entry
	HopCount       int
	SuccList       []HashableString
	ReplicaData    map[ShortURL]URLData
	Keys           map[ShortURL]URLData
	NewPredecessor HashableString
	LastNode       HashableString
	Timestamp      int64
	cacheString    string
}

type Node struct {
	Mu            sync.RWMutex
	ipAddress     HashableString // RPC address (IP:RPC_PORT)
	fixFingerNext int
	fingerTable   []HashableString
	successor     HashableString
	predecessor   HashableString
	UrlMap        map[HashableString]map[ShortURL]URLData
	SuccList      []HashableString
	FailFlag      bool
}

// ============================================================================
// Utility Functions
// ============================================================================
func nilLongURL() LongURL {
	return LongURL("")
}
func (u LongURL) isNil() bool {
	return u == nilLongURL()
}
func (ip HashableString) isNil() bool {
	return ip == HashableString("")
}

func nilHashableString() HashableString {
	return HashableString("")
}

func (ip HashableString) GenerateHash() Hash {
	if ip.isNil() {
		panic("Tried to call GenerateHash() on nil HashableString")
	}

	MAX_RING_SIZE := int64(math.Pow(2, float64(M)))

	h := simpleHash(string(ip)) % MAX_RING_SIZE
	return Hash(h)
}

func simpleHash(s string) int64 {
	var hash int64
	for _, c := range s {
		hash = (hash*31 + int64(c)) & 0x7fffffffffffffff
	}
	return hash
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

// ============================================================================
// RPC Handling
// ============================================================================
func (n *Node) HandleIncomingMessage(msg *RMsg, reply *RMsg) error {
	switch msg.MsgType {
	case JOIN:
		reply.MsgType = ACK
	case FIND_SUCCESSOR:
		succ := n.FindSuccessor(msg.TargetHash)
		reply.TargetIP = succ
	case STORE_URL:
		n.Mu.Lock()
		if _, found := n.UrlMap[n.ipAddress]; !found {
			n.UrlMap[n.ipAddress] = make(map[ShortURL]URLData)
		}
		entry := msg.StoreEntry
		n.UrlMap[n.ipAddress][entry.ShortURL] = URLData{LongURL: entry.LongURL, Timestamp: time.Now().Unix()}
		n.Mu.Unlock()

		reply.TargetIP = n.ipAddress
		reply.MsgType = ACK
	case RETRIEVE_URL:
		short := msg.RetrieveEntry.ShortURL
		n.Mu.RLock()
		URLDataFound, found := n.UrlMap[n.ipAddress][short]
		n.Mu.RUnlock()
		if found {
			reply.RetrieveEntry = Entry{ShortURL: short, LongURL: URLDataFound.LongURL}
		} else {
			reply.RetrieveEntry = Entry{ShortURL: short, LongURL: nilLongURL()}
		}
		reply.MsgType = ACK
	case GET_PREDECESSOR:
		n.Mu.RLock()
		pred := n.predecessor
		n.Mu.RUnlock()
		reply.TargetIP = pred
	case NOTIFY:
		n.Notify(msg.SenderIP)
		reply.MsgType = ACK
	case PING:
		reply.MsgType = ACK
	default:
		reply.MsgType = EMPTY
	}
	return nil
}

func (n *Node) CallRPC(msg RMsg, IP string) (RMsg, error) {
	clnt, err := rpc.Dial("tcp", IP)
	if err != nil {
		return RMsg{}, fmt.Errorf("failed to connect to %s: %w", IP, err)
	}
	defer clnt.Close()

	reply := RMsg{}
	err = clnt.Call("Node.HandleIncomingMessage", &msg, &reply)
	if err != nil {
		return RMsg{}, fmt.Errorf("RPC call to %s failed: %w", IP, err)
	}
	return reply, nil
}

// ============================================================================
// Chord Node Logic
// ============================================================================
func (n *Node) Notify(nPrime HashableString) {
	n.Mu.Lock()
	update := false
	if n.predecessor.isNil() {
		n.predecessor = nPrime
		update = true
	}
	if nPrime.GenerateHash().inBetween(n.predecessor.GenerateHash(), n.ipAddress.GenerateHash(), false) {
		n.predecessor = nPrime
		update = true
	}
	n.Mu.Unlock()

	if update {
		// Transfer keys if needed; omitted for brevity
	}
}

func (n *Node) FindSuccessor(targetID Hash) HashableString {
	n.Mu.RLock()
	ip := n.ipAddress
	succ := n.successor
	n.Mu.RUnlock()

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
	if err != nil || reply.TargetIP.isNil() {
		return succ // fallback
	}
	return reply.TargetIP
}

func (n *Node) ClosestPrecedingNode(targetID Hash) HashableString {
	n.Mu.RLock()
	defer n.Mu.RUnlock()
	for i := len(n.fingerTable) - 1; i >= 0; i-- {
		finger := n.fingerTable[i]
		if !finger.isNil() && finger.GenerateHash().inBetween(n.ipAddress.GenerateHash(), targetID, false) {
			return finger
		}
	}
	return n.ipAddress
}

func (n *Node) Maintain() {
	for {
		if !n.FailFlag {
			n.fixFingers()
			n.stabilise()
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (n *Node) fixFingers() {
	n.Mu.Lock()
	n.fixFingerNext++
	if n.fixFingerNext >= M {
		n.fixFingerNext = 0
	}
	idx := n.fixFingerNext
	start := (n.ipAddress.GenerateHash() + Hash(int64(math.Pow(2, float64(idx))))) % Hash(int64(math.Pow(2, M)))
	successor := n.FindSuccessor(start)
	n.fingerTable[idx] = successor
	n.Mu.Unlock()
}

func (n *Node) stabilise() {
	n.Mu.Lock()
	succ := n.successor
	n.Mu.Unlock()

	getPredMsg := RMsg{MsgType: GET_PREDECESSOR, SenderIP: n.ipAddress, RecieverIP: succ}
	reply, err := n.CallRPC(getPredMsg, string(succ))
	if err == nil && !reply.TargetIP.isNil() {
		succPred := reply.TargetIP
		n.Mu.Lock()
		if succPred.GenerateHash().inBetween(n.ipAddress.GenerateHash(), succ.GenerateHash(), false) {
			n.successor = succPred
		}
		n.Mu.Unlock()
	}

	notifyMsg := RMsg{MsgType: NOTIFY, SenderIP: n.ipAddress, RecieverIP: n.successor}
	n.CallRPC(notifyMsg, string(n.successor))
}

func (n *Node) GenerateShortURL(l LongURL) ShortURL {
	hashVal := simpleHash(string(l))
	return ShortURL(fmt.Sprintf("s%x", hashVal))
}

// ============================================================================
// Node Initialization
// ============================================================================
func InitNode(ip HashableString, rpcPort, httpPort string) *Node {
	// The node's RPC address includes the RPC port
	rpcAddr := ip
	if !strings.Contains(string(rpcAddr), ":") {
		rpcAddr = HashableString(string(ip) + ":" + rpcPort)
	}

	node := &Node{
		ipAddress:   rpcAddr,
		fingerTable: make([]HashableString, M),
		UrlMap:      make(map[HashableString]map[ShortURL]URLData),
		SuccList:    make([]HashableString, REPLICAS),
		FailFlag:    false,
	}

	// Start as single-node ring initially
	node.predecessor = nilHashableString()
	node.successor = node.ipAddress
	node.UrlMap[node.ipAddress] = make(map[ShortURL]URLData)

	go startRPCServer(node, string(node.ipAddress))

	return node
}

func startRPCServer(n *Node, addr string) {
	server := rpc.NewServer()
	server.Register(n)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		if n.FailFlag {
			conn.Close()
			continue
		}
		go server.ServeConn(conn)
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================
func startHTTPServer(node *Node, httpAddr string) {
	r := gin.Default()

	// Bootstrap endpoint for discovery
	r.GET("/bootstrap", func(c *gin.Context) {
		c.JSON(200, gin.H{"node_ip": node.ipAddress})
	})

	// Store a URL
	r.POST("/store", func(c *gin.Context) {
		var req struct {
			LongURL string `json:"long_url" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		shortURL := node.GenerateShortURL(LongURL(req.LongURL))
		targetNodeIP := node.FindSuccessor(HashableString(shortURL).GenerateHash())
		if targetNodeIP == node.ipAddress {
			node.Mu.Lock()
			node.UrlMap[node.ipAddress][ShortURL(shortURL)] = URLData{LongURL: LongURL(req.LongURL), Timestamp: time.Now().Unix()}
			node.Mu.Unlock()
			c.JSON(200, gin.H{"short_url": shortURL})
			return
		}

		storeMsg := RMsg{
			MsgType:    STORE_URL,
			SenderIP:   node.ipAddress,
			RecieverIP: targetNodeIP,
			StoreEntry: Entry{ShortURL: ShortURL(shortURL), LongURL: LongURL(req.LongURL)},
		}

		reply, err := node.CallRPC(storeMsg, string(targetNodeIP))
		if err != nil || reply.MsgType != ACK {
			c.JSON(500, gin.H{"error": "Failed to store URL"})
			return
		}

		c.JSON(200, gin.H{"short_url": shortURL})
	})

	// Retrieve a URL
	r.GET("/retrieve", func(c *gin.Context) {
		shortURL := c.Query("shortUrl")
		if shortURL == "" {
			c.JSON(400, gin.H{"error": "shortUrl query param required"})
			return
		}

		targetNodeIP := node.FindSuccessor(HashableString(shortURL).GenerateHash())
		if targetNodeIP == node.ipAddress {
			node.Mu.RLock()
			data, found := node.UrlMap[node.ipAddress][ShortURL(shortURL)]
			node.Mu.RUnlock()
			if found {
				c.JSON(200, gin.H{"long_url": data.LongURL})
				return
			}
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}

		retrieveMsg := RMsg{
			MsgType:       RETRIEVE_URL,
			SenderIP:      node.ipAddress,
			RecieverIP:    targetNodeIP,
			RetrieveEntry: Entry{ShortURL: ShortURL(shortURL)},
		}

		reply, err := node.CallRPC(retrieveMsg, string(targetNodeIP))
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to retrieve URL"})
			return
		}
		if reply.RetrieveEntry.LongURL.isNil() {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}

		c.JSON(200, gin.H{"long_url": reply.RetrieveEntry.LongURL})
	})

	// Join the ring
	r.POST("/join", func(c *gin.Context) {
		var req struct {
			ExistingNodeIP string `json:"existing_node_ip" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		findSuccMsg := RMsg{
			MsgType:    FIND_SUCCESSOR,
			SenderIP:   node.ipAddress,
			RecieverIP: HashableString(req.ExistingNodeIP),
			TargetHash: node.ipAddress.GenerateHash(),
		}

		reply, err := node.CallRPC(findSuccMsg, req.ExistingNodeIP)
		if err != nil {
			c.JSON(500, gin.H{"error": "Could not find successor"})
			return
		}

		node.Mu.Lock()
		node.successor = reply.TargetIP
		node.Mu.Unlock()

		c.JSON(200, gin.H{"message": "Joined the ring", "successor": node.successor})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	if err := r.Run(httpAddr); err != nil {
		log.Fatalf("Failed to run HTTP server on %s: %v", httpAddr, err)
	}
}

// ============================================================================
// Bootstrapping Logic on Startup with Retries and Jitter
// ============================================================================
func attemptBootstrap(node *Node) {
	svcHost := os.Getenv("CHORD_URL_SHORTENING_SERVICE_HOST")
	svcPort := os.Getenv("CHORD_URL_SHORTENING_SERVICE_PORT")

	// If no service info provided, we assume this node is the first in the ring.
	if svcHost == "" || svcPort == "" {
		log.Println("No CHORD_URL_SHORTENING_SERVICE_HOST/PORT, becoming first node.")
		return
	}

	bootstrapURL := fmt.Sprintf("http://%s:%s/bootstrap", svcHost, svcPort)
	client := &http.Client{Timeout: 2 * time.Second}

	// Seed the RNG for jitter
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < MAX_BOOTSTRAP_RETRIES; i++ {
		resp, err := client.Get(bootstrapURL)
		if err != nil {
			log.Printf("Bootstrap attempt %d: no response. Retrying...\n", i+1)
			addJitterAndSleep()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("Bootstrap attempt %d: received non-OK response. Retrying...\n", i+1)
			addJitterAndSleep()
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Bootstrap attempt %d: error reading body. Retrying...\n", i+1)
			addJitterAndSleep()
			continue
		}

		var data map[string]string
		if err := json.Unmarshal(body, &data); err != nil {
			log.Printf("Bootstrap attempt %d: invalid JSON. Retrying...\n", i+1)
			addJitterAndSleep()
			continue
		}

		existingNodeIP, ok := data["node_ip"]
		if !ok || existingNodeIP == string(node.ipAddress) {
			log.Printf("Bootstrap attempt %d: no valid node found. Retrying...\n", i+1)
			addJitterAndSleep()
			continue
		}

		// We found an existing node. Attempt to join.
		httpPort := os.Getenv("HTTP_PORT")
		if httpPort == "" {
			httpPort = "8080"
		}

		joinURL := fmt.Sprintf("http://localhost:%s/join", httpPort)
		joinReq := map[string]string{"existing_node_ip": existingNodeIP}
		joinPayload, _ := json.Marshal(joinReq)

		resp2, err := client.Post(joinURL, "application/json", bytes.NewReader(joinPayload))
		if err != nil {
			log.Printf("Join attempt after bootstrap %d failed: %v. Retrying...\n", i+1, err)
			addJitterAndSleep()
			continue
		}
		resp2.Body.Close()
		if resp2.StatusCode == http.StatusOK {
			log.Println("Successfully joined the ring via", existingNodeIP)
			return
		} else {
			log.Printf("Join attempt after bootstrap %d returned non-OK status: %d. Retrying...\n", i+1, resp2.StatusCode)
			addJitterAndSleep()
		}
	}

	// If all attempts fail, become the first node.
	log.Println("No existing nodes found after retries, becoming the first node in the ring.")
}

func addJitterAndSleep() {
	// Add random jitter up to 2 seconds (for example)
	jitter := time.Duration(rand.Intn(2000)) * time.Millisecond
	time.Sleep(BOOTSTRAP_RETRY_INTERVAL + jitter)
}

// ============================================================================
// Main
// ============================================================================
func main() {
	podIP := os.Getenv("POD_IP")
	if podIP == "" {
		// default IP if not set
		podIP = "0.0.0.0"
	}

	rpcPort := os.Getenv("GRPC_PORT")
	if rpcPort == "" {
		rpcPort = "50051"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	n := InitNode(HashableString(podIP), rpcPort, httpPort)
	go n.Maintain()

	// Attempt to discover other nodes and join the ring if possible
	attemptBootstrap(n)

	// Run HTTP server on separate port
	httpAddr := "0.0.0.0:" + httpPort
	startHTTPServer(n, httpAddr)
}

// package main

// import (
// 	"chord-url-shortening/internal/api"
// 	"chord-url-shortening/internal/chord"
// 	"chord-url-shortening/internal/utils"
// 	"context"
// 	"time"

// 	"fmt"
// 	"log"
// 	"net"
// 	"net/http"

// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/credentials/insecure"

// 	pb "chord-url-shortening/chordurlshortening"

// 	"github.com/gin-gonic/gin"
// 	"github.com/joho/godotenv"
// )

// var node *chord.Node

// func main() {
// 	// Load environment variables from .env file
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Println("Error loading .env file")
// 	}

// 	fmt.Println("\n\n------------------------------------------------")
// 	fmt.Println("           Environment variables used           ")
// 	fmt.Println("------------------------------------------------")

// 	fmt.Printf("[GRPC_PORT]: %d\n", utils.GRPC_PORT)
// 	fmt.Printf("[HTTP_PORT]: %d\n", utils.HTTP_PORT)
// 	fmt.Printf("[POD_IP]: %s\n", utils.POD_IP)
// 	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_HOST]: %s\n", utils.CHORD_URL_SHORTENING_SERVICE_HOST)
// 	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_PORT]: %d\n", utils.CHORD_URL_SHORTENING_SERVICE_PORT)
// 	fmt.Printf("[CHORD_URL_SHORTENING_SERVICE_PORT_GRPC]: %d\n", utils.CHORD_URL_SHORTENING_SERVICE_PORT_GRPC)

// 	fmt.Println("\n\n------------------------------------------------")
// 	fmt.Println("                Starting node...                ")
// 	fmt.Println("------------------------------------------------")

// 	// Start the gRPC server
// 	go startGRPCServer(utils.GRPC_PORT)

// 	// Initialize Gin router
// 	router := gin.Default()

// 	router.POST("/shorten", api.ShortenURL)
// 	router.GET("/long-url/:short_code", api.GetLongURL)

// 	// Define health check endpoint
// 	router.GET("/api/v1/health", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
// 	})

// 	router.GET("/api/v1/pod_ip", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{"POD_IP": utils.POD_IP})
// 	})

// 	router.GET("/api/v1/other_pod_ip", func(c *gin.Context) {

// 		// Set up a grpc connection to another pod via cluster IP (which pod is dependent on how k8s load balances, since using cluster IP, can be itself)
// 		conn, err := grpc.NewClient(
// 			fmt.Sprintf("%s:%d", utils.CHORD_URL_SHORTENING_SERVICE_HOST, utils.CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
// 			grpc.WithTransportCredentials(insecure.NewCredentials()),
// 		)
// 		if err != nil {
// 			log.Printf("Did not connect: %v", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect via gRPC"})
// 		}
// 		defer conn.Close()

// 		client := pb.NewNodeServiceClient(conn)

// 		// Perform gRPC call to get the IP of another pod
// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 		defer cancel()

// 		req := &pb.GetIpRequest{NodeId: utils.POD_IP}
// 		res, err := client.GetNodeIp(ctx, req)
// 		if err != nil {
// 			log.Printf("Could not get other pod's IP: %v", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get other pod's IP"})
// 		}

// 		var OTHER_POD_IP string = fmt.Sprintf("POD_IP: %s, OTHER_POD_IP: %s", utils.POD_IP, res.GetIpAddress())
// 		log.Printf("OTHER_POD_IP: %s", OTHER_POD_IP)
// 		c.JSON(http.StatusOK, gin.H{"OTHER_POD_IP": OTHER_POD_IP})
// 	})

// 	router.GET("/api/v1/grpc_check", func(c *gin.Context) {
// 		// Get the IP address from query parameters
// 		ipAddress := c.Query("ip")

// 		// Set up a gRPC connection to another pod using the provided IP address
// 		conn, err := grpc.NewClient(
// 			fmt.Sprintf("%s:%d", ipAddress, utils.CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
// 			grpc.WithTransportCredentials(insecure.NewCredentials()),
// 		)
// 		if err != nil {
// 			log.Printf("Did not connect: %v", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect via gRPC"})
// 		}
// 		defer conn.Close()

// 		client := pb.NewNodeServiceClient(conn)

// 		// Perform gRPC call to get the IP of another pod
// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 		defer cancel()

// 		req := &pb.GetIpRequest{NodeId: utils.POD_IP}
// 		res, err := client.GetNodeIp(ctx, req)
// 		if err != nil {
// 			log.Printf("Could not get other pod's IP: %v", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get other pod's IP"})
// 		}

// 		var GRPC_CHECK string = fmt.Sprintf("POD_IP: %s, OTHER_POD_IP: %s", utils.POD_IP, res.GetIpAddress())
// 		log.Printf("GRPC_CHECK: %s", GRPC_CHECK)
// 		c.JSON(http.StatusOK, gin.H{"GRPC_CHECK": GRPC_CHECK})
// 	})

// 	// Start the HTTP server
// 	if err := router.Run(fmt.Sprintf(":%d", utils.HTTP_PORT)); err != nil {
// 		log.Fatalf("Failed to start server: %v", err)
// 	}

// 	node = chord.CreateNode(chord.IPAddress(utils.POD_IP))

// 	node.JoinRingIfExistsElseCreateRing() // create ring if no ring, else join existing ring

// 	log.Printf("%+v", node)
// }

// // Function to start the gRPC server
// func startGRPCServer(port int) {
// 	// Create a new gRPC server
// 	grpcServer := grpc.NewServer()

// 	// Register your gRPC service here
// 	pb.RegisterNodeServiceServer(grpcServer, &NodeService{})

// 	// Listen on the specified port
// 	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
// 	if err != nil {
// 		log.Fatalf("Failed to listen on port %d: %v", port, err)
// 	}

// 	log.Printf("gRPC server is running on port %d", port)

// 	// Start the gRPC server
// 	if err := grpcServer.Serve(listener); err != nil {
// 		log.Fatalf("Failed to serve gRPC server: %v", err)
// 	}
// }

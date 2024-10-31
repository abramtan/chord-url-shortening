package chord

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net/rpc"
	"sync"

	"chord-url-shortening/internal/utils"
)

var GRPC_PORT int = utils.GetEnvInt("GRPC_PORT", 50051)
var HTTP_PORT int = utils.GetEnvInt("HTTP_PORT", 8080)
var POD_IP string = utils.GetEnvString("POD_IP", "0.0.0.0")
var CHORD_URL_SHORTENING_SERVICE_HOST string = utils.GetEnvString("CHORD_URL_SHORTENING_SERVICE_HOST", "0.0.0.0")
var CHORD_URL_SHORTENING_SERVICE_PORT int = utils.GetEnvInt("CHORD_URL_SHORTENING_SERVICE_PORT", 8080)
var CHORD_URL_SHORTENING_SERVICE_PORT_GRPC int = utils.GetEnvInt("CHORD_URL_SHORTENING_SERVICE_PORT_GRPC", 50051)

type KVPair struct {
	key string
	val string
}

type Hash [32]byte

type IPAddress string

func (ip IPAddress) getID() Hash {
	return sha256.Sum256([]byte(ip))
}

// type NodePointer struct {
// 	ipAddress string
// 	id        Hash
// }

// func (np *NodePointer) isNil() bool {
// 	return *np == NodePointer{}
// }

type Node struct {
	Data map[Hash]KVPair
	// id          Hash
	IpAddress   IPAddress
	FingerTable []IPAddress
	Pred        IPAddress
	Succ        IPAddress
	mu          sync.Mutex
}

func CreateNode(podIP IPAddress) *Node {
	ipAddress := podIP
	newNode := Node{
		Data: make(map[Hash]KVPair),
		// id:        sha256.Sum256([]byte(ipAddress)),
		IpAddress: ipAddress,
	}
	return &newNode
}

func ToUInt64(h Hash) uint64 {
	toF64 := float64(binary.BigEndian.Uint64(h[:8]))
	modded := float64(math.Pow(2, 32))
	res := math.Mod(toF64, modded)
	return uint64(res)
}

func (h1 Hash) Compare(h2 Hash) int {
	c1 := ToUInt64(h1)
	c2 := ToUInt64(h2)
	if c1 < c2 {
		return -1
	} else if c1 > c2 {
		return 1
	} else {
		return 0
	}
}

func (n *Node) IdBetween(id Hash) bool {
	nID := n.IpAddress.getID()
	succID := n.Succ.getID()
	if nID.Compare(succID) == 0 {
		return true
	} else if nID.Compare(succID) == -1 {
		return nID.Compare(id) == -1 && (id.Compare(succID) == -1 || id.Compare(succID) == 0)
	} else {
		return nID.Compare(id) == -1 || (id.Compare(succID) == -1 || id.Compare(succID) == 0)
	}
}

// func (n *Node) GetOtherPodIP() IPAddress {
// 	// Set up a grpc connection to another pod via cluster IP (which pod is dependent on how k8s load balances, since using cluster IP, can be itself)
// 	conn, err := grpc.NewClient(
// 		fmt.Sprintf("%s:%d", CHORD_URL_SHORTENING_SERVICE_HOST, CHORD_URL_SHORTENING_SERVICE_PORT_GRPC),
// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
// 	)

// 	if err != nil {
// 		log.Printf("Did not connect: %v", err)
// 	}
// 	defer conn.Close()

// 	client := pb.NewNodeServiceClient(conn)

// 	// Perform gRPC call to get the IP of another pod
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 	defer cancel()

// 	req := &pb.GetIpRequest{NodeId: string(n.ipAddress)}
// 	res, err := client.GetNodeIp(ctx, req)
// 	if err != nil {
// 		log.Printf("Could not get other pod's IP: %v", err)
// 	}

// 	return IPAddress(res.String())
// }

func (n *Node) rpcCall(np IPAddress, funcName string) {
	client, err := rpc.Dial("tcp", string(n.IpAddress))
	if err != nil {
		log.Fatal("Dialing error:", err)
	}
	defer client.Close()

	var reply struct{}
	err = client.Call(funcName, struct{}{}, &reply)
	if err != nil {
		log.Fatal("RPC call error:", err)
	}
}

// func (n *Node) FindSuccessor(id Hash) IPAddress {
// 	if IdBetween(id, n) {
// 		return n.succ
// 	} else {
// 		highestPredOfId := n.ClosestPrecedingNode(id)

// 		// grpc call here to the other node to find findSuccessor?
// 		return highestPredOfId.FindSuccessor(id)
// 	}
// }

func (n *Node) ClosestPrecedingNode(id Hash) IPAddress {
	for i := len(n.FingerTable) - 1; i >= 0; i-- {
		finger := n.FingerTable[i]
		if finger != nil && n.id.Compare(finger.id) == -1 && finger.id.Compare(id) == -1 {
			return finger
		}
	}
	return n
}

// Each incoming node checks 5 times, if its IP is returned 5 times, return ”
func (n *Node) CheckIfRingExists() IPAddress {
	for i := 0; i < 5; i++ {
		otherIP := n.GetOtherPodIP()
		if otherIP != n.IpAddress {
			return otherIP
		}
	}
	return ""
}

func (n *Node) JoinRingIfExistsElseCreateRing() {
	existingRingNodeIP := n.CheckIfRingExists()
	if existingRingNodeIP != n.IpAddress {
		n.JoinRing(existingRingNodeIP)
	} else {
		n.CreateRing()
	}
}

func (n *Node) JoinRing(existingRingNodeIP IPAddress) {
	n.Pred = ""
	//
	n.Succ = existingRingNodeIP.FindSuccessor(n.id)
}

func (n *Node) CreateRing() {
	n.Pred = ""
	n.Succ = n.IpAddress
	fmt.Print("Node %d created a new ring.\n", n.id)
}

func (n *Node) stabilize() {
	succPred := n.Succ.pred
}

func (n *Node) Notify(nNode IPAddress) {
	if n.Pred == nil || (n.Pred.id.Compare(nNode.id) == -1 && nNode.id.Compare(n.id) == -1) {
		n.Pred = nNode
	}
}

// var globalNodeIPAddress int

// func getGlobalNodeIPAddress() string {
// 	globalNodeIPAddress++
// 	return strconv.Itoa(globalNodeIPAddress)
// }

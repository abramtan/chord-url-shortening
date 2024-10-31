package chord

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net/rpc"
	"sync"
	"time"

	"chord-url-shortening/internal/utils"

	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"

	pb "chord-url-shortening/chordurlshortening"
)

type KVPair struct {
	key string
	val string
}

type Hash [32]byte

type IPAddress string

func (ip IPAddress) getID() Hash {
	return sha256.Sum256([]byte(ip))
}

func (ip IPAddress) isNil() bool {
	return ip == ""
}

// type NodePointer struct {
// 	ipAddress string
// 	id        Hash
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

func (n *Node) IdBetween(ipAddress IPAddress) bool {
	nID := n.IpAddress.getID()
	succID := n.Succ.getID()
	id := ipAddress.getID()
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

func (n *Node) ClosestPrecedingNode(id IPAddress) IPAddress {
	idHash := id.getID()
	for i := len(n.FingerTable) - 1; i >= 0; i-- {
		finger := n.FingerTable[i]
		fingerID := n.FingerTable[i].getID()
		if finger.isNil() && n.IpAddress.getID().Compare(idHash) == -1 && fingerID.Compare(idHash) == -1 {
			return finger
		}
	}
	return n.IpAddress
}

// Each incoming node checks 5 times, if its IP is returned 5 times, return ”
func (n *Node) CheckIfRingExists() IPAddress {
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second*1)
        
        otherClient := utils.GetClientPod(utils.CHORD_URL_SHORTENING_SERVICE_HOST)
		req := &pb.GetIpRequest{NodeId: string(n.IpAddress)}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		otherIP, err := otherClient.GetNodeIp(ctx, req)
		if err != nil {
			log.Printf("Could not get other pod's IP: %v", err)
		}
		if IPAddress(otherIP.IpAddress) != n.IpAddress {
			return IPAddress(otherIP.IpAddress)
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
	// create client pod
	// call clientpod (existingRingNodeIP)
	client := utils.GetClientPod(string(existingRingNodeIP))
	req := &pb.FindSuccessorRequest{Id: string(existingRingNodeIP)}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := client.FindSuccessor(ctx, req)
	if err != nil {
		log.Printf("Join Ring - could not get find sucessor: %v", err)
	}
	n.Succ = IPAddress(res.Successor)
}

func (n *Node) CreateRing() {
	n.Pred = ""
	n.Succ = n.IpAddress
	fmt.Print("Node %d created a new ring.\n", n.IpAddress.getID())
}

// func (n *Node) stabilize() {
// 	succPred := n.Succ.pred
// }

// func (n *Node) Notify(nNode IPAddress) {
// 	if n.Pred == nil || (n.Pred.id.Compare(nNode.id) == -1 && nNode.id.Compare(n.id) == -1) {
// 		n.Pred = nNode
// 	}
// }

// var globalNodeIPAddress int

// func getGlobalNodeIPAddress() string {
// 	globalNodeIPAddress++
// 	return strconv.Itoa(globalNodeIPAddress)
// }

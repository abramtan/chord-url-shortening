package proj

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"net/rpc"
	"strconv"
	"sync"
	
)

const NUMBER_OF_NODES = 5

type KVPair struct {
	key string
	val string
}

type Hash [32] byte

type NodePointer struct {
    ipAddress string
    id Hash
}

type Node struct {
	data map [Hash] KVPair
	id Hash
    ipAddress string
    fingerTable []NodePointer
    pred NodePointer
	succ NodePointer
	mu sync.Mutex
}

func createNode() *Node {
    ipAddress := getGlobalNodeIPAddress()
	newNode := Node {
		data: make(map [Hash] KVPair),
        id: sha256.Sum256([]byte(ipAddress)),
        ipAddress: ipAddress,
	}
    return &newNode
}

func toUInt64(h Hash) uint64 {
    toF64 := float64(binary.BigEndian.Uint64(h[:8]))
    modded := float64(math.Pow(2, 32))
    res := math.Mod(toF64, modded)
    return uint64(res)
}

func (h1 Hash) Compare(h2 Hash) int {
    c1 := toUInt64(h1)
    c2 := toUInt64(h2)
    if c1 < c2 {
        return -1
    } else if c1 > c2 {
        return 1
    } else {
        return 0
    }
}

func idBetween(id Hash, n *Node) bool {
    nID := n.id
    succID := n.succ.id
    if nID.Compare(succID) == 0 {
        return true
    } else if nID.Compare(succID) == -1 {
        return nID.Compare(id) == -1 && (id.Compare(succID) == -1 || id.Compare(succID) == 0)
    } else {
        return nID.Compare(id) == -1 || (id.Compare(succID) == -1 || id.Compare(succID) == 0)
    }
}

func (n *Node) initRPC() {
	// create and bind to a tcp port
	l, err := net.Listen("tcp", n.ipAddress) //?
		
}

func (n *Node) rpcCall(np NodePointer, funcName String) {
    client, err := rpc.Dial("tcp", np.ipAddress)
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

func ()

func (n *Node) findSuccessor(id Hash) NodePointer {
    if idBetween(id, n) {
        return n.succ
    } else {
        highestPredOfId := n.closestPrecedingNode(id)

        return highestPredOfId.findSuccessor(id)
    }
}

func (n *Node) closestPrecedingNode(id Hash) NodePointer {
	for i := len(n.fingerTable)-1; i >= 0; i-- {
		finger := n.fingerTable[i]
		if finger != nil && n.id.Compare(finger.id) == -1 && finger.id.Compare(id) == -1 {
			return finger
		}
	}
	return n
}

func (n *Node) joinRing(existingNode NodePointer) {
	if existingNode == nil {
		n.createRing()
	}
    n.pred = nil
    n.succ = existingNode.findSuccessor(n.id)
}

func (n *Node) createRing() {
	n.pred = nil
	n.succ = n
	fmt.Print("Node %d created a new ring.\n", n.id)
}

func (n *Node) stabilize() {
    succPred := n.succ.pred
}

func (n *Node) Notify (nNode NodePointer) {
	if n.pred==nil || (n.pred.id.Compare(nNode.id) == -1 && nNode.id.Compare(n.id) == -1) {
		n.pred = nNode
	}
}

var globalNodeIPAddress int
func getGlobalNodeIPAddress() string {
    globalNodeIPAddress++
    return strconv.Itoa(globalNodeIPAddress)
}

func main() {
    
	

}
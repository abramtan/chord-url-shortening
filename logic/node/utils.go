package node

// Message types.
const (
	PING                   = "ping"                   // Used to check predecessor.
	ACK                    = "ack"                    // Used for general acknowledgements.
	GET_SUCCESSOR          = "get_successor"          // Used in RPC call to get node.Successor
	FIND_SUCCESSOR         = "find_successor"         // Used to find successor.
	CLOSEST_PRECEDING_NODE = "closest_preceding_node" // Used to find the closest preceding node, given a successor id.
	GET_PREDECESSOR        = "get_predecessor"        // Used to get the predecessor of some node.
	NOTIFY                 = "notify"                 // Used to notify a node about a new predecessor.
	EMPTY                  = "empty"                  // Placeholder or undefined message type or errenous communications.
	JOIN                   = "join"                   // testing the join function
)

type RMsg struct {
	MsgType       string
	OutgoingIP    IPAddress // Sender IP
	IncomingIP    IPAddress // Receiver IP
	QueryResponse []string
	Payload       []IPAddress
}

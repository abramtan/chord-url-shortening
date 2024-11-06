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
	STORE_URL              = "store_url"              // Used to store a url in the node.
	RETRIEVE_URL           = "retrieve_url"           // Used to retrieve a url from the node.
	CLIENT_STORE_URL       = "client_store_url"       // Client tells node to store a single short/long URL pair
	CLIENT_RETRIEVE_URL    = "client_retrieve_url"    // Client tells node to retrieve a single short/long URL pair
)

type RMsg struct {
	MsgType       string
	SenderIP      HashableString // Sender IP
	RecieverIP    HashableString // Receiver IP
	TargetHash    Hash           // Hash Value of the value to be found (shortURL or IP Address )
	TargetIP      HashableString // IP of the Found Node
	StoreEntry    Entry          // for passing the short/long URL pair to be stored for a ShortURL request
	RetrieveEntry Entry          // for passing the retrieved longURL for a RetrieveURL request
	// ClientIP        HashableString
	// QueryResponse []string         // ?
}

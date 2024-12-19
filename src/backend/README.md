# Demonstration Overview

Implementation of a TinyURL service.
This project implements a distributed system using the Chord Protocol, with a TinyURL-like application
as the use case. The Chord protocol is a peer-to-peer distributed hash table (DHT) that helps with efficient lookup, insertion and replication of data across a network of nodes. Our TinyURL application leverages this decentralised architecture to provide URL shortening services with fault tolerance, scalability and consistency.

### Storing URLs (`node.StoreURL`)

The process of storing a long URL - short URL pair within the Chord ring follows these steps:

1. Client Request: The `ClientSendStoreURL` function is called to initiate a URL storage request. The storage request consists of a short URL, long URL and its timestamp.
2. RPC Calling: The request is sent via an RPC message to the target node in the Chord ring
3. Incoming Requests: The receiving node processes the RPC request through the `HandleIncomingMessage` method which identifies the message type and routes it to the `StoreURL` function.
4. Storing data: The method saves the URL in the node's local data store (URLMap). It also updates the cache to include the new URL for quicker retrieval during future requests.

### Retrieving URL (`node.RetrieveURL`)

The process of storing retrieving a short URLs within the Chord ring follows these steps:

1. Client Request: Client sends a request to retrieve a URL by calling the `ClientRetrieveURL` function with the Short URL
2. RPC Calling: The request is sent via an RPC message to the target node in the Chord ring
3. Routing the Request: The request is routed through the `HandleIncomingMessage` method which identifies the message type and forwards it to the `RetrieveURL` method for retrieval
4. Retrival: The `RetrieveURL` method first checks the cache for the URL. If it already exists in the cache (cache hit), it is returned immediately. However, if there is a cache miss, it will first check the primary node responsible for the shortURL. If it is not present in the primary node's local store, it will check the replicas which are stored in the primary node's successor list. This has been implemented for fault tolerance.
5. Updating the Cache: When the URL has been retrieved from the primary node or the replica, the cache is updated with the retrieved data, especially the timestamp.

### Additional features on top of the basic Chord functionalities

#### Successor List

To increase robustness, each Chord node maintains a successor list of size `r`, and it contains the first `r` successors. If the immediate successor does not respond, the node can substitute the next entry in the successor list.

#### Data Replication

We have implemented data replication as part of our fault tolerance as this ensures high availability by storing multiple copies of data on successor nodes. We have also used timestamp-based conflict resolution to ensure consistency during updates.

#### Cache

Our application includes a local caching mechanism to optimise URL retrieval and reduce lookup latency. The cache is designed to enhance performance by minimising the need to query the distributed Chord ring for frequenty accessed URLs. Each node maintains a local cache as a key-value store (shortURL: longURL, timestamp) along with other key features:

- Size Limit: The cache has a limit of 10 entries per node and uses the LRU policy to evict the oldest entries when the cache is full
- Cache Lookup:
  - Cache Hit: URL is returned immediately, which means hot count is also reduced
  - Cache Miss: Request to retrieve URL is forwarded to the Chord ring for resolution

# Demo Walkthrough

## Network Creation

Start the interactive menu and enter `ADD`. This will cause the first node to be added, which will call `CreateNetwork()`.

**Correctness:** Enter `SHOW` to see the network's status. It should only show the one node, with its successor and predecessor both set to itself.

## Joining Network

Enter `ADD` three more times. This will cause more nodes to be added, which will call `JoinNetwork()`.

**Correctness:** Enter `SHOW` to see the network's status. It should now show all nodes that have been created thus far. The nodes should all have the correct successors and predecessors, such that it forms a ring.

## Successor List and Data Replication

Enter `ADD` four more times to add some more nodes, for seven in total. Then, enter `STORE` followed by any URL to store it in the network.

**Correctness:** Enter `SHOW` to see the network's status. All nodes except 1 (since we have a successor list length of 5) should have some entry containing the URL.

## Cache

_Continued from "Successor List" above._

**Correctness:** The node that receives the `STORE` request should also have a cache entry for the URL.

# Fault Tolerance

## Voluntary Leaving

In the interactive menu, enter `DEL`, followed by the IP address of the node you wish to trigger to voluntarily leave. The node will send messages to its successor and predecessor to facilitate the voluntary leaving procedure, and then freeze.

**Correctness:** Enter `SHOW` to see the network's status. The node will still show up in the menu with successors and predecessors unchanged (since `DEL` essentially freezes the node). The leaving node's successor predecessor should be set to the leaving node's predecessor, and it should have obtained the keys from the leaving node. The leaving node's predecessor successor list should now have the last node of the leaving node's successor list.

## Fail-Stop Faults: Permanent Fault

In the interactive menu, enter `FAULT`, followed by the IP address of the node you wish to fault. The node will simply freeze.

> The main differences between `FAULT` and `DEL` are:
>
> - `FAULT` does not initiate voluntary leaving procedure, while `DEL` does.
>   - The voluntary leaving procedure is an optimization that allows the Chord network to more predictably account for the leaving node than if it had simply frozen. However, both should eventually be successfully recovered from.
> - In our interactive menu, nodes that get `FAULT`ed can be recovered with `FIX` (elaborated on later), while `DEL` is permanent.

**Correctness:** Enter `SHOW` to see the network's status. The node will still show up in the menu with successors and predecessors unchanged (since `FAULT` essentially freezes the node), but the successor and predecessor of the node that left should themselves have changed their own predecessor resp. successor to form a new, fixed ring.

## Fail-Stop Faults: Intermittent Fault

In the interactive menu, enter `FAULT`, followed by the IP address of another node you wish to fault. The node will simply freeze.

**Correctness:** Same as permanent fault.

Then, enter `FIX`, followed by the IP address of the same node. The node will unfreeze and restart accepting RPC calls.

**Correctness:** Enter `SHOW` to see the network's status. The successor and predecessor of the fixed node should now themselves also list their predecessor resp. successor as the fixed node.

# Scalability

Our main mechanisms for addressing scalability are:

- The Chord system itself
- Performance of our cache

Therefore, we plan to perform some experiments that examine both these aspects.

## Experiments

In the menu, enter `EXPERIMENT` to run an experiment. The experiments are designed to be self-contained, and should be run in isolation of other menu commands. On completion of an experiment and to run another experiment, the script has to be restarted.

### Independent Variables

- Number of nodes
- Number of URLs stored
  - The pool of URLs stored in the network, that can possibly be retrieved during a certain experiment.
- Number of retrieval requests made
  - The number of retrieval requests made.
  - Each request will target a random URL from the pool of URLs stored.

We separately specify the number of URLs stored and the number of retrieval requests made, because we often wish to control the amount of retrieval requests that are made to the same URL.

For instance, if we have 100 URLs stored and 100 retrieval requests made, the average number of requests per URL will be 1.

If, on the other hand, we have 10 URLs stored and 100 retrieval requests made, each URL will on average be retrieved 10 times. This allows us to inspect the impact of the cache, which mostly improves performance of repeat retrievals of the same URL.

### Dependent Variables

- Average time taken per URL retrieval
- Average number of RPC calls per URL retrieval

The above two metrics are recorded once with the cache activated, and another time without. In this way, we may examine the impact of having a cache.

### Experiment 1: Impact of number of nodes in the Chord Ring on the retrieval time

> This experiment is to test how the number of calls made and the time taken scales with an increasing number of nodes in the system.

Independent variable:

- The number of nodes in the Chord Ring

Fixed variables:

- Quantity of URLs: 1000
- Quantity of retrieval calls: 2000

### Experiment 2: Impact of proportion of retrievals which are repeats in the Chord Ring on the retrieval time

> This experiment is to test the difference made by the cache, since the cache comes into play when retrievals are repeated for the same URL.

Independent variable:

- Quantity of URLs stored

Fixed variables:

- Number of nodes in Chord ring
- Quantity of retrieval calls: 2000

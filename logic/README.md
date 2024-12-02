## Currently Implemented
1. **Ring**
   1. `init_node`, `node.create`, `node.join`, `node.initSuccList`
2. **Key Lookup :**
   1. `node.find_successor`, `node.closest_preceding_node`
3. **Periodic Func**
   1. `node.notify`, `node.stabilise`, `node.fix_fingers`, `node.maintainSuccList`
4. **Key-Value Lookups**
   1. `node.storeUrl`, `node.retrieveUrl`
5. **Client Nodes**
   1. `clientNode.sendStoreUrl`, `clientNode.sendRetrieveUrl`, `initClientNode`
6. **Node Failure**
   1. `node.storeReplica` 
7. **Voluntary Node Departure** 
   1. `node.voluntaryLeavingSuccessor`, `voluntaryLeavingPredecessor`, `node.Leave`     

## Fault Tolerance Plan
1. Fault Definition:
   1. Node Failure
   2. Communication Failure
   3. Replica Failures
   4. Lookup Failure
2. Impact of Faults:
   1. Data Unavailability
   2. Ring Inconsistency
3. Fault Tolerance Goals:   
   1. Data Reliability
   2. Ring Resilience
   3. Fault Detection
   4. System Scalability

## To Implement
1. Refinements
   1. Dynamic IPs and Dynamic Join
2. Byzantine Fault Tolerance
3. Cache

## Additional Considerations
- Docker Containers
- Handle Kubernetes shifting IP
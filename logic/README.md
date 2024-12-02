## Currently Implemented
1. **Ring**
   1. `init_node`, `node.create`, `node.join`
2. **Key Lookup :**
   1. `node.find_successor`, `node.closest_preceding_node`
3. **Periodic Func**
   1. `node.notify`, `node.stabilise`, `node.fix_fingers`
4. **Key-Value Lookups**
   1. `node.storeUrl`, `node.retrieveUrl`
5. **Client Nodes**
   1. `clientNode.sendStoreUrl`, `clientNode.sendRetrieveUrl`, `initClientNode`
6. **Node Failure**
   1. `node.storeReplica` 
7. **Voluntary Node Departure** 
   1. `node.voluntaryLeavingSuccessor`, `voluntaryLeavingPredecessor`, `node.Leave`     

## To Implement
1. node.check_predecessor
2. Successor List
3. Data Replication
4. Vountary Node Departures (Shift Data, Inform Pred/Succ)
5. Refinements
   1. Dynamic IPs and Dynamic Join
   2. Error Handling - node failures, etc.
6. Byzantine Fault Tolerance


## Additional Considerations
- Docker Containers
- Handle Kubernetes shifting IP
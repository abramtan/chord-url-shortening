## Currently Implemented
<ul>
    <li><strong>Ring</strong> : initnode, node.create, node.join</li>
    <li><strong>Key Lookup</strong> : node.find_successor, node.closest_preceding_node</li>
    <li><strong>Periodic Func</strong> : node.notify, node.stabilise, node.fix_fingers</li>
    <li><strong>Key-Value Lookups</strong> : node.storeUrl, node.retrieveUrl</li>
    <li><strong>Client Nodes</strong> : clientNode.sendStoreUrl, clientNode.sendRetrieveUrl, initClientNode</li>
</ul>

## To Implement
<ul>
    <li>node.check_predecessor</li>
    <li>Successor List</li>
    <li>Data Replication</li>
    <li> Refinements
        <ul>
            <li>Dynamic IPs and Dynamic Join</li>
            <li>Error Handling - node failures, etc.</li>
        </ul>
    </li>
</ul>

## Additional Considerations
<ul>
    <li>Docker Containers</li>
    <li>Handle Kubernetes shifting IP</li>
</ul>
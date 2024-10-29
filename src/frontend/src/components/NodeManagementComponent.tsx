// components/NodeManagementComponent.tsx
'use client';  // Client-side logic

import { useState } from 'react';

export default function NodeManagementComponent() {
  const [nodeId, setNodeId] = useState('');
  const [message, setMessage] = useState('');

  const handleAddNode = async () => {
    const res = await fetch('/api/nodes', { method: 'POST' });
    const data = await res.json();
    setMessage(`Node added: ${data.nodeId}`);
  };

  const handleRemoveNode = async () => {
    const res = await fetch(`/api/nodes?nodeId=${nodeId}`, { method: 'DELETE' });
    const data = await res.json();
    setMessage(data.message);
  };

  return (
    <div>
      <button onClick={handleAddNode}>Add Node</button>

      <div>
        <input
          type="text"
          value={nodeId}
          onChange={(e) => setNodeId(e.target.value)}
          placeholder="Enter Node ID to remove"
        />
        <button onClick={handleRemoveNode}>Remove Node</button>
      </div>

      {message && <p>{message}</p>}
    </div>
  );
}

// components/VisualizationComponent.tsx
'use client';  // Client-side logic

import { useEffect, useState } from 'react';
import { setupWebSocket } from '../lib/websocket';

export default function VisualizationComponent() {
  const [nodeData, setNodeData] = useState([]);

  useEffect(() => {
    const ws = setupWebSocket((data) => {
      setNodeData(data);
    });
    return () => ws.close();  // Cleanup on component unmount
  }, []);

  return (
    <div>
      <h3>Real-time Chord Node Data Movement</h3>
      {/* Replace this with actual visualizations (e.g., using D3.js) */}
      <pre>{JSON.stringify(nodeData, null, 2)}</pre>
    </div>
  );
}

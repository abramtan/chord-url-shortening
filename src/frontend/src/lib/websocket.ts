// src/lib/websocket.ts
export function setupWebSocket(onMessage: (data: any) => void): WebSocket {
    const wsUrl = process.env.NEXT_PUBLIC_WEBSOCKET_URL || 'ws://localhost:3000/ws';
    const ws = new WebSocket(wsUrl);
  
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      onMessage(data);
    };
  
    return ws;
  }
  
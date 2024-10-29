// app/api/nodes/route.ts
import { NextResponse } from 'next/server';

export async function POST() {
  const nodeId = await addNodeToChordRing();
  return NextResponse.json({ nodeId });
}

export async function DELETE(request: Request) {
  const { searchParams } = new URL(request.url);
  const nodeId = searchParams.get('nodeId');
  
  if (!nodeId) {
    return NextResponse.json({ error: 'Node ID is required' }, { status: 400 });
  }

  const message = await removeNodeFromChordRing(nodeId);
  return NextResponse.json({ message });
}

async function addNodeToChordRing() {
  // Call Go backend to add a node
  return 'newNodeId123';  // Replace with actual node ID
}

async function removeNodeFromChordRing(nodeId: string) {
  // Call Go backend to remove a node
  return `Node ${nodeId} removed successfully`;  // Replace with actual removal message
}

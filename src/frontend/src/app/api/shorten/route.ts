// src/app/api/shorten/route.ts
import { NextResponse } from 'next/server';

export async function POST(request: Request) {
  const { longUrl } = await request.json();
  const shortUrl = await shortenUrlInBackend(longUrl);

  return NextResponse.json({ shortUrl });
}

async function shortenUrlInBackend(longUrl: string) {
  const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:3001';
  const res = await fetch(`${backendUrl}/api/v1/shorten`, {
    method: 'POST',
    body: JSON.stringify({ longUrl }),
    headers: { 'Content-Type': 'application/json' },
  });
  const data = await res.json();
  return data.shortUrl;
}

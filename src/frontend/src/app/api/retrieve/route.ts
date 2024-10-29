// app/api/retrieve/route.ts
import { NextResponse } from 'next/server';

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const shortUrl = searchParams.get('shortUrl');
  if (!shortUrl) {
    return NextResponse.json({ error: 'Short URL is required' }, { status: 400 });
  }
  const longUrl = await retrieveOriginalUrl(shortUrl);

  return NextResponse.json({ longUrl });
}

async function retrieveOriginalUrl(shortUrl: string) {
  // Call Go backend to retrieve the original URL
  return 'http://original-url.com';  // Replace with actual original URL
}

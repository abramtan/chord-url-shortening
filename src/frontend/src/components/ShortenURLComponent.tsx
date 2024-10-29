// src/components/ShortenURLComponent.tsx
'use client';

import { useState } from 'react';

export default function ShortenURLComponent() {
  const [longUrl, setLongUrl] = useState('');
  const [shortUrl, setShortUrl] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const res = await fetch('/api/shorten', {
      method: 'POST',
      body: JSON.stringify({ longUrl }),
      headers: { 'Content-Type': 'application/json' },
    });

    const data = await res.json();
    setShortUrl(data.shortUrl);
  };

  return (
    <div className="space-y-4">
      <form onSubmit={handleSubmit} className="flex space-x-2">
        <input
          type="url"
          value={longUrl}
          onChange={(e) => setLongUrl(e.target.value)}
          placeholder="Enter your long URL"
          className="border border-gray-300 rounded-md px-3 py-2 w-full"
          required
        />
        <button
          type="submit"
          className="bg-blue-600 text-white px-4 py-2 rounded-md"
        >
          Shorten URL
        </button>
      </form>
      {shortUrl && (
        <p>
          Shortened URL: <a href={shortUrl} className="text-blue-600">{shortUrl}</a>
        </p>
      )}
    </div>
  );
}

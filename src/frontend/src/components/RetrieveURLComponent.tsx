// components/RetrieveURLComponent.tsx
'use client';  // Client-side logic

import { useState } from 'react';

export default function RetrieveURLComponent() {
  const [shortUrl, setShortUrl] = useState('');
  const [longUrl, setLongUrl] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const res = await fetch(`/api/retrieve?shortUrl=${shortUrl}`);
    const data = await res.json();
    setLongUrl(data.longUrl);
  };

  return (
    <div>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={shortUrl}
          onChange={(e) => setShortUrl(e.target.value)}
          placeholder="Enter your shortened URL"
          required
        />
        <button type="submit">Retrieve URL</button>
      </form>
      {longUrl && <p>Original URL: <a href={longUrl}>{longUrl}</a></p>}
    </div>
  );
}

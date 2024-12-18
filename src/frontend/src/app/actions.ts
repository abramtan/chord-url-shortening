'use server'

export async function shortenUrl(longUrl: string): Promise<string> {
  // Call the backend API to generate the short URL
  const response = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/shorten`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ long_url: longUrl }),
  });

  if (!response.ok) {
    throw new Error("Failed to shorten URL");
  }

  const data = await response.json();
  return data.short_url; // Return the generated short URL
}

export async function getLongUrl(id: string): Promise<string> {
  // Call the backend API to retrieve the original long URL
  const response = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/long-url/${id}`, {
    method: "GET",
  });

  if (!response.ok) {
    // Return a fallback URL if the short URL does not exist
    return "https://google.com";
  }

  const data = await response.json();
  return data.long_url; // Return the retrieved long URL
}
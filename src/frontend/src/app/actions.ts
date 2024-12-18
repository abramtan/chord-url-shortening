'use server'

export async function shortenUrl(longUrl: string): Promise<string> {
  // Call the backend API to generate the short URL
  const response = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/store`, {
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
  try {
    // Call the backend API to retrieve the original long URL
    const response = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/retrieve?short_url=${encodeURIComponent(id)}`, {
      method: "GET",
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "Failed to fetch data");
    }

    const data = await response.json();
    return data.long_url; // Return the retrieved long URL
  } catch (error: any) {
    console.error("Error fetching short URL data:", error.message);
    throw error;
  }
}
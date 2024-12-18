from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel, HttpUrl
from typing import Optional
import random
import string
from datetime import datetime

app = FastAPI()

# In-memory database (use an actual database in production)
db = {}

class URL(BaseModel):
    long_url: HttpUrl
    short_code: Optional[str] = None  # Optional custom short code

def generate_short_code(length: int = 6) -> str:
    """Generate a random short code of fixed length."""
    chars = string.ascii_letters + string.digits
    return ''.join(random.choices(chars, k=length))

@app.post("/shorten")
async def shorten_url(url: URL):
    """
    Shorten a long URL.
    If `short_code` is provided, use it. Otherwise, generate a random code.
    """
    # Validate custom short code if provided
    if url.short_code:
        if url.short_code in db:
            raise HTTPException(status_code=400, detail="Short code already exists.")
        short_code = url.short_code
    else:
        # Generate a unique short code
        short_code = generate_short_code()
        while short_code in db:
            short_code = generate_short_code()

    # Store the mapping
    db[short_code] = {
        "long_url": url.long_url,
        "created_at": datetime.utcnow(),
    }

    # Construct the short URL
    base_url = "http://localhost:3000"  # Replace with your domain
    short_url = f"{base_url}/{short_code}"
    return {"short_url": short_url, "short_code": short_code}

@app.get("/long-url/{short_code}")
async def get_long_url(short_code: str):
    """
    Retrieve the original long URL for a given short code.
    """
    if short_code not in db:
        raise HTTPException(status_code=404, detail="Short URL not found.")

    return {"long_url": db[short_code]["long_url"]}

@app.get("/")
async def home():
    """Simple home endpoint for testing."""
    return {"message": "Welcome to the URL Shortener API"}
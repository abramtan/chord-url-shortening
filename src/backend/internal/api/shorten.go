package api

import (
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ShortenRequest struct {
	LongURL   string `json:"long_url" binding:"required"`
	ShortCode string `json:"short_code,omitempty"`
}

type ShortenResponse struct {
	ShortURL  string `json:"short_url"`
	ShortCode string `json:"short_code"`
}

type LongURLResponse struct {
	LongURL string `json:"long_url"`
}

var (
	urlDatabase = make(map[string]string) // In-memory storage for short to long URL mappings
	mutex       = &sync.Mutex{}           // Mutex to handle concurrent map access
	baseURL     = "http://localhost:3000" // Base URL for short URLs
)

func ShortenURL(c *gin.Context) {
	var req ShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Check if the custom short code already exists
	if req.ShortCode != "" {
		if _, exists := urlDatabase[req.ShortCode]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Short code already exists"})
			return
		}
	} else {
		// Generate a unique short code
		req.ShortCode = GenerateShortCode(6)
		for _, exists := urlDatabase[req.ShortCode]; exists; _, exists = urlDatabase[req.ShortCode] {
			req.ShortCode = GenerateShortCode(6)
		}
	}

	// Store the mapping in the in-memory database
	urlDatabase[req.ShortCode] = req.LongURL

	// Construct the short URL
	shortURL := baseURL + "/" + req.ShortCode
	c.JSON(http.StatusOK, ShortenResponse{
		ShortURL:  shortURL,
		ShortCode: req.ShortCode,
	})
}

func GetLongURL(c *gin.Context) {
	shortCode := c.Param("short_code")

	mutex.Lock()
	defer mutex.Unlock()

	// Lookup the long URL for the given short code
	longURL, exists := urlDatabase[shortCode]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
		return
	}

	c.JSON(http.StatusOK, LongURLResponse{LongURL: longURL})
}

// generateShortCode generates a random alphanumeric string of the specified length
func GenerateShortCode(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(code)
}

package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var defaultRoom = "hn"

// LoadDefaultRoomFromEnv loads default room from environment variable
func LoadDefaultRoomFromEnv() {
	if room := os.Getenv("DEFAULT_ROOM"); room != "" {
		// Validate room name format
		if len(room) > 0 && len(room) <= 20 {
			defaultRoom = room
		} else {
			fmt.Printf("Invalid DEFAULT_ROOM value: %s (must be 1-20 chars), using default\n", room)
		}
	}
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	LogInterval    int
	BlockThreshold int
}

var rateLimitConfig = RateLimitConfig{
	LogInterval:    50,
	BlockThreshold: 200,
}

// LoadRateLimitConfigFromEnv loads rate limit configuration from environment variables
func LoadRateLimitConfigFromEnv() {
	if envVal := os.Getenv("RATE_LIMIT_LOG_INTERVAL"); envVal != "" {
		val, err := strconv.Atoi(envVal)
		if err != nil || val <= 0 {
			fmt.Printf("Warning: Invalid RATE_LIMIT_LOG_INTERVAL value: %s (must be a positive integer)\n", envVal)
		} else {
			rateLimitConfig.LogInterval = val
		}
	}
	if val, err := strconv.Atoi(os.Getenv("RATE_LIMIT_BLOCK_THRESHOLD")); err == nil && val > 0 {
		rateLimitConfig.BlockThreshold = val
	}
}

var ipLastReset = sync.Map{} // Track when counters were last reset

func rateLimit(c *gin.Context) {
	ip := c.ClientIP()
	now := time.Now()

	// Check if we should reset the counter (e.g., every hour)
	mutexIps.Lock()
	lastReset, exists := ipLastReset.Load(ip)
	if !exists || now.Sub(lastReset.(time.Time)) > 1*time.Hour {
		ips.Reset()
		ipLastReset.Store(ip, now)
		mutexIps.Unlock()
		return
	}
	mutexIps.Unlock()

	mutexIps.Lock()
	value := int(ips.Add(ip, 1))
	mutexIps.Unlock()
	if value%rateLimitConfig.LogInterval == 0 {
		fmt.Printf("ip: %s, count: %d\n", ip, value)
	}
	if value >= rateLimitConfig.BlockThreshold {
		if value%rateLimitConfig.BlockThreshold == 0 {
			fmt.Println("ip blocked")
		}
		c.Abort()
		c.String(http.StatusServiceUnavailable, "you were automatically banned :)")
	}
}

func index(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/room/"+defaultRoom)
}

func roomGET(c *gin.Context) {
	roomid := c.Param("roomid")
	// Validate roomid
	// Only allow alphanumeric and some special characters
	validRoomID := len(roomid) >= 1 && len(roomid) <= 20
	if validRoomID {
		for _, char := range roomid {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '_') {
				validRoomID = false
				break
			}
		}
	}
	if !validRoomID {
		c.String(http.StatusBadRequest, "Invalid room ID")
		return
	}
	nick := c.Query("nick")
	if len(nick) < 2 {
		nick = ""
	}
	if len(nick) > 13 {
		nick = nick[0:12] + "..."
	}
	c.HTML(http.StatusOK, "room_login.templ.html", gin.H{
		"roomid":    roomid,
		"nick":      nick,
		"timestamp": time.Now().Unix(),
	})

}

// Track message timestamps per user per room
var userMessageTimestamps = struct {
	sync.RWMutex
	data map[string]time.Time
}{data: make(map[string]time.Time)}

// Constants for message rate limiting
const (
	messageRateLimit = 1 * time.Second // Minimum time between messages
)

func roomPOST(c *gin.Context) {
	roomid := c.Param("roomid")
	nick := c.Query("nick")
	message := c.PostForm("message")
	message = strings.TrimSpace(message)

	// Create a unique key for this user+room
	userRoomKey := roomid + ":" + nick

	// Check rate limiting for this specific user+room
	userMessageTimestamps.RLock()
	lastMsg, exists := userMessageTimestamps.data[userRoomKey]
	userMessageTimestamps.RUnlock()

	now := time.Now()
	if exists && now.Sub(lastMsg) < messageRateLimit {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"status": "failed",
			"error":  "you're sending messages too quickly",
		})
		return
	}

	// Update timestamp
	userMessageTimestamps.Lock()
	userMessageTimestamps.data[userRoomKey] = now
	userMessageTimestamps.Unlock()

	validMessage := len(message) > 1 && len(message) < 200
	validNick := len(nick) > 1 && len(nick) < 14
	if !validMessage || !validNick {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  "the message or nickname is too long",
		})
		return
	}

	post := gin.H{
		"nick":    html.EscapeString(nick),
		"message": html.EscapeString(message),
	}
	// Create and use a mutex for messages if not already existing
	mutexMessages.Lock()
	messages.Add("inbound", 1)
	mutexMessages.Unlock()
	room(roomid).Submit(post)
	c.JSON(http.StatusOK, post)
}

func streamRoom(c *gin.Context) {
	roomid := c.Param("roomid")
	listener := openListener(roomid)
	ticker := time.NewTicker(1 * time.Second)
	connectionTimeout := time.NewTimer(30 * time.Minute) // Set max connection time
	// Create and use a mutex for users if not already existing
	mutexUsers.Lock()
	users.Add("connected", 1)
	mutexUsers.Unlock()
	defer func() {
		closeListener(roomid, listener)
		ticker.Stop()
		connectionTimeout.Stop()
		mutexUsers.Lock()
		users.Add("disconnected", 1)
		mutexUsers.Unlock()
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg := <-listener:
			// Create and use a mutex for messages if not already existing
			mutexMessages.Lock()
			messages.Add("outbound", 1)
			mutexMessages.Unlock()
			c.SSEvent("message", msg)
		case <-ticker.C:
			c.SSEvent("stats", Stats())
		case <-c.Request.Context().Done():
			// Client disconnected
			return false
		case <-connectionTimeout.C:
			// Connection timeout reached
			c.SSEvent("system", gin.H{"message": "Connection timeout reached"})
			return false
		}
		return true
	})
}

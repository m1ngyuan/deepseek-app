package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
	if val, err := strconv.Atoi(os.Getenv("RATE_LIMIT_LOG_INTERVAL")); err == nil && val > 0 {
		rateLimitConfig.LogInterval = val
	}
	if val, err := strconv.Atoi(os.Getenv("RATE_LIMIT_BLOCK_THRESHOLD")); err == nil && val > 0 {
		rateLimitConfig.BlockThreshold = val
	}
}

func rateLimit(c *gin.Context) {
	ip := c.ClientIP()
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
	c.Redirect(http.StatusMovedPermanently, "/room/hn")
}

func roomGET(c *gin.Context) {
	roomid := c.Param("roomid")
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

func roomPOST(c *gin.Context) {
	roomid := c.Param("roomid")
	nick := c.Query("nick")
	message := c.PostForm("message")
	message = strings.TrimSpace(message)

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
	// Create and use a mutex for users if not already existing
	mutexUsers.Lock()
	users.Add("connected", 1)
	mutexUsers.Unlock()
	defer func() {
		closeListener(roomid, listener)
		ticker.Stop()
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
		}
		return true
	})
}

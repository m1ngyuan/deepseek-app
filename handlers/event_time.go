package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func RegisertEventTimeRoutes(router *gin.Engine) {
	router.GET("/events/time", timeStream)

}

func timeStream(c *gin.Context) {
	req := c.Request
	log.Println(req.Method)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")

	flush, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Something unsupported!")
		return
	}

	for {
		//fmt.Fprintf(c.Writer, "data: %s\n\n", time.Now().Format(time.RFC3339))
		marshal, _ := json.Marshal(map[string]string{"ccc": "ccc"})
		go c.SSEvent("message", marshal)
		flush.Flush()
		time.Sleep(5 * time.Second)
	}
}

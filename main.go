package main

import (
	"context"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"log"
	"os"
	"runtime"

	"github.com/gin-gonic/gin"
)

func main() {

	ConfigRuntime()
	StartWorkers()
	StartGin()

}

func validateAPIKey() {
	if apiKey, ok := os.LookupEnv("DEEPSEEK_API_KEY"); ok {
		baseURL := os.Getenv("DEEPSEEK_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.deepseek.com"
		}
		client := openai.NewClient(
			option.WithAPIKey(apiKey), // defaults to os.LookupEnv("OPENAI_API_KEY")
			option.WithBaseURL(baseURL),
		)
		stream := client.Chat.Completions.NewStreaming(context.TODO(), openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("你如何评价小米ultra"),
			},
			Model: "deepseek-chat",
		})
		// optionally, an accumulator helper can be used
		acc := openai.ChatCompletionAccumulator{}

		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)

			if content, ok := acc.JustFinishedContent(); ok {
				log.Println("Content stream finished:", content)
			}

			// if using tool calls
			if tool, ok := acc.JustFinishedToolCall(); ok {
				log.Println("Tool call stream finished:", tool.Index, tool.Name, tool.Arguments)
			}

			if refusal, ok := acc.JustFinishedRefusal(); ok {
				log.Println("Refusal stream finished:", refusal)
			}

			// it's best to use chunks after handling JustFinished events
			if len(chunk.Choices) > 0 {
				log.Println(chunk.Choices[0].Delta.Content)
			}
		}
		if err := stream.Err(); err != nil {
			log.Fatalf("Stream error: %v", err)
		}
		finalContent := acc.Choices[0].Message.Content

		log.Println(finalContent)
	} else {

		log.Fatal("environment variable DEEPSEEK_API_KEY is not set")
	}
}

// ConfigRuntime sets the number of operating system threads.
func ConfigRuntime() {
	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	log.Printf("Running with %d CPUs\n", nuCPU)
}

// StartWorkers start starsWorker by goroutine.
func StartWorkers() {
	go statsWorker()
}

// StartGin starts gin web server with setting router.
func StartGin() {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(rateLimit, gin.Recovery())
	router.LoadHTMLGlob("resources/*.templ.html")
	router.Static("/static", "resources/static")
	router.GET("/", index)
	router.GET("/room/:roomid", roomGET)
	router.POST("/room-post/:roomid", roomPOST)
	router.GET("/stream/:roomid", streamRoom)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("error: %s", err)
	}
}

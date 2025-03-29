package main

import (
	"context"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"log"
	"os"
)

func main() {
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
			Model: "deepseek-reasoner",
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

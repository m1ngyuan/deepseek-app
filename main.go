package main

import (
	"log"
	"os"
)

func main() {
	content, err := chat("你如何评价小米ultra")
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("./a.md")
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Content stream finished:", content)
}

package main

import "log"

func main() {
	content, eror := chat()
	if eror != nil {
		log.Fatal(eror)
	}
	log.Println("Content stream finished:", content)
}

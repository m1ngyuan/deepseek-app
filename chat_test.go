package main

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestChat(t *testing.T) {
	content, err := chat()
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	log.Println("Content stream finished:", content)
}

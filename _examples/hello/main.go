package main

import (
	"fmt"

	"github.com/Vernacular-ai/godub/audioop"
)

func main() {
	e := audioop.NewError("Hello, world: %d", 100)
	fmt.Println(e.Error())
}

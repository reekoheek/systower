package main

import (
	"fmt"
	"os"

	"github.com/reekoheek/caffeine/internal/caffeine"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}

	c, err := caffeine.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	switch os.Args[1] {
	case "listen":
		if err := c.Listen(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "on":
		c.On()
	case "off":
		c.Off()
	case "toggle":
		c.Toggle()
	case "status":
		fmt.Println(c.Status())
	default:
		os.Exit(1)
	}
}

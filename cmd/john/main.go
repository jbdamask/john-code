package main

import (
	"fmt"
	"os"
	"github.com/jbdamask/john-code/pkg/agent"
	"github.com/jbdamask/john-code/pkg/config"
	"github.com/jbdamask/john-code/pkg/ui"
)

func main() {
	fmt.Println("Starting John Code...")
	
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	ui := ui.New()
	ag := agent.New(cfg, ui)

	if err := ag.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

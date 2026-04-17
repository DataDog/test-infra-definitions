package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{}
)

func init() {
	rootCmd.AddCommand(ServerCmd)
}

func main() {
	// Invoke the Agent
	var err error
	if err = rootCmd.Execute(); err != nil {
		log.Printf("Errors were found in execution: %v", err)
		os.Exit(-1)
	}
	log.Printf("Command finished: %v", err)
}


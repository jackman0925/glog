package main

import (
	"github.com/jackman0925/glog"
)

func main() {
	// Default logger
	glog.Info("This is an info message from the default logger.")
	glog.Warnf("This is a %s message from the default logger.", "warning")

	// Initialize a custom logger
	// Create a logger.yaml file first
	/*
		if err := glog.Init("logger.yaml", "my-app"); err != nil {
			log.Fatalf("failed to initialize logger: %v", err)
		}
		glog.Info("This is an info message from the custom logger.")
	*/

	// Create a new logger instance
	// Create a logger.yaml file first
	/*
		logger, err := glog.New("logger.yaml", "my-other-app")
		if err != nil {
			log.Fatalf("failed to create logger: %v", err)
		}
		logger.Info("This is a message from a new logger instance.")
	*/
}

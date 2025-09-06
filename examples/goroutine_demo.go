package main

// import (
// 	"fmt"
// 	"os"
// 	"sync"
// 	"time"

// 	"github.com/jackman0925/glog"
// )

// func main() {
// 	// Create a logger.yaml file
// 	configContent := `
// encoder: console
// path: ""
// directory: ""
// show_line: true
// show_goroutine: true
// encode_level: CapitalColor
// log_stdout: true
// segment:
//   max_size: 10
//   max_age: 7
//   max_backups: 10
//   compress: false
// `

// 	// Write config file
// 	if err := os.WriteFile("logger.yaml", []byte(configContent), 0644); err != nil {
// 		fmt.Printf("Failed to write config file: %v\n", err)
// 		return
// 	}

// 	// Initialize logger
// 	if err := glog.Init("logger.yaml", "goroutine_demo"); err != nil {
// 		fmt.Printf("Failed to initialize logger: %v\n", err)
// 		return
// 	}

// 	// Create a wait group to wait for all goroutines
// 	var wg sync.WaitGroup

// 	// Start multiple goroutines to demonstrate goroutine ID logging
// 	for i := 0; i < 5; i++ {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()

// 			// Log messages from different goroutines
// 			glog.Info("Starting goroutine", "id", id)
// 			glog.Printf("Goroutine %d is processing data", id)

// 			// Simulate some work
// 			time.Sleep(100 * time.Millisecond)

// 			glog.Warn("Goroutine completed", "id", id)
// 		}(i)
// 	}

// 	// Wait for all goroutines to complete
// 	wg.Wait()

// 	glog.Info("All goroutines completed")
// 	glog.Printf("Demo finished - check the logs to see different goroutine IDs")
// }

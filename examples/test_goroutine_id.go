package main

// import (
// 	"fmt"
// 	"runtime"
// 	"strings"
// 	"sync"
// 	"time"
// )

// // getGoroutineID returns the current goroutine ID
// func getGoroutineID() string {
// 	buf := make([]byte, 64)
// 	buf = buf[:runtime.Stack(buf, false)]
// 	// Extract goroutine ID from stack trace
// 	// Format: "goroutine 123 [running]:"
// 	stackStr := string(buf)
// 	if idx := strings.Index(stackStr, "goroutine "); idx != -1 {
// 		start := idx + len("goroutine ")
// 		if end := strings.Index(stackStr[start:], " "); end != -1 {
// 			return stackStr[start : start+end]
// 		}
// 	}
// 	return "unknown"
// }

// func main() {
// 	fmt.Println("=== Goroutine ID 测试 ===")

// 	// 主 Goroutine ID
// 	fmt.Printf("主 Goroutine ID: %s\n", getGoroutineID())

// 	// 创建多个 Goroutine 来测试
// 	var wg sync.WaitGroup

// 	for i := 0; i < 5; i++ {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()
// 			fmt.Printf("Goroutine %d 的 ID: %s\n", id, getGoroutineID())
// 			time.Sleep(100 * time.Millisecond)
// 		}(i)
// 	}

// 	wg.Wait()
// 	fmt.Println("=== 测试完成 ===")
// }

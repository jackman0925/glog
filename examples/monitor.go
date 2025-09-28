package main

import (
	"bufio"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

// --- 请修改这里的配置 ---
var (
	// 要监控的日志文件路径
	logFilePath = "/Users/jackman/Desktop/projects/glog/logs/glog/error.log"

	// 邮件接收人
	emailTo = "your-recipient-email@example.com"

	// SMTP 服务器配置 (以Gmail为例)
	smtpHost     = "smtp.gmail.com"
	smtpPort     = "587"
	smtpUser     = "your-email@gmail.com" // 您的邮箱地址
	smtpPassword = "your-app-password"    // 您的邮箱应用专用密码, 而不是登录密码
)

// --- 配置结束 ---

func main() {
	// 1. 创建一个新的文件观察者
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("创建文件观察者失败: %v", err)
	}
	defer watcher.Close()

	// 2. 启动一个 goroutine 来处理文件系统事件
	done := make(chan bool)
	go func() {
		var lastOffset int64 = 0 // 记录上次读取的位置
		// 启动时先处理一次，以防有未读日志
		lastOffset = processNewLogs(lastOffset)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// 处理写入事件
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Printf("检测到文件写入事件: %s", event.Name)
					lastOffset = processNewLogs(lastOffset)
				}

				// 处理重命名或删除事件（日志滚动）
				if event.Op&fsnotify.Rename == fsnotify.Rename || event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Printf("日志文件被重命名或删除，可能是日志滚动。尝试重新观察: %s", event.Name)

					// 移除旧的 watch (可能已经失效)
					watcher.Remove(event.Name)

					// 循环尝试重新添加 watch，因为新文件创建可能稍有延迟
					for {
						time.Sleep(100 * time.Millisecond) // 短暂等待
						err := watcher.Add(logFilePath)
						if err == nil {
							log.Printf("已成功重新观察新的日志文件: %s", logFilePath)
							// 重置偏移量，因为这是个新文件
							lastOffset = 0
							break // 成功，跳出循环
						}
						log.Printf("重新观察失败，正在重试: %v", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("观察者错误: %v", err)
			}
		}
	}()

	// 3. 等待并监控文件
	// 检查文件是否存在，如果不存在就等待
	for {
		_, err := os.Stat(logFilePath)
		if err == nil {
			break // 文件存在，跳出循环
		}
		if os.IsNotExist(err) {
			log.Printf("日志文件 '%s' 不存在，将在10秒后重试...", logFilePath)
			time.Sleep(10 * time.Second)
		} else {
			// 其他类型的错误，直接失败
			log.Fatalf("检查日志文件时发生未知错误: %v", err)
		}
	}

	// 将要监控的文件添加到观察者
	err = watcher.Add(logFilePath)
	if err != nil {
		log.Fatalf("添加文件到观察者失败: %v", err)
	}
	log.Printf("开始监控: %s", logFilePath)

	// 阻塞主 goroutine，让程序持续运行
	<-done
}

// processNewLogs 从上次的偏移量开始读取文件，并发送邮件
func processNewLogs(offset int64) int64 {
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Printf("无法打开日志文件: %v", err)
		return offset
	}
	defer file.Close()

	// 移动到上次读取结束的位置
	_, err = file.Seek(offset, 0)
	if err != nil {
		log.Printf("无法定位文件偏移量: %v", err)
		return offset
	}

	// 从当前位置开始逐行读取
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			log.Printf("发现新日志: %s", line)
			// sendEmailAlert(line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("读取文件时出错: %v", err)
	}

	// 获取当前文件末尾的偏移量，作为下次读取的起始位置
	newOffset, err := file.Seek(0, os.SEEK_END)
	if err != nil {
		log.Printf("无法获取新的文件偏移量: %v", err)
		return offset // 出错则保持旧的偏移量
	}
	return newOffset
}

// sendEmailAlert 发送邮件告警
func sendEmailAlert(logLine string) {
	subject := "Glog Error Monitor Alert!"
	body := fmt.Sprintf("Timestamp: %s \n File: %s \n Error Log: %s",
		time.Now().Format("2006-01-02 15:04:05"),
		logFilePath,
		logLine,
	)

	// 构造邮件内容
	msg := "From: " + smtpUser + " " + "To: " + emailTo + " " + "Subject: " + subject + " " + body

	// 设置认证信息
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	// 发送邮件
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUser, []string{emailTo}, []byte(msg))
	if err != nil {
		log.Printf("发送邮件失败: %v", err)
		return
	}

	log.Printf("邮件已成功发送至 %s", emailTo)
}

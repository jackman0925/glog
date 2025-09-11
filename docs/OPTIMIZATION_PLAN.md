# 性能优化方案

## 1. 优化 Goroutine ID 获取函数

### 问题
当前 `getGoroutineID()` 函数存在严重性能问题：
1. 每次调用分配 64 字节缓冲区
2. 调用 `runtime.Stack()` 是昂贵操作
3. 多次字符串操作产生额外内存分配

### 解决方案

#### 方案一：使用缓存机制
```go
// 使用 sync.Pool 缓存 goroutine ID 解析结果
var goroutineIDCache = sync.Map{}

func getGoroutineID() string {
	// 获取当前 goroutine 栈信息作为缓存键
	buf := make([]byte, 32)
	n := runtime.Stack(buf, false)
	key := string(buf[:n])
	
	// 尝试从缓存获取
	if id, ok := goroutineIDCache.Load(key); ok {
		return id.(string)
	}
	
	// 解析并缓存
	stackStr := string(buf[:n])
	if idx := strings.Index(stackStr, "goroutine "); idx != -1 {
		start := idx + len("goroutine ")
		if end := strings.Index(stackStr[start:], " "); end != -1 {
			id := stackStr[start : start+end]
			goroutineIDCache.Store(key, id)
			return id
		}
	}
	return "unknown"
}
```

#### 方案二：使用更高效的解析方法
```go
// 优化的解析方法，减少字符串操作
func getGoroutineID() string {
	buf := make([]byte, 32)
	buf = buf[:runtime.Stack(buf, false)]
	
	// 直接在字节切片中查找，避免字符串转换
	prefix := []byte("goroutine ")
	for i := 0; i < len(buf)-len(prefix); i++ {
		match := true
		for j := 0; j < len(prefix); j++ {
			if buf[i+j] != prefix[j] {
				match = false
				break
			}
		}
		if match {
			// 找到前缀，提取数字
			start := i + len(prefix)
			for end := start; end < len(buf); end++ {
				if buf[end] == ' ' || buf[end] == '[' {
					return string(buf[start:end])
				}
			}
		}
	}
	return "unknown"
}
```

#### 方案三：使用原子操作和线程局部存储
```go
// 使用 goroutine 局部存储方案
type goroutineIDKey struct{}

var goroutineIDCounter int64

func getGoroutineID() string {
	// 从 context 获取或创建 goroutine ID
	return fmt.Sprintf("%d", goroutineIDCounter)
}
```

## 2. 优化默认日志器

### 问题
默认使用 `zap.NewDevelopment()` 创建开发模式日志器，性能较差。

### 解决方案
```go
func init() {
	// 使用生产模式作为默认日志器，性能更好
	logger, _ := zap.NewProduction()
	xLog = logger.Sugar()
}
```

## 3. 减少内存分配

### 问题
日志方法中频繁的字符串操作和对象创建导致内存分配过多。

### 解决方案
```go
// 使用 sync.Pool 复用对象
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// 优化日志方法
func Info(args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			// 复用 buffer 减少分配
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.WriteString("goroutine=")
			buf.WriteString(getGoroutineID())
			defer bufferPool.Put(buf)
			
			xLog.With(buf.String()).Info(args...)
		} else {
			xLog.Info(args...)
		}
	}
}
```

## 4. 提供高性能模式

### 解决方案
添加一个高性能模式配置选项：

```go
// Config 结构体添加高性能模式选项
type Config struct {
	// ... existing fields ...
	HighPerformance bool `yaml:"high_performance"` // 高性能模式开关
}

// 在 newLogger 中根据配置选择不同的实现
func newLogger(cfg *Config) (*zap.SugaredLogger, error) {
	if cfg.HighPerformance {
		// 使用高性能配置
		return newHighPerformanceLogger(cfg)
	}
	// 使用标准配置
	// ... existing implementation ...
}
```

## 5. 优化日志级别核心创建

### 问题
为每个日志级别创建独立的核心和文件，可能导致文件句柄使用过多。

### 解决方案
```go
// 根据配置决定是否分离日志级别到不同文件
func newLogger(cfg *Config) (*zap.SugaredLogger, error) {
	// ... existing code ...
	
	var cores []zapcore.Core
	if cfg.SeparateLevels {
		// 分离日志级别到不同文件
		cores = []zapcore.Core{
			getEncoderCore(path+FileDebug, debugLevel, cfg),
			getEncoderCore(path+FileInfo, infoLevel, cfg),
			getEncoderCore(path+FileWarn, warnLevel, cfg),
			getEncoderCore(path+FileError, errorLevel, cfg),
			getEncoderCore(path+FilePanic, panicLevel, cfg),
		}
	} else {
		// 使用单一核心写入所有日志到一个文件
		cores = []zapcore.Core{
			getEncoderCore(path+"/app.log", zapcore.DebugLevel, cfg),
		}
	}
	
	// ... rest of implementation ...
}
```

## 实施建议

1. **优先级排序**：
   - 首先优化 Goroutine ID 获取函数（影响最大）
   - 然后优化默认日志器配置
   - 最后考虑其他优化

2. **兼容性**：
   - 保持 API 兼容性
   - 通过配置选项启用新特性

3. **测试**：
   - 添加性能测试确保优化效果
   - 保持功能测试的完整性
# 日志库性能分析报告

## 性能测试结果分析

### 基准测试结果

```
BenchmarkDefaultLogger                    288,945     3,985 ns/op     345 B/op      6 allocs/op
BenchmarkGoroutineIDLogger               202,639     6,160 ns/op   1,928 B/op     16 allocs/op
BenchmarkCustomLogger                    389,284     3,123 ns/op      56 B/op      2 allocs/op
BenchmarkCustomLoggerWithGoroutineID     196,744     6,291 ns/op   6,416 B/op     30 allocs/op
BenchmarkGetGoroutineID                  751,056     1,593 ns/op     128 B/op      2 allocs/op
BenchmarkZapDirect                    16,793,756        77.01 ns/op     2 B/op      0 allocs/op
BenchmarkConcurrentLogging               385,302     3,489 ns/op   288,103 logs/sec    95 B/op      3 allocs/op
```

## 性能问题总结

### 1. Goroutine ID 获取性能问题 (严重)

从测试结果可以看出：
- 启用 Goroutine ID 功能会使日志性能下降约 50%
- `getGoroutineID()` 函数单次调用耗时 1,593 ns，且每次调用分配 128 字节内存
- 启用该功能后，每次日志调用分配的内存从 56B 增加到 6,416B

**问题根源：**
```go
func getGoroutineID() string {
    buf := make([]byte, 64)  // 每次分配 64 字节
    buf = buf[:runtime.Stack(buf, false)]  // 调用昂贵的 runtime.Stack()
    // 多次字符串操作和分配
}
```

### 2. 默认日志器性能问题

- 默认开发模式日志器性能较差 (3,985 ns/op)
- 直接使用 zap 生产模式日志器性能最佳 (77.01 ns/op)
- 自定义日志器性能优于默认日志器，但仍远低于直接使用 zap

### 3. 内存分配问题

- 启用 Goroutine ID 功能后，单次日志调用内存分配增加超过 100 倍
- 默认日志器每次调用分配 345 字节内存
- 自定义日志器优化后仅分配 56 字节内存

### 4. 并发性能

- 在高并发场景下，日志库能保持稳定的性能
- 100 个 goroutine 并发写入日志，速度可达 288,103 logs/sec

## 优化建议

### 1. 优化 Goroutine ID 获取

使用更高效的方法获取 Goroutine ID，例如：
- 使用 sync.Map 缓存 Goroutine ID
- 使用 atomic 包和 goroutine 局部存储
- 或者完全避免使用 Goroutine ID 功能

### 2. 改进默认日志器

- 默认使用 zap 的生产模式而非开发模式
- 提供更好的初始化选项

### 3. 减少内存分配

- 避免不必要的字符串操作
- 使用对象池减少内存分配
- 优化配置结构

### 4. 提供高性能模式

- 提供无格式化、无额外信息的快速日志模式
- 允许用户选择启用/禁用特定功能
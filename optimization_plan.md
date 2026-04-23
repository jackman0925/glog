# glog 性能优化与 Bug 修复实施文档

## 一、 优化目标与范围
本阶段的调整严格遵循**“不重写已有功能架构、只修复 Bug 和性能缺陷”**的原则。调整重点集中在日志级别的无效开销、`sync.Pool` 内存分配失效，以及实例对象的行号偏移错误。

## 二、 具体修改计划

### 1. 修复全局函数中极其严重的性能损耗 (Performance)
**问题：** `glog.Debug()`, `glog.Info()` 等封装函数中，如果在配置里开启了 `showGoroutine: true`，即使当前日志级别不需要打印（例如当前级别为 Info，代码调用 `glog.Debug()`），依然会**无条件先执行**极度耗时的 `getGoroutineID()` 并克隆整个 Logger (`s.logger.With()`)，随后才在内部判定并丢弃。
**修复方案：** 在这些封装函数中，执行前置操作之前，先通过 `zap` 的 `Enabled(Level)` 进行拦截。
```go
// 修复前：
if s.showGoroutine {
    s.logger.With("goroutine", getGoroutineID()).Debug(args...)
}

// 修复后：
if !s.logger.Desugar().Core().Enabled(zap.DebugLevel) {
    return
}
if s.showGoroutine { ... }
```

### 2. 修复 `goroutineIDBufPool` 内存池失效问题 (Performance)
**问题：** `getGoroutineID` 中试图复用 64 字节的 buffer。但 `runtime.Stack` 获取堆栈几乎总会填满 64 字节，导致 `n == len(buf)` 永远成立，随后代码会丢弃复用的 buf，每次都在堆上重新 `make([]byte, 256)`。
**修复方案：** 我们只需解析第一行的协程编号。因此，无论是否截断，64 字节内都已包含了协程号信息。直接去掉长度等于 64 就重新分配的逻辑，永远只取前 `n` 个字节给 `parseGoroutineID` 即可。

### 3. 修复 Instance Logger 的行号向上偏移 Bug (Bug Fix)
**问题：** `newLogger()` 函数内硬编码了 `zap.AddCallerSkip(1)`。这对于 `glog.Info()` 全局函数是正确的（需要跳过包级包装层）。但是当用户调用 `logger, _ := glog.New()` 拿到实例，然后调用 `logger.Info()` 时，并没有这一层包装。此时 Skip(1) 会导致打印出来的代码行号是错误的（跳到了更上一级的调用者，如 `runtime.main`）。
**修复方案：** `newLogger` 中只加 `AddCaller()`，不加 `AddCallerSkip(1)`。`AddCallerSkip(1)` 应该在 `Init` 和 `New` (当 `setGlobal` 为 `true` 时) 手动对即将设为全局 `loggerState` 的那个 logger 补充添加。

### 四、 实施步骤
1. 修改 `logger.go` 中所有的全局导出函数 (`Debug`, `Info`, `Warn` 等)。
2. 修改 `logger.go` 中的 `getGoroutineID`。
3. 修改 `logger.go` 中的 `newLogger`，剥离 `AddCallerSkip` 逻辑，挪至 `Init` 和 `New` 内部按需应用。
4. 运行 `go test`，确保一切正确无误，功能行为保持向后兼容。

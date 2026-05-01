### TL;DR
下一步：基于当前 benchmark，优化 `Resolve` 在“最新版本命中大多数 feature”场景下的冷查询性能。

### 全局事实
- 当前冷查询基准在 `registry_benchmark_test.go`：注册离散 feature，10% 老 feature 在新版本移除，查询最新版本。
- `Resolve` 是 exported 缓存层；unexported `resolve` 默认不走缓存，benchmark 直接调用 `resolve`，避免缓存命中干扰。
- 当前计算逻辑仍是遍历所有 registered feature，复杂度约 `O(n)`。
- 目标优化方向：`Freeze()` 后构建版本断点/区间索引；最新版本走 fast path；普通版本二分定位区间；FeatureSet 尽量复用不可变 backing。

### WIP
当前子任务：实现 Resolve 索引优化。
- 优先保持公开 API 和既有行为不变。
- 重点优化 latest 区间：当 version >= 最后一个断点时，避免遍历全部 feature。
- 需要兼容 `RegisterRange` 的移除语义 `[since, until)`。

### TODO
- 设计并实现 Freeze 阶段的断点索引结构。
- 调整 frozen 后 `Resolve`/`resolve` 计算路径使用索引。
- 跑 `go test ./...` 和现有 benchmark，对比优化前后结果。

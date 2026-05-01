### TL;DR
Autoresearch 可暂停：当前最佳 `resolve_10000_ns` 从 baseline ~409,491 ns/op 优化到 ~20 ns/op。

### 全局事实
- 实验配置在 `autoresearch.md`，运行脚本 `autoresearch.sh`，检查脚本 `autoresearch.checks.sh`，日志 `autoresearch.jsonl`。
- Primary metric：`resolve_10000_ns`，越低越好；benchmark 直接调用 unexported `resolve` 避免 exported cache 命中。
- 已保留主要优化提交：
  - `7a74cae`：Freeze 预计算 latest interval，共享 active feature map。
  - `f2a8ba8`：latest stable version 先走轻量 parser，避免 semver allocation。
  - `73236ea`：internal resolve 读取 immutable latest index 时跳过锁。
  - `36c9fab`：内联 stable semver core parser。
  - `01f00ff`：single-digit minor/patch fast path。
  - `d6c96e1`：single-digit fast path 只处理 plain stable，metadata 回落通用 parser。
  - `0bae562`：major 超过 latest breakpoint 时短路比较。
  - `65fadd9`：在完整 parser 前内联 high-major plain `x.d.d` latest fast path，当前最佳约 20 ns/op、24 B/op、1 alloc/op。
- 多个 parser/layout 微优化已 discard，详见 `autoresearch.jsonl`；不要重复尝试。
- `autoresearch.ideas.md` 记录了非 latest 版本完整 breakpoint interval index 方案；它可能优化旧版本查询，但当前 primary 不覆盖。

### WIP
当前无正在编辑的代码；最后一轮实验已完成并 keep。工作区可能还有 autoresearch 记录文件更新。

### TODO
- 如要收尾：运行一次 `git status`，按需整理/提交或归档 autoresearch 记录。
- 如继续优化 primary：剩余成本主要是 `FeatureSet` 返回对象的 1 次 allocation；需谨慎，避免改变语义/并发安全。
- 如扩展优化范围：先明确是否把 older-version cold resolve 纳入 primary/secondary，并重新初始化实验目标。

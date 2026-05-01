# semverflags 设计

## 目标

提供一个轻量的 Go library，让调用方可以：

1. **注册阶段**：声明某个特性在哪个语义化版本被引入（可选地声明在哪个版本被移除）。
2. **解析阶段**：给定一个具体的版本号，得到该版本所支持的特性集合。
3. **断言阶段**：以 `Has(feature)` / `MustHave(feature)` 等方式判断某个特性是否被支持。

典型场景：客户端 / 服务端 / 设备固件等需要按版本做兼容降级时，把"哪个版本起支持什么"这件事集中维护，避免散落在业务代码里的 `if version >= "1.2.3"` 判断。

## 依赖与约束

- 语义化版本解析使用 [`github.com/Masterminds/semver/v3`](https://github.com/Masterminds/semver)。
- Go 1.21+（用到泛型）。
- 包名 `semverflags`；模块路径为 `github.com/sjet47/semverflags`，导入时无需 alias。
- Feature 标识在使用方应保持**全局唯一**；本库不存任何描述/元信息，元信息应由调用方在外部以 feature 为 key 索引。

## 核心概念

| 概念 | 类型 | 说明 |
|---|---|---|
| Feature | 用户自定义 `comparable` 类型（通常是 `string` 或 int 常量） | 特性标识 |
| Registry | `*Registry[F]` | 注册表，初始化阶段构建，`Freeze` 后只读 |
| FeatureSet | `*FeatureSet[F]` | 某个具体版本解析出的特性集合，runtime 使用 |
| Options | `Option` | 控制 prerelease 处理等行为 |

## API

### Registry

```go
package semverflags

type Registry[F comparable] struct { /* ... */ }

// NewRegistry 创建一个空的注册表。可选 Option 控制行为，例如 prerelease 处理。
func NewRegistry[F comparable](opts ...Option) *Registry[F]

// Register 声明一个特性从 sinceVersion 起被支持（含该版本，无上界）。
// sinceVersion 必须是合法 semver；不合法 panic（仅在初始化阶段调用，fail-fast）。
// 同一个 feature 重复注册 panic。
// Freeze 之后调用 panic。
func (r *Registry[F]) Register(feature F, sinceVersion string) *Registry[F]

// RegisterRange 声明一个特性在 [sinceVersion, untilVersion) 区间内支持，untilVersion 不含。
// 用于"曾经支持、后被移除"的特性。
// sinceVersion / untilVersion 必须是合法 semver，且 untilVersion 必须大于 sinceVersion；不合法 panic。
// untilVersion 必填；无上界时应使用 Register。
// 同一个 feature 重复注册 panic。
// Freeze 之后调用 panic。
func (r *Registry[F]) RegisterRange(feature F, sinceVersion, untilVersion string) *Registry[F]

// Freeze 切换到只读状态：
//   - 阻止后续 Register*；
//   - 允许并发安全地 Resolve；
//   - 启用按版本号缓存的 FeatureSet（首次 Resolve 计算，后续命中缓存）。
func (r *Registry[F]) Freeze()

// Resolve 返回该版本支持的特性集合。version 不合法时返回 error。
// 调用前应已 Freeze（未 Freeze 也能用，但每次都重新计算且并发不安全）。
func (r *Registry[F]) Resolve(version string) (*FeatureSet[F], error)

// MustResolve 同 Resolve，失败 panic。适合启动期对已知合法版本号使用。
func (r *Registry[F]) MustResolve(version string) *FeatureSet[F]

// SinceOf 返回某个特性最早被支持的版本号；若该特性未注册，返回 (_, false)。
func (r *Registry[F]) SinceOf(feature F) (string, bool)

// UntilOf 返回某个特性的上界版本号（不含）。
//   - 若该特性未注册：("", false)
//   - 若注册但无上界：("latest", true)
//   - 若注册且有上界：(untilVersion, true)
func (r *Registry[F]) UntilOf(feature F) (string, bool)

// FeatureRange 描述一个特性的版本范围。
type FeatureRange[F comparable] struct {
    Feature F
    Since   string // 起始版本（含）
    Until   string // 上界版本（不含）；无上界时为 "latest"
}

// Dump 返回当前 Registry 中所有已注册的特性及其版本范围，便于调试 / 日志输出。
// 顺序按 Feature 的字符串化排序（string 类型直接排，其他类型用 fmt.Sprint 排）。
func (r *Registry[F]) Dump() []FeatureRange[F]
```

### FeatureSet

```go
// FeatureSet 表示某个具体版本支持的特性集合，只读、并发安全。
type FeatureSet[F comparable] struct { /* ... */ }

func (s *FeatureSet[F]) Version() string

func (s *FeatureSet[F]) Has(feature F) bool
func (s *FeatureSet[F]) HasAll(features ...F) bool
func (s *FeatureSet[F]) HasAny(features ...F) bool

// MustHave 当不支持时 panic；用于"理应支持否则就是 bug"的断言。
func (s *FeatureSet[F]) MustHave(feature F)

// All 返回当前版本支持的全部特性（顺序不保证稳定）。
func (s *FeatureSet[F]) All() []F
```

### Options

```go
type Option func(*options)

// WithIgnorePrerelease 让 Resolve 时忽略输入版本号的 prerelease 部分，
// 例如 "1.2.3-rc.1" 会被当作 "1.2.3" 比较。默认不忽略，遵循 semver
// 语义（"1.2.3-rc.1" < "1.2.3"）。
func WithIgnorePrerelease() Option

// 后续可加 WithIgnoreBuildMetadata 等。
```

### 默认全局 Registry

为常见用法提供包级别的便捷入口，内部维护一个 `Feature = string` 的默认 registry：

```go
// 默认 registry 的 Feature 类型固定为 string。

func Register(feature string, sinceVersion string)
func RegisterRange(feature, sinceVersion, untilVersion string)
func Freeze()

func Resolve(version string) (*FeatureSet[string], error)
func MustResolve(version string) *FeatureSet[string]

func SinceOf(feature string) (string, bool)
func UntilOf(feature string) (string, bool)

func Dump() []FeatureRange[string]

// 用于希望复用全局注册但自定义 Options 的场景（不常用）。
func ConfigureDefault(opts ...Option) // 必须在任何 Register* 之前调用，否则 panic
```

要使用自定义 Feature 类型或独立隔离多套 registry 时，使用 `NewRegistry[F]()`。

## 使用示例

### 包级默认 registry

```go
package boot

import "github.com/sjet47/semverflags"

func init() {
    semverflags.Register("dark_mode", "1.2.0")
    semverflags.Register("push_notify", "1.5.0")
    semverflags.RegisterRange("legacy_login", "1.0.0", "2.0.0")
    semverflags.Freeze()
}
```

```go
func handle(clientVersion string) {
    fs, err := semverflags.Resolve(clientVersion)
    if err != nil {
        return // 版本号非法，调用方决定降级策略
    }
    if fs.Has("dark_mode") { /* ... */ }
}
```

### 自定义 Feature 类型

```go
type Feature string

const (
    FeatureDarkMode    Feature = "dark_mode"
    FeaturePushNotify  Feature = "push_notify"
    FeatureLegacyLogin Feature = "legacy_login"
)

var registry = func() *semverflags.Registry[Feature] {
    r := semverflags.NewRegistry[Feature]()
    r.Register(FeatureDarkMode, "1.2.0")
    r.Register(FeaturePushNotify, "1.5.0")
    r.RegisterRange(FeatureLegacyLogin, "1.0.0", "2.0.0")
    r.Freeze()
    return r
}()
```

### 可运行示例

仓库内提供了一个 Go doc 可运行示例：

```bash
go test -run Example .
```

示例文件：`example_test.go`，在 Go doc / pkg.go.dev 中可直接运行查看输出。

## 关键实现要点

- **Freeze 与缓存**：`Freeze` 后，Registry 的注册数据不可变；`Resolve` 首次按版本号计算 `FeatureSet`，结果以 version 字符串为 key 缓存（lazy）。
  缓存使用 `sync.RWMutex + map`：
  1. 读锁查询缓存命中则直接返回；
  2. 未命中升级为写锁，**进入临界区后再 double-check 一次**，避免并发下多个 goroutine 重复计算并相互覆盖；
  3. 计算结果写入 map 后释放写锁。
  不使用 `sync.Map`，因为它无法保证"同一 key 只计算一次"。
- **未 Freeze 时**：每次 Resolve 都重算，且不保证并发安全（视作未完成初始化）。文档明确写出，避免误用。
- **错误处理**：`Resolve` 仅在版本号非法时返 error；不存在"该版本不支持任何特性"这种 error，会正常返回空 `FeatureSet`。
- **零值/空 registry**：未注册任何 feature 的 registry，`Resolve` 合法版本号返回空 `FeatureSet`，不报错。
- **不实现特性依赖**：feature 与版本绑定，只要版本支持就一定满足"依赖"，不是 lib 的职责。

## Roadmap（暂不实现）

1. **配置文件 / 结构体批量注册**，以及随之而来的 constraint 表达式支持，例如：

   ```yaml
   features:
     dark_mode: ">=1.2.0"
     legacy_login: ">=1.0.0, <2.0.0"
     experimental: ">=1.5.0, <2.0.0 || >=3.0.0"
   ```

   届时新增 `LoadFromYAML` / `LoadFromStruct` 等入口，并在 Registry 层引入
   `RegisterConstraint(feature, constraintExpr)`。

2. `WithIgnoreBuildMetadata` 等更多 Options。

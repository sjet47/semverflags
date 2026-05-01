package semverflags

var defaultRegistry = NewRegistry[string]()

// ConfigureDefault configures the package-level default registry. It must be
// called before any package-level Register/RegisterRange call.
func ConfigureDefault(opts ...Option) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	if defaultRegistry.frozen || len(defaultRegistry.entries) > 0 {
		panic("semverflags: ConfigureDefault must be called before any default registry Register call")
	}
	defaultRegistry.opts = applyOptions(opts)
}

// Register declares a string feature on the default registry.
func Register(feature string, sinceVersion string) {
	defaultRegistry.Register(feature, sinceVersion)
}

// RegisterRange declares a string feature range on the default registry.
func RegisterRange(feature, sinceVersion, untilVersion string) {
	defaultRegistry.RegisterRange(feature, sinceVersion, untilVersion)
}

// Freeze freezes the default registry.
func Freeze() {
	defaultRegistry.Freeze()
}

// Resolve resolves version against the default registry.
func Resolve(version string) (*FeatureSet[string], error) {
	return defaultRegistry.Resolve(version)
}

// MustResolve resolves version against the default registry and panics on
// invalid version.
func MustResolve(version string) *FeatureSet[string] {
	return defaultRegistry.MustResolve(version)
}

// SinceOf returns the since version for a string feature in the default
// registry.
func SinceOf(feature string) (string, bool) {
	return defaultRegistry.SinceOf(feature)
}

// UntilOf returns the until version for a string feature in the default
// registry.
func UntilOf(feature string) (string, bool) {
	return defaultRegistry.UntilOf(feature)
}

// Dump returns all feature ranges in the default registry.
func Dump() []FeatureRange[string] {
	return defaultRegistry.Dump()
}

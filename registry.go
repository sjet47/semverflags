package semverflags

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Masterminds/semver/v3"
)

const latestVersion = "latest"

type featureEntry struct {
	since    *semver.Version
	until    *semver.Version
	sinceStr string
	untilStr string
}

// FeatureRange describes the version range of a feature.
type FeatureRange[F comparable] struct {
	Feature F
	Since   string // inclusive lower bound
	Until   string // exclusive upper bound; "latest" means no upper bound
}

// Registry stores feature version ranges. After Freeze it is read-only and safe
// for concurrent Resolve calls.
type Registry[F comparable] struct {
	mu               sync.RWMutex
	opts             options
	frozen           bool
	entries          map[F]featureEntry
	cache            map[string]*FeatureSet[F]
	latestBreakpoint *semver.Version
	latestFeatures   map[F]struct{}
}

// NewRegistry creates an empty registry.
func NewRegistry[F comparable](opts ...Option) *Registry[F] {
	return &Registry[F]{
		opts:    applyOptions(opts),
		entries: make(map[F]featureEntry),
	}
}

// Register declares feature support starting from sinceVersion, inclusive.
// It panics when sinceVersion is invalid or when called after Freeze.
func (r *Registry[F]) Register(feature F, sinceVersion string) *Registry[F] {
	return r.register(feature, sinceVersion, "", false)
}

// RegisterRange declares feature support in [sinceVersion, untilVersion). It
// panics when versions are invalid, untilVersion is empty or not greater than
// sinceVersion, or when called after Freeze.
func (r *Registry[F]) RegisterRange(feature F, sinceVersion, untilVersion string) *Registry[F] {
	return r.register(feature, sinceVersion, untilVersion, true)
}

func (r *Registry[F]) register(feature F, sinceVersion, untilVersion string, requireUntil bool) *Registry[F] {
	r.mustNotBeNil()

	since := mustStrictVersion(sinceVersion)
	var until *semver.Version
	untilStr := latestVersion
	if requireUntil && untilVersion == "" {
		panic("semverflags: untilVersion is required")
	}
	if untilVersion != "" {
		until = mustStrictVersion(untilVersion)
		untilStr = until.String()
		if until.Compare(since) <= 0 {
			panic(fmt.Sprintf("semverflags: untilVersion %q must be greater than sinceVersion %q", untilVersion, sinceVersion))
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		panic("semverflags: register after Freeze")
	}
	if _, exists := r.entries[feature]; exists {
		panic(fmt.Sprintf("semverflags: feature %v already registered", feature))
	}
	if r.entries == nil {
		r.entries = make(map[F]featureEntry)
	}
	r.entries[feature] = featureEntry{
		since:    since,
		until:    until,
		sinceStr: since.String(),
		untilStr: untilStr,
	}
	return r
}

// Freeze switches the registry to read-only mode and enables lazy Resolve
// caching. It is safe to call multiple times.
func (r *Registry[F]) Freeze() {
	r.mustNotBeNil()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return
	}
	if r.entries == nil {
		r.entries = make(map[F]featureEntry)
	}
	r.cache = make(map[string]*FeatureSet[F])
	r.buildLatestIndexLocked()
	r.frozen = true
}

// Resolve returns the set of features supported by version. Invalid versions
// return an error. Before Freeze, Resolve recomputes every time and should be
// treated as not concurrency-safe.
func (r *Registry[F]) Resolve(version string) (*FeatureSet[F], error) {
	r.mustNotBeNil()

	parsed, key, err := r.parseResolveVersion(version)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	frozen := r.frozen
	if frozen {
		if set, ok := r.cache[key]; ok {
			r.mu.RUnlock()
			return set, nil
		}
	}
	r.mu.RUnlock()

	if !frozen {
		return r.resolveParsed(key, parsed), nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if set, ok := r.cache[key]; ok {
		return set, nil
	}
	set := r.resolveParsedLocked(key, parsed)
	r.cache[key] = set
	return set, nil
}

func (r *Registry[F]) resolve(version string) (*FeatureSet[F], error) {
	r.mustNotBeNil()

	if set, ok := r.resolveLatestStable(version); ok {
		return set, nil
	}

	parsed, key, err := r.parseResolveVersion(version)
	if err != nil {
		return nil, err
	}
	return r.resolveParsed(key, parsed), nil
}

// MustResolve is like Resolve but panics on invalid version.
func (r *Registry[F]) MustResolve(version string) *FeatureSet[F] {
	set, err := r.Resolve(version)
	if err != nil {
		panic(err)
	}
	return set
}

// SinceOf returns the earliest supported version of a registered feature.
func (r *Registry[F]) SinceOf(feature F) (string, bool) {
	if r == nil {
		return "", false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[feature]
	if !ok {
		return "", false
	}
	return entry.sinceStr, true
}

// UntilOf returns the exclusive upper bound of a registered feature, or
// "latest" when there is no upper bound.
func (r *Registry[F]) UntilOf(feature F) (string, bool) {
	if r == nil {
		return "", false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[feature]
	if !ok {
		return "", false
	}
	return entry.untilStr, true
}

// Dump returns all registered feature ranges sorted by fmt.Sprint(feature).
func (r *Registry[F]) Dump() []FeatureRange[F] {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.entries) == 0 {
		return nil
	}
	ranges := make([]FeatureRange[F], 0, len(r.entries))
	for feature, entry := range r.entries {
		ranges = append(ranges, FeatureRange[F]{
			Feature: feature,
			Since:   entry.sinceStr,
			Until:   entry.untilStr,
		})
	}
	sort.Slice(ranges, func(i, j int) bool {
		left := fmt.Sprint(ranges[i].Feature)
		right := fmt.Sprint(ranges[j].Feature)
		if left != right {
			return left < right
		}
		if ranges[i].Since != ranges[j].Since {
			return ranges[i].Since < ranges[j].Since
		}
		return ranges[i].Until < ranges[j].Until
	})
	return ranges
}

func (r *Registry[F]) mustNotBeNil() {
	if r == nil {
		panic("semverflags: nil Registry")
	}
}

func (r *Registry[F]) parseResolveVersion(version string) (*semver.Version, string, error) {
	parsed, err := semver.StrictNewVersion(version)
	if err != nil {
		return nil, "", fmt.Errorf("semverflags: invalid version %q: %w", version, err)
	}
	if !r.opts.ignorePrerelease || parsed.Prerelease() == "" {
		return parsed, parsed.String(), nil
	}

	normalized := fmt.Sprintf("%d.%d.%d", parsed.Major(), parsed.Minor(), parsed.Patch())
	if metadata := parsed.Metadata(); metadata != "" {
		normalized += "+" + metadata
	}
	withoutPrerelease, err := semver.StrictNewVersion(normalized)
	if err != nil {
		return nil, "", fmt.Errorf("semverflags: invalid normalized version %q: %w", normalized, err)
	}
	return withoutPrerelease, withoutPrerelease.String(), nil
}

func (r *Registry[F]) buildLatestIndexLocked() {
	r.latestBreakpoint = nil
	latestFeatures := make(map[F]struct{})
	for feature, entry := range r.entries {
		if r.latestBreakpoint == nil || entry.since.Compare(r.latestBreakpoint) > 0 {
			r.latestBreakpoint = entry.since
		}
		if entry.until != nil {
			if entry.until.Compare(r.latestBreakpoint) > 0 {
				r.latestBreakpoint = entry.until
			}
			continue
		}
		latestFeatures[feature] = struct{}{}
	}
	r.latestFeatures = latestFeatures
}

func (r *Registry[F]) resolveLatestStable(version string) (*FeatureSet[F], bool) {
	major, minor, patch, ok := parseStableCore(version)
	if !ok {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.frozen || r.latestBreakpoint == nil {
		return nil, false
	}
	if compareStableCoreToVersion(major, minor, patch, r.latestBreakpoint) < 0 {
		return nil, false
	}
	return newFeatureSet(version, r.latestFeatures), true
}

func (r *Registry[F]) resolveParsed(version string, parsed *semver.Version) *FeatureSet[F] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resolveParsedLocked(version, parsed)
}

func (r *Registry[F]) resolveParsedLocked(version string, parsed *semver.Version) *FeatureSet[F] {
	if r.frozen && r.latestBreakpoint != nil && parsed.Compare(r.latestBreakpoint) >= 0 {
		return newFeatureSet(version, r.latestFeatures)
	}
	return r.computeFeatureSetLocked(version, parsed)
}

func (r *Registry[F]) computeFeatureSet(version string, parsed *semver.Version) *FeatureSet[F] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.computeFeatureSetLocked(version, parsed)
}

func (r *Registry[F]) computeFeatureSetLocked(version string, parsed *semver.Version) *FeatureSet[F] {
	features := make(map[F]struct{})
	for feature, entry := range r.entries {
		if parsed.Compare(entry.since) < 0 {
			continue
		}
		if entry.until != nil && parsed.Compare(entry.until) >= 0 {
			continue
		}
		features[feature] = struct{}{}
	}
	return newFeatureSet(version, features)
}

func parseStableCore(version string) (uint64, uint64, uint64, bool) {
	major, rest, ok := parseCorePart(version)
	if !ok || len(rest) == 0 || rest[0] != '.' {
		return 0, 0, 0, false
	}
	minor, rest, ok := parseCorePart(rest[1:])
	if !ok || len(rest) == 0 || rest[0] != '.' {
		return 0, 0, 0, false
	}
	patch, rest, ok := parseCorePart(rest[1:])
	if !ok {
		return 0, 0, 0, false
	}
	if rest == "" {
		return major, minor, patch, true
	}
	if rest[0] != '+' {
		return 0, 0, 0, false
	}
	if len(rest) == 1 {
		return 0, 0, 0, false
	}
	for i := 1; i < len(rest); i++ {
		c := rest[i]
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '-' || c == '.' {
			continue
		}
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

func parseCorePart(s string) (uint64, string, bool) {
	if s == "" {
		return 0, "", false
	}
	if s[0] < '0' || s[0] > '9' {
		return 0, "", false
	}
	if len(s) > 1 && s[0] == '0' && s[1] >= '0' && s[1] <= '9' {
		return 0, "", false
	}

	var value uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return value, s[i:], true
		}
		value = value*10 + uint64(c-'0')
	}
	return value, "", true
}

func compareStableCoreToVersion(major, minor, patch uint64, version *semver.Version) int {
	if major != version.Major() {
		if major < version.Major() {
			return -1
		}
		return 1
	}
	if minor != version.Minor() {
		if minor < version.Minor() {
			return -1
		}
		return 1
	}
	if patch != version.Patch() {
		if patch < version.Patch() {
			return -1
		}
		return 1
	}
	if version.Prerelease() != "" {
		return 1
	}
	return 0
}

func mustStrictVersion(version string) *semver.Version {
	parsed, err := semver.StrictNewVersion(version)
	if err != nil {
		panic(fmt.Sprintf("semverflags: invalid version %q: %v", version, err))
	}
	return parsed
}

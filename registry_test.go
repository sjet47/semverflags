package semverflags

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistryResolveRanges(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().
		Register("base", "1.0.0").
		Register("dark_mode", "1.2.0").
		RegisterRange("legacy_login", "1.0.0", "2.0.0")
	r.Freeze()

	fs, err := r.Resolve("1.5.0")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "1.5.0", fs.Version())
	assert.True(t, fs.Has("base"))
	assert.True(t, fs.Has("dark_mode"))
	assert.True(t, fs.Has("legacy_login"))
	assert.True(t, fs.HasAll("base", "dark_mode"))
	assert.True(t, fs.HasAny("missing", "legacy_login"))
	assert.False(t, fs.Has("missing"))

	fs, err = r.Resolve("2.0.0")
	if !assert.NoError(t, err) {
		return
	}
	assert.False(t, fs.Has("legacy_login"))
	assert.True(t, fs.HasAll("base", "dark_mode"))
}

func TestRegistryResolvePrerelease(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().Register("feature", "1.2.3")
	fs, err := r.Resolve("1.2.3-rc.1")
	if !assert.NoError(t, err) {
		return
	}
	assert.False(t, fs.Has("feature"))

	r = NewRegistry[string](WithIgnorePrerelease()).Register("feature", "1.2.3")
	fs, err = r.Resolve("1.2.3-rc.1")
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, fs.Has("feature"))
	assert.Equal(t, "1.2.3", fs.Version())
}

func TestFreezeEnablesCacheAndRejectsRegister(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().Register("feature", "1.0.0")
	beforeA := r.MustResolve("1.0.0")
	beforeB := r.MustResolve("1.0.0")
	assert.NotSame(t, beforeA, beforeB)

	r.Freeze()
	a := r.MustResolve("1.0.0")
	b := r.MustResolve("1.0.0")
	assert.Same(t, a, b)

	assert.Panics(t, func() { r.Register("other", "1.0.0") })
}

func TestConcurrentResolveAfterFreeze(t *testing.T) {
	t.Parallel()

	r := NewRegistry[int]()
	for i := 0; i < 100; i++ {
		r.Register(i, "1.0.0")
	}
	r.Freeze()

	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	sets := make(chan *FeatureSet[int], goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			sets <- r.MustResolve("1.0.0")
		}()
	}
	wg.Wait()
	close(sets)

	var first *FeatureSet[int]
	for set := range sets {
		assert.Len(t, set.All(), 100)
		if first == nil {
			first = set
			continue
		}
		assert.Same(t, first, set)
	}
}

func TestSinceUntilDump(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().
		Register("zeta", "1.0.0").
		RegisterRange("alpha", "1.1.0", "2.0.0")

	since, ok := r.SinceOf("alpha")
	assert.True(t, ok)
	assert.Equal(t, "1.1.0", since)

	until, ok := r.UntilOf("alpha")
	assert.True(t, ok)
	assert.Equal(t, "2.0.0", until)

	until, ok = r.UntilOf("zeta")
	assert.True(t, ok)
	assert.Equal(t, "latest", until)

	_, ok = r.SinceOf("missing")
	assert.False(t, ok)

	want := []FeatureRange[string]{
		{Feature: "alpha", Since: "1.1.0", Until: "2.0.0"},
		{Feature: "zeta", Since: "1.0.0", Until: "latest"},
	}
	assert.Equal(t, want, r.Dump())
}

func TestValidationPanicsAndErrors(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { NewRegistry[string]().Register("bad", "1.2") })
	assert.Panics(t, func() { NewRegistry[string]().RegisterRange("bad", "1.0.0", "") })
	assert.Panics(t, func() { NewRegistry[string]().RegisterRange("bad", "2.0.0", "1.0.0") })
	assert.Panics(t, func() {
		NewRegistry[string]().
			Register("dup", "1.0.0").
			Register("dup", "1.1.0")
	})

	_, err := NewRegistry[string]().Resolve("1.2")
	assert.Error(t, err)
	assert.Panics(t, func() { NewRegistry[string]().MustResolve("1.2") })
}

func TestFeatureSetHelpers(t *testing.T) {
	t.Parallel()

	fs := NewRegistry[string]().
		Register("b", "1.0.0").
		Register("a", "1.0.0").
		MustResolve("1.0.0")

	all := fs.All()
	sort.Strings(all)
	assert.Equal(t, []string{"a", "b"}, all)
	fs.MustHave("a")
	assert.Panics(t, func() { fs.MustHave("missing") })

	var nilSet *FeatureSet[string]
	assert.Empty(t, nilSet.Version())
	assert.False(t, nilSet.Has("x"))
	assert.False(t, nilSet.HasAny("x"))
	assert.True(t, nilSet.HasAll())
	assert.Nil(t, nilSet.All())
}

func TestDefaultRegistry(t *testing.T) {
	resetDefaultRegistryForTest()
	t.Cleanup(resetDefaultRegistryForTest)

	ConfigureDefault(WithIgnorePrerelease())
	Register("dark_mode", "1.2.0")
	RegisterRange("legacy_login", "1.0.0", "2.0.0")
	Freeze()

	fs, err := Resolve("1.2.0-rc.1")
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, fs.HasAll("dark_mode", "legacy_login"))

	since, ok := SinceOf("dark_mode")
	assert.True(t, ok)
	assert.Equal(t, "1.2.0", since)

	assert.Len(t, Dump(), 2)
	assert.Panics(t, func() { ConfigureDefault() })
}

func resetDefaultRegistryForTest() {
	defaultRegistry = NewRegistry[string]()
}

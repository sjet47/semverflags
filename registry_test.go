package semverflags

import (
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestRegistryResolveRanges(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().
		Register("base", "1.0.0").
		Register("dark_mode", "1.2.0").
		RegisterRange("legacy_login", "1.0.0", "2.0.0")
	r.Freeze()

	fs, err := r.Resolve("1.5.0")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if fs.Version() != "1.5.0" {
		t.Fatalf("Version() = %q", fs.Version())
	}
	if !fs.Has("base") || !fs.Has("dark_mode") || !fs.Has("legacy_login") {
		t.Fatalf("expected all features at 1.5.0, got %#v", fs.All())
	}
	if !fs.HasAll("base", "dark_mode") {
		t.Fatalf("HasAll returned false")
	}
	if !fs.HasAny("missing", "legacy_login") {
		t.Fatalf("HasAny returned false")
	}
	if fs.Has("missing") {
		t.Fatalf("unexpected missing feature")
	}

	fs, err = r.Resolve("2.0.0")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if fs.Has("legacy_login") {
		t.Fatalf("legacy_login should be removed at exclusive upper bound")
	}
	if !fs.HasAll("base", "dark_mode") {
		t.Fatalf("base and dark_mode should still be supported")
	}
}

func TestRegistryResolvePrerelease(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().Register("feature", "1.2.3")
	fs, err := r.Resolve("1.2.3-rc.1")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if fs.Has("feature") {
		t.Fatalf("prerelease should compare lower than final release by default")
	}

	r = NewRegistry[string](WithIgnorePrerelease()).Register("feature", "1.2.3")
	fs, err = r.Resolve("1.2.3-rc.1")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !fs.Has("feature") {
		t.Fatalf("WithIgnorePrerelease should treat rc as final release")
	}
	if fs.Version() != "1.2.3" {
		t.Fatalf("Version() = %q, want normalized 1.2.3", fs.Version())
	}
}

func TestFreezeEnablesCacheAndRejectsRegister(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().Register("feature", "1.0.0")
	beforeA := r.MustResolve("1.0.0")
	beforeB := r.MustResolve("1.0.0")
	if beforeA == beforeB {
		t.Fatalf("unfrozen registry should recompute feature sets")
	}

	r.Freeze()
	a := r.MustResolve("1.0.0")
	b := r.MustResolve("1.0.0")
	if a != b {
		t.Fatalf("frozen registry should return cached feature set for same version")
	}

	assertPanic(t, func() { r.Register("other", "1.0.0") })
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
		if len(set.All()) != 100 {
			t.Fatalf("expected 100 features, got %d", len(set.All()))
		}
		if first == nil {
			first = set
			continue
		}
		if first != set {
			t.Fatalf("expected all goroutines to observe same cached set pointer")
		}
	}
}

func TestSinceUntilDump(t *testing.T) {
	t.Parallel()

	r := NewRegistry[string]().
		Register("zeta", "1.0.0").
		RegisterRange("alpha", "1.1.0", "2.0.0")

	since, ok := r.SinceOf("alpha")
	if !ok || since != "1.1.0" {
		t.Fatalf("SinceOf(alpha) = %q, %v", since, ok)
	}
	until, ok := r.UntilOf("alpha")
	if !ok || until != "2.0.0" {
		t.Fatalf("UntilOf(alpha) = %q, %v", until, ok)
	}
	until, ok = r.UntilOf("zeta")
	if !ok || until != "latest" {
		t.Fatalf("UntilOf(zeta) = %q, %v", until, ok)
	}
	if _, ok := r.SinceOf("missing"); ok {
		t.Fatalf("missing feature should not exist")
	}

	want := []FeatureRange[string]{
		{Feature: "alpha", Since: "1.1.0", Until: "2.0.0"},
		{Feature: "zeta", Since: "1.0.0", Until: "latest"},
	}
	if got := r.Dump(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Dump() = %#v, want %#v", got, want)
	}
}

func TestValidationPanicsAndErrors(t *testing.T) {
	t.Parallel()

	assertPanic(t, func() { NewRegistry[string]().Register("bad", "1.2") })
	assertPanic(t, func() { NewRegistry[string]().RegisterRange("bad", "1.0.0", "") })
	assertPanic(t, func() { NewRegistry[string]().RegisterRange("bad", "2.0.0", "1.0.0") })
	assertPanic(t, func() {
		NewRegistry[string]().
			Register("dup", "1.0.0").
			Register("dup", "1.1.0")
	})

	if _, err := NewRegistry[string]().Resolve("1.2"); err == nil {
		t.Fatalf("Resolve should reject invalid semver")
	}
	assertPanic(t, func() { NewRegistry[string]().MustResolve("1.2") })
}

func TestFeatureSetHelpers(t *testing.T) {
	t.Parallel()

	fs := NewRegistry[string]().
		Register("b", "1.0.0").
		Register("a", "1.0.0").
		MustResolve("1.0.0")

	all := fs.All()
	sort.Strings(all)
	if !reflect.DeepEqual(all, []string{"a", "b"}) {
		t.Fatalf("All() = %#v", all)
	}
	fs.MustHave("a")
	assertPanic(t, func() { fs.MustHave("missing") })

	var nilSet *FeatureSet[string]
	if nilSet.Version() != "" || nilSet.Has("x") || nilSet.HasAny("x") || !nilSet.HasAll() || nilSet.All() != nil {
		t.Fatalf("nil FeatureSet helpers returned unexpected values")
	}
}

func TestDefaultRegistry(t *testing.T) {
	resetDefaultRegistryForTest()
	t.Cleanup(resetDefaultRegistryForTest)

	ConfigureDefault(WithIgnorePrerelease())
	Register("dark_mode", "1.2.0")
	RegisterRange("legacy_login", "1.0.0", "2.0.0")
	Freeze()

	fs, err := Resolve("1.2.0-rc.1")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !fs.HasAll("dark_mode", "legacy_login") {
		t.Fatalf("default registry did not resolve expected features")
	}
	if since, ok := SinceOf("dark_mode"); !ok || since != "1.2.0" {
		t.Fatalf("SinceOf(default) = %q, %v", since, ok)
	}
	if got := Dump(); len(got) != 2 {
		t.Fatalf("Dump(default) length = %d", len(got))
	}
	assertPanic(t, func() { ConfigureDefault() })
}

func assertPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}

func resetDefaultRegistryForTest() {
	defaultRegistry = NewRegistry[string]()
}

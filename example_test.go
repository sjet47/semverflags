package semverflags_test

import (
	"fmt"
	"sort"

	"github.com/sjet47/semverflags"
)

func Example() {
	type Feature string

	const (
		FeatureDarkMode    Feature = "dark_mode"
		FeaturePushNotify  Feature = "push_notify"
		FeatureLegacyLogin Feature = "legacy_login"
	)

	registry := semverflags.NewRegistry[Feature](semverflags.WithIgnorePrerelease())
	registry.Register(FeatureDarkMode, "1.2.0")
	registry.Register(FeaturePushNotify, "1.5.0")
	registry.RegisterRange(FeatureLegacyLogin, "1.0.0", "2.0.0")
	registry.Freeze()

	features := registry.MustResolve("1.5.0-rc.1")

	fmt.Println("version:", features.Version())
	fmt.Println("has dark mode:", features.Has(FeatureDarkMode))
	fmt.Println("has push notify:", features.Has(FeaturePushNotify))
	fmt.Println("has legacy login:", features.Has(FeatureLegacyLogin))

	all := features.All()
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	fmt.Println("all features:", all)

	for _, featureRange := range registry.Dump() {
		fmt.Printf("%s: [%s, %s)\n", featureRange.Feature, featureRange.Since, featureRange.Until)
	}

	// Output:
	// version: 1.5.0
	// has dark mode: true
	// has push notify: true
	// has legacy login: true
	// all features: [dark_mode legacy_login push_notify]
	// dark_mode: [1.2.0, latest)
	// legacy_login: [1.0.0, 2.0.0)
	// push_notify: [1.5.0, latest)
}

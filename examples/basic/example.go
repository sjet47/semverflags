package main

import (
	"fmt"
	"sort"

	"github.com/sjet47/semverflags"
)

type Feature string

const (
	FeatureDarkMode    Feature = "dark_mode"
	FeaturePushNotify  Feature = "push_notify"
	FeatureLegacyLogin Feature = "legacy_login"
)

func main() {
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
}

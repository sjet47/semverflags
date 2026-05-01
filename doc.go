// Package semverflags maps semantic versions to supported feature flags.
//
// A Registry is populated during initialization, optionally frozen, and then
// resolved for concrete versions to obtain read-only FeatureSet values.
package semverflags

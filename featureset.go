package semverflags

import "fmt"

// FeatureSet represents the read-only set of features supported by a specific
// version. It is safe for concurrent reads.
type FeatureSet[F comparable] struct {
	version  string
	features map[F]struct{}
}

func newFeatureSet[F comparable](version string, features map[F]struct{}) *FeatureSet[F] {
	if features == nil {
		features = make(map[F]struct{})
	}
	return &FeatureSet[F]{
		version:  version,
		features: features,
	}
}

// Version returns the normalized version string this set was resolved for.
func (s *FeatureSet[F]) Version() string {
	if s == nil {
		return ""
	}
	return s.version
}

// Has reports whether feature is supported.
func (s *FeatureSet[F]) Has(feature F) bool {
	if s == nil {
		return false
	}
	_, ok := s.features[feature]
	return ok
}

// HasAll reports whether all features are supported.
func (s *FeatureSet[F]) HasAll(features ...F) bool {
	if s == nil {
		return len(features) == 0
	}
	for _, feature := range features {
		if !s.Has(feature) {
			return false
		}
	}
	return true
}

// HasAny reports whether at least one of the given features is supported.
func (s *FeatureSet[F]) HasAny(features ...F) bool {
	if s == nil {
		return false
	}
	for _, feature := range features {
		if s.Has(feature) {
			return true
		}
	}
	return false
}

// MustHave panics when feature is not supported.
func (s *FeatureSet[F]) MustHave(feature F) {
	if !s.Has(feature) {
		panic(fmt.Sprintf("semverflags: feature %v is not supported by version %q", feature, s.Version()))
	}
}

// All returns all supported features. The order is not guaranteed to be stable.
func (s *FeatureSet[F]) All() []F {
	if s == nil || len(s.features) == 0 {
		return nil
	}
	features := make([]F, 0, len(s.features))
	for feature := range s.features {
		features = append(features, feature)
	}
	return features
}

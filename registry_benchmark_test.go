package semverflags

import (
	"fmt"
	"testing"
)

var benchmarkFeatureSetSink *FeatureSet[int]

func BenchmarkRegistryResolveColdLatest(b *testing.B) {
	for _, featureCount := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("features=%d", featureCount), func(b *testing.B) {
			r := newBenchmarkRegistry(featureCount)
			r.Freeze()

			// Sanity check outside the measured section: latest versions should hit
			// most features, while a small fraction of old features has been removed.
			fs := r.MustResolve("2000000.0.0+sanity")
			wantActive := featureCount - removedBenchmarkFeatureCount(featureCount)
			if got := len(fs.features); got != wantActive {
				b.Fatalf("active feature count = %d, want %d", got, wantActive)
			}

			b.ReportAllocs()
			b.ReportMetric(float64(featureCount), "registered_features")
			b.ReportMetric(float64(wantActive), "active_features")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fs, err := r.resolve("2000000.0.0")
				if err != nil {
					b.Fatal(err)
				}
				benchmarkFeatureSetSink = fs
			}
		})
	}
}

func newBenchmarkRegistry(featureCount int) *Registry[int] {
	r := NewRegistry[int]()
	for i := 0; i < featureCount; i++ {
		feature := i
		since := fmt.Sprintf("%d.0.0", i+1)
		if isRemovedBenchmarkFeature(i) {
			r.RegisterRange(feature, since, "1000000.0.0")
			continue
		}
		r.Register(feature, since)
	}
	return r
}

func isRemovedBenchmarkFeature(i int) bool {
	return i%10 == 0
}

func removedBenchmarkFeatureCount(featureCount int) int {
	if featureCount <= 0 {
		return 0
	}
	return (featureCount + 9) / 10
}

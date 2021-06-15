package blackscholes

import "testing"

/**
 * goos: windows
 * goarch: amd64
 * pkg: github.com/charlerive/library/blackscholes
 * BenchmarkRpc-12           706713              1706 ns/op             160 B/op          1 allocs/op
 * BenchmarkBs-12           4471878               268 ns/op             160 B/op          1 allocs/op
 * PASS
 * ok      github.com/charlerive/library/blackscholes      3.199s
 */
func BenchmarkRpc(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewBS("p", 4600, 5000, 0.1644, 0.025, 996.27, 0.01, 3, 0.3)
	}
}

func BenchmarkBs(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewBSWithIv("p", 4600, 5000, 0.1644, 0.025, 996.27, 0.01, 3, 0.3, 1.0304)
	}
}

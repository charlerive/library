package blackscholes

import (
	"math"
	"strings"
	"testing"
)

func TestNewBS(t *testing.T) {
	bs := NewBS("p", 4600, 5000, 0.1644, 0.025, 996.27, 0.01, 3, 0.3)
	t.Logf("bs: %+v", bs)
}

func TestNewBSWithIv(t *testing.T) {
	bs := NewBSWithIv("p", 4600, 5000, 0.1644, 0.025, 996.27, 0.01, 3, 0.3, 1.0304)
	t.Logf("bs: %+v", bs)
}

/**
 * goos: windows
 * goarch: amd64
 * pkg: github.com/charlerive/library/blackscholes
 * BenchmarkRpc-12          1348222               833 ns/op             176 B/op          1 allocs/op
 * BenchmarkBs-12           6683582               181 ns/op             176 B/op          1 allocs/op
 * PASS
 * ok      github.com/charlerive/library/blackscholes      3.199s
 */

/**
 * goos: darwin
 * goarch: arm64
 * pkg: github.com/charlerive/library/blackscholes
 * BenchmarkRpc-8          2768335               413.6 ns/op
 * BenchmarkBs-8           9026594               132 ns/op
 * PASS
 */

func BenchmarkRpc(b *testing.B) {
	bsm := BSM{
		D:         strings.ToLower("p"),
		S:         4600,
		X:         5000,
		T:         0.1644,
		R:         0.025,
		Op:        996.27,
		OpEpsilon: 0.01,
		IvMax:     3,
		IvMin:     0.3,
		ExtractT:  math.Sqrt(0.1644),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bsm.Iv = 0
		bsm.Init()
	}
}

func BenchmarkBs(b *testing.B) {
	bsm := BSM{
		D:         strings.ToLower("p"),
		S:         4600,
		X:         5000,
		T:         0.1644,
		R:         0.025,
		Op:        996.27,
		OpEpsilon: 0.01,
		Iv:        1.0304,
		IvMax:     3,
		IvMin:     0.3,
		ExtractT:  math.Sqrt(0.1644),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bsm.Init()
	}
}

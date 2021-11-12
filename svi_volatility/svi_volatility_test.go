package svi_volatility

import (
	/*"github.com/go-nlopt/nlopt"*/
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
	"log"
	"math"
	"math/rand"
	"testing"
)

func TestSviVolatility_InitialParam(t *testing.T) {
	s := generateS()
	s.InitParams(marketDataList)
	t.Logf("sviParam: %+v", s.SviParamsOld)

	s1 := generateS1()
	s1.InitParams(marketDataList1)
	t.Logf("sviParam1: %+v", s1.SviParams)
}

var marketDataList = []*MarketDataOld{
	{
		StrikePrice: 3500,
		ImVol:       0.31203,
	},
	{
		StrikePrice: 4000,
		ImVol:       0.25041,
	},
	{
		StrikePrice: 4500,
		ImVol:       0.19897,
	},
	{
		StrikePrice: 5000,
		ImVol:       0.15795,
	},
	{
		StrikePrice: 5500,
		ImVol:       0.13803,
	},
	{
		StrikePrice: 6000,
		ImVol:       0.14575,
	},
	{
		StrikePrice: 6400,
		ImVol:       0.17007,
	},
}

var marketDataList1 = []*MarketData{
	{
		K:     math.Log(3500 / 5066.5),
		ImVol: 0.31203,
	},
	{
		K:     math.Log(4000 / 5066.5),
		ImVol: 0.25041,
	},
	{
		K:     math.Log(4500 / 5066.5),
		ImVol: 0.19897,
	},
	{
		K:     math.Log(5000 / 5066.5),
		ImVol: 0.15795,
	},
	{
		K:     math.Log(5500 / 5066.5),
		ImVol: 0.13803,
	},
	{
		K:     math.Log(6000 / 5066.5),
		ImVol: 0.14575,
	},
	{
		K:     math.Log(6400 / 5066.5),
		ImVol: 0.17007,
	},
}

func generateS() *SviVolatilityOld {
	s := NewSviVolatilityOld(5066.5, 0.00194)
	return s
}

func generateS1() *SviVolatility {
	// 5066.5
	s := NewSviVolatility(0.00194)
	return s
}

func TestDot(t *testing.T) {
	//arr := []float64{-0.004122108414143455, -0.007765255474848909, 0.00093046102290259, 0.011055138146279508, -0.0038147592962383403, -0.017327769969704115, -0.01141037002450157}
	arr := []float64{-0.006839912841306139, -0.013589024749367351, -0.010900779696139368, -0.005214032253557627, -0.005017253760173035, -0.008549567065650965, -0.00246918537765195}
	transpose := mat.NewDense(1, 7, arr)
	log.Printf("rTranspose.RowView(0): %+v", transpose.RowView(0))
	log.Printf("mat.Dot(rTransposeV, rTransposeV): %+v", mat.Dot(transpose.RowView(0), transpose.RowView(0)))

	sum := 0.0
	for _, v := range arr {
		sum += v * v
	}
	log.Printf("sum: %+v", math.Sqrt(sum))
}

func TestSviVolatility_FVector(t *testing.T) {
	s := generateS()
	s.InitParams(marketDataList)
	s.FVector(s.xMatrix, s.SviParamsOld)
}

func TestSviVolatility_GradFMatrix(t *testing.T) {
	s := generateS()
	s.InitParams(marketDataList)
	ma := s.GradFMatrix(s.xMatrix, s.SviParamsOld)
	r, _ := ma.Dims()
	for i := 0; i < r; i++ {
		log.Printf("ma.Row: %+v", ma.RowView(i))
	}
}

func TestSviVolatility_FitVol(t *testing.T) {
	s := generateS()
	s.InitParams(marketDataList)
	t.Logf("sviParam: %+v", s.FitVol())

	s1 := generateS1()
	s1.InitParams(marketDataList1)
	t.Logf("sviParam1: %+v", s1.FitVol())
}

// goos: windows
// goarch: amd64
// pkg: github.com/charlerive/library/svi_volatility
// BenchmarkSviVolatility_FitVol
// BenchmarkSviVolatility_FitVol-12            7077            157725 ns/op
// PASS
func BenchmarkSviVolatility_FitVol(b *testing.B) {
	s := generateS()
	s.InitParams(marketDataList)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.FitVol()
	}
}

func BenchmarkSviVolatility1_FitVol(b *testing.B) {
	s1 := generateS1()
	s1.InitParams(marketDataList1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1.FitVol()
	}
}

// goos: windows
// goarch: amd64
// pkg: github.com/charlerive/library/svi_volatility
// BenchmarkSviVolatility_FitVol
// BenchmarkSviVolatility_FitVol-12            57229791            20.7 ns/op
// PASS
func BenchmarkSviVolatility_GetImVol(b *testing.B) {
	b.ResetTimer()
	p := &SviParamsOld{
		A:   -0.00011482356495239037,
		B:   0.0005530503392993171,
		C:   0.2740588979212063,
		Rho: -0.03953196370937801,
		Eta: 0.08143389727211306,
	}
	s := generateS()
	for i := 0; i < b.N; i++ {
		s.GetImVol(3500, p)
	}
	log.Printf("%+v", s.GetImVol(3500, p))
}

func BenchmarkSviVolatility1_GetImVol(b *testing.B) {
	b.ResetTimer()
	p := &SviParams{
		A:   -0.00011482356495239037,
		B:   0.0005530503392993171,
		C:   0.2740588979212063,
		Rho: -0.03953196370937801,
		Eta: 0.08143389727211306,
	}
	s := generateS1()
	k := math.Log(3500 / 5066.5)
	for i := 0; i < b.N; i++ {
		s.GetImVol(k, p)
	}
	log.Printf("%+v", s.GetImVol(k, p))
}

func TestNorm(t *testing.T) {
	vDense := mat.NewVecDense(7, []float64{-0.00058974, 0.00034847, 0.00037835, 0.00069335, -0.00083113, 0.00111637, -0.00064025})
	t.Logf("norm vDense: %f", mat.Norm(vDense, 2))
}

func TestLeastSquares(t *testing.T) {
	// f(x) = a * x^2 + b
	// f(x) = 1.5 * x^2 + 3
	xList, yList := make([]float64, 0), make([]float64, 0)
	for i := 0; i < 100; i++ {
		cur := rand.Float64()
		xList = append(xList, cur)
		yList = append(yList, 1.5*cur*cur+3)
	}

	pro := optimize.Problem{
		Func: func(x []float64) float64 {
			res := 0.0
			for i, item := range xList {
				cur := x[0]*item*item + x[1] - yList[i]
				res += cur * cur
			}
			return res
		},
	}
	result, err := optimize.Minimize(pro, []float64{1, 1}, &optimize.Settings{}, nil)
	if err == nil {
		log.Printf("result: %+v", result)
	}
	if err != nil {
		log.Printf("err: %+v", err)
	}
}

func TestSviVolatility_PyMinimizeSLSQP(t *testing.T) {
	marketDataList = []*MarketDataOld{
		{
			StrikePrice: 30000,
			ImVol:       1.076,
		},
		{
			StrikePrice: 32000,
			ImVol:       0.966,
		},
		{
			StrikePrice: 34000,
			ImVol:       0.905,
		},
		{
			StrikePrice: 36000,
			ImVol:       0.824,
		},
		{
			StrikePrice: 38000,
			ImVol:       0.868,
		},
		{
			StrikePrice: 40000,
			ImVol:       0.804,
		},
		{
			StrikePrice: 45000,
			ImVol:       1.211,
		},
	}
	s := NewSviVolatilityOld(34940.32, 0.008789954)
	param, err := s.PyMinimizeSLSQP(marketDataList)
	if err != nil {
		t.Errorf("s.PyMinimizeSLSQP fail. err: %s", err)
		return
	}
	t.Logf("s.PyMinimizeSLSQP success. param: %+v", param)
}

func TestSviVolatility1_PyMinimizeSLSQP(t *testing.T) {
	marketDataList1 = []*MarketData{
		{
			K:     math.Log(30000 / 34940.32),
			ImVol: 1.076,
		},
		{
			K:     math.Log(32000 / 34940.32),
			ImVol: 0.966,
		},
		{
			K:     math.Log(34000 / 34940.32),
			ImVol: 0.905,
		},
		{
			K:     math.Log(36000 / 34940.32),
			ImVol: 0.824,
		},
		{
			K:     math.Log(38000 / 34940.32),
			ImVol: 0.868,
		},
		{
			K:     math.Log(40000 / 34940.32),
			ImVol: 0.804,
		},
		{
			K:     math.Log(45000 / 34940.32),
			ImVol: 1.211,
		},
	}
	s := NewSviVolatility(0.008789954)
	param, err := s.PyMinimizeSLSQP(marketDataList1)
	if err != nil {
		t.Errorf("s.PyMinimizeSLSQP fail. err: %s", err)
		return
	}
	t.Logf("s.PyMinimizeSLSQP success. param: %+v", param)
}

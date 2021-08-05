package svi_volatility

import (
	"github.com/go-nlopt/nlopt"
	"gonum.org/v1/gonum/mat"
	"log"
	"math"
	"testing"
)

func TestSviVolatility_InitialParam(t *testing.T) {
	s := generateS()
	s.InitParams(marketDataList)
	t.Logf("sviParam: %+v", s.SviParams)
}

var marketDataList = []*MarketData{
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

func generateS() *SviVolatility {
	s := NewSviVolatility(5066.5, 0.00194)
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
	s.FVector(s.xMatrix, s.SviParams)
}

func TestSviVolatility_GradFMatrix(t *testing.T) {
	s := generateS()
	s.InitParams(marketDataList)
	ma := s.GradFMatrix(s.xMatrix, s.SviParams)
	r, _ := ma.Dims()
	for i := 0; i < r; i++ {
		log.Printf("ma.Row: %+v", ma.RowView(i))
	}
}

func TestSviVolatility_FitVol(t *testing.T) {
	s := generateS()
	s.InitParams(marketDataList)
	t.Logf("sviParam: %+v", s.FitVol())
}

// goos: windows
// goarch: amd64
// pkg: github.com/charlerive/library/svi_volatility
// BenchmarkSviVolatility_FitVol
// BenchmarkSviVolatility_FitVol-12            7077            157725 ns/op
// PASS
func BenchmarkSviVolatility_FitVol(b *testing.B) {
	b.ResetTimer()
	s := generateS()
	s.InitParams(marketDataList)
	for i := 0; i < b.N; i++ {
		s.FitVol()
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
	p := &SviParams{
		A:   -0.00011482356495239037,
		B:   0.0005530503392993171,
		C:   0.2740588979212063,
		Rho: -0.03953196370937801,
		Eta: 0.08143389727211306,
	}
	for i := 0; i < b.N; i++ {
		s := generateS()
		s.GetImVol(3500, p)
	}
}

func TestNorm(t *testing.T) {
	vDense := mat.NewVecDense(7, []float64{-0.00058974, 0.00034847, 0.00037835, 0.00069335, -0.00083113, 0.00111637, -0.00064025})
	t.Logf("norm vDense: %f", mat.Norm(vDense, 2))
}

func TestSviVolatility_InitParamsForSLSQP(t *testing.T) {
	marketDataList = []*MarketData{
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
	s := NewSviVolatility(34940.32, 0.008789954)
	s.InitParamsForSLSQP(marketDataList)
	t.Logf("svi: %+v", s.SviParams)
}

func TestSviVolatility_GetImVol(t *testing.T) {
	s := NewSviVolatility(34940.32, 0.008789954)
	p := &SviParams{
		A:   -1.7939244727839825,
		B:   1.5006777096275756,
		C:   1.7567226362270612,
		Rho: 0.7306938504972463,
		Eta: 1.920311710946073,
	}
	log.Printf("%+v", s.GetImVol(30000, p))
	log.Printf("%+v", s.GetImVol(32000, p))
	log.Printf("%+v", s.GetImVol(34000, p))
	log.Printf("%+v", s.GetImVol(36000, p))
	log.Printf("%+v", s.GetImVol(38000, p))
	log.Printf("%+v", s.GetImVol(40000, p))
	log.Printf("%+v", s.GetImVol(45000, p))
}

func TestLeastSquares(t *testing.T) {
	// f(x) = x1^2 + x2^2
	var minFunc = func(x, gradient []float64) float64 {
		if len(gradient) > 0 {
			gradient[0] = 2 * x[0]
			gradient[1] = 2 * x[1]
		}
		log.Printf("x: %+v", x)
		log.Printf("gradient: %+v", gradient)
		return x[0]*x[0] + x[1]*x[1]
	}
	opt, err := nlopt.NewNLopt(nlopt.LD_SLSQP, 2)
	if err != nil {
		return
	}
	defer opt.Destroy()

	err = opt.SetMinObjective(minFunc)

	if err != nil {
		log.Printf("err: %+v", err)
	}
	_ = opt.SetXtolRel(1e-6)
	_ = opt.SetFtolRel(1e-6)
	param, f, err := opt.Optimize([]float64{100.0, 100.0})
	log.Printf("param: %+v, f: %+v, err: %+v", param, f, err)
}

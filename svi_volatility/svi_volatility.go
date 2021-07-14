package svi_volatility

import (
	"gonum.org/v1/gonum/mat"
	"log"
	"math"
)

type SviParams struct {
	A   float64 // 方差大小
	B   float64 // 渐近线夹角
	C   float64 // 平滑度
	Rho float64 // 旋转
	Eta float64 // 平移
}

func (s *SviParams) Copy() *SviParams {
	return &SviParams{
		A:   s.A,
		B:   s.B,
		C:   s.C,
		Rho: s.Rho,
		Eta: s.Eta,
	}
}

type MarketData struct {
	StrikePrice float64
	ImVol       float64
}

type SviVolatility struct {
	*SviParams
	MarketDataList []*MarketData
	xMatrix        *mat.VecDense
	yMatrix        *mat.VecDense
	ForwardPrice   float64 // 远期价格
	T              float64 // （期权到期日-当前时间）/365天
	kMap           map[float64]float64
}

func NewSviVolatility(ForwardPrice float64, T float64) *SviVolatility {
	s := &SviVolatility{
		ForwardPrice: ForwardPrice,
		T:            T,
	}
	return s
}

// 曲线拟合返回参数
func (s *SviVolatility) FitVol() *SviParams {
	// prepare to call the Levenberg-Marquardt method
	return s.LMFit(s.xMatrix, s.yMatrix, s.SviParams)
}

// 根据参数和行权价格找到波动率
func (s *SviVolatility) GetImVol(strikePrice float64, p *SviParams) float64 {
	kM := math.Log(strikePrice/s.ForwardPrice) - p.Eta
	return math.Sqrt(math.Abs(Variance(kM, p.A, p.B, p.C, p.Rho) / s.T))
}

func Variance(kM, a, b, c, rho float64) float64 {
	return a + b*(rho*kM+math.Sqrt(kM*kM+c*c))
}

func (s *SviVolatility) InitParams(marketDataList []*MarketData) {

	s.SviParams = &SviParams{}
	s.kMap = make(map[float64]float64)
	s.MarketDataList = marketDataList
	s.xMatrix = mat.NewVecDense(len(marketDataList), nil)
	s.yMatrix = mat.NewVecDense(len(marketDataList), nil)
	for i, marketData := range marketDataList {
		s.xMatrix.SetVec(i, marketData.StrikePrice)
		s.yMatrix.SetVec(i, marketData.ImVol)
		k := math.Log(marketData.StrikePrice / s.ForwardPrice)
		s.kMap[marketData.StrikePrice] = k
	}

	moneynessArr := make([]float64, 0)
	// 方差
	varianceArr := make([]float64, 0)
	dataLen := len(s.MarketDataList)
	for _, marketData := range s.MarketDataList {
		moneynessArr = append(moneynessArr, math.Log(marketData.StrikePrice/s.ForwardPrice))
		varianceArr = append(varianceArr, marketData.ImVol*marketData.ImVol*s.T)
	}

	x1, x2, y1, y2 := 0.0, 0.0, 0.0, 0.0
	al, ar, bl, br := 0.0, 0.0, 0.0, 0.0

	// coefficients left asymptotics
	x1, x2, y1, y2 = moneynessArr[0], moneynessArr[1], varianceArr[0], varianceArr[1]
	al = (x1*y2 - y1*x2) / (x1 - x2)
	bl = -math.Abs((y2 - y1) / (x2 - x1))

	// coefficients right asymptotics
	x1, x2, y1, y2 = moneynessArr[dataLen-2], moneynessArr[dataLen-1], varianceArr[dataLen-2], varianceArr[dataLen-1]
	ar = (x1*y2 - y1*x2) / (x1 - x2)
	br = math.Abs((y2 - y1) / (x2 - x1))

	// params through asymptotics
	s.B = (br - bl) / 2
	s.Rho = (bl + br) / (br - bl)
	s.A = al + bl*(-al+ar)/(bl-br)
	s.Eta = bl * (-al + ar) / (bl - br) / s.B / (s.Rho - 1)

	// s.C = smoothing the vertex at the minimum
	// minimum by brute force instead of WorksheetFunction.min
	miniVariance := y1
	for _, v := range varianceArr {
		if miniVariance > v {
			miniVariance = v
		}
	}
	s.C = -(-miniVariance + al + bl*(-al+ar)/(bl-br))
	s.C = s.C / s.B / math.Sqrt(math.Abs(1-s.Rho*s.Rho))
}

const (
	Nu0           = 1000
	Res0          = 1e9
	MaxIterations = 25
	Tolerance     = 1e-8
)

// Levenberg-Marquardt 最小二乘法
func (s *SviVolatility) LMFit(x, y *mat.VecDense, pStart *SviParams) *SviParams {
	resZero := Res0
	nu := float64(Nu0)
	dataLen := len(s.MarketDataList)
	p := pStart.Copy()
	rTranspose := mat.NewDense(1, dataLen, nil)
	for i := 0; i < MaxIterations; i++ {
		fv := s.FVector(x, p)
		for j := 0; j < dataLen; j++ {
			rTranspose.Set(0, j, y.At(j, 0)-fv.AtVec(j))
		}

		res1 := math.Sqrt(mat.Dot(rTranspose.RowView(0), rTranspose.RowView(0)))

		tmpGradFMatrix := s.GradFMatrix(x, p)

		betaTranspose := &mat.Dense{}
		betaTranspose.Mul(rTranspose, tmpGradFMatrix)

		beta := mat.DenseCopyOf(betaTranspose.T())

		alpha := &mat.Dense{}
		alpha.Mul(tmpGradFMatrix.T(), tmpGradFMatrix)

		for j := 0; j < 5; j++ {
			alpha.Set(j, j, alpha.At(j, j)*(1+1/nu))
		}

		dp := &mat.Dense{}
		err := dp.Solve(alpha, beta)
		if err != nil {
			log.Printf("LMFit dp.solve fail, err: %s, alpha: %+v, beta: %+v ", err, alpha, beta)
		}

		pNew := p.Copy()
		pNew.A = p.A + dp.At(0, 0)
		pNew.B = p.B + dp.At(1, 0)
		pNew.C = p.C + dp.At(2, 0)
		pNew.Rho = p.Rho + dp.At(3, 0)
		pNew.Eta = p.Eta + dp.At(4, 0)

		fv = s.FVector(x, pNew)
		for j := 0; j < dataLen; j++ {
			rTranspose.Set(0, j, y.At(j, 0)-fv.At(j, 0))
		}

		res := math.Sqrt(mat.Dot(rTranspose.RowView(0), rTranspose.RowView(0)))

		if res1 <= res {
			nu = nu / 10
		} else {
			nu = nu * 10
			p = pNew
		}

		if math.Abs(res-resZero) < Tolerance {
			break
		}
		resZero = res
	}
	return p
}

func (s *SviVolatility) FVector(x *mat.VecDense, p *SviParams) *mat.VecDense {
	dataLen := len(s.MarketDataList)
	outline := mat.NewVecDense(dataLen, nil)
	for i := 0; i < x.Len(); i++ {
		outline.SetVec(i, s.F(x.AtVec(i), p))
	}
	return mat.VecDenseCopyOf(outline.TVec())
}

func (s *SviVolatility) F(strikePrice float64, p *SviParams) float64 {
	k := s.kMap[strikePrice]
	kM := k - p.Eta
	return math.Sqrt(math.Abs((p.A + p.B*(p.Rho*kM+math.Sqrt(kM*kM+p.C*p.C))) / s.T))
}

func (s *SviVolatility) GradFMatrix(x *mat.VecDense, p *SviParams) *mat.Dense {
	outMatrixTranspose := mat.NewDense(5, x.Len(), nil)
	for i := 0; i < x.Len(); i++ {
		tmpGradFI := s.GradF(x.AtVec(i), p)
		for j := 0; j < 5; j++ {
			outMatrixTranspose.Set(j, i, tmpGradFI[j])
		}
	}
	return mat.DenseCopyOf(outMatrixTranspose.T())
}

func (s *SviVolatility) GradF(strikePrice float64, p *SviParams) []float64 {
	res := make([]float64, 5)
	k := s.kMap[strikePrice]
	kM := k - p.Eta
	ff := s.F(strikePrice, p)
	tmp := 1 / (2 * ff * s.T)
	tmpB := tmp * p.B
	tmp1 := Variance(kM, 0, 1, p.C, p.Rho)
	tmp2 := Variance(kM, 0, 1, p.C, 0)

	res[0] = tmp
	res[1] = tmp * tmp2
	res[2] = tmpB * p.C / tmp1
	res[3] = tmpB * kM
	res[4] = -tmpB * (p.Rho + kM/tmp1)

	return res
}

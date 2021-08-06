package svi_volatility

import (
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
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
	if s.A == 0 && s.B == 0 && s.C == 0 && s.Eta == 0 {
		return s.SviParams
	}
	return s.LMFit(s.xMatrix, s.yMatrix, s.SviParams)
}

// 根据参数和行权价格找到波动率
func (s *SviVolatility) GetImVol(strikePrice float64, p *SviParams) float64 {
	kM := math.Log(strikePrice/s.ForwardPrice) - p.Eta
	return math.Sqrt(math.Abs(Variance(kM, p.A, p.B, p.C, p.Rho) / s.T))
}

func (s *SviVolatility) GetVariance(strikePrice float64, p *SviParams) float64 {
	kM := math.Log(strikePrice/s.ForwardPrice) - p.Eta
	return Variance(kM, p.A, p.B, p.C, p.Rho)
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

	moneynessArr := make([]float64, 0)
	// 方差
	varianceArr := make([]float64, 0)
	for i, marketData := range s.MarketDataList {
		s.xMatrix.SetVec(i, marketData.StrikePrice)
		s.yMatrix.SetVec(i, marketData.ImVol)
		k := math.Log(marketData.StrikePrice / s.ForwardPrice)
		s.kMap[marketData.StrikePrice] = k
		moneynessArr = append(moneynessArr, k)
		varianceArr = append(varianceArr, marketData.ImVol*marketData.ImVol*s.T)
	}

	lx1, lx2, ly1, ly2, rx1, rx2, ry1, ry2 := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	al, ar, bl, br := 0.0, 0.0, 0.0, 0.0

	i := 0
	for ; varianceArr[i] == varianceArr[i+1]; i++ {
		if moneynessArr[i+1] > 0 {
			return
		}
	}
	j := len(varianceArr)
	for ; varianceArr[j] == varianceArr[j-1]; j-- {
		if moneynessArr[j-1] < 0 {
			return
		}
	}

	// coefficients left asymptotics
	lx1, lx2, ly1, ly2 = moneynessArr[i], moneynessArr[i+1], varianceArr[i], varianceArr[i+1]
	al = (lx1*ly2 - ly1*lx2) / (lx1 - lx2)
	bl = -math.Abs((ly2 - ly1) / (lx2 - lx1))

	// coefficients right asymptotics
	rx1, rx2, ry1, ry2 = moneynessArr[j-1], moneynessArr[j], varianceArr[j-1], varianceArr[j]
	ar = (rx1*ry2 - ry1*rx2) / (rx1 - rx2)
	br = math.Abs((ry2 - ry1) / (rx2 - rx1))

	// params through asymptotics
	s.B = (br - bl) / 2
	s.Rho = (bl + br) / (br - bl)
	if s.Rho > 0.99 {
		s.Rho = 0.99
	} else if s.Rho < -0.99 {
		s.Rho = -0.99
	}

	s.A = al + bl*(-al+ar)/(bl-br)
	s.Eta = bl * (-al + ar) / (bl - br) / s.B / (s.Rho - 1)

	// s.C = smoothing the vertex at the minimum
	// minimum by brute force instead of WorksheetFunction.min
	miniVariance := ry1
	for _, v := range varianceArr {
		if miniVariance > v {
			miniVariance = v
		}
	}
	s.C = -(-miniVariance + al + bl*(-al+ar)/(bl-br))
	s.C = s.C / s.B / math.Sqrt(math.Abs(1-s.Rho*s.Rho))
	if math.IsNaN(s.A) {
		s.A = 0
	}
	if math.IsNaN(s.B) {
		s.B = 0
	}
	if math.IsNaN(s.C) {
		s.C = 0
	}
	if math.IsNaN(s.Rho) {
		s.Rho = 0
	}
	if math.IsNaN(s.Eta) {
		s.Eta = 0
	}
}

const (
	Nu0           = 1000
	Res0          = 1e9
	MaxIterations = 25
	Tolerance     = 1e-8
	ParamsLen     = 5
)

// Levenberg-Marquardt 最小二乘法
func (s *SviVolatility) LMFit(x, y *mat.VecDense, pStart *SviParams) *SviParams {
	resZero := Res0
	nu := float64(Nu0)
	dataLen := x.Len()
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

		for j := 0; j < ParamsLen; j++ {
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
	if math.IsNaN(p.A) {
		p.A = 0
	}
	if math.IsNaN(p.B) {
		p.B = 0
	}
	if math.IsNaN(p.C) {
		p.C = 0
	}
	if math.IsNaN(p.Rho) {
		p.Rho = 0
	}
	if math.IsNaN(p.Eta) {
		p.Eta = 0
	}

	return p
}

func (s *SviVolatility) FVector(x *mat.VecDense, p *SviParams) *mat.VecDense {
	dataLen := x.Len()
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
	outMatrixTranspose := mat.NewDense(ParamsLen, x.Len(), nil)
	for i := 0; i < x.Len(); i++ {
		tmpGradFI := s.GradF(x.AtVec(i), p)
		for j := 0; j < ParamsLen; j++ {
			outMatrixTranspose.Set(j, i, tmpGradFI[j])
		}
	}
	return mat.DenseCopyOf(outMatrixTranspose.T())
}

func (s *SviVolatility) GradF(strikePrice float64, p *SviParams) []float64 {
	res := make([]float64, ParamsLen)
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

func TotalVariance(kList []float64, p *SviParams) []float64 {
	res := make([]float64, 0)
	for _, k := range kList {
		res = append(res, Variance(k-p.Eta, p.A, p.B, p.C, p.Rho))
	}
	return res
}

/** Right constraints **/
func RightConstraint1(x, gradient []float64) float64 {
	p := &SviParams{
		A:   x[0],
		B:   x[1],
		C:   x[2],
		Rho: x[3],
		Eta: x[4],
	}
	return ((4 - p.A + p.B*p.Eta*(p.Rho+1)) * (p.A - p.B*p.Eta*(p.Rho+1))) - (p.B * p.B * (p.Rho + 1) * (p.Rho + 1))
}
func RightConstraint2(x, gradient []float64) float64 {
	p := &SviParams{
		A:   x[0],
		B:   x[1],
		C:   x[2],
		Rho: x[3],
		Eta: x[4],
	}
	return 4 - (p.B * p.B * (p.Rho + 1) * (p.Rho + 1))
}

/** Left constraints **/
func LeftConstraint1(x, gradient []float64) float64 {
	p := &SviParams{
		A:   x[0],
		B:   x[1],
		C:   x[2],
		Rho: x[3],
		Eta: x[4],
	}
	return ((4 - p.A + p.B*p.Eta*(p.Rho-1)) * (p.A - p.B*p.Eta*(p.Rho-1))) - (p.B * p.B * (p.Rho - 1) * (p.Rho - 1))
}
func LeftConstraint2(x, gradient []float64) float64 {
	p := &SviParams{
		A:   x[0],
		B:   x[1],
		C:   x[2],
		Rho: x[3],
		Eta: x[4],
	}
	return 4 - (p.B * p.B * (p.Rho - 1) * (p.Rho - 1))
}

func Constraint(x, gradient []float64) float64 {
	p := &SviParams{
		A:   x[0],
		B:   x[1],
		C:   x[2],
		Rho: x[3],
		Eta: x[4],
	}
	return p.C * p.C
}

func Constraint2(x, gradient []float64) float64 {
	p := &SviParams{
		A:   x[0],
		B:   x[1],
		C:   x[2],
		Rho: x[3],
		Eta: x[4],
	}
	return p.Eta * p.Eta
}

// Objective function to optimize
func LeastSquares(x, kList, totImpliedVariance []float64) float64 {
	p := &SviParams{
		A:   x[0],
		B:   x[1],
		C:   x[2],
		Rho: x[3],
		Eta: x[4],
	}
	vList := TotalVariance(kList, p)
	for i, v := range vList {
		vList[i] = v - totImpliedVariance[i]
	}
	vDense := mat.NewVecDense(len(vList), vList)
	return mat.Norm(vDense, 2)
}

func (s *SviVolatility) InitParamsForSLSQP(marketDataList []*MarketData) *SviParams {
	kList := make([]float64, 0)
	vList := make([]float64, 0)

	kMin, kMax, vMin, vMax := math.MaxFloat64, math.MaxFloat64*-1, math.MaxFloat64, math.MaxFloat64*-1
	for _, marketData := range marketDataList {
		k := math.Log(marketData.StrikePrice / s.ForwardPrice)
		v := marketData.ImVol * marketData.ImVol * s.T
		kList = append(kList, k)
		vList = append(vList, v)
		if kMin > k {
			kMin = k
		}
		if kMax < k {
			kMax = k
		}
		if vMin > v {
			vMin = v
		}
		if vMax < v {
			vMax = v
		}
	}
	aLow, bLow, cLow, rhoLow, etaLow := 0.000001, 0.001, 0.001, -0.999999, 2*kMin
	aHigh, bHigh, cHigh, rhoHigh, etaHigh := vMax, 1., 2., 0.999999, 2*kMax
	aInit, bInit, cInit, rhoInit, etaInit := vMin/2, 0.1, 0.1, -0.5, 0.1
	//lowBounds := []float64{aLow, bLow, cLow, rhoLow, etaLow}
	//log.Printf("lowBounds: %+v", lowBounds)
	//upBounds := []float64{aHigh, bHigh, cHigh, rhoHigh, etaHigh}
	//log.Printf("upBounds: %+v", upBounds)
	paramInit := []float64{aInit, bInit, cInit, rhoInit, etaInit}
	//log.Printf("paramInit: %+v", paramInit)
	//log.Printf("klist: %+v", kList)
	//log.Printf("vlist: %+v", vList)
	pro := optimize.Problem{
		Func: func(x []float64) float64 {
			p := &SviParams{
				A:   x[0],
				B:   x[1],
				C:   x[2],
				Rho: x[3],
				Eta: x[4],
			}
			// bounds
			if p.A > aHigh || p.A < aLow || p.B > bHigh || p.B < bLow || p.C > cHigh || p.C < cLow || p.Rho > rhoHigh || p.Rho < rhoLow || p.Eta > etaHigh || p.Eta < etaLow {
				return math.MaxFloat64
			}
			// Constraint
			if LeftConstraint1(x, []float64{}) <= 0 || LeftConstraint2(x, []float64{}) <= 0 || RightConstraint1(x, []float64{}) <= 0 || RightConstraint2(x, []float64{}) <= 0 {
				return math.MaxFloat64
			}
			return LeastSquares(x, kList, vList)
		},
	}
	result, err := optimize.Minimize(pro, paramInit, &optimize.Settings{}, nil)
	if err == nil {
		s.SviParams = &SviParams{
			A:   result.X[0],
			B:   result.X[1],
			C:   result.X[2],
			Rho: result.X[3],
			Eta: result.X[4],
		}
		log.Printf("result: %+v", result)
		return nil
	} else {
		log.Printf("err: %+v", err)
		return nil
	}

	//paramInit = []float64{0.0054421079489133046,0.03117294131686459,0.001,0.4853198041450475,0.10595859238899136}
	//svi: &{A:0.012890652035712746 B:0.2946183363201443 C:0.005139194991262575 Rho:0.47910891553009877 Eta:-0.3048881608663741}
	//	opt, err := nlopt.NewNLopt(nlopt.LD_SLSQP, 5)
	//	if err != nil {
	//		return nil
	//	}
	//	defer opt.Destroy()
	//	// (a + b*(rho*(k-eta)+math.Sqrt((k-eta)*(k-eta)+c*c)))
	//	err = opt.SetMinObjective(func(x, gradient []float64) float64 {
	//		if len(gradient) > 0 {
	//			p := &SviParams{
	//				A:   x[0],
	//				B:   x[1],
	//				C:   x[2],
	//				Rho: x[3],
	//				Eta: x[4],
	//			}
	//
	//			/*gradient[0], gradient[1], gradient[2], gradient[3], gradient[4] = 0.0, 0.0, 0.0, 0.0 ,0.0
	//			bc := p.B * p.C
	//			bEta := p.B - p.Eta
	//			for i, k := range kList {
	//				km := k-p.Eta
	//				d1 := math.Sqrt(km * km + p.C * p.C)
	//				d2 := p.Rho * km + d1
	//				d3 := p.A + p.B * d2 - vList[i]
	//				gradient[0] += d3
	//				gradient[1] += d3 * d2
	//				gradient[2] += d3 * bc / d1
	//				gradient[3] += d3 * km * p.B
	//				gradient[4] += d3 * (-km / d1 -bEta)
	//			}
	//			gradient[0] *= 2
	//			gradient[1] *= 2
	//			gradient[2] *= 2
	//			gradient[3] *= 2
	//			gradient[4] *= 2*/
	//
	//			/*bc := p.B * p.C
	//			bEta := p.B - p.Eta
	//			for _, k := range kList {
	//				km := k-p.Eta
	//				d1 := math.Sqrt(km * km + p.C * p.C)
	//				d2 := p.Rho * km + d1
	//				//d3 := p.A + p.B * d2 - vList[i]
	//				gradient[0] += 1
	//				gradient[1] += d2
	//				gradient[2] += bc / d1
	//				gradient[3] += km * p.B
	//				gradient[4] += km / d1 -bEta
	//			}*/
	//
	//			// K当成0
	//			eta := 0 - p.Eta
	//			d1 := math.Sqrt(eta*eta + p.C*p.C)
	//			if d1 == 0 {
	//				panic("d1 == 0")
	//			}
	//			gradient[0] = 1
	//			gradient[1] = p.Rho*eta + d1
	//			gradient[2] = p.B * p.C / d1
	//			gradient[3] = p.B * eta
	//			gradient[4] = p.Eta/d1 - p.B*p.Eta
	//		}
	//
	//		log.Printf("x: %+v", x)
	//		log.Printf("gradient: %+v", gradient)
	//		log.Printf("LeastSquares(x, kList, vList): %+v", LeastSquares(x, kList, vList))
	//		return LeastSquares(x, kList, vList)
	//	})
	//	if err != nil {
	//		log.Printf("err: %+v", err)
	//	}

	/*_ = opt.SetLowerBounds(lowBounds)
	_ = opt.SetUpperBounds(upBounds)
	//_ = opt.SetXtolRel(1e-9)
	_ = opt.SetFtolRel(1e-9)
	_ = opt.SetMaxEval(1000)*/

	/*err = opt.AddInequalityMConstraint(func(result, x, gradient []float64) {
		result[0] = RightConstraint1(x, kList)
		result[1] = RightConstraint2(x, kList)
		result[2] = LeftConstraint1(x, kList)
		result[3] = LeftConstraint2(x, kList)
		return
	}, []float64{1e-9, 1e-9, 1e-9, 1e-9})
	if err != nil {
		log.Printf("err: %+v", err)
	}*/

	/*_ = opt.AddInequalityConstraint(RightConstraint1, 1e-9)
	_ = opt.AddInequalityConstraint(RightConstraint2, 1e-9)
	_ = opt.AddInequalityConstraint(LeftConstraint1, 1e-9)
	_ = opt.AddInequalityConstraint(LeftConstraint2, 1e-9)
	_ = opt.AddInequalityConstraint(Constraint, 1e-9)
	_ = opt.AddInequalityConstraint(Constraint2, 1e-9)

	log.Printf("opt: %+v", opt)
	param, f, err := opt.Optimize(paramInit)
	if err != nil {
		log.Printf("param: %+v, f: %+v, err: %s", param, f, err)
		return nil
	}
	s.SviParams = &SviParams{}
	s.A = param[0]
	s.B = param[1]
	s.C = param[2]
	s.Rho = param[3]
	s.Eta = param[4]

	return s.SviParams*/
}

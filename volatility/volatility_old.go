package volatility

import (
	"bytes"
	"fmt"
	/*"github.com/go-nlopt/nlopt"*/
	"gonum.org/v1/gonum/mat"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ParamsOld struct {
	A   float64 // 方差大小
	B   float64 // 渐近线夹角
	C   float64 // 平滑度
	Rho float64 // 旋转
	Eta float64 // 平移
}

func (s *ParamsOld) Copy() *ParamsOld {
	return &ParamsOld{
		A:   s.A,
		B:   s.B,
		C:   s.C,
		Rho: s.Rho,
		Eta: s.Eta,
	}
}

type MarketDataOld struct {
	StrikePrice float64
	ImVol       float64
}

type VolatilityOld struct {
	*ParamsOld
	MarketDataList []*MarketDataOld
	xMatrix        *mat.VecDense
	yMatrix        *mat.VecDense
	ForwardPrice   float64 // 远期价格
	T              float64 // （期权到期日-当前时间）/365天
	kMap           map[float64]float64
}

func NewVolatilityOld(ForwardPrice float64, T float64) *VolatilityOld {
	s := &VolatilityOld{
		ForwardPrice: ForwardPrice,
		T:            T,
	}
	return s
}

// 曲线拟合返回参数
func (s *VolatilityOld) FitVol() *ParamsOld {
	// prepare to call the Levenberg-Marquardt method
	if s.A == 0 && s.B == 0 && s.C == 0 && s.Eta == 0 {
		return s.ParamsOld
	}
	return s.LMFit(s.xMatrix, s.yMatrix, s.ParamsOld)
}

// 根据参数和行权价格找到波动率
func (s *VolatilityOld) GetImVol(strikePrice float64, p *ParamsOld) float64 {
	kM := math.Log(strikePrice/s.ForwardPrice) - p.Eta
	return math.Sqrt(math.Abs(VarianceOld(kM, p.A, p.B, p.C, p.Rho) / s.T))
}

func (s *VolatilityOld) GetVariance(strikePrice float64, p *ParamsOld) float64 {
	kM := math.Log(strikePrice/s.ForwardPrice) - p.Eta
	return VarianceOld(kM, p.A, p.B, p.C, p.Rho)
}

func VarianceOld(kM, a, b, c, rho float64) float64 {
	return a + b*(rho*kM+math.Sqrt(kM*kM+c*c))
}

func (s *VolatilityOld) InitParams(marketDataList []*MarketDataOld) {

	s.ParamsOld = &ParamsOld{}
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
	j := len(varianceArr) - 1
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
	Nu0Old           = 1000
	Res0Old          = 1e9
	MaxIterationsOld = 25
	ToleranceOld     = 1e-8
	ParamsLenOld     = 5
)

// Levenberg-Marquardt 最小二乘法
func (s *VolatilityOld) LMFit(x, y *mat.VecDense, pStart *ParamsOld) *ParamsOld {
	resZero := Res0Old
	nu := float64(Nu0Old)
	dataLen := x.Len()
	p := pStart.Copy()
	rTranspose := mat.NewDense(1, dataLen, nil)
	for i := 0; i < MaxIterationsOld; i++ {
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

		for j := 0; j < ParamsLenOld; j++ {
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

		if math.Abs(res-resZero) < ToleranceOld {
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

func (s *VolatilityOld) FVector(x *mat.VecDense, p *ParamsOld) *mat.VecDense {
	dataLen := x.Len()
	outline := mat.NewVecDense(dataLen, nil)
	for i := 0; i < x.Len(); i++ {
		outline.SetVec(i, s.F(x.AtVec(i), p))
	}
	return mat.VecDenseCopyOf(outline.TVec())
}

func (s *VolatilityOld) F(strikePrice float64, p *ParamsOld) float64 {
	k := s.kMap[strikePrice]
	kM := k - p.Eta
	return math.Sqrt(math.Abs((p.A + p.B*(p.Rho*kM+math.Sqrt(kM*kM+p.C*p.C))) / s.T))
}

func (s *VolatilityOld) GradFMatrix(x *mat.VecDense, p *ParamsOld) *mat.Dense {
	outMatrixTranspose := mat.NewDense(ParamsLenOld, x.Len(), nil)
	for i := 0; i < x.Len(); i++ {
		tmpGradFI := s.GradF(x.AtVec(i), p)
		for j := 0; j < ParamsLenOld; j++ {
			outMatrixTranspose.Set(j, i, tmpGradFI[j])
		}
	}
	return mat.DenseCopyOf(outMatrixTranspose.T())
}

func (s *VolatilityOld) GradF(strikePrice float64, p *ParamsOld) []float64 {
	res := make([]float64, ParamsLenOld)
	k := s.kMap[strikePrice]
	kM := k - p.Eta
	ff := s.F(strikePrice, p)
	tmp := 1 / (2 * ff * s.T)
	tmpB := tmp * p.B
	tmp1 := VarianceOld(kM, 0, 1, p.C, p.Rho)
	tmp2 := VarianceOld(kM, 0, 1, p.C, 0)

	res[0] = tmp
	res[1] = tmp * tmp2
	res[2] = tmpB * p.C / tmp1
	res[3] = tmpB * kM
	res[4] = -tmpB * (p.Rho + kM/tmp1)

	return res
}

// 调用python的scipy实现slsqp
// need install python3,numpy,scipy
func (s *VolatilityOld) PyMinimizeSLSQP(marketDataList []*MarketDataOld) (*ParamsOld, error) {
	kBuf, vBuf := bytes.Buffer{}, bytes.Buffer{}
	for _, marketData := range marketDataList {
		k := math.Log(marketData.StrikePrice / s.ForwardPrice)
		v := marketData.ImVol * marketData.ImVol * s.T
		kBuf.WriteString(",")
		kBuf.WriteString(strconv.FormatFloat(k, 'f', -1, 64))

		vBuf.WriteString(",")
		vBuf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	}

	command, args := "python3", []string{"minimize_slsqp.py", "-k " + kBuf.String()[1:], "-v " + vBuf.String()[1:]}
	//command, args := "ls", []string{"-a"}
	//command, args := "python3", []string{"minimize_slsqp.py"}
	cmd := exec.Command(command, args...)
	log.Printf("%+v", cmd.String())
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("exec.Command fail, err: %s, out: %s", err, out.String())
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("exec.Command fail, err: %s, out: %s", err, out.String())
	}

	res := strings.Split(strings.Trim(strings.Replace(out.String(), " ", "", -1), "\n"), ",")
	if len(res) != 5 {
		return nil, fmt.Errorf("exec.Command fail, err: %s, out: %s", err, out.String())
	}

	s.ParamsOld = &ParamsOld{}
	s.A, _ = strconv.ParseFloat(res[0], 64)
	s.B, _ = strconv.ParseFloat(res[1], 64)
	s.C, _ = strconv.ParseFloat(res[2], 64)
	s.Rho, _ = strconv.ParseFloat(res[3], 64)
	s.Eta, _ = strconv.ParseFloat(res[4], 64)
	return s.ParamsOld, nil
}

// TODO optimize with nlopt-slsqp function in cgo

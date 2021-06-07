package blackscholes

import (
	"math"
	"strings"
)

const MaxExecTimes = 100

// Black–Scholes model
// see wiki: https://en.wikipedia.org/wiki/Black%E2%80%93Scholes_model
type BSM struct {
	D         string  `json:"direction"`     // direction 期权方向 看涨：c 看跌：p
	S         float64 `json:"subject_price"` // subjectPrice期权标的价格（指数价格）
	X         float64 `json:"strike_price"`  // 期权行权价格（敲定价格）
	T         float64 `json:"rest_time"`     // （期权到期日-当前时间）/365天
	R         float64 `json:"price_rate"`    // 计价币种的利率
	Iv        float64 `json:"volatility"`    // 年化波动率
	IvMax     float64 `json:"iv_max"`        // 年化波动最大值
	IvMin     float64 `json:"iv_min"`        // 年化波动最小
	Op        float64 `json:"option_price"`  // 期权报价
	OpEpsilon float64 `json:"op_epsilon"`    // 价格精度0.0001
	D1        float64 `json:"d1"`            // 中间值d1
	Nd1       float64 `json:"nd1"`           // 中间值nd1
	D2        float64 `json:"d2"`            // 中间值d2
	Delta     float64 `json:"delta"`         // 希腊值delta, 期权价格对underlying价格的敏感度
	Gamma     float64 `json:"gamma"`         // 希腊值gamma, delta对underlying价格的敏感度
	Vega      float64 `json:"vega"`          // 希腊值vega, 期权价格对隐含波动率的敏感度
	Theta     float64 `json:"theta"`         // 希腊值theta, 期权价格对剩余期限的敏感度
	Rho       float64 `json:"rho"`           // 希腊值rho
}

func NewBS(direction string, S float64, X float64, T float64, r float64, op float64, opEpsilon float64, ivMax float64, ivMin float64) *BSM {
	bsm := BSM{
		D:         strings.ToLower(direction),
		S:         S,
		X:         X,
		T:         T,
		R:         r,
		Op:        op,
		OpEpsilon: opEpsilon,
		IvMax:     ivMax,
		IvMin:     ivMin,
	}
	bsm.init()
	return &bsm
}

func NewBSWithIv(direction string, S float64, X float64, T float64, r float64, op float64, opEpsilon float64, ivMax float64, ivMin float64, iv float64) *BSM {
	bsm := BSM{
		D:         strings.ToLower(direction),
		S:         S,
		X:         X,
		T:         T,
		R:         r,
		Op:        op,
		OpEpsilon: opEpsilon,
		Iv:        iv,
		IvMax:     ivMax,
		IvMin:     ivMin,
	}
	bsm.init()
	return &bsm
}

func (bsm *BSM) init() {
	// 计算波动率
	if bsm.Iv == 0 {
		bsm.ImVolBisection()
	}

	// 计算d1
	bsm.calcD1()
	// 计算d2
	bsm.calcD2()
	// 计算nd1
	bsm.calcNd1()
	// 计算delta
	bsm.calcDelta()
	// 计算gamma
	bsm.calcGamma()
	// 计算vega
	bsm.calcVega()
	// 计算theta
	bsm.calcTheta()
	// 计算rho
	bsm.calcRho()
}

func (bsm *BSM) ImVolBisection() {
	ivMax, ivMin := bsm.IvMax, bsm.IvMin
	opMax, opMin := 0.0, 0.0
	opEpsilon := 0.000001 // 价格精度

	if bsm.Op < opEpsilon {
		bsm.Iv = 0
		return
	}

	// 处理边界
	opMax = bsm.GetOptionPriceFromIv(ivMax)
	if bsm.Op > opMax-opEpsilon {
		bsm.Iv = ivMax
		return
	}
	opMin = bsm.GetOptionPriceFromIv(ivMin)
	if bsm.Op < opMin+opEpsilon {
		bsm.Iv = ivMin
		return
	}

	execCount := 0
	iv := (ivMax + ivMin) / 2
	op := bsm.GetOptionPriceFromIv(iv)
	for math.Abs(bsm.Op-op) > opEpsilon && execCount < MaxExecTimes {
		execCount++

		if op < bsm.Op {
			ivMin = iv
			opMin = bsm.GetOptionPriceFromIv(ivMin)
		} else {
			ivMax = iv
			opMax = bsm.GetOptionPriceFromIv(ivMax)
		}

		if execCount > 5 {
			iv = (ivMax + ivMin) / 2
		} else {
			iv = ivMin + (bsm.Op-opMin)*(ivMax-ivMin)/(opMax-opMin)
		}
		op = bsm.GetOptionPriceFromIv(iv)
	}
	bsm.Iv = iv
}

// 通过隐含波动率找到对应的期权报价
func (bsm *BSM) GetOptionPriceFromIv(iv float64) (optionPrice float64) {
	bsm.Iv = iv
	bsm.calcD1()
	bsm.calcD2()
	if bsm.D == "c" {
		optionPrice = bsm.S*Cdf(bsm.D1) - bsm.X*math.Exp(-bsm.R*bsm.T)*Cdf(bsm.D2)
	} else if bsm.D == "p" {
		optionPrice = bsm.X*math.Exp(-bsm.R*bsm.T)*Cdf(-bsm.D2) - bsm.S*Cdf(-bsm.D1)
	}
	return
}

func (bsm *BSM) calcD1() {
	bsm.D1 = (math.Log(bsm.S/bsm.X) + (bsm.R+math.Pow(bsm.Iv, 2)/2)*bsm.T) / (bsm.Iv * math.Sqrt(bsm.T))
}

func (bsm *BSM) calcD2() {
	bsm.D2 = bsm.D1 - bsm.Iv*math.Sqrt(bsm.T)
}

func (bsm *BSM) calcNd1() {
	bsm.Nd1 = 1 / math.Sqrt(2*math.Pi) * math.Exp(-(math.Pow(bsm.D1, 2) / 2))
}

func (bsm *BSM) calcDelta() {
	if bsm.D == "c" {
		bsm.Delta = Cdf(bsm.D1)
	} else if bsm.D == "p" {
		bsm.Delta = Cdf(bsm.D1) - 1
	}
}

func (bsm *BSM) calcGamma() {
	bsm.Gamma = 1 / (bsm.S * bsm.Iv * math.Sqrt(bsm.T)) * bsm.Nd1
}

func (bsm *BSM) calcVega() {
	bsm.Vega = bsm.S * math.Sqrt(bsm.T) * bsm.Nd1 / 100
}

func (bsm *BSM) calcTheta() {
	if bsm.D == "c" {
		bsm.Theta = (-bsm.S*bsm.Iv/(2*math.Sqrt(bsm.T))*bsm.Nd1 - bsm.R*bsm.X*math.Exp(-bsm.R*bsm.T)*Cdf(bsm.D2)) / 365
	} else if bsm.D == "p" {
		bsm.Theta = (-bsm.S*bsm.Iv/(2*math.Sqrt(bsm.T))*bsm.Nd1 + bsm.R*bsm.X*math.Exp(-bsm.R*bsm.T)*Cdf(-bsm.D2)) / 365
	}
}

func (bsm *BSM) calcRho() {
	if bsm.D == "c" {
		bsm.Rho = bsm.T * bsm.X * math.Exp(-bsm.R*bsm.T) * Cdf(bsm.D2) / 100
	} else if bsm.D == "p" {
		bsm.Rho = -bsm.T * bsm.X * math.Exp(-bsm.R*bsm.T) * Cdf(-bsm.D2) / 100
	}
}

/**
 * cumulative normal distribution function
 */
func Cdf(x float64) float64 {
	a := []float64{0.31938153, -0.356563782, 1.781477937, -1.821255978, 1.330274429}
	var (
		res float64
	)
	l := math.Abs(x)
	k := 1 / (1 + 0.2316419*l)
	res = 1 - 1/math.Sqrt(2*math.Pi)*math.Exp(-math.Pow(l, 2)/2)*(a[0]*k+a[1]*math.Pow(k, 2)+a[2]*math.Pow(k, 3)+a[3]*math.Pow(k, 4)+a[4]*math.Pow(k, 5))

	if x < 0 {
		res = 1 - res
	}
	return res
}

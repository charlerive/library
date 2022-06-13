package main

import (
	"fmt"
	"github.com/charlerive/library/authenticator"
	"github.com/charlerive/library/blackscholes"
	"log"
	"time"
)

func main() {
	deliveryTime, _ := time.Parse("2006-01-02 15:04:05", "2022-06-24 16:00:00")
	nowTime, _ := time.Parse("2006-01-02 15:04:05", "2022-06-13 11:44:00")
	t := float64(deliveryTime.Sub(nowTime)) / float64(time.Hour*24*365)
	bsm := blackscholes.NewBS("c", 25490.88, 25000, t, 0, 0.1, 0.01, 3, 0.01)
	fmt.Println(fmt.Sprintf("bsm: %+v", bsm))
	log.Printf("optionsPrice: %+v", bsm.GetOptionPriceFromIv(1.0303446214295107))

	return

	bsm = blackscholes.NewBS("p", 4600, 5000, 0.1644, 0.025, 996.27, 0.01, 3, 0.3)
	fmt.Println(fmt.Sprintf("bsm: %+v", bsm))
	log.Printf("optionsPrice: %+v", bsm.GetOptionPriceFromIv(1.0303446214295107))
	if authenticator.GetGoogleAuthService().Auth(552861) {
		log.Printf("auth success")
	}
	authenticator.GetGoogleAuthService().Quit()

	bsm = blackscholes.NewBS("c", 1000, 1001, 30.0/365, 0.0, 0.0, 0.01, 3, 0.01)
	log.Printf("optionsPrice: %+v", bsm.GetOptionPriceFromIv(0.25))

	bsm = blackscholes.NewBS("c", 40579, 39680, 3.5/365, 0.025, 1961, 0.01, 3, 0.01)
	log.Printf("optionsPrice: %+v, bsm: %+v", bsm.GetOptionPriceFromIv(0.56), bsm)
	bsm = blackscholes.NewBS("p", 40579, 39680, 3.5/365, 0.025, 1961, 0.01, 3, 0.01)
	log.Printf("optionsPrice: %+v, bsm: %+v", bsm.GetOptionPriceFromIv(0.56), bsm)

}

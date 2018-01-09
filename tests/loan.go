package main

import (
	"fmt"
	"math"
)

type Loan struct {
	Month     int
	Paid      float64
	Left      float64
	PrevPaid  float64
	PrevLeft  float64
	FixRate   float64
	FixMonths int
	FixPay    float64
	VarRate   float64
	VarPay    float64
}

func (x *Loan) Pay(amount float64) bool {
	x.PrevPaid = x.Paid
	x.PrevLeft = x.Left
	x.Month++
	rate := x.VarRate
	topay := x.VarPay
	x.VarRate += 0.001/12
	if x.FixMonths > 0 {
		rate = x.FixRate
		topay = x.FixPay
		x.FixMonths--
	}
	if amount < topay {
		amount = topay
	}
	if amount > x.Left {
		x.Paid += x.Left
		x.Left = 0
		return true
	}
	x.Paid += amount
	x.Left -= amount
	// pow(1+r, 1/12)*amount => exp(log(1+r)*1/12) => log1p(r)/12
	x.Left *= math.Exp(math.Log1p(rate)/12.0)
	return false
}

func (x *Loan) String() string {
	return fmt.Sprintf("%3d %4.1f %8.f %8.f %6.f %6.f", x.Month, float64(x.Month) / 12.0, x.Left, x.Paid, x.PrevLeft-x.Left, x.Paid-x.PrevPaid)
}

func main() {
	loan := &Loan{
		Paid:      0.,
		PrevPaid:  0.,
		Left:      604000.,
		PrevLeft:  604000.,
		FixRate:   0.0399,
		FixMonths: 36,
		FixPay:    2880,
		VarRate:   0.0359,
		VarPay:    2753,
	}
	fmt.Println(loan)
	for {
		paid := loan.Pay(6000.)
		if paid {
			break
		}
		fmt.Println(loan)
	}
}

package gasprice

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
)

func TestBaseFee_ExamplePriceAdjustments(t *testing.T) {
	approxExp := func(f, n, d int64) uint64 {
		return uint64(float64(f) * math.Exp(float64(n)/float64(d)))
	}
	const targetRate = 1e6

	tests := map[string]struct {
		parentBaseFee  uint64
		parentGasUsed  uint64
		parentDuration time.Duration
		targetRate     uint64
		wantBaseFee    uint64
	}{
		"base fee remains the same": {
			parentBaseFee:  1e8,
			parentGasUsed:  targetRate,
			parentDuration: 1 * time.Second,
			targetRate:     targetRate,
			wantBaseFee:    approxExp(1e8, 0, 128), // max change rate per second ~1/128
		},
		"base fee increases": {
			parentBaseFee:  1e8,
			parentGasUsed:  2 * targetRate,
			parentDuration: 1 * time.Second,
			targetRate:     targetRate,
			wantBaseFee:    approxExp(1e8, 1, 128),
		},
		"base fee decreases": {
			parentBaseFee:  1e8,
			parentGasUsed:  0,
			parentDuration: 1 * time.Second,
			targetRate:     targetRate,
			wantBaseFee:    approxExp(1e8, -1, 128),
		},
		"base fee increase is limited": {
			parentBaseFee:  1e8,
			parentGasUsed:  3 * targetRate,
			parentDuration: 1 * time.Second,
			targetRate:     targetRate,
			wantBaseFee:    approxExp(1e8, 1, 128), // same as for 2x target rate
		},
		"short time bursts have capped impact on base fee price": {
			parentBaseFee:  1e8,
			parentGasUsed:  targetRate,
			parentDuration: 1 * time.Millisecond,
			targetRate:     targetRate,
			wantBaseFee:    approxExp(1e8, 1, 128*1000), // the max increase in 1 millisecond
		},
		"long durations are ignored": {
			parentBaseFee:  123456789,
			parentGasUsed:  0, // < no gas used, should reduce the price
			parentDuration: 61 * time.Second,
			targetRate:     targetRate,
			wantBaseFee:    123456789, // < since the duration is too long, the price should not change
		},
		"target rate is zero": {
			parentBaseFee:  123456789,
			parentGasUsed:  0, // < no gas used, should reduce the price
			parentDuration: time.Second,
			targetRate:     0, // < since the target rate is zero, the price should not change
			wantBaseFee:    123456789,
		},
		"zero duration has no effect": {
			parentBaseFee:  123456789,
			parentGasUsed:  targetRate,
			parentDuration: 0,
			targetRate:     targetRate,
			wantBaseFee:    123456789,
		},
		"negative duration has no effect": {
			parentBaseFee:  123456789,
			parentGasUsed:  targetRate,
			parentDuration: -time.Second,
			targetRate:     targetRate,
			wantBaseFee:    123456789,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			header := ParentBlockInfo{
				BaseFee:  big.NewInt(int64(test.parentBaseFee)),
				GasUsed:  test.parentGasUsed,
				Duration: test.parentDuration,
			}

			rules := opera.EconomyRules{
				ShortGasPower: opera.GasPowerRules{
					AllocPerSec: 2 * test.targetRate,
				},
			}

			gotBaseFee := GetBaseFeeForNextBlock(header, rules)
			wantBaseFee := big.NewInt(int64(test.wantBaseFee))
			if gotBaseFee.Cmp(wantBaseFee) != 0 {
				t.Fatalf("base fee is incorrect; got %v, want %v, diff %d",
					gotBaseFee, wantBaseFee, sub(gotBaseFee, wantBaseFee),
				)
			}

			if header.BaseFee == gotBaseFee {
				t.Fatalf("new base fee is not a copy; got %p, want %p",
					header.BaseFee, gotBaseFee,
				)
			}
		})
	}
}

func TestBaseFee_PriceCanRecoverFromPriceZero(t *testing.T) {
	target := uint64(1e6)
	header := ParentBlockInfo{
		BaseFee:  big.NewInt(0),
		GasUsed:  target + 1,
		Duration: time.Second,
	}

	rules := opera.EconomyRules{
		ShortGasPower: opera.GasPowerRules{
			AllocPerSec: 2 * target,
		},
	}

	newPrice := GetBaseFeeForNextBlock(header, rules)
	if newPrice.Cmp(big.NewInt(1)) < 0 {
		t.Errorf("failed to increase price from zero, new price %v", newPrice)
	}
}

func TestBaseFee_GrowsAtMostTwelveAndAHalfPercentPer15Seconds(t *testing.T) {
	const targetRate = 1e6
	rules := opera.EconomyRules{
		ShortGasPower: opera.GasPowerRules{
			AllocPerSec: targetRate * 2,
		},
	}

	// This property should be true for any block time.
	blockTimes := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
	}

	for _, blockTime := range blockTimes {
		t.Run(fmt.Sprintf("blockTime=%s", blockTime.String()), func(t *testing.T) {
			const initialPrice = 100_000_000
			// Define a header using 2x the target rate of gas in the given block time.
			header := ParentBlockInfo{
				BaseFee:  big.NewInt(initialPrice),
				GasUsed:  uint64((2 * targetRate * blockTime) / time.Second),
				Duration: blockTime,
			}
			duration := time.Duration(0)
			for duration < 15*time.Second {
				header.BaseFee = GetBaseFeeForNextBlock(header, rules)
				duration += blockTime
			}

			want := uint64(initialPrice + initialPrice/8) // < 12.5% growth
			got := header.BaseFee.Uint64()
			diff := got - want
			if got < want {
				diff = want - got
			}
			if float64(diff)/float64(100_000_000) > 0.015 {
				t.Errorf(
					"price growth is incorrect; wanted %d, got %d, diff %d",
					want, got, diff,
				)
			}
		})
	}
}

func TestBaseFee_ShrinksAtMostTwelveAndAHalfPercentPer15Seconds(t *testing.T) {
	const targetRate = 1e6
	rules := opera.EconomyRules{
		ShortGasPower: opera.GasPowerRules{
			AllocPerSec: targetRate * 2,
		},
	}

	// This property should be true for any block time.
	blockTimes := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
	}

	for _, blockTime := range blockTimes {
		t.Run(fmt.Sprintf("blockTime=%s", blockTime.String()), func(t *testing.T) {
			const initialPrice = 100_000_000
			// Define a header using no gas at all.
			header := ParentBlockInfo{
				BaseFee:  big.NewInt(initialPrice),
				GasUsed:  0,
				Duration: blockTime,
			}
			duration := time.Duration(0)
			for duration < 15*time.Second {
				header.BaseFee = GetBaseFeeForNextBlock(header, rules)
				duration += blockTime
			}

			want := uint64(initialPrice - initialPrice/8) // < 12.5% reduction
			got := header.BaseFee.Uint64()
			diff := got - want
			if got < want {
				diff = want - got
			}
			if float64(diff)/float64(initialPrice) > 0.015 {
				t.Errorf(
					"price reduction is incorrect; wanted %d, got %d, diff %d",
					want, got, diff,
				)
			}
		})
	}
}

func TestBaseFee_DecayTimeFromInitialToZeroIsApproximately40Minutes(t *testing.T) {
	rules := opera.EconomyRules{
		ShortGasPower: opera.GasPowerRules{
			AllocPerSec: 1e6,
		},
	}

	// This property should be true for any block time.
	blockTimes := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
	}
	for _, blockTime := range blockTimes {
		t.Run(fmt.Sprintf("blockTime=%s", blockTime.String()), func(t *testing.T) {
			header := ParentBlockInfo{
				BaseFee:  GetInitialBaseFee(opera.EconomyRules{}),
				GasUsed:  0,
				Duration: blockTime,
			}
			decayDuration := time.Duration(0)
			for header.BaseFee.Sign() > 0 {
				header.BaseFee = GetBaseFeeForNextBlock(header, rules)
				decayDuration += blockTime
			}

			if decayDuration < 35*time.Minute || decayDuration > 45*time.Minute {
				t.Errorf(
					"time to decay from initial to zero is incorrect; got %v",
					decayDuration,
				)
			}
		})
	}
}

func TestBaseFee_InitialPriceIsAtLeastMinBaseFee(t *testing.T) {
	for _, minPrice := range []int64{-1, 0, 1, 1e8, 1e9, 1e10, 1e12} {
		rules := opera.EconomyRules{MinBaseFee: big.NewInt(int64(minPrice))}
		got := GetInitialBaseFee(rules).Int64()
		if got < minPrice {
			t.Errorf("initial price is below min gas price; got %d, min %d", got, minPrice)
		}
	}
}

func TestBaseFee_DoesNotSinkBelowMinBaseFee(t *testing.T) {
	for _, minPrice := range []int64{-1, 0, 1, 1e8, 1e9, 1e10} {
		t.Run(fmt.Sprintf("minPrice=%d", minPrice), func(t *testing.T) {
			minPrice := big.NewInt(minPrice)
			rules := opera.EconomyRules{
				MinBaseFee: minPrice,
				ShortGasPower: opera.GasPowerRules{
					AllocPerSec: 1e6,
				},
			}

			t.Run("price does not sink below minimum", func(t *testing.T) {
				header := ParentBlockInfo{
					BaseFee:  minPrice,
					GasUsed:  0, // < should reduce the price
					Duration: time.Second,
				}
				newPrice := GetBaseFeeForNextBlock(header, rules)
				if newPrice.Cmp(minPrice) < 0 {
					t.Errorf("price sank below minimum; got %v, min %v", newPrice, minPrice)
				}
			})

			t.Run("at threshold price is capped at minimum", func(t *testing.T) {
				header := ParentBlockInfo{
					BaseFee:  new(big.Int).Add(minPrice, big.NewInt(1)),
					GasUsed:  0, // < should reduce the price
					Duration: time.Second,
				}
				newPrice := GetBaseFeeForNextBlock(header, rules)
				if newPrice.Cmp(minPrice) < 0 {
					t.Errorf("price sank below minimum; got %v, min %v", newPrice, minPrice)
				}
			})
		})
	}
}

func TestApproximateExponential_KnownValues(t *testing.T) {
	tests := map[string]struct {
		factor      int64
		numerator   int64
		denominator int64
		want        int64
	}{
		"e^0": {
			factor:      1,
			numerator:   0,
			denominator: 1,
			want:        1,
		},
		"e^1": {
			factor:      1,
			numerator:   1,
			denominator: 1,
			want:        2,
		},
		"e^2": {
			factor:      1,
			numerator:   2,
			denominator: 1,
			want:        6, // < should be 7, but the function just approximates
		},
		"e^(1/2)": {
			factor:      1,
			numerator:   1,
			denominator: 2,
			want:        1,
		},
		"e^-1": {
			factor:      1,
			numerator:   -1,
			denominator: 1,
			want:        0,
		},
		"10*e^2": {
			factor:      10,
			numerator:   2,
			denominator: 1,
			want:        71, // < should be 73, but the function just approximates
		},
		"100*e^(1/2)": {
			factor:      100,
			numerator:   1,
			denominator: 2,
			want:        164,
		},
		"100*e^(-1/2)": {
			factor:      100,
			numerator:   -1,
			denominator: 2,
			want:        60,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotBaseFee := approximateExponential(
				big.NewInt(int64(test.factor)),
				big.NewInt(test.numerator),
				big.NewInt(test.denominator),
			)
			wantBaseFee := big.NewInt(int64(test.want))
			if gotBaseFee.Cmp(wantBaseFee) != 0 {
				t.Fatalf("base fee is incorrect; got %v, want %v", gotBaseFee, wantBaseFee)
			}
		})
	}
}

func TestApproximateExponential_RandomInputs(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	for range 100 {
		factor := int64(r.Int31n(100))

		// Our practical use cases for gas computations are fractions in the range [-1,1]
		denominator := int64(r.Int31n(1e9) + 1)
		numerator := int64((r.Float64()*2 - 1) * float64(denominator))

		want := big.NewInt(int64(float64(factor) * math.Exp(float64(numerator)/float64(denominator))))
		got := approximateExponential(big.NewInt(factor), big.NewInt(numerator), big.NewInt(denominator))

		diff := new(big.Int).Abs(sub(got, want))
		if diff.Cmp(big.NewInt(1)) > 0 {
			t.Errorf(
				"incorrect approximation for f=%d, n=%d, d=%d; got %v, want %v, error %v",
				factor, numerator, denominator, got, want, diff,
			)
		}
	}
}

func BenchmarkBaseFeeComputation(b *testing.B) {
	header := ParentBlockInfo{
		BaseFee:  big.NewInt(1e9),
		GasUsed:  1e6,
		Duration: time.Second,
	}
	rules := opera.EconomyRules{
		ShortGasPower: opera.GasPowerRules{
			AllocPerSec: 1e6,
		},
	}
	for range b.N {
		GetBaseFeeForNextBlock(header, rules)
	}
}

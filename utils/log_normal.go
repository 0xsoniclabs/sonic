package utils

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

// LogNormalDistribution produces samples from a log-normal distribution.
//
// Background:
//   - A log-normal distribution is the distribution of e^X where
//     X ~ Normal(μ, σ). This means: take a random variable X from a
//     normal distribution with mean = μ and standard deviation = σ,
//     then exponentiate it. The result is always positive.
//   - This means sampled values are always positive and skewed to the right.
//   - It's useful for modeling real-world latencies and delays, which cannot
//     be negative and often have occasional "long-tail" high values.
//   - "Long-tail": Unlike a symmetric bell curve, a log-normal distribution
//     has a heavy right tail. Most samples cluster around smaller values, but
//     occasionally you get very large values. This models the fact that
//     network or processing delays are usually short, but sometimes
//     unexpectedly long.
//
// Parameters:
//   - μ (mu): the mean of the underlying normal distribution (before
//     exponentiation).
//   - σ (sigma): the standard deviation of the underlying normal distribution.
//   - Important: μ and σ do not directly correspond to the mean and std
//     dev of the samples of the distribution themselves. Instead, they
//     influence them indirectly. Larger μ shifts the whole distribution to
//     the right (increasing the typical delay), while larger σ increases
//     the spread and the likelihood of very large "long-tail" samples
//     (delays).
//   - Median: The median of the distribution is exactly exp(μ).
//   - Mean (E[X]): The mean is exp(μ + σ²/2). Note that the mean is
//     always greater than the median.
//   - Variance (Var[X]): The variance is (exp(σ²) - 1) * exp(2μ + σ²).
//     A larger σ drastically increases the variance and the "long-tail"
//     effect.
//
// Units:
//   - The values sampled from the distribution are raw float64 numbers.
//   - They must be scaled to a concrete time unit (e.g., milliseconds).
//
// Example:
//
//	m := NewLogNormalDistribution(3.0, 0.5, nil)
//	d := m.SampleDuration(time.Millisecond) // returns ~exp(N(3,0.5^2)) ms
//
// Note on "~exp(N(3,0.5^2))":
//   - This shorthand means "a sample drawn from the exponential of a normal
//     distribution with mean = 3 and variance = 0.25 [σ^2 = 0.5^2]".
//   - Mathematically: X = exp(μ + σ * Z), where Z ~ Normal(0,1).
//   - So with μ = 3.0 and σ = 0.5, X = exp(3 + 0.5*Z).
//   - For example, if Z = 0.2, then X ≈ exp(3.1) ≈ 22.2.
//   - If unit = time.Millisecond, then SampleDuration returns ≈ 22.2 ms.
type LogNormalDistribution struct {
	dist distuv.LogNormal
}

// NewLogNormalDistribution creates a new log-normal delay model. If seed is
// nil, it uses the current timestamp as the random source.
func NewLogNormalDistribution(
	mu, sigma float64,
	seed *int64,
) *LogNormalDistribution {
	var src rand.Source
	if seed != nil {
		src = rand.NewSource(*seed)
	} else {
		src = rand.NewSource(time.Now().UnixNano())
	}

	return &LogNormalDistribution{
		dist: distuv.LogNormal{
			Mu:    mu,
			Sigma: sigma,
			Src:   rand.New(src),
		},
	}
}

// NewFromMedianAndPercentile creates a new log-normal distribution by
// specifying its median (P50) and one other upper-tail percentile (e.g., P95).
//
// This is often more intuitive than specifying μ (mu) and σ (sigma) directly,
// as it allows you to define the distribution based on its observable behavior
// (e.g., "the median delay is 50ms, and P95 is 200ms").
//
// --- Derivation of Formulas ---
//
// Our goal is to find the μ (mu) and σ (sigma) parameters for the
// underlying normal distribution, given two percentile points:
//  1. The median (m), which is the 50th percentile (p=0.50).
//  2. A target percentile (x_p) at a given percentile p (e.g., p=0.95).
//
// We use two key relationships:
//
//  1. The Log-Normal to Normal Relationship:
//
//     A log-normal distribution is defined by its construction from a
//     normal distribution. A variable X is log-normal if it is the
//     exponential of a normally distributed variable Y.
//
//     - Start with a normal variable: Y ~ Normal(μ, σ)
//     - Create the log-normal variable: X = exp(Y)
//
//     To find the underlying normal variable Y from our log-normal
//     variable X, we just invert this definition by taking the
//     natural logarithm (ln) of both sides:
//
//     ln(X) = ln(exp(Y))
//     ln(X) = Y
//
//     This gives us our first key insight: any percentile of X (let's
//     call it x_p) can be mapped to the corresponding percentile of Y (y_p)
//     by taking its natural log:
//
//     y_p = ln(x_p)
//
//  2. The Normal to "Standard" Normal (Z-Score) Relationship:
//
//     Any percentile y_p from our specific normal distribution Y ~ Normal(μ, σ)
//     can be related back to the standard normal distribution Z ~ Normal(0, 1).
//     The p-th percentile of Z is called the Z-score, z_p.
//     (This z_p value is what distuv.UnitNormal.Quantile(p) gives us).
//
//     The formula to convert from the Z-score (z_p) to our percentile (y_p) is:
//
//     y_p = μ + (σ * z_p)
//
// By combining (1) and (2), we get our main equation:
//
//	ln(x_p) = μ + (σ * z_p)
//
// Step 1: Solve for μ (mu) using the Median
//
//   - We use our master equation with the median point,
//   - p = 0.50 (the 50th percentile)
//   - x_p = m (the median value)
//   - The Z-score for the 50th percentile (z_0.5) is 0. This is because
//     the 50th percentile of a standard normal(0,1) is its mean, which is 0.
//   - Now, plug these values into the main equation:
//     ln(m) = μ + (σ * 0)
//     ln(m) = μ
//   - Result: μ = ln(median)
//
// Step 2: Solve for σ (sigma) using the Target Percentile
//
//   - Now we use the main equation again, this time with our second
//     point (p, x_p):
//     ln(x_p) = μ + (σ * z_p)
//   - Substitute the value of μ we just found in Step 1:
//     ln(x_p) = ln(m) + (σ * z_p)
//   - Now, we just solve for σ:
//     ln(x_p) - ln(m) = σ * z_p
//     (ln(x_p) - ln(m)) / z_p = σ
//   - Result: σ = (ln(x_p) - ln(m)) / z_p = ln(x_p / m) / z_p)
//
// Parameters:
//   - median: The desired 50th percentile (e.g., 50ms). Must be > 0.
//   - p: The percentile to specify (e.g., 0.95 for P95). Must be > 0.5, < 1.0.
//   - pTarget: The target duration for that percentile (e.g., 200ms for P95).
//     Must be > median.
//   - timeUnit: The base unit for the distribution (e.g., time.Millisecond).
//   - seed: Optional random seed.
//
// Returns:
//   - A configured *LogNormalDistribution.
//   - An error if parameters are invalid.
func NewFromMedianAndPercentile(
	median float64,
	p float64,
	pTarget float64,
	seed *int64,
) (*LogNormalDistribution, error) {
	if median <= 0 {
		return nil, errors.New("median must be positive")
	}
	if pTarget <= 0 {
		return nil, errors.New("pTarget must be positive")
	}
	if p <= 0.5 || p >= 1.0 {
		return nil, fmt.Errorf("percentile p must be in the (0.5, 1.0) range, but got %f", p)
	}
	if pTarget <= median {
		return nil, fmt.Errorf(
			"pTarget (%f) must be greater than the median (%f) for p > 0.5",
			pTarget, median,
		)
	}

	// m is the median
	m := median
	// x_p is the target value for the percentile p
	x_p := pTarget

	// μ = ln(median)
	mu := math.Log(m)

	// Get the Z-score for the percentile p.
	// This is the Inverse CDF (or Quantile function) of the standard normal
	// distribution N(0,1).
	z_p := distuv.UnitNormal.Quantile(p)

	// σ = (ln(x_p) / ln(m)) / z_p
	sigma := math.Log(x_p/m) / z_p

	return NewLogNormalDistribution(mu, sigma, seed), nil
}

// Sample returns a positive random float64 drawn from the log-normal
// distribution.
func (l *LogNormalDistribution) Sample() float64 {
	return l.dist.Rand()
}

package utils

import "math/big"

// BigMin returns the smallest of the provided big.Ints.
// None of the arguments must be nil. If no arguments
// are provided, nil is returned.
func BigMin(values ...*big.Int) *big.Int {
	if len(values) == 0 {
		return nil
	}
	res := values[0]
	for _, b := range values[1:] {
		if res.Cmp(b) > 0 {
			res = b
		}
	}
	return res
}

// BigMax returns the largest of the provided big.Ints.
// None of the arguments must be nil. If no arguments
// are provided, nil is returned.
func BigMax(values ...*big.Int) *big.Int {
	if len(values) == 0 {
		return nil
	}
	res := values[0]
	for _, b := range values[1:] {
		if res.Cmp(b) < 0 {
			res = b
		}
	}
	return res
}

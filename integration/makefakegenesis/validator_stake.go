package makefakegenesis

// CreateEqualValidatorStake creates a slice of validator stakes where each
// validator has the same stake amount.
func CreateEqualValidatorStake(numValidators int) []uint64 {
	res := make([]uint64, numValidators)
	for i := range numValidators {
		res[i] = 5_000_000
	}
	return res
}

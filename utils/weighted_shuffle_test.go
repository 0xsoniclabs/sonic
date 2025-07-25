// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"crypto/sha256"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/common/littleendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/assert"
)

func getTestWeightsIncreasing(num int) []pos.Weight {
	weights := make([]pos.Weight, num)
	for i := 0; i < num; i++ {
		weights[i] = pos.Weight(i+1) * 1000
	}
	return weights
}

func getTestWeightsEqual(num int) []pos.Weight {
	weights := make([]pos.Weight, num)
	for i := 0; i < num; i++ {
		weights[i] = 1000
	}
	return weights
}

// Test average distribution of the shuffle
func Test_Permutation_distribution(t *testing.T) {
	weightsArr := getTestWeightsIncreasing(30)

	weightHits := make(map[int]int) // weight -> number of occurrences
	for roundSeed := 0; roundSeed < 3000; roundSeed++ {
		seed := hashOf(hash.Hash{}, uint32(roundSeed))
		perm := WeightedPermutation(len(weightsArr)/10, weightsArr, seed)
		for _, p := range perm {
			weight := weightsArr[p]
			weightFactor := int(weight / 1000)

			_, ok := weightHits[weightFactor]
			if !ok {
				weightHits[weightFactor] = 0
			}
			weightHits[weightFactor]++
		}
	}

	assertar := assert.New(t)
	for weightFactor, hits := range weightHits {
		//fmt.Printf("Test_RandomElection_distribution: %d \n", hits/weightFactor)
		assertar.Equal((hits/weightFactor) > 20-8, true)
		assertar.Equal((hits/weightFactor) < 20+8, true)
		if t.Failed() {
			return
		}
	}
}

// test that WeightedPermutation provides a correct permaition
func testCorrectPermutation(t *testing.T, weightsArr []pos.Weight) {
	assertar := assert.New(t)

	perm := WeightedPermutation(len(weightsArr), weightsArr, hash.Hash{})
	assertar.Equal(len(weightsArr), len(perm))

	met := make(map[int]bool)
	for _, p := range perm {
		assertar.True(p >= 0)
		assertar.True(p < len(weightsArr))
		assertar.False(met[p])
		met[p] = true
	}
}

func Test_Permutation_correctness(t *testing.T) {
	testCorrectPermutation(t, getTestWeightsIncreasing(1))
	testCorrectPermutation(t, getTestWeightsIncreasing(30))
	testCorrectPermutation(t, getTestWeightsEqual(1000))
}

func hashOf(a hash.Hash, b uint32) hash.Hash {
	hasher := sha256.New()
	hasher.Write(a.Bytes())
	hasher.Write(littleendian.Uint32ToBytes(uint32(b)))
	return hash.FromBytes(hasher.Sum(nil))
}

func Test_Permutation_determinism(t *testing.T) {
	weightsArr := getTestWeightsIncreasing(5)

	assertar := assert.New(t)

	assertar.Equal([]int{4, 0, 1, 2, 3}, WeightedPermutation(len(weightsArr), weightsArr, hashOf(hash.Hash{}, 0)))
	assertar.Equal([]int{2, 4, 3, 1, 0}, WeightedPermutation(len(weightsArr), weightsArr, hashOf(hash.Hash{}, 1)))
	assertar.Equal([]int{4, 2, 3, 1, 0}, WeightedPermutation(len(weightsArr), weightsArr, hashOf(hash.Hash{}, 2)))
	assertar.Equal([]int{0, 2, 1, 3, 4}, WeightedPermutation(len(weightsArr), weightsArr, hashOf(hash.Hash{}, 3)))
	assertar.Equal([]int{1, 2}, WeightedPermutation(len(weightsArr)/2, weightsArr, hashOf(hash.Hash{}, 4)))
}

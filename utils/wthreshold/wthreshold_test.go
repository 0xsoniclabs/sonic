package wthreshold

import (
	"testing"

	"github.com/0xsoniclabs/consensus/inter/pos"
)

type testWeightedValue struct {
	weight pos.Weight
	value  string
}

func (twv testWeightedValue) Weight() pos.Weight {
	return twv.weight
}

func TestFindWeightedThresholdValue(t *testing.T) {
	tests := []struct {
		name      string
		values    []WeightedValue
		threshold pos.Weight
		want      string
		panic     bool
	}{
		{
			name: "simple case",
			values: []WeightedValue{
				testWeightedValue{weight: 1, value: "test1"},
				testWeightedValue{weight: 2, value: "test2"},
				testWeightedValue{weight: 3, value: "test3"},
			},
			threshold: 2,
			want:      "test2",
			panic:     false,
		},
		{
			name: "first element",
			values: []WeightedValue{
				testWeightedValue{weight: 5, value: "test4"},
				testWeightedValue{weight: 2, value: "test5"},
				testWeightedValue{weight: 3, value: "test6"},
			},
			threshold: 3,
			want:      "test4",
			panic:     false,
		},
		{
			name: "last element",
			values: []WeightedValue{
				testWeightedValue{weight: 1, value: "test7"},
				testWeightedValue{weight: 2, value: "test8"},
				testWeightedValue{weight: 3, value: "test9"},
			},
			threshold: 6,
			want:      "test9",
			panic:     false,
		},
		{
			name: "exact threshold",
			values: []WeightedValue{
				testWeightedValue{weight: 1, value: "test10"},
				testWeightedValue{weight: 2, value: "test11"},
				testWeightedValue{weight: 3, value: "test12"},
			},
			threshold: 3,
			want:      "test11",
			panic:     false,
		},
		{
			name: "panic case",
			values: []WeightedValue{
				testWeightedValue{weight: 1, value: "test13"},
				testWeightedValue{weight: 2, value: "test14"},
			},
			threshold: 4,
			panic:     true,
		},
		{
			name:      "empty values",
			values:    []WeightedValue{},
			threshold: 1,
			panic:     true,
		},
		{
			name: "zero weight",
			values: []WeightedValue{
				testWeightedValue{weight: 0, value: "test15"},
				testWeightedValue{weight: 2, value: "test16"},
			},
			threshold: 1,
			want:      "test16",
			panic:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("FindThresholdValue() did not panic as expected")
					}
				}()
				FindThresholdValue(tt.values, tt.threshold)
			} else {
				result := FindThresholdValue(tt.values, tt.threshold).(testWeightedValue).value
				if result != tt.want {
					t.Errorf("FindThresholdValue() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

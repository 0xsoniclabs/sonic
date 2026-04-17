// Copyright 2026 Sonic Operations Ltd
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

package sonicapi

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_NewRPCExecutionPlanComposable_FromBundleExecutionPlan(t *testing.T) {

	ref1 := bundle.TxReference{
		From: common.Address{1},
		Hash: common.Hash{2},
	}

	ref2 := bundle.TxReference{
		From: common.Address{3},
		Hash: common.Hash{4},
	}

	step1 := bundle.NewTxStep(ref1)
	step2 := bundle.NewTxStep(ref2)

	tests := map[string]struct {
		plan         bundle.ExecutionPlan
		expectedJson string
	}{
		"plan with single step": {
			plan: bundle.ExecutionPlan{Root: step1},
			expectedJson: `{
    			"steps":[
					{
						"from":"0x0100000000000000000000000000000000000000",
						"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
					}
				]
			}`,
		},
		"plan with different single step": {
			plan: bundle.ExecutionPlan{Root: step2},
			expectedJson: `{
		 		"steps":[
		 			{
		 				"from":"0x0300000000000000000000000000000000000000",
		 				"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
		 			}
		 		]
		 	}`,
		},
		"plan with single step and execution flags 1": {
			plan: bundle.ExecutionPlan{Root: step1.WithFlags(bundle.EF_TolerateFailed)},
			expectedJson: `{
		 		"steps":[
		 			{
		 				"tolerateFailed":true,
		 				"from":"0x0100000000000000000000000000000000000000",
		 				"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
		 			}
		 		]
		 	}`,
		},
		"plan with single step and execution flags 2": {
			plan: bundle.ExecutionPlan{Root: step1.WithFlags(bundle.EF_TolerateInvalid)},
			expectedJson: `{
		 		"steps":[
		 			{
		 				"tolerateInvalid":true,
		 				"from":"0x0100000000000000000000000000000000000000",
		 				"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
		 			}
		 		]
		 	}`,
		},
		"plan with single step and execution flags 3": {
			plan: bundle.ExecutionPlan{Root: step2.WithFlags(bundle.EF_TolerateFailed | bundle.EF_TolerateInvalid)},
			expectedJson: `{
		 		"steps":[
		 			{
		 				"tolerateFailed":true,
		 				"tolerateInvalid":true,
		 				"from":"0x0300000000000000000000000000000000000000",
		 				"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
		 			}
		 		]
		 	}`,
		},
		"plan with all-of group": {
			plan: bundle.ExecutionPlan{Root: bundle.NewAllOfStep(step1, step2)},
			expectedJson: `{
		 		"steps": [
					{
						"steps":[
							{
								"from":"0x0100000000000000000000000000000000000000",
								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
							},
							{
								"from":"0x0300000000000000000000000000000000000000",
								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
							}
						]
					}
				]
		 	}`,
		},
		"plan with different all-of group": {
			plan: bundle.ExecutionPlan{Root: bundle.NewAllOfStep(step2, step1)},
			expectedJson: `{
		  		"steps":[
					{
		  				"steps":[
		  					{
								"from":"0x0300000000000000000000000000000000000000",
								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
		  					},
		  					{
								"from":"0x0100000000000000000000000000000000000000",
								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
		  					}
		  				]
		  			}
		  		]
		  	}`,
		},
		"plan with all-of group tolerating failed": {
			plan: bundle.ExecutionPlan{Root: bundle.NewAllOfStep(step1, step2).WithFlags(bundle.EF_TolerateFailed)},
			expectedJson: `{
				"steps":[
					{
						"tolerateFailures":true,
						"steps":[
							{
								"from":"0x0100000000000000000000000000000000000000",
								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
							},
							{
								"from":"0x0300000000000000000000000000000000000000",
								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
							}
						]
					}
				]
			}`,
		},
		"plan with one-of group": {
			plan: bundle.ExecutionPlan{Root: bundle.NewOneOfStep(step1, step2)},
			expectedJson: `{
				"steps":[
					{
						"oneOf":true,
						"steps":[
							{
								"from":"0x0100000000000000000000000000000000000000",
								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
							},
							{
								"from":"0x0300000000000000000000000000000000000000",
								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
							}
						]
					}
				]
			}`,
		},
		"plan with different one-of group": {
			plan: bundle.ExecutionPlan{Root: bundle.NewOneOfStep(step2, step1)},
			expectedJson: `{
				"steps":[
					{
						"oneOf":true,
						"steps":[
							{
								"from":"0x0300000000000000000000000000000000000000",
								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
							},
							{
								"from":"0x0100000000000000000000000000000000000000",
								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
							}
						]
					}
				]
			}`,
		},
		"plan with one-of group and tolerating failed": {
			plan: bundle.ExecutionPlan{Root: bundle.NewOneOfStep(step1, step2).WithFlags(bundle.EF_TolerateFailed)},
			expectedJson: `{
				"steps":[
					{
						"tolerateFailures":true,
						"oneOf":true,
						"steps":[
							{
								"from":"0x0100000000000000000000000000000000000000",
								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
							},
							{
								"from":"0x0300000000000000000000000000000000000000",
								"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
							}
						]
					}
				]
			}`,
		},
		"plan with nested groups": {
			plan: bundle.ExecutionPlan{Root: bundle.NewOneOfStep(
				bundle.NewAllOfStep(step1, step2),
				bundle.NewAllOfStep(step2, step1),
			)},
			expectedJson: `{
				"steps":[
					{
						"oneOf":true,
						"steps":[
							{
								"steps":[
									{
										"from":"0x0100000000000000000000000000000000000000",
										"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
									},
									{
										"from":"0x0300000000000000000000000000000000000000",
										"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
									}
								]
							},
							{
								"steps":[
									{
										"from":"0x0300000000000000000000000000000000000000",
										"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
									},
									{
										"from":"0x0100000000000000000000000000000000000000",
										"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
									}
								]
							}
						]
					}
				]
			}`,
		},
		"plan with different nested groups": {
			plan: bundle.ExecutionPlan{Root: bundle.NewOneOfStep(
				bundle.NewAllOfStep(step2, step1),
				bundle.NewAllOfStep(step1, step2),
			)},
			expectedJson: `{
				"steps":[
					{
						"oneOf":true,
						"steps":[
							{
								"steps":[
									{
										"from":"0x0300000000000000000000000000000000000000",
										"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
									},
									{
										"from":"0x0100000000000000000000000000000000000000",
										"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
									}
								]
							},
							{
								"steps":[
									{
										"from":"0x0100000000000000000000000000000000000000",
										"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
									},
									{
										"from":"0x0300000000000000000000000000000000000000",
										"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
									}
								]
							}
						]
					}
				]
			}`,
		},
		"plan with block range": {
			plan: bundle.ExecutionPlan{Root: step1, Range: bundle.BlockRange{Earliest: 10, Latest: 20}},
			expectedJson: `{
				"blockRange":{"earliest":"0xa","latest":"0x14"},
				"steps":[
					{
						"from":"0x0100000000000000000000000000000000000000",
						"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
					}
				]
			}`,
		},
		"plan with different start": {
			plan: bundle.ExecutionPlan{Root: step1, Range: bundle.BlockRange{Earliest: 11, Latest: 20}},
			expectedJson: `{
				"blockRange":{"earliest":"0xb","latest":"0x14"},
				"steps":[
					{
						"from":"0x0100000000000000000000000000000000000000",
						"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
					}
				]
			}`,
		},
		"plan with different end": {
			plan: bundle.ExecutionPlan{Root: step1, Range: bundle.BlockRange{Earliest: 10, Latest: 21}},
			expectedJson: `{
				"blockRange":{"earliest":"0xa","latest":"0x15"},
				"steps":[
					{
						"from":"0x0100000000000000000000000000000000000000",
						"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
					}
				]
			}`,
		},
		"mixed": {
			plan: bundle.ExecutionPlan{
				Root: bundle.NewOneOfStep(
					bundle.NewTxStep(ref1).WithFlags(bundle.EF_TolerateFailed),
					bundle.NewAllOfStep(
						bundle.NewTxStep(ref1),
						bundle.NewTxStep(ref2).WithFlags(bundle.EF_TolerateInvalid),
					),
				),
				Range: bundle.BlockRange{Earliest: 12345678, Latest: 12345778},
			},
			expectedJson: `{
				"blockRange":{"earliest":"0xbc614e","latest":"0xbc61b2"},
				"steps":[
					{
						"oneOf":true,
						"steps":[
							{
								"tolerateFailed":true,
								"from":"0x0100000000000000000000000000000000000000",
								"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
							},
							{
								"steps":[
									{
										"from":"0x0100000000000000000000000000000000000000",
										"hash":"0x0200000000000000000000000000000000000000000000000000000000000000"
									},
									{
										"tolerateInvalid":true,
										"from":"0x0300000000000000000000000000000000000000",
										"hash":"0x0400000000000000000000000000000000000000000000000000000000000000"
									}
								]
							}
						]
					}
				]
			}`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rpcPlan, err := NewRPCExecutionPlanComposable(tc.plan)
			require.NoError(t, err)

			expectJsonEqual(t, tc.expectedJson, rpcPlan)

			recreated, err := toBundleExecutionPlan(rpcPlan)
			if err != nil {
				t.Fatalf("failed to convert back to bundle.ExecutionPlan: %v", err)
			}
			require.Equal(t, recreated, tc.plan)
		})
	}
}

func Test_toJsonExecutionPlanVisitor_CanReturnErrors(t *testing.T) {

	visitor := &toJsonExecutionPlanVisitor{
		toLeaf: func(flags bundle.ExecutionFlags, txRef bundle.TxReference) (any, error) {
			return nil, fmt.Errorf("test error")
		},
	}

	err := visitor.Step(0, bundle.TxReference{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "test error")
}

package sonicapi

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetBundleInfo_UnknownBundle_ReturnsNonErrorEmptyAnswer(t *testing.T) {

	ctr := gomock.NewController(t)
	be := NewMockbackend(ctr)
	api := NewPublicBundleAPI(be)

	hash := common.Hash{123}
	be.EXPECT().GetBundleExecutionInfo(hash)
	res, err := api.GetBundleInfo(t.Context(), hash)
	require.NoError(t, err)
	require.Nil(t, res)
}

func Test_GetBundleInfo_KnownBundle_ReturnsInfo(t *testing.T) {

	ctr := gomock.NewController(t)
	be := NewMockbackend(ctr)
	api := NewPublicBundleAPI(be)

	hash := common.Hash{123}
	be.EXPECT().GetBundleExecutionInfo(hash).Return(&bundle.ExecutionInfo{
		BlockNumber: 123,
		Position: bundle.PositionInBlock{
			Offset: 1,
			Count:  2,
		},
	})
	res, err := api.GetBundleInfo(t.Context(), hash)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, 123, res.Block.Int64())
	require.EqualValues(t, 1, uint64(*res.Position))
	require.EqualValues(t, 2, uint64(*res.Count))
}

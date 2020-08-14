package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/harmony-one/harmony/hmy"
	"github.com/harmony-one/harmony/rosetta/common"
)

// NetworkAPIService implements the server.NetworkAPIServicer interface.
type NetworkAPIService struct {
	hmy *hmy.Harmony
}

// NewNetworkAPIService creates a new instance of a NetworkAPIService.
func NewNetworkAPIService(hmy *hmy.Harmony) server.NetworkAPIServicer {
	return &NetworkAPIService{
		hmy: hmy,
	}
}

// NetworkList implements the /network/list endpoint (placeholder)
// TODO (dm): Update Node API to support beacon shard functionality for all nodes.
func (s *NetworkAPIService) NetworkList(
	ctx context.Context,
	request *types.MetadataRequest,
) (*types.NetworkListResponse, *types.Error) {
	network, err := common.GetNetwork(s.hmy.ShardID)
	if err != nil {
		return nil, &types.Error{
			Code:      common.CatchAllError.Code(),
			Message:   err.Error(),
			Retriable: false,
		}
	}
	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{
			network,
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint (placeholder)
// FIXME: remove placeholder & implement block endpoint
func (s *NetworkAPIService) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: 1000,
			Hash:  "block 1000",
		},
		CurrentBlockTimestamp: int64(1586483189000),
		GenesisBlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  "block 0",
		},
		Peers: []*types.Peer{
			{
				PeerID: "peer 1",
			},
		},
	}, nil
}

// NetworkOptions implements the /network/options endpoint (placeholder)
// FIXME: remove placeholder & implement block endpoint
func (s *NetworkAPIService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion: "1.4.0",
			NodeVersion:    "0.0.1",
		},
		Allow: &types.Allow{
			OperationStatuses: []*types.OperationStatus{
				{
					Status:     "Success",
					Successful: true,
				},
				{
					Status:     "Reverted",
					Successful: false,
				},
			},
			OperationTypes: []string{
				"Transfer",
				"Reward",
			},
			Errors: []*types.Error{
				{
					Code:      1,
					Message:   "not implemented",
					Retriable: false,
				},
			},
		},
	}, nil
}

package hmyapi

import (
	"context"
	"fmt"
	"time"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/harmony-one/harmony/hmy"
)

type BlockAPIService struct {
	harmony *hmy.Harmony
	network *types.NetworkIdentifier
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if *request.BlockIdentifier.Index != 1000 {
		previousBlockIndex := *request.BlockIdentifier.Index - 1
		if previousBlockIndex < 0 {
			previousBlockIndex = 0
		}

		blockNum := *request.BlockIdentifier.Index
		blk, err := s.harmony.APIBackend.BlockByNumber(ctx, rpc.BlockNumber(blockNum))
		if err != nil {
			fmt.Println(err)
		}

		prevBlk, err := s.harmony.APIBackend.BlockByNumber(ctx, rpc.BlockNumber(previousBlockIndex))
		if err != nil {
			fmt.Println(err)
		}

		return &types.BlockResponse{
			Block: &types.Block{
				BlockIdentifier: &types.BlockIdentifier{
					Index: *request.BlockIdentifier.Index,
					Hash:  blk.Hash().String(),
				},
				ParentBlockIdentifier: &types.BlockIdentifier{
					Index: previousBlockIndex,
					Hash:  prevBlk.Hash().String(),
				},
				Timestamp:    time.Now().UnixNano() / 1000000,
				Transactions: []*types.Transaction{},
			},
		}, nil
	}

	blk, err := s.harmony.APIBackend.BlockByNumber(ctx, rpc.BlockNumber(1000))
	if err != nil {
		fmt.Println(err)
	}

	prevBlk, err := s.harmony.APIBackend.BlockByNumber(ctx, rpc.BlockNumber(999))
	if err != nil {
		fmt.Println(err)
	}

	return &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 1000,
				Hash:  blk.Hash().String(),
			},
			ParentBlockIdentifier: &types.BlockIdentifier{
				Index: 999,
				Hash:  prevBlk.Hash().String(),
			},
			Timestamp: 1586483189000,
			Transactions: []*types.Transaction{
				{
					TransactionIdentifier: &types.TransactionIdentifier{
						Hash: "transaction 0",
					},
					Operations: []*types.Operation{
						{
							OperationIdentifier: &types.OperationIdentifier{
								Index: 0,
							},
							Type:   "Transfer",
							Status: "Success",
							Account: &types.AccountIdentifier{
								Address: "account 0",
							},
							Amount: &types.Amount{
								Value: "-1000",
								Currency: &types.Currency{
									Symbol:   "ROS",
									Decimals: 2,
								},
							},
						},
						{
							OperationIdentifier: &types.OperationIdentifier{
								Index: 1,
							},
							RelatedOperations: []*types.OperationIdentifier{
								{
									Index: 0,
								},
							},
							Type:   "Transfer",
							Status: "Reverted",
							Account: &types.AccountIdentifier{
								Address: "account 1",
							},
							Amount: &types.Amount{
								Value: "1000",
								Currency: &types.Currency{
									Symbol:   "ROS",
									Decimals: 2,
								},
							},
						},
					},
				},
			},
		},
		OtherTransactions: []*types.TransactionIdentifier{
			{
				Hash: "transaction 1",
			},
		},
	}, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	return &types.BlockTransactionResponse{
		Transaction: &types.Transaction{
			TransactionIdentifier: &types.TransactionIdentifier{
				Hash: "transaction 1",
			},
			Operations: []*types.Operation{
				{
					OperationIdentifier: &types.OperationIdentifier{
						Index: 0,
					},
					Type:   "Reward",
					Status: "Success",
					Account: &types.AccountIdentifier{
						Address: "account 2",
					},
					Amount: &types.Amount{
						Value: "1000",
						Currency: &types.Currency{
							Symbol:   "ROS",
							Decimals: 2,
						},
					},
				},
			},
		},
	}, nil
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(network *types.NetworkIdentifier, harmony *hmy.Harmony) server.BlockAPIServicer {
	return &BlockAPIService{
		network: network,
		harmony: harmony,
	}
}

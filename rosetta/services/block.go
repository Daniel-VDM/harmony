package services

import (
	"context"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/harmony-one/harmony/core/rawdb"
	hmytypes "github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/hmy"
	"github.com/harmony-one/harmony/rosetta/common"
	"github.com/harmony-one/harmony/rpc"
	stakingTypes "github.com/harmony-one/harmony/staking/types"
)

// BlockAPI implements the server.BlockAPIServicer interface.
type BlockAPI struct {
	hmy *hmy.Harmony
}

// NewBlockAPI creates a new instance of a BlockAPI.
func NewBlockAPI(hmy *hmy.Harmony) server.BlockAPIServicer {
	return &BlockAPI{
		hmy: hmy,
	}
}

// Block implements the /block endpoint
func (s *BlockAPI) Block(
	ctx context.Context, request *types.BlockRequest,
) (response *types.BlockResponse, rosettaError *types.Error) {
	if err := assertValidNetworkIdentifier(request.NetworkIdentifier, s.hmy.ShardID); err != nil {
		return nil, err
	}

	var blk *hmytypes.Block
	var currBlockID, prevBlockID *types.BlockIdentifier
	if blk, rosettaError = s.getBlock(ctx, request.BlockIdentifier); rosettaError != nil {
		return nil, rosettaError
	}

	// Format genesis block if it is requested.
	if blk.Number().Uint64() == 0 {
		return s.genesisBlock(ctx, request, blk)
	}

	currBlockID = &types.BlockIdentifier{
		Index: blk.Number().Int64(),
		Hash:  blk.Hash().String(),
	}
	prevBlock, err := s.hmy.BlockByNumber(ctx, rpc.BlockNumber(blk.Number().Int64()-1).EthBlockNumber())
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	prevBlockID = &types.BlockIdentifier{
		Index: prevBlock.Number().Int64(),
		Hash:  prevBlock.Hash().String(),
	}

	// Report undelegation payouts as transactions to fit API.
	// Report all transactions here since all undelegation payout amounts are known after fetching payouts.
	transactions, rosettaError := s.getAllUndelegationPayoutTransactions(ctx, blk)
	if rosettaError != nil {
		return nil, rosettaError
	}

	responseBlock := &types.Block{
		BlockIdentifier:       currBlockID,
		ParentBlockIdentifier: prevBlockID,
		Timestamp:             blk.Time().Int64() * 1e3, // Timestamp must be in ms.
		Transactions:          transactions,
	}

	otherTransactions := []*types.TransactionIdentifier{}
	for _, tx := range blk.Transactions() {
		otherTransactions = append(otherTransactions, &types.TransactionIdentifier{
			Hash: tx.Hash().String(),
		})
	}
	for _, tx := range blk.StakingTransactions() {
		otherTransactions = append(otherTransactions, &types.TransactionIdentifier{
			Hash: tx.Hash().String(),
		})
	}
	// Report cross-shard transaction payouts.
	for _, cxReceipts := range blk.IncomingReceipts() {
		// Report cross-shard transaction payouts.
		for _, cxReceipt := range cxReceipts.Receipts {
			otherTransactions = append(otherTransactions, &types.TransactionIdentifier{
				Hash: cxReceipt.TxHash.String(),
			})
		}
	}
	// Report pre-staking era block rewards as transactions to fit API.
	if !s.hmy.IsStakingEpoch(blk.Epoch()) {
		preStakingRewardTxIDs, rosettaError := s.getPreStakingRewardTransactionIdentifiers(ctx, blk)
		if rosettaError != nil {
			return nil, rosettaError
		}
		otherTransactions = append(otherTransactions, preStakingRewardTxIDs...)
	}

	return &types.BlockResponse{
		Block:             responseBlock,
		OtherTransactions: otherTransactions,
	}, nil
}

// getPreStakingRewardTransactionIdentifiers is only used for the /block endpoint
func (s *BlockAPI) getPreStakingRewardTransactionIdentifiers(
	ctx context.Context, blk *hmytypes.Block,
) ([]*types.TransactionIdentifier, *types.Error) {
	txIDs := []*types.TransactionIdentifier{}
	blockSigInfo, err := s.hmy.GetDetailedBlockSignerInfo(ctx, blk)
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	for acc, signedBlsKeys := range blockSigInfo.Signers {
		if len(signedBlsKeys) > 0 {
			txIDs = append(txIDs, getSpecialCaseTransactionIdentifier(blk.Hash(), acc, SpecialPreStakingRewardTxID))
		}
	}
	return txIDs, nil
}

// getBlock ..
func (s *BlockAPI) getBlock(
	ctx context.Context, request *types.PartialBlockIdentifier,
) (blk *hmytypes.Block, rosettaError *types.Error) {
	var err error
	if request.Hash != nil {
		requestBlockHash := ethcommon.HexToHash(*request.Hash)
		blk, err = s.hmy.GetBlock(ctx, requestBlockHash)
	} else if request.Index != nil {
		blk, err = s.hmy.BlockByNumber(ctx, rpc.BlockNumber(*request.Index).EthBlockNumber())
	} else {
		return nil, &common.BlockNotFoundError
	}
	if err != nil {
		return nil, common.NewError(common.BlockNotFoundError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	if blk == nil {
		return nil, &common.BlockNotFoundError
	}
	return blk, nil
}

// BlockTransaction implements the /block/transaction endpoint
func (s *BlockAPI) BlockTransaction(
	ctx context.Context, request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	if err := assertValidNetworkIdentifier(request.NetworkIdentifier, s.hmy.ShardID); err != nil {
		return nil, err
	}

	// Format genesis block transaction request
	if request.BlockIdentifier.Index == 0 {
		return s.specialGenesisBlockTransaction(ctx, request)
	}

	blockHash := ethcommon.HexToHash(request.BlockIdentifier.Hash)
	txHash := ethcommon.HexToHash(request.TransactionIdentifier.Hash)
	txInfo, rosettaError := s.getTransactionInfo(ctx, blockHash, txHash)
	if rosettaError != nil {
		// If no transaction info is found, check for special case transactions.
		response, rosettaError2 := s.specialBlockTransaction(ctx, request)
		if rosettaError2 != nil && rosettaError2.Code != common.TransactionNotFoundError.Code {
			return nil, common.NewError(common.TransactionNotFoundError, map[string]interface{}{
				"from_error": rosettaError2,
			})
		}
		return response, rosettaError2
	}

	var transaction *types.Transaction
	if txInfo.tx != nil && txInfo.receipt != nil {
		transaction, rosettaError = FormatTransaction(txInfo.tx, txInfo.receipt)
		if rosettaError != nil {
			return nil, rosettaError
		}
	} else if txInfo.cxReceipt != nil {
		transaction, rosettaError = FormatCrossShardReceiverTransaction(txInfo.cxReceipt)
		if rosettaError != nil {
			return nil, rosettaError
		}
	} else {
		return nil, &common.TransactionNotFoundError
	}
	return &types.BlockTransactionResponse{Transaction: transaction}, nil
}

// transactionInfo stores all related information for any transaction on the Harmony chain
// Note that some elements can be nil if not applicable
type transactionInfo struct {
	tx        hmytypes.PoolTransaction
	txIndex   uint64
	receipt   *hmytypes.Receipt
	cxReceipt *hmytypes.CXReceipt
}

// getTransactionInfo given the block hash and transaction hash
func (s *BlockAPI) getTransactionInfo(
	ctx context.Context, blockHash, txHash ethcommon.Hash,
) (txInfo *transactionInfo, rosettaError *types.Error) {
	// Look for all of the possible transactions
	var index uint64
	var plainTx *hmytypes.Transaction
	var stakingTx *stakingTypes.StakingTransaction
	plainTx, _, _, index = rawdb.ReadTransaction(s.hmy.ChainDb(), txHash)
	if plainTx == nil {
		stakingTx, _, _, index = rawdb.ReadStakingTransaction(s.hmy.ChainDb(), txHash)
	}
	cxReceipt, _, _, _ := rawdb.ReadCXReceipt(s.hmy.ChainDb(), txHash)

	if plainTx == nil && stakingTx == nil && cxReceipt == nil {
		return nil, &common.TransactionNotFoundError
	}

	var receipt *hmytypes.Receipt
	receipts, err := s.hmy.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	if int(index) < len(receipts) {
		receipt = receipts[index]
	}

	// Use pool transaction for concise formatting
	var tx hmytypes.PoolTransaction
	if stakingTx != nil {
		tx = stakingTx
	} else if plainTx != nil {
		tx = plainTx
	}

	return &transactionInfo{
		tx:        tx,
		txIndex:   index,
		receipt:   receipt,
		cxReceipt: cxReceipt,
	}, nil
}

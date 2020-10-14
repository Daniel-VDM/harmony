package services

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	hmytypes "github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/hmy"
	internalCommon "github.com/harmony-one/harmony/internal/common"
	"github.com/harmony-one/harmony/rosetta/common"
	stakingNetwork "github.com/harmony-one/harmony/staking/network"
	stakingTypes "github.com/harmony-one/harmony/staking/types"
)

var (
	// DefaultSenderAddress ..
	DefaultSenderAddress = ethcommon.HexToAddress("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE")
)

// FormatTransaction for staking, cross-shard sender, and plain transactions
func FormatTransaction(
	tx hmytypes.PoolTransaction, receipt *hmytypes.Receipt,
) (fmtTx *types.Transaction, rosettaError *types.Error) {
	var operations []*types.Operation
	var isCrossShard, isStaking bool
	var toShard uint32

	switch tx.(type) {
	case *stakingTypes.StakingTransaction:
		isStaking = true
		stakingTx := tx.(*stakingTypes.StakingTransaction)
		operations, rosettaError = GetStakingOperations(stakingTx, receipt)
		if rosettaError != nil {
			return nil, rosettaError
		}
		isCrossShard = false
		toShard = stakingTx.ShardID()
	case *hmytypes.Transaction:
		isStaking = false
		plainTx := tx.(*hmytypes.Transaction)
		operations, rosettaError = GetOperations(plainTx, receipt)
		if rosettaError != nil {
			return nil, rosettaError
		}
		isCrossShard = plainTx.ShardID() != plainTx.ToShardID()
		toShard = plainTx.ToShardID()
	default:
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "unknown transaction type",
		})
	}
	fromShard := tx.ShardID()
	txID := &types.TransactionIdentifier{Hash: tx.Hash().String()}

	// Set all possible metadata
	var txMetadata TransactionMetadata
	if isCrossShard {
		txMetadata.CrossShardIdentifier = txID
		txMetadata.ToShardID = &toShard
		txMetadata.FromShardID = &fromShard
	}
	if len(tx.Data()) > 0 && !isStaking {
		hexData := hex.EncodeToString(tx.Data())
		txMetadata.Data = &hexData
		txMetadata.Logs = receipt.Logs
	}
	metadata, err := types.MarshalMap(txMetadata)
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}

	return &types.Transaction{
		TransactionIdentifier: txID,
		Operations:            operations,
		Metadata:              metadata,
	}, nil
}

// FormatCrossShardReceiverTransaction for cross-shard payouts on destination shard
func FormatCrossShardReceiverTransaction(
	cxReceipt *hmytypes.CXReceipt,
) (txs *types.Transaction, rosettaError *types.Error) {
	ctxID := &types.TransactionIdentifier{Hash: cxReceipt.TxHash.String()}
	senderAccountID, rosettaError := newAccountIdentifier(cxReceipt.From)
	if rosettaError != nil {
		return nil, rosettaError
	}
	receiverAccountID, rosettaError := newAccountIdentifier(*cxReceipt.To)
	if rosettaError != nil {
		return nil, rosettaError
	}
	metadata, err := types.MarshalMap(TransactionMetadata{
		CrossShardIdentifier: ctxID,
		ToShardID:            &cxReceipt.ToShardID,
		FromShardID:          &cxReceipt.ShardID,
	})
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	opMetadata, err := types.MarshalMap(common.CrossShardTransactionOperationMetadata{
		From: senderAccountID,
		To:   receiverAccountID,
	})
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}

	return &types.Transaction{
		TransactionIdentifier: ctxID,
		Metadata:              metadata,
		Operations: []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 0, // There is no gas expenditure for cross-shard transaction payout
				},
				Type:    common.CrossShardTransferOperation,
				Status:  common.SuccessOperationStatus.Status,
				Account: receiverAccountID,
				Amount: &types.Amount{
					Value:    cxReceipt.Amount.String(),
					Currency: &common.Currency,
				},
				Metadata: opMetadata,
			},
		},
	}, nil
}

// FormatGenesisTransaction for genesis block's initial balances
func FormatGenesisTransaction(
	txID *types.TransactionIdentifier, targetAddr ethcommon.Address, shardID uint32,
) (fmtTx *types.Transaction, rosettaError *types.Error) {
	var b32Addr string
	targetB32Addr := internalCommon.MustAddressToBech32(targetAddr)
	for _, tx := range getPseudoTransactionForGenesis(getGenesisSpec(shardID)) {
		if tx.To() == nil {
			return nil, common.NewError(common.CatchAllError, nil)
		}
		b32Addr = internalCommon.MustAddressToBech32(*tx.To())
		if targetB32Addr == b32Addr {
			accID, rosettaError := newAccountIdentifier(*tx.To())
			if rosettaError != nil {
				return nil, rosettaError
			}
			return &types.Transaction{
				TransactionIdentifier: txID,
				Operations: []*types.Operation{
					{
						OperationIdentifier: &types.OperationIdentifier{
							Index: 0,
						},
						Type:    common.GenesisFundsOperation,
						Status:  common.SuccessOperationStatus.Status,
						Account: accID,
						Amount: &types.Amount{
							Value:    tx.Value().String(),
							Currency: &common.Currency,
						},
					},
				},
			}, nil
		}
	}
	return nil, &common.TransactionNotFoundError
}

// FormatPreStakingRewardTransaction for block rewards in pre-staking era for a given Bech-32 address.
func FormatPreStakingRewardTransaction(
	txID *types.TransactionIdentifier, blockSigInfo *hmy.DetailedBlockSignerInfo, address ethcommon.Address,
) (*types.Transaction, *types.Error) {
	signatures, ok := blockSigInfo.Signers[address]
	if !ok || len(signatures) == 0 {
		return nil, &common.TransactionNotFoundError
	}
	accID, rosettaError := newAccountIdentifier(address)
	if rosettaError != nil {
		return nil, rosettaError
	}

	// Calculate rewards exactly like `AccumulateRewardsAndCountSigs` but short circuit when possible.
	i := 0
	last := big.NewInt(0)
	rewardsForThisBlock := big.NewInt(0)
	count := big.NewInt(int64(blockSigInfo.TotalKeysSigned))
	for sigAddr, keys := range blockSigInfo.Signers {
		rewardsForThisAddr := big.NewInt(0)
		for range keys {
			cur := big.NewInt(0)
			cur.Mul(stakingNetwork.BlockReward, big.NewInt(int64(i+1))).Div(cur, count)
			reward := big.NewInt(0).Sub(cur, last)
			rewardsForThisAddr = new(big.Int).Add(reward, rewardsForThisAddr)
			last = cur
			i++
		}
		if sigAddr == address {
			rewardsForThisBlock = rewardsForThisAddr
			if !(rewardsForThisAddr.Cmp(big.NewInt(0)) > 0) {
				return nil, common.NewError(common.SanityCheckError, map[string]interface{}{
					"message": "expected non-zero block reward in pre-staking era for block signer",
				})
			}
			break
		}
	}

	return &types.Transaction{
		TransactionIdentifier: txID,
		Operations: []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 0,
				},
				Type:    common.PreStakingBlockRewardOperation,
				Status:  common.SuccessOperationStatus.Status,
				Account: accID,
				Amount: &types.Amount{
					Value:    rewardsForThisBlock.String(),
					Currency: &common.Currency,
				},
			},
		},
	}, nil
}

// FormatUndelegationPayoutTransaction for undelegation payouts at committee selection block
func FormatUndelegationPayoutTransaction(
	txID *types.TransactionIdentifier, delegatorPayouts hmy.UndelegationPayouts, address ethcommon.Address,
) (*types.Transaction, *types.Error) {
	accID, rosettaError := newAccountIdentifier(address)
	if rosettaError != nil {
		return nil, rosettaError
	}
	payout, ok := delegatorPayouts[address]
	if !ok {
		return nil, &common.TransactionNotFoundError
	}
	return &types.Transaction{
		TransactionIdentifier: txID,
		Operations: []*types.Operation{
			{
				OperationIdentifier: &types.OperationIdentifier{
					Index: 0,
				},
				Type:    common.UndelegationPayoutOperation,
				Status:  common.SuccessOperationStatus.Status,
				Account: accID,
				Amount: &types.Amount{
					Value:    payout.String(),
					Currency: &common.Currency,
				},
			},
		},
	}, nil

}

// negativeBigValue formats a transaction value as a string
func negativeBigValue(num *big.Int) string {
	value := "0"
	if num != nil && num.Cmp(big.NewInt(0)) != 0 {
		value = fmt.Sprintf("-%v", new(big.Int).Abs(num))
	}
	return value
}
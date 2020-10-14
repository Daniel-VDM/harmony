package services

import (
	"fmt"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	hmytypes "github.com/harmony-one/harmony/core/types"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/rosetta/common"
	rpcV2 "github.com/harmony-one/harmony/rpc/v2"
	"github.com/harmony-one/harmony/staking"
	stakingTypes "github.com/harmony-one/harmony/staking/types"
)

// GetOperationsFromTransaction for one of the following transactions:
// contract creation, cross-shard sender, same-shard transfer
func GetOperationsFromTransaction(
	tx *hmytypes.Transaction, receipt *hmytypes.Receipt,
) ([]*types.Operation, *types.Error) {
	senderAddress, err := tx.SenderAddress()
	if err != nil {
		senderAddress = FormatDefaultSenderAddress
	}
	accountID, rosettaError := newAccountIdentifier(senderAddress)
	if rosettaError != nil {
		return nil, rosettaError
	}

	// All operations excepts for cross-shard tx payout expend gas
	gasExpended := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), tx.GasPrice())
	gasOperations := newOperations(gasExpended, accountID)

	// Handle different cases of plain transactions
	var txOperations []*types.Operation
	if tx.To() == nil {
		txOperations, rosettaError = newContractCreationOperations(
			gasOperations[0].OperationIdentifier, tx, receipt, senderAddress,
		)
	} else if tx.ShardID() != tx.ToShardID() {
		txOperations, rosettaError = newCrossShardSenderTransferOperations(
			gasOperations[0].OperationIdentifier, tx, senderAddress,
		)
	} else {
		txOperations, rosettaError = newTransferOperations(
			gasOperations[0].OperationIdentifier, tx, receipt, senderAddress,
		)
	}
	if rosettaError != nil {
		return nil, rosettaError
	}

	return append(gasOperations, txOperations...), nil
}

// GetOperationsFromStakingTransaction for all staking directives
func GetOperationsFromStakingTransaction(
	tx *stakingTypes.StakingTransaction, receipt *hmytypes.Receipt,
) ([]*types.Operation, *types.Error) {
	senderAddress, err := tx.SenderAddress()
	if err != nil {
		senderAddress = FormatDefaultSenderAddress
	}
	accountID, rosettaError := newAccountIdentifier(senderAddress)
	if rosettaError != nil {
		return nil, rosettaError
	}

	// All operations excepts for cross-shard tx payout expend gas
	gasExpended := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), tx.GasPrice())
	gasOperations := newOperations(gasExpended, accountID)

	// Format staking message for metadata using decimal numbers (hence usage of rpcV2)
	rpcStakingTx, err := rpcV2.NewStakingTransaction(tx, ethcommon.Hash{}, 0, 0, 0)
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	metadata, err := types.MarshalMap(rpcStakingTx.Msg)
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}

	// Set correct amount depending on staking message directive that apply balance changes INSTANTLY
	var amount *types.Amount
	switch tx.StakingType() {
	case stakingTypes.DirectiveCreateValidator:
		if amount, rosettaError = getAmountFromCreateValidatorMessage(tx.Data()); rosettaError != nil {
			return nil, rosettaError
		}
	case stakingTypes.DirectiveDelegate:
		if amount, rosettaError = getAmountFromDelegateMessage(receipt, tx.Data()); rosettaError != nil {
			return nil, rosettaError
		}
	case stakingTypes.DirectiveCollectRewards:
		if amount, rosettaError = getAmountFromCollectRewards(receipt, senderAddress); rosettaError != nil {
			return nil, rosettaError
		}
	default:
		amount = &types.Amount{
			Value:    "0", // All other staking transactions do not apply balance changes instantly or at all
			Currency: &common.Currency,
		}
	}

	return append(gasOperations, &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: gasOperations[0].OperationIdentifier.Index + 1,
		},
		RelatedOperations: []*types.OperationIdentifier{
			gasOperations[0].OperationIdentifier,
		},
		Type:     tx.StakingType().String(),
		Status:   common.SuccessOperationStatus.Status,
		Account:  accountID,
		Amount:   amount,
		Metadata: metadata,
	}), nil
}

func getAmountFromCreateValidatorMessage(data []byte) (*types.Amount, *types.Error) {
	msg, err := stakingTypes.RLPDecodeStakeMsg(data, stakingTypes.DirectiveCreateValidator)
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	stkMsg, ok := msg.(*stakingTypes.CreateValidator)
	if !ok {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "unable to parse staking message for create validator tx",
		})
	}
	return &types.Amount{
		Value:    negativeBigValue(stkMsg.Amount),
		Currency: &common.Currency,
	}, nil
}

func getAmountFromDelegateMessage(receipt *hmytypes.Receipt, data []byte) (*types.Amount, *types.Error) {
	msg, err := stakingTypes.RLPDecodeStakeMsg(data, stakingTypes.DirectiveDelegate)
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	stkMsg, ok := msg.(*stakingTypes.Delegate)
	if !ok {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "unable to parse staking message for delegate tx",
		})
	}

	stkAmount := stkMsg.Amount
	logs := hmytypes.FindLogsWithTopic(receipt, staking.DelegateTopic)
	for _, log := range logs {
		if len(log.Data) > ethcommon.AddressLength {
			validatorAddress := ethcommon.BytesToAddress(log.Data[:ethcommon.AddressLength])
			if log.Address == stkMsg.DelegatorAddress && stkMsg.ValidatorAddress == validatorAddress {
				// Remove re-delegation amount as funds were never credited to account's balance.
				stkAmount = new(big.Int).Sub(stkAmount, new(big.Int).SetBytes(log.Data[ethcommon.AddressLength:]))
				break
			}
		}
	}
	return &types.Amount{
		Value:    negativeBigValue(stkAmount),
		Currency: &common.Currency,
	}, nil
}

func getAmountFromCollectRewards(
	receipt *hmytypes.Receipt, senderAddress ethcommon.Address,
) (*types.Amount, *types.Error) {
	var amount *types.Amount
	logs := hmytypes.FindLogsWithTopic(receipt, staking.CollectRewardsTopic)
	for _, log := range logs {
		if log.Address == senderAddress {
			amount = &types.Amount{
				Value:    big.NewInt(0).SetBytes(log.Data).String(),
				Currency: &common.Currency,
			}
			break
		}
	}
	if amount == nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": fmt.Sprintf("collect rewards amount not found for %v", senderAddress),
		})
	}
	return amount, nil
}

// newTransferOperations extracts & formats the operation(s) for plain transaction,
// including contract transactions.
func newTransferOperations(
	startingOperationID *types.OperationIdentifier,
	tx *hmytypes.Transaction, receipt *hmytypes.Receipt, senderAddress ethcommon.Address,
) ([]*types.Operation, *types.Error) {
	if tx.To() == nil {
		return nil, common.NewError(common.CatchAllError, nil)
	}
	receiverAddress := *tx.To()

	// Common elements
	opType := common.TransferOperation
	opStatus := common.SuccessOperationStatus.Status
	if receipt.Status == hmytypes.ReceiptStatusFailed {
		if len(tx.Data()) > 0 {
			opStatus = common.ContractFailureOperationStatus.Status
		} else {
			// Should never see a failed non-contract related transaction on chain
			opStatus = common.FailureOperationStatus.Status
			utils.Logger().Warn().Msgf("Failed transaction on chain: %v", tx.Hash().String())
		}
	}

	// Subtraction operation elements
	subOperationID := &types.OperationIdentifier{
		Index: startingOperationID.Index + 1,
	}
	subRelatedID := []*types.OperationIdentifier{
		startingOperationID,
	}
	subAccountID, rosettaError := newAccountIdentifier(senderAddress)
	if rosettaError != nil {
		return nil, rosettaError
	}
	subAmount := &types.Amount{
		Value:    negativeBigValue(tx.Value()),
		Currency: &common.Currency,
	}

	// Addition operation elements
	addOperationID := &types.OperationIdentifier{
		Index: subOperationID.Index + 1,
	}
	addRelatedID := []*types.OperationIdentifier{
		subOperationID,
	}
	addAccountID, rosettaError := newAccountIdentifier(receiverAddress)
	if rosettaError != nil {
		return nil, rosettaError
	}
	addAmount := &types.Amount{
		Value:    tx.Value().String(),
		Currency: &common.Currency,
	}

	return []*types.Operation{
		{
			OperationIdentifier: subOperationID,
			RelatedOperations:   subRelatedID,
			Type:                opType,
			Status:              opStatus,
			Account:             subAccountID,
			Amount:              subAmount,
		},
		{
			OperationIdentifier: addOperationID,
			RelatedOperations:   addRelatedID,
			Type:                opType,
			Status:              opStatus,
			Account:             addAccountID,
			Amount:              addAmount,
		},
	}, nil
}

// newCrossShardSenderTransferOperations extracts & formats the operation(s) for cross-shard-tx
// on the sender's shard.
func newCrossShardSenderTransferOperations(
	startingOperationID *types.OperationIdentifier,
	tx *hmytypes.Transaction, senderAddress ethcommon.Address,
) ([]*types.Operation, *types.Error) {
	if tx.To() == nil {
		return nil, common.NewError(common.CatchAllError, nil)
	}
	senderAccountID, rosettaError := newAccountIdentifier(senderAddress)
	if rosettaError != nil {
		return nil, rosettaError
	}
	receiverAccountID, rosettaError := newAccountIdentifier(*tx.To())
	if rosettaError != nil {
		return nil, rosettaError
	}
	metadata, err := types.MarshalMap(common.CrossShardTransactionOperationMetadata{
		From: senderAccountID,
		To:   receiverAccountID,
	})
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}

	return []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: startingOperationID.Index + 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				startingOperationID,
			},
			Type:    common.CrossShardTransferOperation,
			Status:  common.SuccessOperationStatus.Status,
			Account: senderAccountID,
			Amount: &types.Amount{
				Value:    negativeBigValue(tx.Value()),
				Currency: &common.Currency,
			},
			Metadata: metadata,
		},
	}, nil
}

// newContractCreationOperations extracts & formats the operation(s) for a contract creation tx
func newContractCreationOperations(
	startingOperationID *types.OperationIdentifier,
	tx *hmytypes.Transaction, txReceipt *hmytypes.Receipt, senderAddress ethcommon.Address,
) ([]*types.Operation, *types.Error) {
	senderAccountID, rosettaError := newAccountIdentifier(senderAddress)
	if rosettaError != nil {
		return nil, rosettaError
	}

	// Set execution status as necessary
	status := common.SuccessOperationStatus.Status
	if txReceipt.Status == hmytypes.ReceiptStatusFailed {
		status = common.ContractFailureOperationStatus.Status
	}
	contractAddressID, rosettaError := newAccountIdentifier(txReceipt.ContractAddress)
	if rosettaError != nil {
		return nil, rosettaError
	}

	return []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: startingOperationID.Index + 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				startingOperationID,
			},
			Type:    common.ContractCreationOperation,
			Status:  status,
			Account: senderAccountID,
			Amount: &types.Amount{
				Value:    negativeBigValue(tx.Value()),
				Currency: &common.Currency,
			},
			Metadata: map[string]interface{}{
				"contract_address": contractAddressID,
			},
		},
	}, nil
}

// newOperations creates a new operation with the gas fee as the first operation.
// Note: the gas fee is gasPrice * gasUsed.
func newOperations(
	gasFeeInATTO *big.Int, accountID *types.AccountIdentifier,
) []*types.Operation {
	return []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0, // gas operation is always first
			},
			Type:    common.ExpendGasOperation,
			Status:  common.SuccessOperationStatus.Status,
			Account: accountID,
			Amount: &types.Amount{
				Value:    negativeBigValue(gasFeeInATTO),
				Currency: &common.Currency,
			},
		},
	}
}

package services

import (
	"math/big"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/crypto"

	internalCommon "github.com/harmony-one/harmony/internal/common"
	"github.com/harmony-one/harmony/rosetta/common"
)

func TestGetContractCreationOperationComponents(t *testing.T) {
	refAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.Currency,
	}
	refKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}

	// test valid operations
	refOperation := &types.Operation{
		Type:    common.ContractCreationOperation,
		Amount:  refAmount,
		Account: refFrom,
	}
	testComponents, rosettaError := getContractCreationOperationComponents(refOperation)
	if rosettaError != nil {
		t.Error(rosettaError)
	}
	if testComponents.Type != refOperation.Type {
		t.Error("expected same operation")
	}
	if testComponents.From == nil || types.Hash(testComponents.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testComponents.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test nil amount
	_, rosettaError = getContractCreationOperationComponents(&types.Operation{
		Type:    common.ContractCreationOperation,
		Amount:  nil,
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test positive amount
	_, rosettaError = getContractCreationOperationComponents(&types.Operation{
		Type: common.ContractCreationOperation,
		Amount: &types.Amount{
			Value:    "12000",
			Currency: &common.Currency,
		},
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test different/unsupported currency
	_, rosettaError = getContractCreationOperationComponents(&types.Operation{
		Type: common.ContractCreationOperation,
		Amount: &types.Amount{
			Value: "-12000",
			Currency: &types.Currency{
				Symbol:   "bad",
				Decimals: 9,
			},
		},
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil account
	_, rosettaError = getContractCreationOperationComponents(&types.Operation{
		Type:    common.ContractCreationOperation,
		Amount:  refAmount,
		Account: nil,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}
}

func TestGetCrossShardOperationComponents(t *testing.T) {
	refAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.Currency,
	}
	refFromKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refFromKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refToKey := internalCommon.MustGeneratePrivateKey()
	refTo, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refToKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refMetadata := common.CrossShardTransactionOperationMetadata{
		From: refFrom,
		To:   refTo,
	}
	refMetadataMap, err := types.MarshalMap(refMetadata)
	if err != nil {
		t.Fatal(err)
	}

	// test valid operations
	refOperation := &types.Operation{
		Type:     common.CrossShardTransferOperation,
		Amount:   refAmount,
		Account:  refFrom,
		Metadata: refMetadataMap,
	}
	testComponents, rosettaError := getCrossShardOperationComponents(refOperation)
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if testComponents.Type != refOperation.Type {
		t.Error("expected same operation")
	}
	if testComponents.From == nil || types.Hash(testComponents.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testComponents.To == nil || types.Hash(testComponents.To) != types.Hash(refTo) {
		t.Error("expected same sender")
	}
	if testComponents.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test nil amount
	_, rosettaError = getCrossShardOperationComponents(&types.Operation{
		Type:     common.CrossShardTransferOperation,
		Amount:   nil,
		Account:  refFrom,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test positive amount
	_, rosettaError = getCrossShardOperationComponents(&types.Operation{
		Type: common.CrossShardTransferOperation,
		Amount: &types.Amount{
			Value:    "12000",
			Currency: &common.Currency,
		},
		Account:  refFrom,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test different/unsupported currency
	_, rosettaError = getCrossShardOperationComponents(&types.Operation{
		Type: common.CrossShardTransferOperation,
		Amount: &types.Amount{
			Value: "-12000",
			Currency: &types.Currency{
				Symbol:   "bad",
				Decimals: 9,
			},
		},
		Account:  refFrom,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil account
	_, rosettaError = getCrossShardOperationComponents(&types.Operation{
		Type:     common.CrossShardTransferOperation,
		Amount:   refAmount,
		Account:  nil,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test no metadata
	_, rosettaError = getCrossShardOperationComponents(&types.Operation{
		Type:    common.CrossShardTransferOperation,
		Amount:  refAmount,
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test bad metadata
	randomKey := internalCommon.MustGeneratePrivateKey()
	randomID, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(randomKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	badMetadata := common.CrossShardTransactionOperationMetadata{
		From: randomID,
		To:   refTo,
	}
	badMetadataMap, err := types.MarshalMap(badMetadata)
	if err != nil {
		t.Fatal(err)
	}
	_, rosettaError = getCrossShardOperationComponents(&types.Operation{
		Type:     common.CrossShardTransferOperation,
		Amount:   refAmount,
		Account:  refFrom,
		Metadata: badMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}
}

func TestGetTransferOperationComponents(t *testing.T) {
	refFromAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.Currency,
	}
	refToAmount := &types.Amount{
		Value:    "12000",
		Currency: &common.Currency,
	}
	refFromKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refFromKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refToKey := internalCommon.MustGeneratePrivateKey()
	refTo, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refToKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}

	// test valid operations
	refOperations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type:    common.TransferOperation,
			Amount:  refFromAmount,
			Account: refFrom,
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
			Type:    common.TransferOperation,
			Amount:  refToAmount,
			Account: refTo,
		},
	}
	testComponents, rosettaError := getTransferOperationComponents(refOperations)
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if testComponents.Type != refOperations[0].Type {
		t.Error("expected same operation")
	}
	if testComponents.From == nil || types.Hash(testComponents.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testComponents.To == nil || types.Hash(testComponents.To) != types.Hash(refTo) {
		t.Error("expected same sender")
	}
	if testComponents.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test valid operations flipped
	refOperations[0].Amount = refToAmount
	refOperations[0].Account = refTo
	refOperations[1].Amount = refFromAmount
	refOperations[1].Account = refFrom
	testComponents, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if testComponents.Type != refOperations[0].Type {
		t.Error("expected same operation")
	}
	if testComponents.From == nil || types.Hash(testComponents.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testComponents.To == nil || types.Hash(testComponents.To) != types.Hash(refTo) {
		t.Error("expected same sender")
	}
	if testComponents.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test no sender
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = nil
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test no receiver
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = nil
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid operation
	refOperations[0].Type = common.ExpendGasOperation
	refOperations[1].Type = common.TransferOperation
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid operation sender
	refOperations[0].Type = common.TransferOperation
	refOperations[1].Type = common.ExpendGasOperation
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[1].Type = common.TransferOperation

	// test nil amount
	refOperations[0].Amount = nil
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil amount sender
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = nil
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test uneven amount
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = &types.Amount{
		Value:    "0",
		Currency: &common.Currency,
	}
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test uneven amount sender
	refOperations[0].Amount = &types.Amount{
		Value:    "0",
		Currency: &common.Currency,
	}
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil amount
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = nil
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil amount sender
	refOperations[0].Amount = nil
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid currency
	refOperations[0].Amount = refFromAmount
	refOperations[0].Amount.Currency = &types.Currency{
		Symbol:   "bad",
		Decimals: 9,
	}
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[0].Amount.Currency = &common.Currency

	// test invalid currency sender
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Amount.Currency = &types.Currency{
		Symbol:   "bad",
		Decimals: 9,
	}
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[1].Amount.Currency = &common.Currency

	// test invalid related operation
	refOperations[1].RelatedOperations[0].Index = 2
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[1].RelatedOperations[0].Index = 0

	// test cyclic related operation
	refOperations[0].RelatedOperations = []*types.OperationIdentifier{
		{
			Index: 1,
		},
	}
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// Test invalid related operation sender
	refOperations[1].RelatedOperations = nil
	refOperations[0].RelatedOperations[0].Index = 3
	_, rosettaError = getTransferOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
}

func TestGetOperationComponents(t *testing.T) {
	refFromAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.Currency,
	}
	refToAmount := &types.Amount{
		Value:    "12000",
		Currency: &common.Currency,
	}
	refFromKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refFromKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refToKey := internalCommon.MustGeneratePrivateKey()
	refTo, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refToKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}

	// test valid transaction operation
	// Detailed test in TestGetTransferOperationComponents
	_, rosettaError = GetOperationComponents([]*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type:    common.TransferOperation,
			Amount:  refFromAmount,
			Account: refFrom,
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
			Type:    common.TransferOperation,
			Amount:  refToAmount,
			Account: refTo,
		},
	})
	if rosettaError != nil {
		t.Error(rosettaError)
	}

	// test valid cross-shard transaction operation
	// Detailed test in TestGetCrossShardOperationComponents
	refMetadata := common.CrossShardTransactionOperationMetadata{
		From: refFrom,
		To:   refTo,
	}
	refMetadataMap, err := types.MarshalMap(refMetadata)
	if err != nil {
		t.Fatal(err)
	}
	_, rosettaError = GetOperationComponents([]*types.Operation{
		{
			Type:     common.CrossShardTransferOperation,
			Amount:   refFromAmount,
			Account:  refFrom,
			Metadata: refMetadataMap,
		},
	})
	if rosettaError != nil {
		t.Error(rosettaError)
	}

	// test valid contract creation operation
	// Detailed test in TestGetContractCreationOperationComponents
	_, rosettaError = GetOperationComponents([]*types.Operation{
		{
			Type:    common.ContractCreationOperation,
			Amount:  refFromAmount,
			Account: refFrom,
		},
	})
	if rosettaError != nil {
		t.Error(rosettaError)
	}

	// test invalid number of operations
	refOperations := []*types.Operation{}
	_, rosettaError = GetOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid number of operations pas max number of operations
	for i := 0; i <= maxNumOfConstructionOps+1; i++ {
		refOperations = append(refOperations, &types.Operation{})
	}
	_, rosettaError = GetOperationComponents(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid operation
	_, rosettaError = GetOperationComponents([]*types.Operation{
		{
			Type: common.ExpendGasOperation,
		},
	})
	if rosettaError == nil {
		t.Error("expected error")
	}
}

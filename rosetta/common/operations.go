package common

import (
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/types"

	rpcV2 "github.com/harmony-one/harmony/rpc/v2"
	staking "github.com/harmony-one/harmony/staking/types"
)

// Invariant: A transaction can only contain 1 type of operation(s) other than gas expenditure.
const (
	// ExpendGasOperation ..
	ExpendGasOperation = "Gas"

	// TransferOperation ..
	TransferOperation = "Transfer"

	// CrossShardTransferOperation ..
	CrossShardTransferOperation = "CrossShardTransfer"

	// ContractCreationOperation ..
	ContractCreationOperation = "ContractCreation"

	// GenesisFundsOperation ..
	GenesisFundsOperation = "Genesis"

	// PreStakingBlockRewardOperation ..
	PreStakingBlockRewardOperation = "PreStakingBlockReward"

	// UndelegationPayoutOperation ..
	UndelegationPayoutOperation = "UndelegationPayout"
)

var (
	// PlainOperationTypes ..
	PlainOperationTypes = []string{
		ExpendGasOperation,
		TransferOperation,
		CrossShardTransferOperation,
		ContractCreationOperation,
		GenesisFundsOperation,
		PreStakingBlockRewardOperation,
		UndelegationPayoutOperation,
	}

	// StakingOperationTypes ..
	StakingOperationTypes = []string{
		staking.DirectiveCreateValidator.String(),
		staking.DirectiveEditValidator.String(),
		staking.DirectiveDelegate.String(),
		staking.DirectiveUndelegate.String(),
		staking.DirectiveCollectRewards.String(),
	}
)

var (
	// SuccessOperationStatus for tx operations who's amount affects the account
	SuccessOperationStatus = &types.OperationStatus{
		Status:     "success",
		Successful: true,
	}

	// ContractFailureOperationStatus for tx operations who's amount does not affect the account
	// due to a contract call failure (but still incurs gas).
	ContractFailureOperationStatus = &types.OperationStatus{
		Status:     "contract_failure",
		Successful: false,
	}

	// FailureOperationStatus ..
	FailureOperationStatus = &types.OperationStatus{
		Status:     "failure",
		Successful: false,
	}
)

// CreateValidatorOperationMetadata ..
type CreateValidatorOperationMetadata rpcV2.CreateValidatorMsg

// EditValidatorOperationMetadata ..
type EditValidatorOperationMetadata rpcV2.EditValidatorMsg

// DelegateOperationMetadata ..
type DelegateOperationMetadata rpcV2.DelegateMsg

// UndelegateOperationMetadata ..
type UndelegateOperationMetadata rpcV2.UndelegateMsg

// CollectRewardsMetadata ..
type CollectRewardsMetadata rpcV2.CollectRewardsMsg

// CrossShardTransactionOperationMetadata ..
type CrossShardTransactionOperationMetadata struct {
	From *types.AccountIdentifier `json:"from"`
	To   *types.AccountIdentifier `json:"to"`
}

// UnmarshalFromInterface ..
func (t *CrossShardTransactionOperationMetadata) UnmarshalFromInterface(metaData interface{}) error {
	var args CrossShardTransactionOperationMetadata
	dat, err := json.Marshal(metaData)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, &args); err != nil {
		return err
	}
	*t = args
	return nil
}

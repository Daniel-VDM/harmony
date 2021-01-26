package network

import (
	"errors"
	"fmt"
	"github.com/harmony-one/harmony/block"
	"math/big"
	"sync"

	"github.com/harmony-one/harmony/common/denominations"
	"github.com/harmony-one/harmony/consensus/engine"
	"github.com/harmony-one/harmony/consensus/reward"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/numeric"
	"github.com/harmony-one/harmony/shard"
)

var (
	// BlockReward is the block reward, to be split evenly among block signers in pre-staking era.
	BlockReward = new(big.Int).Mul(big.NewInt(24), big.NewInt(denominations.One))
	// BaseStakedReward is the flat-rate block reward for epos staking launch.
	// 28 ONE per block
	BaseStakedReward = numeric.NewDecFromBigInt(new(big.Int).Mul(
		big.NewInt(28), big.NewInt(denominations.One),
	))
	// FiveSecondsBaseStakedReward is the flat-rate block reward after epoch 230.
	// 17.5 ONE per block
	FiveSecondsBaseStakedReward = numeric.NewDecFromBigInt(new(big.Int).Mul(
		big.NewInt(17.5*denominations.Nano), big.NewInt(denominations.Nano),
	))
	// TwoSecondsBaseStakedReward is the flat-rate block reward after epoch 360.
	// 7 ONE per block
	TwoSecondsBaseStakedReward = numeric.NewDecFromBigInt(new(big.Int).Mul(
		big.NewInt(7*denominations.Nano), big.NewInt(denominations.Nano),
	))
	// TotalInitialTokens in the network across all shards
	TotalInitialTokens = numeric.NewDecFromBigInt(
		new(big.Int).Mul(big.NewInt(12600000000), big.NewInt(denominations.One)),
	)
	targetStakedPercentage = numeric.MustNewDecFromStr("0.35")
	dynamicAdjust          = numeric.MustNewDecFromStr("0.4")
	// ErrPayoutNotEqualBlockReward ..
	ErrPayoutNotEqualBlockReward = errors.New(
		"total payout not equal to blockreward",
	)
	// NoReward ..
	NoReward = big.NewInt(0)
	// EmptyPayout ..
	EmptyPayout = noReward{}
	once        sync.Once
)

type ignoreMissing struct{}

func (ignoreMissing) MissingSigners() shard.SlotList {
	return shard.SlotList{}
}

type noReward struct{ ignoreMissing }

func (noReward) ReadRoundResult() *reward.CompletedRound {
	return &reward.CompletedRound{
		Total:            big.NewInt(0),
		BeaconchainAward: []reward.Payout{},
		ShardChainAward:  []reward.Payout{},
	}
}

type preStakingEra struct {
	ignoreMissing
	payout *big.Int
}

// NewPreStakingEraRewarded ..
func NewPreStakingEraRewarded(totalAmount *big.Int) reward.Reader {
	return &preStakingEra{ignoreMissing{}, totalAmount}
}

func (p *preStakingEra) ReadRoundResult() *reward.CompletedRound {
	return &reward.CompletedRound{
		Total:            p.payout,
		BeaconchainAward: []reward.Payout{},
		ShardChainAward:  []reward.Payout{},
	}
}

type stakingEra struct {
	reward.CompletedRound
	missingSigners shard.SlotList
}

// NewStakingEraRewardForRound ..
func NewStakingEraRewardForRound(
	totalPayout *big.Int,
	mia shard.SlotList,
	beaconP, shardP []reward.Payout,
) reward.Reader {
	return &stakingEra{
		CompletedRound: reward.CompletedRound{
			Total:            totalPayout,
			BeaconchainAward: beaconP,
			ShardChainAward:  shardP,
		},
		missingSigners: mia,
	}
}

// MissingSigners ..
func (r *stakingEra) MissingSigners() shard.SlotList {
	return r.missingSigners
}

// ReadRoundResult ..
func (r *stakingEra) ReadRoundResult() *reward.CompletedRound {
	return &r.CompletedRound
}

func adjust(amount numeric.Dec) numeric.Dec {
	return amount.MulTruncate(
		numeric.NewDecFromBigInt(big.NewInt(denominations.One)),
	)
}

// Adjustment ..
func Adjustment(percentageStaked numeric.Dec) (numeric.Dec, numeric.Dec) {
	howMuchOff := targetStakedPercentage.Sub(percentageStaked)
	adjustBy := adjust(
		howMuchOff.MulTruncate(numeric.NewDec(100)).Mul(dynamicAdjust),
	)
	return howMuchOff, adjustBy
}

// WhatPercentStakedNow ..
func WhatPercentStakedNow(
	beaconchain engine.ChainReader,
	timestamp int64,
) (*big.Int, *numeric.Dec, error) {
	fmt.Printf("WhatPercentStakedNow CALLED\n")
	blk, err := beaconchain.ReadFirstBlockNumberOfStakingEra(1)
	fmt.Printf("1st block of era: %v, err: %v\n", blk, err)
	stakedNow := numeric.ZeroDec()
	// Only elected validators' stake is counted in stake ratio because only their stake is under slashing risk
	active, err := beaconchain.ReadShardState(beaconchain.CurrentBlock().Epoch())
	if err != nil {
		return nil, nil, err
	}

	soFarDoledOut, err := beaconchain.ReadBlockRewardAccumulator(
		beaconchain.CurrentHeader().Number().Uint64(),
	)

	if err != nil {
		return nil, nil, err
	}

	dole := numeric.NewDecFromBigInt(soFarDoledOut)

	for _, electedValAdr := range active.StakedValidators().Addrs {
		wrapper, err := beaconchain.ReadValidatorInformation(electedValAdr)
		if err != nil {
			return nil, nil, err
		}
		stakedNow = stakedNow.Add(
			numeric.NewDecFromBigInt(wrapper.TotalDelegation()),
		)
	}
	percentage := stakedNow.Quo(TotalInitialTokens.Mul(
		reward.PercentageForTimeStamp(timestamp),
	).Add(dole))
	utils.Logger().Info().
		Str("so-far-doled-out", dole.String()).
		Str("staked-percentage", percentage.String()).
		Str("currently-staked", stakedNow.String()).
		Msg("Computed how much staked right now")
	return soFarDoledOut, &percentage, nil
}

// SetInitTotalSupply once
func SetInitTotalSupply(supply *big.Int) {
	// TODO: hook this into the node init...
	once.Do(func() {
		TotalInitialTokens = numeric.NewDecFromBigInt(supply)
	})
}

// GetTotalSupply of the entire network (on all shards) at the
// latest header of the given beacon chain.
//
// May introduce some slight inaccuracies if NOT in staking era.
// Specifically, non-beacon shards may be de-synced in terms of block height & epoch,
// resulting in a slight over or under estimate. However, in staking era it is
// accurate (once cross links finalized for current header) due to the known
// number of blocks in pre-staking era and reward accumulator.
func GetTotalSupply(
	beaconchain engine.ChainReader,
) (*big.Int, error) {
	header := beaconchain.CurrentHeader()
	numShard := shard.Schedule.InstanceForEpoch(header.Epoch()).NumShards()
	_ = numShard
	return nil, nil
}

// getPreStakingBlockRewards across all shards.
//
// WARNING: This assumes that the number of shards is constant in the pre-staking era.
// A slight under or over estimate is possible if the network is still in pre-staking era.
func getPreStakingBlockRewards(
	beaconchain engine.ChainReader, currHeader *block.Header,
) (*big.Int, error) {
	chainConfig := beaconchain.Config()
	numShards := shard.Schedule.InstanceForEpoch(big.NewInt(0)).NumShards()
	if !chainConfig.IsStaking(currHeader.Epoch()) {
		return new(big.Int).Mul(new(big.Int).Mul(BlockReward, currHeader.Number()), big.NewInt(int64(numShards))), nil
	}

	lastBlocksInPreStaking := make([]*big.Int, numShards)
	for i := uint32(0); i < numShards; i++ {
		firstBlockInStakingEra, err := beaconchain.ReadFirstBlockNumberOfStakingEra(i)
		if err != nil {
			return nil, err
		}
		lastBlocksInPreStaking[i] = new(big.Int).Sub(firstBlockInStakingEra, big.NewInt(1))
	}
	// TODO: rest of impl...

	return nil, nil
}

func init() {
	// TODO: hook this into the node init...
	//numShards := shard.Schedule.InstanceForEpoch(big.NewInt(0)).NumShards()
	//totalInitTokens := big.NewInt(0)
	//for i := uint32(0); i < numShards; i++ {
	//	totalInitTokens = new(big.Int).Add(genspec.GetInitialSupply(i), totalInitTokens)
	//}
	//TotalInitialTokens = numeric.NewDecFromBigInt(totalInitTokens)
}

package economics

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/harmony-one/harmony/common/denominations"
	"github.com/harmony-one/harmony/consensus/engine"
	"github.com/harmony-one/harmony/consensus/reward"
	"github.com/harmony-one/harmony/consensus/votepower"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/numeric"
)

const (
	numBlocksPerYear = 300_000_000
)

// Produced is a record rewards given out after a successful round of consensus
type Produced struct {
	blockNumber uint64
	accum       votepower.RewardAccumulation
}

// NewProduced ..
func NewProduced(b uint64, r []votepower.VoterReward, t *big.Int) *Produced {
	return &Produced{b, votepower.RewardAccumulation{t, r}}
}

// ReadBlockNumber ..
func (p *Produced) ReadBlockNumber() uint64 {
	return p.blockNumber
}

// ReadRewarded ..
func (p *Produced) ReadRewarded() []votepower.VoterReward {
	return p.accum.ValidatorReward
}

// ReadTotalPayout ..
func (p *Produced) ReadTotalPayout() *big.Int {
	return p.accum.NetworkTotalPayout
}

var (
	// BlockReward is the block reward, to be split evenly among block signers.
	BlockReward = new(big.Int).Mul(big.NewInt(24), big.NewInt(denominations.One))
	// BaseStakedReward is the base block reward for epos.
	BaseStakedReward = numeric.NewDecFromBigInt(new(big.Int).Mul(
		big.NewInt(18), big.NewInt(denominations.One),
	))
	totalTokens = numeric.NewDecFromBigInt(
		new(big.Int).Mul(big.NewInt(12_600_000_000), big.NewInt(denominations.One)),
	)
	targetStakedPercentage = numeric.MustNewDecFromStr("0.35")
	dynamicAdjust          = numeric.MustNewDecFromStr("0.4")
	oneHundred             = numeric.NewDec(100)
	potentialAdjust        = oneHundred.Mul(dynamicAdjust)
	zero                   = numeric.ZeroDec()
	blocksPerYear          = numeric.NewDec(numBlocksPerYear)
	// ErrPayoutNotEqualBlockReward ..
	ErrPayoutNotEqualBlockReward = errors.New("total payout not equal to blockreward")
)

// NewNoReward ..
func NewNoReward(blockNum uint64) *Produced {
	return &Produced{blockNum, votepower.RewardAccumulation{common.Big0, nil}}
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
		howMuchOff.MulTruncate(oneHundred).Mul(dynamicAdjust),
	)
	return howMuchOff, adjustBy
}

// Snapshot ..
type Snapshot struct {
	Rewards          *votepower.RewardAccumulation `json:"accumulated-rewards"`
	APR              []ComputedAPR                 `json:"active-validators-apr"`
	StakedPercentage *numeric.Dec                  `json:"current-percent-token-staked"`
}

// NewSnapshot returns a record with metrics on
// the network accumulated rewards,
// and by validator.
func NewSnapshot(
	beaconchain engine.ChainReader,
	timestamp int64,
	includeAPRs bool,
) (*Snapshot, error) {
	stakedNow, rates, junk :=
		numeric.ZeroDec(), []ComputedAPR{}, numeric.ZeroDec()
	// Only active validators' stake is counted in
	// stake ratio because only their stake is under slashing risk
	active, err := beaconchain.ReadActiveValidatorList()
	if err != nil {
		return nil, err
	}
	if includeAPRs {
		rates = make([]ComputedAPR, len(active))
	}
	soFarDoledOut, err := beaconchain.ReadBlockRewardAccumulator(
		beaconchain.CurrentHeader().Number().Uint64(),
	)

	if err != nil {
		return nil, err
	}

	dole := numeric.NewDecFromBigInt(soFarDoledOut.NetworkTotalPayout)

	for i := range active {
		wrapper, err := beaconchain.ReadValidatorInformation(active[i])
		if err != nil {
			return nil, err
		}
		total := wrapper.TotalDelegation()
		stakedNow = stakedNow.Add(numeric.NewDecFromBigInt(total))
		if includeAPRs {
			rates[i] = ComputedAPR{active[i], total, junk, numeric.ZeroDec()}
		}
	}

	circulatingSupply := totalTokens.Mul(
		reward.PercentageForTimeStamp(timestamp),
	).Add(dole)

	for i := range rates {
		rates[i].StakeRatio = numeric.NewDecFromBigInt(
			rates[i].TotalStakedToken,
		).Quo(circulatingSupply)
		if reward := BaseStakedReward.Sub(
			rates[i].StakeRatio.Sub(targetStakedPercentage).Mul(potentialAdjust),
		); reward.GT(zero) {
			rates[i].APR = blocksPerYear.Mul(reward).Quo(stakedNow)
		}
	}

	percentage := stakedNow.Quo(circulatingSupply)
	utils.Logger().Info().
		Str("so-far-doled-out", dole.String()).
		Str("staked-percentage", percentage.String()).
		Str("currently-staked", stakedNow.String()).
		Msg("Computed how much staked right now")
	return &Snapshot{soFarDoledOut, rates, &percentage}, nil
}

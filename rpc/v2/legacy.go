package v2

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/harmony-one/harmony/hmy"
	internal_common "github.com/harmony-one/harmony/internal/common"
)

// PublicLegacyService provides an API to access the Harmony blockchain.
// Services here are legacy methods, specific to the V1 RPC that can be deprecated in the future.
type PublicLegacyService struct {
	hmy *hmy.Harmony
}

// NewPublicLegacyService creates a new API for the RPC interface
func NewPublicLegacyAPI(hmy *hmy.Harmony) rpc.API {
	return rpc.API{
		Namespace: "hmyv2",
		Version:   "1.0",
		Service:   &PublicLegacyService{hmy},
		Public:    true,
	}
}

// GetBalance returns the amount of Atto for the given address in the state of the
// given block number.
func (s *PublicLegacyService) GetBalance(
	ctx context.Context, address string,
) (*big.Int, error) {
	addr := internal_common.ParseAddr(address)
	balance, err := s.hmy.GetBalance(ctx, addr, rpc.BlockNumber(-1))
	if err != nil {
		return nil, err
	}
	return balance, nil
}

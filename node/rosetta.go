package node

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/event"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/examples/server/services"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/harmony-one/harmony/hmy"
	"github.com/harmony-one/harmony/internal/hmyapi"
)

const (
	serverPort = 8080
)

// newBlockchainRouter creates a Mux http.Handler from a collection
// of server controllers.
func newBlockchainRouter(
	network *types.NetworkIdentifier,
	asserter *asserter.Asserter,
	hmy *hmy.Harmony,
) http.Handler {
	networkAPIService := services.NewNetworkAPIService(network)
	networkAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
	)

	blockAPIService := hmyapi.NewBlockAPIService(network, hmy)
	blockAPIController := server.NewBlockAPIController(
		blockAPIService,
		asserter,
	)

	return server.NewRouter(networkAPIController, blockAPIController)
}

func (node *Node) StartRosettaServer() {
	go func() {
		network := &types.NetworkIdentifier{
			Blockchain: "Rosetta",
			Network:    "Testnet",
		}
		// The asserter automatically rejects incorrectly formatted
		// requests.
		asserter, err := asserter.NewServer(
			[]*types.NetworkIdentifier{network},
		)
		if err != nil {
			fmt.Println(err)
		}

		harmony, _ = hmy.New(
			node, node.TxPool, node.CxPool, new(event.TypeMux), node.Consensus.ShardID,
		)

		router := newBlockchainRouter(network, asserter, harmony)
		fmt.Printf("Listening on port %d\n", serverPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", serverPort), router); err != nil {
			fmt.Println(err.Error())
		}
	}()
}

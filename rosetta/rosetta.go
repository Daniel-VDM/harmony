package rosetta

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	rosettaServer "github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/harmony-one/harmony/rosetta/server"

	"github.com/harmony-one/harmony/hmy"
	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
	"github.com/harmony-one/harmony/internal/utils"
	"github.com/harmony-one/harmony/rosetta/common"
	"github.com/harmony-one/harmony/rosetta/services"
)

var (
	listener net.Listener
)

// StartServers starts the rosetta http server
func StartServers(hmy *hmy.Harmony, config nodeconfig.RosettaServerConfig) error {
	if !config.HTTPEnabled {
		utils.Logger().Info().Msg("Rosetta http server disabled...")
		return nil
	}

	network, err := common.GetNetwork(hmy.ShardID)
	if err != nil {
		return err
	}
	serverAsserter, err := asserter.NewServer(
		append(common.PlainOperationTypes, common.StakingOperationTypes...),
		nodeconfig.GetShardConfig(hmy.ShardID).Role() == nodeconfig.ExplorerNode,
		[]*types.NetworkIdentifier{network},
	)
	if err != nil {
		return err
	}

	router := getRouter(serverAsserter, hmy)
	router = server.RecoverMiddleware(rosettaServer.CorsMiddleware(rosettaServer.LoggerMiddleware(router)))
	utils.Logger().Info().
		Int("port", config.HTTPPort).
		Str("ip", config.HTTPIp).
		Msg("Starting Rosetta server")
	endpoint := fmt.Sprintf("%s:%d", config.HTTPIp, config.HTTPPort)
	go runHTTPServer(router, endpoint)
	return nil
}

// StopServers stops the rosetta http server
func StopServers() error {
	if err := listener.Close(); err != nil {
		return err
	}
	return nil
}

func runHTTPServer(handler http.Handler, endpoint string) {
	s := http.Server{
		Handler:      handler,
		ReadTimeout:  common.ReadTimeout,
		WriteTimeout: common.WriteTimeout,
		IdleTimeout:  common.IdleTimeout,
	}
	var err error
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr, "Unable to start Rosetta server at: %v - err: %v", endpoint, err.Error(),
		)
	}
	if err := s.Serve(listener); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr, "Unable to start Rosetta server at: %v - err: %v", endpoint, err.Error(),
		)
	}
	fmt.Printf("Started Rosetta server at: %v\n", endpoint)
}

func getRouter(asserter *asserter.Asserter, hmy *hmy.Harmony) http.Handler {
	return rosettaServer.NewRouter(
		rosettaServer.NewAccountAPIController(services.NewAccountAPI(hmy), asserter),
		rosettaServer.NewBlockAPIController(services.NewBlockAPI(hmy), asserter),
		rosettaServer.NewMempoolAPIController(services.NewMempoolAPI(hmy), asserter),
		rosettaServer.NewNetworkAPIController(services.NewNetworkAPI(hmy), asserter),
		rosettaServer.NewConstructionAPIController(services.NewConstructionAPI(hmy), asserter),
	)
}

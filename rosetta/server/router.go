package server

import (
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"

	"github.com/harmony-one/harmony/hmy"
	"github.com/harmony-one/harmony/rosetta/services"
)

func GetRouter(asserter *asserter.Asserter, hmy *hmy.Harmony) http.Handler {
	return server.NewRouter(
		NewAccountAPIController(services.NewAccountAPI(hmy), asserter),
		server.NewBlockAPIController(services.NewBlockAPI(hmy), asserter),
		server.NewMempoolAPIController(services.NewMempoolAPI(hmy), asserter),
		server.NewNetworkAPIController(services.NewNetworkAPI(hmy), asserter),
		server.NewConstructionAPIController(services.NewConstructionAPI(hmy), asserter),
	)
}

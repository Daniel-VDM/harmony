package server

import (
	"net/http"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/golang/groupcache/singleflight"
)

// AccountAPIController is an API controller with optimizations/changes specific to harmony.
type AccountAPIController struct {
	baseController *server.AccountAPIController
	exeGroup       singleflight.Group
}

// NewAccountAPIController creates an api controller.
// It panics if the base rosetta controller is an unexpected type.
func NewAccountAPIController(
	s server.AccountAPIServicer,
	asserter *asserter.Asserter,
) server.Router {
	controller, ok := server.NewAccountAPIController(s, asserter).(*server.AccountAPIController)
	if !ok {
		panic("unknown base account API controller")
	}

	return &AccountAPIController{
		baseController: controller,
	}
}

// Routes returns all of the api route for the AccountAPIController
func (c *AccountAPIController) Routes() server.Routes {
	return server.Routes{
		{
			"AccountBalance",
			strings.ToUpper("Post"),
			"/account/balance",
			c.AccountBalance,
		},
	}
}

// AccountBalance - Get an Account Balance
func (c *AccountAPIController) AccountBalance(w http.ResponseWriter, r *http.Request) {
	c.baseController.AccountBalance(w, r)
}

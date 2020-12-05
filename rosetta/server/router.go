package server

import (
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/golang/groupcache/singleflight"
	"github.com/gorilla/mux"

	"github.com/harmony-one/harmony/hmy"
	"github.com/harmony-one/harmony/rosetta/services"
)

func GetRouter(asserter *asserter.Asserter, hmy *hmy.Harmony) http.Handler {
	server.NewRouter()
	return NewRouterWithSingleFlight(
		server.NewAccountAPIController(services.NewAccountAPI(hmy), asserter),
		server.NewBlockAPIController(services.NewBlockAPI(hmy), asserter),
		server.NewMempoolAPIController(services.NewMempoolAPI(hmy), asserter),
		server.NewNetworkAPIController(services.NewNetworkAPI(hmy), asserter),
		server.NewConstructionAPIController(services.NewConstructionAPI(hmy), asserter),
	)
}

func NewRouterWithSingleFlight(routers ...server.Router) http.Handler {
	router := mux.NewRouter().StrictSlash(true)
	for _, api := range routers {
		for _, route := range api.Routes() {
			router.
				Methods(route.Method).
				Path(route.Pattern).
				Name(route.Name).
				Handler(singleFlightHandler(route.HandlerFunc))
		}
	}
	return router
}

// singleFlightHandlerBuf implements http.ResponseWriter and
// is only used for the singleFlightHandler.
type singleFlightHandlerBuf struct {
	data   []byte
	header http.Header
	code   int
}

func (b *singleFlightHandlerBuf) Header() http.Header {
	return b.header
}

func (b *singleFlightHandlerBuf) WriteHeader(statusCode int) {
	b.code = statusCode
}

func (b *singleFlightHandlerBuf) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func singleFlightHandler(baseHandler http.HandlerFunc) http.HandlerFunc {
	var group singleflight.Group
	return func(w http.ResponseWriter, r *http.Request) {
		//if true {
		//	baseHandler(w, r)
		//	return
		//}
		ret, _ := group.Do(types.Hash(r.Body), func() (interface{}, error) {
			buf := &singleFlightHandlerBuf{
				header: w.Header(),
				code:   http.StatusOK,
			}
			baseHandler(buf, r)
			return buf, nil
		})
		singleBuf, ok := ret.(*singleFlightHandlerBuf)
		if !ok {
			http.Error(w, "unknown buffer for single flight handler", http.StatusInternalServerError)
		}
		w.WriteHeader(singleBuf.code)
		if _, err := w.Write(singleBuf.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

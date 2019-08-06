package v1alpha1

import (
	api "github.com/dexidp/dex/api/v1alpha1"
	"github.com/dexidp/dex/pkg/log"
	"github.com/dexidp/dex/storage"
)

// apiVersion increases every time a new call is added to the API. Clients should use this info
// to determine if the server supports specific features.
const apiVersion = "v1alpha1"

const (
	// recCost is the recommended bcrypt cost, which balances hash strength and
	// efficiency.
	recCost = 12

	// upBoundCost is a sane upper bound on bcrypt cost determined by benchmarking:
	// high enough to ensure secure encryption, low enough to not put unnecessary
	// load on a dex server.
	upBoundCost = 16
)

// NewAPI returns a server which implements the gRPC API interface.
func NewAPI(s storage.Storage, logger log.Logger) api.DexServer {
	return dexAPI{
		s:      s,
		logger: logger,
	}
}

type dexAPI struct {
	s      storage.Storage
	logger log.Logger
}

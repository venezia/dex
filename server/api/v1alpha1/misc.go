package v1alpha1

import (
	api "github.com/dexidp/dex/api/v1alpha1"
	"github.com/dexidp/dex/version"
	"golang.org/x/net/context"
)

func (d dexAPI) GetVersion(ctx context.Context, req *api.VersionReq) (*api.VersionResp, error) {
	return &api.VersionResp{
		Server: version.Version,
		Api:    apiVersion,
	}, nil
}

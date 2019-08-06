package v1alpha1

import (
	api "github.com/dexidp/dex/api/v1alpha1"
	"github.com/dexidp/dex/storage"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d dexAPI) CreateClient(ctx context.Context, req *api.CreateClientReq) (*api.CreateClientResp, error) {
	if req.Client == nil {
		return nil, status.Error(codes.InvalidArgument, "no client supplied")
	}

	if req.Client.Id == "" {
		req.Client.Id = storage.NewID()
	}
	if req.Client.Secret == "" {
		req.Client.Secret = storage.NewID() + storage.NewID()
	}

	c := storage.Client{
		ID:           req.Client.Id,
		Secret:       req.Client.Secret,
		RedirectURIs: req.Client.RedirectUris,
		TrustedPeers: req.Client.TrustedPeers,
		Public:       req.Client.Public,
		Name:         req.Client.Name,
		LogoURL:      req.Client.LogoUrl,
	}
	if err := d.s.CreateClient(c); err != nil {
		if err == storage.ErrAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, "client already exists")
		}
		d.logger.Errorf("api: failed to create client: %v", err)
		return nil, status.Errorf(codes.Internal, "create client: %v", err)
	}

	return &api.CreateClientResp{}, nil
}

func (d dexAPI) UpdateClient(ctx context.Context, req *api.UpdateClientReq) (*api.UpdateClientResp, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "update client: no client ID supplied")
	}

	err := d.s.UpdateClient(req.Id, func(old storage.Client) (storage.Client, error) {
		if req.RedirectUris != nil {
			old.RedirectURIs = req.RedirectUris
		}
		if req.TrustedPeers != nil {
			old.TrustedPeers = req.TrustedPeers
		}
		if req.Name != "" {
			old.Name = req.Name
		}
		if req.LogoUrl != "" {
			old.LogoURL = req.LogoUrl
		}
		return old, nil
	})

	if err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdateClientResp{}, status.Error(codes.NotFound, "client not found, cannot update")
		}
		d.logger.Errorf("api: failed to update the client: %v", err)
		return nil, status.Errorf(codes.Internal, "update client: %v", err)
	}
	return &api.UpdateClientResp{}, nil
}

func (d dexAPI) DeleteClient(ctx context.Context, req *api.DeleteClientReq) (*api.DeleteClientResp, error) {
	err := d.s.DeleteClient(req.Id)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeleteClientResp{}, status.Error(codes.NotFound, "client not found, cannot delete")
		}
		d.logger.Errorf("api: failed to delete client: %v", err)
		return nil, status.Errorf(codes.Internal, "delete client: %v", err)
	}
	return &api.DeleteClientResp{}, nil
}

func (d dexAPI) ListClients(ctx context.Context, req *api.ListClientsReq) (*api.ListClientsResp, error) {
	clientList, err := d.s.ListClients()
	if err != nil {
		d.logger.Errorf("api: failed to list clients: %v", err)
		return nil, status.Errorf(codes.Internal, "list clients: %v", err)
	}

	var clients []*api.Client
	for _, client := range clientList {
		c := api.Client{
			Id:           client.ID,
			Secret:       client.Secret,
			RedirectUris: client.RedirectURIs,
			TrustedPeers: client.TrustedPeers,
			Public:       client.Public,
			Name:         client.Name,
			LogoUrl:      client.LogoURL,
		}
		clients = append(clients, &c)
	}

	return &api.ListClientsResp{
		Clients: clients,
	}, nil

}

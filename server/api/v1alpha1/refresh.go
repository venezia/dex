package v1alpha1

import (
	api "github.com/dexidp/dex/api/v1alpha1"
	"github.com/dexidp/dex/server/internal"
	"github.com/dexidp/dex/storage"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d dexAPI) ListRefreshForUser(ctx context.Context, req *api.ListRefreshForUserReq) (*api.ListRefreshForUserResp, error) {
	id := new(internal.IDTokenSubject)
	if err := internal.Unmarshal(req.UserId, id); err != nil {
		d.logger.Errorf("api: failed to unmarshal ID Token subject: %v", err)
		return nil, status.Errorf(codes.Internal, "api: failed to unmarshal ID Token subject: %v", err)
	}

	var refreshTokenRefs []*api.RefreshTokenRef
	offlineSessions, err := d.s.GetOfflineSessions(id.UserId, id.ConnId)
	if err != nil {
		if err == storage.ErrNotFound {
			// This means that this user-client pair does not have a refresh token yet.
			// An empty list should be returned instead of an error.
			return &api.ListRefreshForUserResp{
				RefreshTokens: refreshTokenRefs,
			}, nil
		}
		d.logger.Errorf("api: failed to list refresh tokens %t here : %v", err == storage.ErrNotFound, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, session := range offlineSessions.Refresh {
		r := api.RefreshTokenRef{
			Id:        session.ID,
			ClientId:  session.ClientID,
			CreatedAt: session.CreatedAt.Unix(),
			LastUsed:  session.LastUsed.Unix(),
		}
		refreshTokenRefs = append(refreshTokenRefs, &r)
	}

	return &api.ListRefreshForUserResp{
		RefreshTokens: refreshTokenRefs,
	}, nil
}

func (d dexAPI) RevokeRefresh(ctx context.Context, req *api.RevokeRefreshReq) (*api.RevokeRefreshResp, error) {
	id := new(internal.IDTokenSubject)
	if err := internal.Unmarshal(req.UserId, id); err != nil {
		d.logger.Errorf("api: failed to unmarshal ID Token subject: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	var (
		refreshID string
		notFound  bool
	)
	updater := func(old storage.OfflineSessions) (storage.OfflineSessions, error) {
		refreshRef := old.Refresh[req.ClientId]
		if refreshRef == nil || refreshRef.ID == "" {
			d.logger.Errorf("api: refresh token issued to client %q for user %q not found for deletion", req.ClientId, id.UserId)
			notFound = true
			return old, storage.ErrNotFound
		}

		refreshID = refreshRef.ID

		// Remove entry from Refresh list of the OfflineSession object.
		delete(old.Refresh, req.ClientId)

		return old, nil
	}

	if err := d.s.UpdateOfflineSessions(id.UserId, id.ConnId, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.RevokeRefreshResp{}, status.Error(codes.NotFound, "could not revoke refresh token, user id + client id match not found")
		}
		d.logger.Errorf("api: failed to update offline session object: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	if notFound {
		return &api.RevokeRefreshResp{}, status.Error(codes.NotFound, "could not revoke refresh token, user id + client id match not found")
	}

	// Delete the refresh token from the storage
	//
	// TODO(ericchiang): we don't have any good recourse if this call fails.
	// Consider garbage collection of refresh tokens with no associated ref.
	if err := d.s.DeleteRefresh(refreshID); err != nil {
		d.logger.Errorf("failed to delete refresh token: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.RevokeRefreshResp{}, nil
}

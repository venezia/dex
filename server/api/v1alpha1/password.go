package v1alpha1

import (
	"fmt"
	api "github.com/dexidp/dex/api/v1alpha1"
	"github.com/dexidp/dex/storage"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d dexAPI) CreatePassword(ctx context.Context, req *api.CreatePasswordReq) (*api.CreatePasswordResp, error) {
	if req.Password == nil {
		return nil, status.Error(codes.InvalidArgument, "no password supplied")
	}
	if req.Password.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "no user ID supplied")
	}
	if req.Password.Hash != nil {
		if err := checkCost(req.Password.Hash); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	} else {
		return nil, status.Error(codes.InvalidArgument, "no hash of password supplied")
	}

	p := storage.Password{
		Email:    req.Password.Email,
		Hash:     req.Password.Hash,
		Username: req.Password.Username,
		UserID:   req.Password.UserId,
	}
	if err := d.s.CreatePassword(p); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreatePasswordResp{}, status.Error(codes.AlreadyExists, "cannot create, username/password already exists")
		}
		d.logger.Errorf("api: failed to create password: %v", err)
		return nil, status.Errorf(codes.Internal, "create password: %v", err)
	}

	return &api.CreatePasswordResp{}, nil
}

func (d dexAPI) UpdatePassword(ctx context.Context, req *api.UpdatePasswordReq) (*api.UpdatePasswordResp, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "no email supplied")
	}
	if req.NewHash == nil && req.NewUsername == "" {
		return nil, status.Error(codes.InvalidArgument, "nothing to update")
	}

	if req.NewHash != nil {
		if err := checkCost(req.NewHash); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	updater := func(old storage.Password) (storage.Password, error) {
		if req.NewHash != nil {
			old.Hash = req.NewHash
		}

		if req.NewUsername != "" {
			old.Username = req.NewUsername
		}

		return old, nil
	}

	if err := d.s.UpdatePassword(req.Email, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdatePasswordResp{}, status.Error(codes.NotFound, "cannot update, email not found")
		}
		d.logger.Errorf("api: failed to update password: %v", err)
		return nil, status.Errorf(codes.Internal, "update password: %v", err)
	}

	return &api.UpdatePasswordResp{}, nil
}

func (d dexAPI) DeletePassword(ctx context.Context, req *api.DeletePasswordReq) (*api.DeletePasswordResp, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "no email supplied")
	}

	err := d.s.DeletePassword(req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeletePasswordResp{}, status.Error(codes.NotFound, "cannot delete, email not found")
		}
		d.logger.Errorf("api: failed to delete password: %v", err)
		return nil, status.Errorf(codes.Internal, "delete password: %v", err)
	}
	return &api.DeletePasswordResp{}, nil

}

func (d dexAPI) ListPasswords(ctx context.Context, req *api.ListPasswordReq) (*api.ListPasswordResp, error) {
	passwordList, err := d.s.ListPasswords()
	if err != nil {
		d.logger.Errorf("api: failed to list passwords: %v", err)
		return nil, status.Errorf(codes.Internal, "list passwords: %v", err)
	}

	var passwords []*api.Password
	for _, password := range passwordList {
		p := api.Password{
			Email:    password.Email,
			Username: password.Username,
			UserId:   password.UserID,
		}
		passwords = append(passwords, &p)
	}

	return &api.ListPasswordResp{
		Passwords: passwords,
	}, nil

}

func (d dexAPI) VerifyPassword(ctx context.Context, req *api.VerifyPasswordReq) (*api.VerifyPasswordResp, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "no email supplied")
	}

	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "no password to verify supplied")
	}

	password, err := d.s.GetPassword(req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.VerifyPasswordResp{}, status.Error(codes.NotFound, "cannot verify password, email cannot be found")
		}
		d.logger.Errorf("api: there was an error retrieving the password: %v", err)
		return nil, status.Errorf(codes.Internal, "verify password: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(password.Hash, []byte(req.Password)); err != nil {
		d.logger.Infof("api: password check failed: %v", err)
		return &api.VerifyPasswordResp{
			Verified: false,
		}, nil
	}
	return &api.VerifyPasswordResp{
		Verified: true,
	}, nil
}

// checkCost returns an error if the hash provided does not meet lower or upper
// bound cost requirements.
func checkCost(hash []byte) error {
	actual, err := bcrypt.Cost(hash)
	if err != nil {
		return fmt.Errorf("parsing bcrypt hash: %v", err)
	}
	if actual < bcrypt.DefaultCost {
		return fmt.Errorf("given hash cost = %d does not meet minimum cost requirement = %d", actual, bcrypt.DefaultCost)
	}
	if actual > upBoundCost {
		return fmt.Errorf("given hash cost = %d is above upper bound cost = %d, recommended cost = %d", actual, upBoundCost, recCost)
	}
	return nil
}

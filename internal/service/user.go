package service

import (
	"context"
	"os"
	"strconv"

	resourcev1 "bff/api/service/resource/v1"
	serviceuserv1 "bff/api/service/user/v1"
	pb "bff/api/user/v1"
	"bff/internal/data"
	"bff/internal/pkg/middleware/auth"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

type UserService struct {
	pb.UnimplementedUserServer

	userClient     serviceuserv1.UserServiceClient
	resourceClient resourcev1.ResourceServiceClient
	jwtKey         string
	log            *log.Helper
}

func NewUserService(clients *data.ServiceClients, logger log.Logger) *UserService {
	jwtKey := os.Getenv("BFF_JWT_KEY")
	if jwtKey == "" {
		jwtKey = "is_a_very_secret_key_and_it_is_this"
	}
	return &UserService{
		userClient:     clients.UserClient,
		resourceClient: clients.ResourceClient,
		jwtKey:         jwtKey,
		log:            log.NewHelper(logger),
	}
}

func (s *UserService) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterReply, error) {
	if s.userClient == nil {
		return nil, errors.InternalServer("NO_USER_CLIENT", "downstream user client is not initialized")
	}
	resp, err := s.userClient.Register(ctx, &serviceuserv1.RegisterReq{
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		return nil, err
	}
	return &pb.RegisterReply{UserId: resp.UserId}, nil
}

func (s *UserService) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginReply, error) {
	if s.userClient == nil {
		return nil, errors.InternalServer("NO_USER_CLIENT", "downstream user client is not initialized")
	}

	loginResp, err := s.userClient.Login(ctx, &serviceuserv1.LoginReq{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	if loginResp == nil || loginResp.Token == "" {
		return nil, errors.InternalServer("EMPTY_LOGIN_TOKEN", "downstream login returned empty token")
	}

	if _, err := auth.ParseToken(loginResp.Token, s.jwtKey); err != nil {
		s.log.Errorf("downstream login returned invalid token: %v", err)
		return nil, errors.InternalServer("INVALID_LOGIN_TOKEN", "downstream login returned invalid token")
	}
	return &pb.LoginReply{Token: loginResp.Token}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserReq) (*pb.GetUserReply, error) {
	if s.userClient == nil {
		return nil, errors.InternalServer("NO_USER_CLIENT", "downstream user client is not initialized")
	}
	if _, err := s.authenticateAndAuthorize(ctx, req.UserId); err != nil {
		return nil, err
	}

	resp, err := s.userClient.GetUser(ctx, &serviceuserv1.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, err
	}
	return &pb.GetUserReply{
		UserId:     resp.UserId,
		Email:      resp.Email,
		Nickname:   resp.Nickname,
		UserStatus: pb.Status(resp.UserStatus),
	}, nil
}

func (s *UserService) ListResources(ctx context.Context, req *pb.ListResourcesReq) (*pb.ListResourcesReply, error) {
	if s.resourceClient == nil {
		return nil, errors.InternalServer("NO_RESOURCE_CLIENT", "downstream resource client is not initialized")
	}
	claims, err := s.authenticateAndAuthorize(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	var userID string
	if claims.UserID != "" {
		userID = claims.UserID
	} else {
		return nil, errors.InternalServer("EMPTY_USER_ID", "downstream user id is empty")
	}

	resourceResp, err := s.resourceClient.ListResources(ctx, &resourcev1.ListResourcesReq{
		UserId:    &userID,
		Start:     req.Start,
		End:       req.End,
		Type:      req.Type,
		FieldMask: req.FieldMask,
	})
	if err != nil {
		return nil, err
	}

	out := &pb.ListResourcesReply{
		Resources: make([]*pb.Resource, 0, len(resourceResp.Resources)),
		Specs:     make(map[string]*pb.ResourceSpec, len(resourceResp.Specs)),
	}
	for _, r := range resourceResp.Resources {
		out.Resources = append(out.Resources, &pb.Resource{
			InstanceId: r.InstanceId,
			Name:       r.Name,
			UserId:     userID,
			Type:       r.Type,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
		})
	}
	for k, v := range resourceResp.Specs {
		out.Specs[k] = &pb.ResourceSpec{
			InstanceId:   v.InstanceId,
			CpuCores:     v.CpuCores,
			MemorySize:   v.MemorySize,
			Gpu:          v.Gpu,
			Image:        v.Image,
			CustomConfig: v.CustomConfig,
		}
	}
	return out, nil
}

func (s *UserService) authenticateAndAuthorize(ctx context.Context, requestedUserID string) (*auth.CustomClaims, error) {
	token, ok := auth.BearerTokenFromContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "missing bearer token")
	}
	claims, err := auth.ParseToken(token, s.jwtKey)
	if err != nil {
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid token")
	}

	if requestedUserID == "" {
		return claims, nil
	}
	if claims.UserID != "" && requestedUserID == claims.UserID {
		return claims, nil
	}
	if claims.Subject != "" && requestedUserID == claims.Subject {
		return claims, nil
	}
	if claims.ID != 0 && requestedUserID == strconv.FormatInt(claims.ID, 10) {
		return claims, nil
	}
	return nil, errors.Unauthorized("UNAUTHORIZED", "token user mismatch")
}

package data

import (
	"errors"

	resourcev1 "bff/api/service/resource/v1"
	userv1 "bff/api/service/user/v1"
	"bff/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceClients manages downstream service clients.
type ServiceClients struct {
	UserClient     userv1.UserServiceClient
	ResourceClient resourcev1.ResourceServiceClient
}

// NewServiceClients creates downstream service clients based on config.
func NewServiceClients(c *conf.Service, logger log.Logger) (*ServiceClients, func(), error) {
	if c == nil || c.User == nil || c.Resource == nil {
		return nil, nil, errors.New("service config is required")
	}
	if c.User.Addr == "" {
		return nil, nil, errors.New("service.user.addr is required")
	}
	if c.Resource.Addr == "" {
		return nil, nil, errors.New("service.resource.addr is required")
	}

	helper := log.NewHelper(logger)

	userConn, err := grpc.Dial(c.User.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		helper.Errorf("failed to dial user service: %v", err)
		return nil, nil, err
	}

	resourceConn, err := grpc.Dial(c.Resource.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = userConn.Close()
		helper.Errorf("failed to dial resource service: %v", err)
		return nil, nil, err
	}

	cleanup := func() {
		if err := resourceConn.Close(); err != nil {
			helper.Errorf("failed to close resource client conn: %v", err)
		}
		if err := userConn.Close(); err != nil {
			helper.Errorf("failed to close user client conn: %v", err)
		}
		helper.Info("closing the service clients")
	}

	return &ServiceClients{
		UserClient:     userv1.NewUserServiceClient(userConn),
		ResourceClient: resourcev1.NewResourceServiceClient(resourceConn),
	}, cleanup, nil
}

package data

import (
	"context"
	"crypto/tls"
	"errors"
	"os"
	"strings"

	resourcev1 "bff/api/service/resource/v1"
	userv1 "bff/api/service/user/v1"
	"bff/internal/conf"
	"bff/internal/pkg/grpcquic"

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

	userConn, err := dialServiceConn(context.Background(), c.User.Addr)
	if err != nil {
		helper.Errorf("failed to dial user service: %v", err)
		return nil, nil, err
	}

	resourceConn, err := dialServiceConn(context.Background(), c.Resource.Addr)
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

func dialServiceConn(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	if target, ok := quicTarget(addr); ok {
		tlsConf := quicTLSConfig()
		creds := grpcquic.NewCredentials(tlsConf)
		dialer := grpcquic.NewQuicDialer(tlsConf, nil)
		return grpc.DialContext(ctx, target,
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(creds),
		)
	}

	return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func quicTarget(addr string) (string, bool) {
	a := strings.TrimSpace(addr)
	for _, prefix := range []string{"quic://", "quic-grpc://", "grpc-quic://", "quic+grpc://"} {
		if strings.HasPrefix(a, prefix) {
			return strings.TrimPrefix(a, prefix), true
		}
	}
	return "", false
}

func quicTLSConfig() *tls.Config {
	alpn := strings.TrimSpace(os.Getenv("BFF_QUIC_ALPN"))
	if alpn == "" {
		alpn = "grpc-quic"
	}

	insecureSkipVerify := true
	if envFalse(os.Getenv("BFF_QUIC_INSECURE_SKIP_VERIFY")) {
		insecureSkipVerify = false
	}

	return &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		NextProtos:         []string{alpn},
	}
}

func envFalse(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	return s == "0" || s == "false" || s == "no" || s == "off"
}

package data

import (
	"bff/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewServiceClients, NewData)

// Data .
type Data struct {
	ServiceClients *ServiceClients
}

// NewData .
func NewData(c *conf.Data, serviceClients *ServiceClients, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{ServiceClients: serviceClients}, cleanup, nil
}

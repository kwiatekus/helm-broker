package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/coreos/etcd/integration"
	"github.com/coreos/pkg/capnslog"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/helm-broker/internal/storage"
	"github.com/kyma-project/helm-broker/internal/storage/driver/etcd"
	"github.com/kyma-project/helm-broker/internal/storage/driver/memory"
)

var allDrivers = map[storage.DriverType]func() storage.ConfigList{
	storage.DriverMemory: func() storage.ConfigList {
		return storage.ConfigList{storage.Config{
			Driver:  storage.DriverMemory,
			Provide: storage.ProviderConfigMap{storage.EntityAll: storage.ProviderConfig{}},
			Memory: memory.Config{
				// Ignored for now
				MaxKeys: 666,
			},
		}}
	},
	storage.DriverEtcd: func() storage.ConfigList {

		return storage.ConfigList{storage.Config{
			Driver:  storage.DriverEtcd,
			Provide: storage.ProviderConfigMap{storage.EntityAll: storage.ProviderConfig{}},
			Etcd:    etcd.Config{},
		}}
	},
}

func tRunDrivers(t *testing.T, tName string, f func(*testing.T, storage.Factory)) bool {
	result := true
	for dt, clGen := range allDrivers {
		cl := clGen()

		fT := func(t *testing.T) {
			if dt == storage.DriverEtcd {
				// silence logs for all coreos packages to silence etcd
				ft := capnslog.NewNilFormatter()

				// enable verbose logging
				//ft := capnslog.NewPrettyFormatter(os.Stdout, true)
				capnslog.SetFormatter(ft)

				cfg := integration.ClusterConfig{
					Size:              1,
					QuotaBackendBytes: 10 * 1024 * 1024,
					UseGRPC:           true,
				}

				clus := integration.NewClusterByConfig(t, &cfg)
				m := clus.Members[0]

				// lower cluster startup time
				m.BootstrapTimeout = time.Millisecond
				m.ElectionTicks = 2
				m.TickMs = 1
				m.ServerConfig.TickMs = 1

				clus.Launch(t)
				client, err := integration.NewClientV3(m)
				require.NoError(t, err)

				defer clus.Terminate(t)

				cl[0].Etcd.ForceClient = client
			}

			sf, err := storage.NewFactory(&cl)
			require.NoError(t, err)

			f(t, sf)
		}
		result = t.Run(fmt.Sprintf("%s/%s", dt, tName), fT) && result
	}

	return result
}

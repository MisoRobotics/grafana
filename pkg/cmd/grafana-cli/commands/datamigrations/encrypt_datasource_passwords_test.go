package datamigrations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/cmd/grafana-cli/commands/commandstest"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
)

func TestPasswordMigrationCommand(t *testing.T) {
	// setup datasources with password, basic_auth and none
	store := sqlstore.InitTestDB(t)
	err := store.WithDbSession(context.Background(), func(sess *sqlstore.DBSession) error {
		passwordMigration(t, sess, store)
		return nil
	})
	require.NoError(t, err)
}

func passwordMigration(t *testing.T, session *sqlstore.DBSession, sqlstore *sqlstore.SQLStore) {
	ds := []*datasources.DataSource{
		{Type: "influxdb", Name: "influxdb", Password: "foobar", Uid: "influx"},
		{Type: "graphite", Name: "graphite", BasicAuthPassword: "foobar", Uid: "graphite"},
		{Type: "prometheus", Name: "prometheus", Uid: "prom"},
		{Type: "elasticsearch", Name: "elasticsearch", Password: "pwd", Uid: "elastic"},
	}

	// set required default values
	for _, ds := range ds {
		ds.Created = time.Now()
		ds.Updated = time.Now()

		if ds.Name == "elasticsearch" {
			key, err := util.Encrypt([]byte("value"), setting.SecretKey)
			require.NoError(t, err)

			ds.SecureJsonData = map[string][]byte{"key": key}
		} else {
			ds.SecureJsonData = map[string][]byte{}
		}
	}

	_, err := session.Insert(&ds)
	require.NoError(t, err)

	// force secure_json_data to be null to verify that migration can handle that
	_, err = session.Exec("update data_source set secure_json_data = null where name = 'influxdb'")
	require.NoError(t, err)

	// run migration
	c, err := commandstest.NewCliContext(map[string]string{})
	require.Nil(t, err)
	err = EncryptDatasourcePasswords(c, sqlstore)
	require.NoError(t, err)

	// verify that no datasources still have password or basic_auth
	var dss []*datasources.DataSource
	err = session.SQL("select * from data_source").Find(&dss)
	require.NoError(t, err)
	assert.Equal(t, len(dss), 4)

	for _, ds := range dss {
		sj, err := DecryptSecureJsonData(ds)
		require.NoError(t, err)

		if ds.Name == "influxdb" {
			assert.Equal(t, ds.Password, "")
			v, exist := sj["password"]
			assert.True(t, exist)
			assert.Equal(t, v, "foobar", "expected password to be moved to securejson")
		}

		if ds.Name == "graphite" {
			assert.Equal(t, ds.BasicAuthPassword, "")
			v, exist := sj["basicAuthPassword"]
			assert.True(t, exist)
			assert.Equal(t, v, "foobar", "expected basic_auth_password to be moved to securejson")
		}

		if ds.Name == "prometheus" {
			assert.Equal(t, len(sj), 0)
		}

		if ds.Name == "elasticsearch" {
			assert.Equal(t, ds.Password, "")
			key, exist := sj["key"]
			assert.True(t, exist)
			password, exist := sj["password"]
			assert.True(t, exist)
			assert.Equal(t, password, "pwd", "expected password to be moved to securejson")
			assert.Equal(t, key, "value", "expected existing key to be kept intact in securejson")
		}
	}
}

func DecryptSecureJsonData(ds *datasources.DataSource) (map[string]string, error) {
	decrypted := make(map[string]string)
	for key, data := range ds.SecureJsonData {
		decryptedData, err := util.Decrypt(data, setting.SecretKey)
		if err != nil {
			return nil, err
		}

		decrypted[key] = string(decryptedData)
	}
	return decrypted, nil
}

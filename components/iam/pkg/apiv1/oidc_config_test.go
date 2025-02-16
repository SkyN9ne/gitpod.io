// Copyright (c) 2023 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package apiv1

import (
	"context"
	"testing"

	"github.com/gitpod-io/gitpod/common-go/baseserver"
	db "github.com/gitpod-io/gitpod/components/gitpod-db/go"
	"github.com/gitpod-io/gitpod/components/gitpod-db/go/dbtest"
	v1 "github.com/gitpod-io/gitpod/components/iam-api/go/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestOIDCClientConfig_Create(t *testing.T) {
	client, dbConn := setupOIDCClientConfigService(t)

	config := &v1.OIDCClientConfig{
		Oauth2Config: &v1.OAuth2Config{
			ClientId:              "some-client-id",
			ClientSecret:          "some-client-secret",
			AuthorizationEndpoint: "http://some-endpoint.here",
			ScopesSupported:       []string{"my-scope"},
		},
		OidcConfig: &v1.OIDCConfig{
			Issuer: "some-issuer",
		},
	}
	response, err := client.CreateClientConfig(context.Background(), &v1.CreateClientConfigRequest{
		Config: config,
	})
	require.NoError(t, err)
	require.Equal(t, codes.OK, status.Code(err))

	t.Cleanup(func() {
		dbtest.HardDeleteOIDCClientConfigs(t, response.Config.Id)
	})

	retrieved, err := db.GetOIDCClientConfig(context.Background(), dbConn, uuid.MustParse(response.Config.Id))
	require.NoError(t, err)

	decrypted, err := retrieved.Data.Decrypt(dbtest.CipherSet(t))
	require.NoError(t, err)
	require.Equal(t, toDBSpec(config.Oauth2Config, config.OidcConfig), decrypted)
}

func setupOIDCClientConfigService(t *testing.T) (v1.OIDCServiceClient, *gorm.DB) {
	t.Helper()

	dbConn := dbtest.ConnectForTests(t)
	cipher := dbtest.CipherSet(t)

	srv := baseserver.NewForTests(t, baseserver.WithGRPC(
		baseserver.MustUseRandomLocalAddress(t)),
	)

	svc := NewOIDCClientConfigService(dbConn, cipher)
	v1.RegisterOIDCServiceServer(srv.GRPC(), svc)

	t.Cleanup(func() {
		require.NoError(t, srv.Close())
	})

	go func(t *testing.T) {
		require.NoError(t, srv.ListenAndServe())
	}(t)

	conn, err := grpc.Dial(srv.GRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return v1.NewOIDCServiceClient(conn), dbConn
}

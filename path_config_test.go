package gitlab_test

import (
	"context"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gitlab "github.com/ilijamt/vault-plugin-secrets-gitlab"
)

func TestPathConfig(t *testing.T) {
	t.Run("initial config should be empty fail with backend not configured", func(t *testing.T) {
		ctx := getCtxGitlabClient(t)
		b, l, err := getBackend(ctx)
		require.NoError(t, err)
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.ReadOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Error(t, resp.Error())
		require.EqualValues(t, resp.Error(), gitlab.ErrBackendNotConfigured)
	})

	t.Run("deleting uninitialized config should fail with backend not configured", func(t *testing.T) {
		ctx := getCtxGitlabClient(t)
		b, l, err := getBackend(ctx)
		require.NoError(t, err)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Error(t, resp.Error())
		require.True(t, resp.IsError())
		require.EqualValues(t, resp.Error(), gitlab.ErrBackendNotConfigured)
	})

	t.Run("write, read, delete and read config", func(t *testing.T) {
		httpClient, url := getClient(t)
		ctx := gitlab.HttpClientNewContext(context.Background(), httpClient)

		b, l, events, err := getBackendWithEvents(ctx)
		require.NoError(t, err)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
			Data: map[string]any{
				"token":    "glpat-secret-random-token",
				"base_url": url,
				"type":     gitlab.TypeSelfManaged.String(),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NoError(t, resp.Error())

		resp, err = b.HandleRequest(ctx, &logical.Request{
			Operation: logical.ReadOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NoError(t, resp.Error())
		assert.NotEmpty(t, resp.Data["token_sha1_hash"])
		assert.NotEmpty(t, resp.Data["base_url"])
		require.Len(t, events.eventsProcessed, 1)

		resp, err = b.HandleRequest(ctx, &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
		})
		require.NoError(t, err)
		require.Nil(t, resp)
		require.Len(t, events.eventsProcessed, 2)

		resp, err = b.HandleRequest(ctx, &logical.Request{
			Operation: logical.ReadOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Error(t, resp.Error())

		events.expectEvents(t, []expectedEvent{
			{eventType: "gitlab/config-write"},
			{eventType: "gitlab/config-delete"},
		})
	})

	t.Run("invalid token", func(t *testing.T) {
		httpClient, url := getClient(t)
		ctx := gitlab.HttpClientNewContext(context.Background(), httpClient)

		b, l, events, err := getBackendWithEvents(ctx)
		require.NoError(t, err)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
			Data: map[string]any{
				"token":    "invalid-token",
				"base_url": url,
				"type":     gitlab.TypeSelfManaged.String(),
			},
		})

		require.Error(t, err)
		require.Nil(t, resp)

		events.expectEvents(t, []expectedEvent{})
	})

	t.Run("missing token from the request", func(t *testing.T) {
		ctx := getCtxGitlabClient(t)
		b, l, err := getBackend(ctx)
		require.NoError(t, err)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
			Data: map[string]any{},
		})

		require.Error(t, err)
		require.Nil(t, resp)

		var errorMap = countErrByName(err.(*multierror.Error))
		assert.EqualValues(t, 3, errorMap[gitlab.ErrFieldRequired.Error()])
		require.Len(t, errorMap, 1)
	})

	t.Run("patch a config with no storage", func(t *testing.T) {
		httpClient, url := getClient(t)
		ctx := gitlab.HttpClientNewContext(context.Background(), httpClient)

		b, _, err := getBackend(ctx)
		require.NoError(t, err)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.PatchOperation,
			Path:      gitlab.PathConfigStorage, Storage: nil,
			Data: map[string]any{
				"token":    "glpat-secret-random-token",
				"base_url": url,
				"type":     gitlab.TypeSelfManaged.String(),
			},
		})

		require.ErrorIs(t, err, gitlab.ErrNilValue)
		require.Nil(t, resp)
	})

	t.Run("patch a config no backend", func(t *testing.T) {
		httpClient, url := getClient(t)
		ctx := gitlab.HttpClientNewContext(context.Background(), httpClient)

		b, l, err := getBackend(ctx)
		require.NoError(t, err)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.PatchOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
			Data: map[string]any{
				"token":    "glpat-secret-random-token",
				"base_url": url,
				"type":     gitlab.TypeSelfManaged.String(),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.EqualValues(t, resp.Error(), gitlab.ErrBackendNotConfigured)
	})

	t.Run("patch a config", func(t *testing.T) {
		httpClient, url := getClient(t)
		ctx := gitlab.HttpClientNewContext(context.Background(), httpClient)

		b, l, events, err := getBackendWithEvents(ctx)
		require.NoError(t, err)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
			Data: map[string]any{
				"token":    "glpat-secret-random-token",
				"base_url": url,
				"type":     gitlab.TypeSelfManaged.String(),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NoError(t, resp.Error())

		resp, err = b.HandleRequest(ctx, &logical.Request{
			Operation: logical.ReadOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NoError(t, resp.Error())
		tokenOriginalSha1Hash := resp.Data["token_sha1_hash"].(string)
		require.NotEmpty(t, tokenOriginalSha1Hash)
		require.Equal(t, gitlab.TypeSelfManaged.String(), resp.Data["type"])
		require.NotNil(t, b.GetClient().GitlabClient())

		resp, err = b.HandleRequest(ctx, &logical.Request{
			Operation: logical.PatchOperation,
			Path:      gitlab.PathConfigStorage, Storage: l,
			Data: map[string]interface{}{
				"type":  gitlab.TypeSaaS.String(),
				"token": "glpat-secret-admin-token",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NoError(t, resp.Error())
		tokenNewSha1Hash := resp.Data["token_sha1_hash"].(string)
		require.NotEmpty(t, tokenNewSha1Hash)
		require.NotEqual(t, tokenOriginalSha1Hash, tokenNewSha1Hash)

		require.Equal(t, gitlab.TypeSaaS.String(), resp.Data["type"])
		require.NotNil(t, b.GetClient().GitlabClient())

		events.expectEvents(t, []expectedEvent{
			{eventType: "gitlab/config-write"},
			{eventType: "gitlab/config-patch"},
		})

	})

}

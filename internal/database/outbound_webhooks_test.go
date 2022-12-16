package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/sourcegraph/log/logtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/encryption"
	et "github.com/sourcegraph/sourcegraph/internal/encryption/testing"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestOutboundWebhooks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := logtest.Scoped(t)

	for _, encrypted := range []bool{true, false} {
		t.Run(fmt.Sprintf("encrypted=%t", encrypted), func(t *testing.T) {
			db := NewDB(logger, dbtest.NewDB(logger, t))
			user, err := db.Users().Create(ctx, NewUser{
				Username: "test",
			})
			require.NoError(t, err)

			var key encryption.Key
			if encrypted {
				key = et.ByteaTestKey{}
			}
			store := db.OutboundWebhooks(key)

			var createdWebhook *types.OutboundWebhook

			t.Run("Create", func(t *testing.T) {
				t.Run("no event types", func(t *testing.T) {
					webhook := newTestWebhook(t, user)
					err := store.Create(ctx, webhook)
					assert.ErrorIs(t, err, errOutboundWebhookHasNoEventTypes)
				})

				t.Run("encryption failure", func(t *testing.T) {
					store := db.OutboundWebhooks(&et.BadKey{})
					webhook := newTestWebhook(t, user)
					err := store.Create(ctx, webhook)
					assert.Error(t, err)
				})

				t.Run("success", func(t *testing.T) {
					createdWebhook = newTestWebhook(
						t, user,
						ScopedEventType{EventType: "foo"},
						ScopedEventType{EventType: "bar"},
						ScopedEventType{EventType: "quux", Scope: stringPtr("123")},
					)
					err := store.Create(ctx, createdWebhook)
					assert.NoError(t, err)
					assert.NotZero(t, createdWebhook.ID)
					assert.NotZero(t, createdWebhook.CreatedAt)
					assert.NotZero(t, createdWebhook.UpdatedAt)
					for _, eventType := range createdWebhook.EventTypes {
						assert.NotZero(t, eventType.ID)
						assert.Equal(t, createdWebhook.ID, eventType.OutboundWebhookID)
					}
					if encrypted {
						assertEncryptedFieldsAreEncrypted(t, ctx, store, createdWebhook)
					}
				})
			})

			t.Run("GetByID", func(t *testing.T) {
				t.Run("not found", func(t *testing.T) {
					webhook, err := store.GetByID(ctx, 0)
					assert.True(t, errcode.IsNotFound(err))
					assert.Nil(t, webhook)
				})

				t.Run("found", func(t *testing.T) {
					webhook, err := store.GetByID(ctx, createdWebhook.ID)
					assert.NoError(t, err)
					assertEqualWebhooks(t, ctx, createdWebhook, webhook)
				})
			})

			t.Run("List/Count", func(t *testing.T) {
				// OK, let's create a few more webhooks now for testing
				// purposes.
				newSavedTestWebhook := func(t *testing.T, user *types.User, scopes ...ScopedEventType) *types.OutboundWebhook {
					t.Helper()
					webhook := newTestWebhook(t, user, scopes...)
					require.NoError(t, store.Create(ctx, webhook))
					return webhook
				}

				fooOnlyWebhook := newSavedTestWebhook(t, user, ScopedEventType{EventType: "foo"})
				barOnlyWebhook := newSavedTestWebhook(t, user, ScopedEventType{EventType: "bar"})
				quuxWithSameScopeWebhook := newSavedTestWebhook(
					t, user,
					ScopedEventType{EventType: "quux", Scope: stringPtr("123")},
				)
				quuxWithDifferentScopeWebhook := newSavedTestWebhook(
					t, user,
					ScopedEventType{EventType: "quux", Scope: stringPtr("456")},
				)

				allWebhooks := []*types.OutboundWebhook{
					createdWebhook,
					fooOnlyWebhook,
					barOnlyWebhook,
					quuxWithSameScopeWebhook,
					quuxWithDifferentScopeWebhook,
				}

				t.Run("unpaginated", func(t *testing.T) {
					for name, tc := range map[string]struct {
						opts OutboundWebhookListOpts
						want []*types.OutboundWebhook
					}{
						"no matches based on event type": {
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									EventTypes: []string{"not found"},
								},
							},
							want: []*types.OutboundWebhook{},
						},
						"scoped, missing type": {
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									ScopedEventTypes: []ScopedEventType{
										{EventType: "not found"},
									},
								},
							},
							want: []*types.OutboundWebhook{},
						},
						"scoped, no scopes in type": {
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									ScopedEventTypes: []ScopedEventType{
										{EventType: "foo", Scope: stringPtr("bar")},
									},
								},
							},
							want: []*types.OutboundWebhook{},
						},
						"scoped, missing scope in type": {
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									ScopedEventTypes: []ScopedEventType{
										{EventType: "quux", Scope: stringPtr("789")},
									},
								},
							},
							want: []*types.OutboundWebhook{},
						},
						"all": {
							opts: OutboundWebhookListOpts{},
							want: allWebhooks,
						},
						"unscoped": {
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									EventTypes: []string{"foo", "bar"},
								},
							},
							want: []*types.OutboundWebhook{
								createdWebhook, fooOnlyWebhook, barOnlyWebhook,
							},
						},
						"scoped with null scopes": {
							// This should return the foos, but no quuxs, since
							// they have scopes attached.
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									ScopedEventTypes: []ScopedEventType{
										{EventType: "foo"},
										{EventType: "quux"},
									},
								},
							},
							want: []*types.OutboundWebhook{
								createdWebhook, fooOnlyWebhook,
							},
						},
						"scoped with non-null scopes": {
							// This should return the quuxs, but no foos, since
							// the foos don't have scopes.
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									ScopedEventTypes: []ScopedEventType{
										{EventType: "foo", Scope: stringPtr("no match")},
										{EventType: "quux", Scope: stringPtr("123")},
										{EventType: "quux", Scope: stringPtr("456")},
										{EventType: "quux", Scope: stringPtr("789")},
									},
								},
							},
							want: []*types.OutboundWebhook{
								createdWebhook,
								quuxWithSameScopeWebhook,
								quuxWithDifferentScopeWebhook,
							},
						},
						"scoped with only one scope": {
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									ScopedEventTypes: []ScopedEventType{
										{EventType: "quux", Scope: stringPtr("123")},
									},
								},
							},
							want: []*types.OutboundWebhook{
								createdWebhook,
								quuxWithSameScopeWebhook,
							},
						},
						"mixed unscoped and scoped": {
							opts: OutboundWebhookListOpts{
								OutboundWebhookCountOpts: OutboundWebhookCountOpts{
									EventTypes: []string{"bar"},
									ScopedEventTypes: []ScopedEventType{
										{EventType: "quux", Scope: stringPtr("123")},
									},
								},
							},
							want: []*types.OutboundWebhook{
								createdWebhook,
								barOnlyWebhook,
								quuxWithSameScopeWebhook,
							},
						},
					} {
						t.Run(name, func(t *testing.T) {
							have, err := store.List(ctx, tc.opts)
							assert.NoError(t, err)
							assertEqualWebhookSlices(t, ctx, tc.want, have)

							count, err := store.Count(ctx, tc.opts.OutboundWebhookCountOpts)
							assert.NoError(t, err)
							assert.EqualValues(t, len(tc.want), count)
						})
					}
				})

				t.Run("pagination", func(t *testing.T) {
					// We won't rehash all the unpaginated tests above, since
					// that was really exercising the filtering; instead, we'll
					// just ensure all results are paginated as we expect.
					for i, want := range [][]*types.OutboundWebhook{
						{createdWebhook, fooOnlyWebhook},
						{barOnlyWebhook, quuxWithSameScopeWebhook},
						{quuxWithDifferentScopeWebhook},
						{},
					} {
						t.Run(fmt.Sprintf("page %d", i+1), func(t *testing.T) {
							have, err := store.List(ctx, OutboundWebhookListOpts{
								LimitOffset: &LimitOffset{
									Offset: i * 2,
									Limit:  2,
								},
							})
							assert.NoError(t, err)
							assertEqualWebhookSlices(t, ctx, want, have)
						})
					}
				})
			})

			t.Run("Update", func(t *testing.T) {
				t.Run("fail due to missing event types", func(t *testing.T) {
					createdWebhook.EventTypes = []types.OutboundWebhookEventType{}
					err := store.Update(ctx, createdWebhook)
					assert.ErrorIs(t, err, errOutboundWebhookHasNoEventTypes)
				})

				t.Run("replace all event types", func(t *testing.T) {
					createdWebhook.EventTypes = []types.OutboundWebhookEventType{
						{EventType: "new"},
					}
					err := store.Update(ctx, createdWebhook)
					assert.NoError(t, err)

					have, err := store.GetByID(ctx, createdWebhook.ID)
					require.NoError(t, err)
					assertEqualEventTypes(t, have.ID, createdWebhook.EventTypes, have.EventTypes)
				})

				t.Run("append to the current event types", func(t *testing.T) {
					createdWebhook.EventTypes = append(
						createdWebhook.EventTypes,
						types.OutboundWebhookEventType{EventType: "newer", Scope: stringPtr("abc")},
					)
					err := store.Update(ctx, createdWebhook)
					assert.NoError(t, err)

					have, err := store.GetByID(ctx, createdWebhook.ID)
					require.NoError(t, err)
					assertEqualEventTypes(t, have.ID, createdWebhook.EventTypes, have.EventTypes)
				})

				t.Run("update other fields", func(t *testing.T) {
					createdWebhook.URL.Set("https://a.new.value")
					createdWebhook.Secret.Set("a whole new secret")
					err := store.Update(ctx, createdWebhook)
					assert.NoError(t, err)

					have, err := store.GetByID(ctx, createdWebhook.ID)
					require.NoError(t, err)
					assertEqualWebhooks(t, ctx, createdWebhook, have)

					if encrypted {
						assertEncryptedFieldsAreEncrypted(t, ctx, store, have)
					}
				})
			})

			t.Run("Delete", func(t *testing.T) {
				err := store.Delete(ctx, createdWebhook.ID)
				assert.NoError(t, err)

				_, err = store.GetByID(ctx, createdWebhook.ID)
				assert.True(t, errcode.IsNotFound(err))
			})
		})
	}
}

func assertEncryptedFieldsAreEncrypted(t *testing.T, ctx context.Context, store basestore.ShareableStore, webhook *types.OutboundWebhook) {
	t.Helper()

	url, err := webhook.URL.Decrypt(ctx)
	require.NoError(t, err)

	secret, err := webhook.Secret.Decrypt(ctx)
	require.NoError(t, err)

	row := store.Handle().QueryRowContext(
		ctx,
		"SELECT url, secret, encryption_key_id FROM outbound_webhooks WHERE id = $1",
		webhook.ID,
	)
	var (
		dbURL    string
		dbSecret string
		keyID    string
	)
	err = row.Scan(&dbURL, &dbSecret, &dbutil.NullString{S: &keyID})
	assert.NoError(t, err)
	assert.NotEmpty(t, keyID)
	assert.NotEqual(t, dbURL, url)
	assert.NotEqual(t, dbSecret, secret)
}

func assertEqualEventTypes(t *testing.T, webhookID int64, want, have []types.OutboundWebhookEventType) {
	t.Helper()

	type unidentifiedEventType struct {
		outboundWebhookID int64
		eventType         string
		scope             *string
	}

	comparableEventTypes := func(eventTypes []types.OutboundWebhookEventType) []unidentifiedEventType {
		t.Helper()

		comp := make([]unidentifiedEventType, len(eventTypes))
		for i, eventType := range eventTypes {
			assert.Equal(t, webhookID, eventType.OutboundWebhookID)
			comp[i] = unidentifiedEventType{
				outboundWebhookID: eventType.OutboundWebhookID,
				eventType:         eventType.EventType,
				scope:             eventType.Scope,
			}
		}

		return comp
	}

	assert.ElementsMatch(t, comparableEventTypes(want), comparableEventTypes(have))
}

func assertEqualWebhooks(t *testing.T, ctx context.Context, want, have *types.OutboundWebhook) {
	t.Helper()

	valueOf := func(e *encryption.Encryptable) string {
		t.Helper()

		value, err := e.Decrypt(ctx)
		require.NoError(t, err)
		return value
	}

	// We need this helper because the encryptable values need to be decrypted
	// before it makes sense to compare them, and because event type IDs are (in
	// practice) ephemeral, so we only really care about the actual values.
	assert.Equal(t, want.ID, have.ID)
	assert.Equal(t, want.CreatedBy, have.CreatedBy)
	assert.Equal(t, want.CreatedAt, have.CreatedAt)
	assert.Equal(t, want.UpdatedBy, have.UpdatedBy)
	assert.Equal(t, want.UpdatedAt, have.UpdatedAt)
	assert.Equal(t, valueOf(want.URL), valueOf(have.URL))
	assert.Equal(t, valueOf(want.Secret), valueOf(have.Secret))
	assertEqualEventTypes(t, want.ID, want.EventTypes, have.EventTypes)
}

func assertEqualWebhookSlices(t *testing.T, ctx context.Context, want, have []*types.OutboundWebhook) {
	assert.Equal(t, len(want), len(have))
	for i := range want {
		assertEqualWebhooks(t, ctx, want[i], have[i])
	}
}

func newTestWebhook(t *testing.T, user *types.User, scopes ...ScopedEventType) *types.OutboundWebhook {
	t.Helper()

	webhook := &types.OutboundWebhook{
		CreatedBy:  user.ID,
		UpdatedBy:  user.ID,
		URL:        encryption.NewUnencrypted("https://example.com/"),
		Secret:     encryption.NewUnencrypted("super secret"),
		EventTypes: make([]types.OutboundWebhookEventType, 0, len(scopes)),
	}

	for _, scope := range scopes {
		webhook.EventTypes = append(webhook.EventTypes, types.OutboundWebhookEventType{
			EventType: scope.EventType,
			Scope:     scope.Scope,
		})
	}

	return webhook
}

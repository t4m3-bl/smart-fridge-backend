//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "smart-fridge-backend/internal/repository/postgres"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func productData() *domain.ProductData {
	return &domain.ProductData{
		Name:      "Молоко",
		ExpiresAt: day("2026-11-10"),
	}
}

func fullProductData(categoryID domain.CategoryID) *domain.ProductData {
	return &domain.ProductData{
		Name:                  "Кефир",
		CategoryID:            &categoryID,
		Description:           new("1%"),
		Quantity:              new("900 мл"),
		ImageLink:             new("https://example.com/kefir.jpg"),
		ManufacturedAt:        new(day("2026-11-01")),
		ExpiresAt:             day("2026-11-15"),
		OpenedAt:              new(day("2026-11-03")),
		ShelfLifeAfterOpening: new(int32(3)),
	}
}

func mutation(typ domain.MutationType, productID domain.ProductID, data *domain.ProductData) domain.Mutation {
	return domain.Mutation{
		ID:        domain.MutationID(newUUID()),
		Type:      typ,
		ProductID: productID,
		Data:      data,
	}
}

func newProductID() domain.ProductID {
	return domain.ProductID(newUUID())
}

func seedCategory(t *testing.T, userID domain.UserID) domain.CategoryID {
	t.Helper()

	id := newUUID()
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO categories (id, user_id, name) VALUES ($1, $2, $3)`,
		id, uuid.UUID(userID), "category-"+uuid.NewString(),
	)
	require.NoError(t, err)

	return domain.CategoryID(id)
}

func seedProduct(t *testing.T, repo *db.SyncRepository, userID domain.UserID) domain.ProductID {
	t.Helper()

	id := newProductID()
	res, err := repo.Apply(context.Background(), userID, mutation(domain.MutationCreateProduct, id, productData()))
	require.NoError(t, err)
	require.Equal(t, domain.MutationApplied, res.Status)

	return id
}

type productRow struct {
	Name                  string
	CategoryID            *uuid.UUID
	Description           *string
	Quantity              *string
	ImageLink             *string
	ManufacturedAt        *time.Time
	ExpiresAt             time.Time
	OpenedAt              *time.Time
	ShelfLifeAfterOpening *int32
	EffectiveExpiresAt    time.Time
}

func findProduct(t *testing.T, id domain.ProductID) *productRow {
	t.Helper()

	var row productRow
	err := testPool.QueryRow(context.Background(),
		`SELECT name, category_id, description, quantity, image_link, manufactured_at,
		        expires_at, opened_at, shelf_life_after_opening, effective_expires_at
		 FROM products WHERE id = $1`,
		uuid.UUID(id),
	).Scan(&row.Name, &row.CategoryID, &row.Description, &row.Quantity, &row.ImageLink, &row.ManufacturedAt,
		&row.ExpiresAt, &row.OpenedAt, &row.ShelfLifeAfterOpening, &row.EffectiveExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	require.NoError(t, err)

	return &row
}

func countMutations(t *testing.T, userID domain.UserID) int {
	t.Helper()

	var n int
	require.NoError(t, testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM mutations WHERE user_id = $1`, uuid.UUID(userID),
	).Scan(&n))

	return n
}

func TestSyncRepository_Apply(t *testing.T) {
	t.Parallel()

	repo := db.NewSyncRepository(testPool)

	tests := []struct {
		name       string
		prepare    func(t *testing.T, userID domain.UserID) domain.Mutation
		wantStatus domain.MutationStatus
		wantReason domain.RejectReason
		check      func(t *testing.T, m domain.Mutation)
	}{
		{
			name: "create product with minimal data",
			prepare: func(_ *testing.T, _ domain.UserID) domain.Mutation {
				return mutation(domain.MutationCreateProduct, newProductID(), productData())
			},
			wantStatus: domain.MutationApplied,
			check: func(t *testing.T, m domain.Mutation) {
				t.Helper()

				got := findProduct(t, m.ProductID)
				require.NotNil(t, got)
				assert.Equal(t, "Молоко", got.Name)
				assert.Equal(t, day("2026-11-10"), got.ExpiresAt)
				assert.Nil(t, got.CategoryID)
			},
		},
		{
			name: "create product with all fields",
			prepare: func(t *testing.T, userID domain.UserID) domain.Mutation {
				t.Helper()
				return mutation(domain.MutationCreateProduct, newProductID(), fullProductData(seedCategory(t, userID)))
			},
			wantStatus: domain.MutationApplied,
			check: func(t *testing.T, m domain.Mutation) {
				t.Helper()

				got := findProduct(t, m.ProductID)
				require.NotNil(t, got)
				require.NotNil(t, got.CategoryID)
				assert.Equal(t, uuid.UUID(*m.Data.CategoryID), *got.CategoryID)
				assert.Equal(t, m.Data.Quantity, got.Quantity)
				assert.Equal(t, m.Data.ShelfLifeAfterOpening, got.ShelfLifeAfterOpening)
				assert.Equal(t, day("2026-11-06"), got.EffectiveExpiresAt)
			},
		},
		{
			name: "create product with existing id",
			prepare: func(t *testing.T, userID domain.UserID) domain.Mutation {
				t.Helper()
				return mutation(domain.MutationCreateProduct, seedProduct(t, repo, userID), productData())
			},
			wantStatus: domain.MutationRejected,
			wantReason: domain.RejectAlreadyExists,
		},
		{
			name: "create product with unknown category",
			prepare: func(_ *testing.T, _ domain.UserID) domain.Mutation {
				data := productData()
				data.CategoryID = new(domain.CategoryID(newUUID()))
				return mutation(domain.MutationCreateProduct, newProductID(), data)
			},
			wantStatus: domain.MutationRejected,
			wantReason: domain.RejectNotFound,
		},
		{
			name: "create product with category of another user",
			prepare: func(t *testing.T, _ domain.UserID) domain.Mutation {
				t.Helper()
				data := productData()
				data.CategoryID = new(seedCategory(t, seedUser(t).ID))
				return mutation(domain.MutationCreateProduct, newProductID(), data)
			},
			wantStatus: domain.MutationRejected,
			wantReason: domain.RejectNotFound,
		},
		{
			name: "update replaces all fields",
			prepare: func(t *testing.T, userID domain.UserID) domain.Mutation {
				t.Helper()

				id := newProductID()
				_, err := repo.Apply(context.Background(), userID,
					mutation(domain.MutationCreateProduct, id, fullProductData(seedCategory(t, userID))))
				require.NoError(t, err)

				data := productData()
				data.Name = "Ряженка"
				return mutation(domain.MutationUpdateProduct, id, data)
			},
			wantStatus: domain.MutationApplied,
			check: func(t *testing.T, m domain.Mutation) {
				t.Helper()

				got := findProduct(t, m.ProductID)
				require.NotNil(t, got)
				assert.Equal(t, "Ряженка", got.Name)
				assert.Nil(t, got.CategoryID)
				assert.Nil(t, got.Quantity)
				assert.Nil(t, got.OpenedAt)
				assert.Equal(t, day("2026-11-10"), got.EffectiveExpiresAt)
			},
		},
		{
			name: "update unknown product",
			prepare: func(_ *testing.T, _ domain.UserID) domain.Mutation {
				return mutation(domain.MutationUpdateProduct, newProductID(), productData())
			},
			wantStatus: domain.MutationRejected,
			wantReason: domain.RejectNotFound,
		},
		{
			name: "update product of another user",
			prepare: func(t *testing.T, _ domain.UserID) domain.Mutation {
				t.Helper()
				return mutation(domain.MutationUpdateProduct, seedProduct(t, repo, seedUser(t).ID), productData())
			},
			wantStatus: domain.MutationRejected,
			wantReason: domain.RejectNotFound,
		},
		{
			name: "delete product",
			prepare: func(t *testing.T, userID domain.UserID) domain.Mutation {
				t.Helper()
				return mutation(domain.MutationDeleteProduct, seedProduct(t, repo, userID), nil)
			},
			wantStatus: domain.MutationApplied,
			check: func(t *testing.T, m domain.Mutation) {
				t.Helper()
				assert.Nil(t, findProduct(t, m.ProductID))
			},
		},
		{
			name: "delete already deleted product",
			prepare: func(t *testing.T, userID domain.UserID) domain.Mutation {
				t.Helper()

				id := seedProduct(t, repo, userID)
				_, err := repo.Apply(context.Background(), userID, mutation(domain.MutationDeleteProduct, id, nil))
				require.NoError(t, err)

				return mutation(domain.MutationDeleteProduct, id, nil)
			},
			wantStatus: domain.MutationRejected,
			wantReason: domain.RejectNotFound,
		},
		{
			name: "update deleted product",
			prepare: func(t *testing.T, userID domain.UserID) domain.Mutation {
				t.Helper()

				id := seedProduct(t, repo, userID)
				_, err := repo.Apply(context.Background(), userID, mutation(domain.MutationDeleteProduct, id, nil))
				require.NoError(t, err)

				return mutation(domain.MutationUpdateProduct, id, productData())
			},
			wantStatus: domain.MutationRejected,
			wantReason: domain.RejectNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID := seedUser(t).ID
			m := tt.prepare(t, userID)

			got, err := repo.Apply(context.Background(), userID, m)

			require.NoError(t, err)
			assert.Equal(t, m.ID, got.MutationID)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantReason, got.Reason)
			assert.Positive(t, got.Seq)
			if tt.wantStatus == domain.MutationRejected {
				assert.NotEmpty(t, got.Message)
			}
			if tt.check != nil {
				tt.check(t, m)
			}
		})
	}
}

func TestSyncRepository_Apply_IsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewSyncRepository(testPool)
	userID := seedUser(t).ID
	m := mutation(domain.MutationCreateProduct, newProductID(), productData())

	first, err := repo.Apply(ctx, userID, m)
	require.NoError(t, err)

	second, err := repo.Apply(ctx, userID, m)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, domain.MutationApplied, second.Status)
	assert.Equal(t, 1, countMutations(t, userID))
}

func TestSyncRepository_Apply_SameMutationIDForDifferentUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewSyncRepository(testPool)
	m := mutation(domain.MutationCreateProduct, newProductID(), productData())
	other := m
	other.ProductID = newProductID()

	first, err := repo.Apply(ctx, seedUser(t).ID, m)
	require.NoError(t, err)
	second, err := repo.Apply(ctx, seedUser(t).ID, other)
	require.NoError(t, err)

	assert.Equal(t, domain.MutationApplied, first.Status)
	assert.Equal(t, domain.MutationApplied, second.Status)
	assert.NotEqual(t, first.Seq, second.Seq)
}

func TestSyncRepository_Apply_UnknownUser(t *testing.T) {
	t.Parallel()

	repo := db.NewSyncRepository(testPool)

	_, err := repo.Apply(context.Background(), domain.UserID(newUUID()),
		mutation(domain.MutationCreateProduct, newProductID(), productData()))

	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSyncRepository_Apply_ConcurrentPushesOfOneUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewSyncRepository(testPool)
	userID := seedUser(t).ID

	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Go(func() {
			res, err := repo.Apply(ctx, userID, mutation(domain.MutationCreateProduct, newProductID(), productData()))
			if err == nil && res.Status != domain.MutationApplied {
				err = errors.New("mutation was not applied: " + res.Message)
			}
			errs <- err
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	got, err := repo.ListApplied(ctx, userID, 0, n+1)
	require.NoError(t, err)
	assert.Len(t, got, n)
}

func TestSyncRepository_Reject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewSyncRepository(testPool)
	userID := seedUser(t).ID
	m := mutation(domain.MutationCreateProduct, newProductID(), &domain.ProductData{})

	got, err := repo.Reject(ctx, userID, m, domain.RejectValidation, "name is empty")
	require.NoError(t, err)

	assert.Equal(t, domain.MutationResult{
		MutationID: m.ID,
		Status:     domain.MutationRejected,
		Seq:        got.Seq,
		Reason:     domain.RejectValidation,
		Message:    "name is empty",
	}, got)
	assert.Nil(t, findProduct(t, m.ProductID))

	m.Data = productData()
	retried, err := repo.Apply(ctx, userID, m)
	require.NoError(t, err)
	assert.Equal(t, got, retried)
	assert.Nil(t, findProduct(t, m.ProductID))
}

func TestSyncRepository_ListApplied(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewSyncRepository(testPool)
	userID := seedUser(t).ID
	otherID := seedUser(t).ID

	productID := newProductID()
	full := fullProductData(seedCategory(t, userID))
	created := mutation(domain.MutationCreateProduct, productID, full)
	rejected := mutation(domain.MutationUpdateProduct, newProductID(), productData())
	deleted := mutation(domain.MutationDeleteProduct, productID, nil)

	batch := []domain.Mutation{created, rejected, deleted}
	seqs := make([]int64, 0, len(batch))
	for _, m := range batch {
		res, err := repo.Apply(ctx, userID, m)
		require.NoError(t, err)
		seqs = append(seqs, res.Seq)
	}
	seedProduct(t, repo, otherID)

	t.Run("returns only applied mutations of the user in order", func(t *testing.T) {
		t.Parallel()

		got, err := repo.ListApplied(ctx, userID, 0, 10)
		require.NoError(t, err)
		require.Len(t, got, 2)

		assert.Equal(t, seqs[0], got[0].Seq)
		assert.Equal(t, created.ID, got[0].ID)
		assert.Equal(t, domain.MutationCreateProduct, got[0].Type)
		assert.Equal(t, productID, got[0].ProductID)
		assert.Equal(t, full, got[0].Data)
		assert.False(t, got[0].CreatedAt.IsZero())

		assert.Equal(t, seqs[2], got[1].Seq)
		assert.Equal(t, domain.MutationDeleteProduct, got[1].Type)
		assert.Nil(t, got[1].Data)
	})

	t.Run("since excludes earlier mutations", func(t *testing.T) {
		t.Parallel()

		got, err := repo.ListApplied(ctx, userID, seqs[0], 10)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, deleted.ID, got[0].ID)
	})

	t.Run("limit", func(t *testing.T) {
		t.Parallel()

		got, err := repo.ListApplied(ctx, userID, 0, 1)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, created.ID, got[0].ID)
	})

	t.Run("cursor after the last mutation", func(t *testing.T) {
		t.Parallel()

		got, err := repo.ListApplied(ctx, userID, seqs[2], 10)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestCategoryDeletion_ClearsProductCategory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewSyncRepository(testPool)
	userID := seedUser(t).ID
	categoryID := seedCategory(t, userID)
	productID := newProductID()

	_, err := repo.Apply(ctx, userID, mutation(domain.MutationCreateProduct, productID, fullProductData(categoryID)))
	require.NoError(t, err)

	_, err = testPool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, uuid.UUID(categoryID))
	require.NoError(t, err)

	got := findProduct(t, productID)
	require.NotNil(t, got)
	assert.Nil(t, got.CategoryID)
}

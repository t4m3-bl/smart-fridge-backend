package postgres

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
	pgDataExceptionClass  = "22"
)

type SyncRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewSyncRepository(pool *pgxpool.Pool) *SyncRepository {
	return &SyncRepository{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *SyncRepository) Apply(ctx context.Context, userID domain.UserID, m domain.Mutation) (domain.MutationResult, error) {
	return r.inUserTx(ctx, userID, m, func(qtx *sqlc.Queries, tx pgx.Tx) (domain.MutationResult, error) {
		applyErr := r.applyInSavepoint(ctx, tx, userID, m)
		if applyErr == nil {
			return r.save(ctx, qtx, userID, m, domain.MutationResult{Status: domain.MutationApplied})
		}

		reason, ok := domain.RejectReasonOf(applyErr)
		if !ok {
			return domain.MutationResult{}, applyErr
		}

		return r.save(ctx, qtx, userID, m, domain.MutationResult{
			Status:  domain.MutationRejected,
			Reason:  reason,
			Message: applyErr.Error(),
		})
	})
}

func (r *SyncRepository) Reject(
	ctx context.Context,
	userID domain.UserID,
	m domain.Mutation,
	reason domain.RejectReason,
	message string,
) (domain.MutationResult, error) {
	return r.inUserTx(ctx, userID, m, func(qtx *sqlc.Queries, _ pgx.Tx) (domain.MutationResult, error) {
		return r.save(ctx, qtx, userID, m, domain.MutationResult{
			Status:  domain.MutationRejected,
			Reason:  reason,
			Message: message,
		})
	})
}

func (r *SyncRepository) ListApplied(ctx context.Context, userID domain.UserID, since int64, limit int32) ([]domain.AppliedMutation, error) {
	rows, err := r.q.ListAppliedMutations(ctx, sqlc.ListAppliedMutationsParams{
		UserID:  uuid.UUID(userID),
		Since:   since,
		MaxRows: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list applied mutations: %w", err)
	}

	result := make([]domain.AppliedMutation, 0, len(rows))
	for _, row := range rows {
		data, err := decodePayload(row.Payload)
		if err != nil {
			return nil, fmt.Errorf("list applied mutations: seq %d: %w", row.Seq, err)
		}

		result = append(result, domain.AppliedMutation{
			Seq:       row.Seq,
			ID:        domain.MutationID(row.ID),
			Type:      domain.MutationType(row.Type),
			ProductID: domain.ProductID(row.ProductID),
			Data:      data,
			CreatedAt: row.CreatedAt.UTC(),
		})
	}

	return result, nil
}

func (r *SyncRepository) inUserTx(
	ctx context.Context,
	userID domain.UserID,
	m domain.Mutation,
	fn func(qtx *sqlc.Queries, tx pgx.Tx) (domain.MutationResult, error),
) (domain.MutationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.MutationResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.q.WithTx(tx)

	if _, err := qtx.LockUser(ctx, uuid.UUID(userID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MutationResult{}, fmt.Errorf("lock user: %w", domain.ErrNotFound)
		}
		return domain.MutationResult{}, fmt.Errorf("lock user: %w", err)
	}

	stored, err := qtx.GetMutationResult(ctx, sqlc.GetMutationResultParams{
		UserID: uuid.UUID(userID),
		ID:     uuid.UUID(m.ID),
	})
	switch {
	case err == nil:
		return storedResult(m.ID, stored), nil
	case !errors.Is(err, pgx.ErrNoRows):
		return domain.MutationResult{}, fmt.Errorf("get mutation result: %w", err)
	}

	result, err := fn(qtx, tx)
	if err != nil {
		return domain.MutationResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.MutationResult{}, fmt.Errorf("commit: %w", err)
	}

	return result, nil
}

func (r *SyncRepository) applyInSavepoint(ctx context.Context, tx pgx.Tx, userID domain.UserID, m domain.Mutation) error {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin savepoint: %w", err)
	}
	defer func() {
		_ = sp.Rollback(ctx)
	}()

	if err := applyMutation(ctx, r.q.WithTx(sp), userID, m); err != nil {
		return mapProductErr(err)
	}

	if err := sp.Commit(ctx); err != nil {
		return fmt.Errorf("release savepoint: %w", err)
	}

	return nil
}

func applyMutation(ctx context.Context, q *sqlc.Queries, userID domain.UserID, m domain.Mutation) error {
	switch m.Type {
	case domain.MutationCreateProduct:
		return q.CreateProduct(ctx, sqlc.CreateProductParams(productParams(userID, m.ProductID, m.Data)))

	case domain.MutationUpdateProduct:
		n, err := q.UpdateProduct(ctx, sqlc.UpdateProductParams(productParams(userID, m.ProductID, m.Data)))
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("product: %w", domain.ErrNotFound)
		}
		return nil

	case domain.MutationDeleteProduct:
		n, err := q.DeleteProduct(ctx, sqlc.DeleteProductParams{
			ID:     uuid.UUID(m.ProductID),
			UserID: uuid.UUID(userID),
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("product: %w", domain.ErrNotFound)
		}
		return nil

	default:
		return domain.NewValidationErr("type", "unknown mutation type")
	}
}

func mapProductErr(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}

	switch {
	case pgErr.Code == pgUniqueViolation:
		return fmt.Errorf("product: %w", domain.ErrAlreadyExists)
	case pgErr.Code == pgForeignKeyViolation:
		return fmt.Errorf("category: %w", domain.ErrNotFound)
	case pgErr.Code == pgCheckViolation, pgErr.Code[:2] == pgDataExceptionClass:
		return domain.NewValidationErr("data", pgErr.Message)
	default:
		return err
	}
}

func (r *SyncRepository) save(
	ctx context.Context,
	qtx *sqlc.Queries,
	userID domain.UserID,
	m domain.Mutation,
	result domain.MutationResult,
) (domain.MutationResult, error) {
	var payload []byte
	if result.Status == domain.MutationApplied && m.Data != nil {
		var err error
		if payload, err = encodePayload(m.Data); err != nil {
			return domain.MutationResult{}, fmt.Errorf("encode payload: %w", err)
		}
	}

	params := sqlc.CreateMutationParams{
		ID:        uuid.UUID(m.ID),
		UserID:    uuid.UUID(userID),
		Type:      string(m.Type),
		ProductID: uuid.UUID(m.ProductID),
		Payload:   payload,
		Status:    string(result.Status),
	}
	if result.Status == domain.MutationRejected {
		params.Reason = new(string(result.Reason))
		params.Message = &result.Message
	}

	seq, err := qtx.CreateMutation(ctx, params)
	if err != nil {
		return domain.MutationResult{}, fmt.Errorf("create mutation: %w", err)
	}

	result.MutationID = m.ID
	result.Seq = seq
	return result, nil
}

func storedResult(id domain.MutationID, row sqlc.GetMutationResultRow) domain.MutationResult {
	result := domain.MutationResult{
		MutationID: id,
		Status:     domain.MutationStatus(row.Status),
		Seq:        row.Seq,
	}
	if row.Reason != nil {
		result.Reason = domain.RejectReason(*row.Reason)
	}
	if row.Message != nil {
		result.Message = *row.Message
	}
	return result
}

type productRow struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	Name                  string
	CategoryID            *uuid.UUID
	Description           *string
	Quantity              *string
	ImageLink             *string
	ManufacturedAt        *time.Time
	ExpiresAt             time.Time
	OpenedAt              *time.Time
	ShelfLifeAfterOpening *int32
}

func productParams(userID domain.UserID, id domain.ProductID, d *domain.ProductData) productRow {
	row := productRow{
		ID:                    uuid.UUID(id),
		UserID:                uuid.UUID(userID),
		Name:                  d.Name,
		Description:           d.Description,
		Quantity:              d.Quantity,
		ImageLink:             d.ImageLink,
		ManufacturedAt:        d.ManufacturedAt,
		ExpiresAt:             d.ExpiresAt,
		OpenedAt:              d.OpenedAt,
		ShelfLifeAfterOpening: d.ShelfLifeAfterOpening,
	}
	if d.CategoryID != nil {
		row.CategoryID = new(uuid.UUID(*d.CategoryID))
	}
	return row
}

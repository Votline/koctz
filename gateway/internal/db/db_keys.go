package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type KeysPsql struct {
	log *zap.Logger
	db  *sqlx.DB
	bd  sq.StatementBuilderType
}

func NewKeysPsql(log *zap.Logger) (KeysRepository, error) {
	const op = "db_keys.NewKeysPsql"

	db, err := GetDB()
	if err != nil {
		return nil, fmt.Errorf("%s: get db: %w", op, err)
	}

	return &KeysPsql{
		log: log,
		db:  db,
		bd:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

func (r *KeysPsql) GetByRequestID(ctx context.Context, requestID string) (*Key, error) {
	const op = "db_keys.GetByRequestID"

	query, args, err := r.bd.Select("id", "sku", "code", "status", "request_id").
		From("keys").
		Where(sq.Eq{"request_id": requestID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}

	var key Key
	if err := r.db.GetContext(ctx, &key, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: key not found", op)
		}
		return nil, fmt.Errorf("%s: get: %w", op, err)
	}

	return &key, nil
}

func (r *KeysPsql) ReserveAndIssueKey(ctx context.Context, sku, requestID string) (*Key, error) {
	const op = "db_keys.ReserveAndIssueKey"

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	subSelect, subArgs, err := r.bd.Select("id").
		From("keys").
		Where(sq.Eq{"sku": sku, "status": "available"}).
		Limit(1).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build subquery: %w", op, err)
	}

	query, args, err := r.bd.Update("keys").
		Set("status", "issued").
		Set("request_id", requestID).
		Where(fmt.Sprintf("id = (%s)", subSelect), subArgs...).
		Suffix("RETURNING id, sku, code, status, request_id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build update query: %w", op, err)
	}

	var key Key
	if err := tx.GetContext(ctx, &key, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: out of stock", op)
		}
		return nil, fmt.Errorf("%s: reserve key: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: commit tx: %w", op, err)
	}

	return &key, nil
}

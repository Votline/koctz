package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type OrdersPsql struct {
	log *zap.Logger
	db  *sqlx.DB
	bd  sq.StatementBuilderType
}

func NewOrdersPsql(log *zap.Logger) (OrdersRepository, error) {
	const op = "db_orders.NewOrdersPsql"

	db, err := GetDB()
	if err != nil {
		return nil, fmt.Errorf("%s: get db: %w", op, err)
	}

	return &OrdersPsql{
		log: log,
		db:  db,
		bd:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

func (r *OrdersPsql) CreateOrder(ctx context.Context, order *Order) error {
	const op = "db_orders.CreateOrder"

	query, args, err := r.bd.Insert("orders").
		Columns("id", "sku", "status", "created_at").
		Values(order.ID, order.SKU, order.Status, order.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	if _, err = r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	return nil
}

func (r *OrdersPsql) GetOrderByID(ctx context.Context, id string) (*Order, error) {
	const op = "db_orders.GetOrderByID"

	query, args, err := r.bd.Select("id", "sku", "status", "COALESCE(code, '') as code", "created_at").
		From("orders").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}

	var ord Order
	if err := r.db.GetContext(ctx, &ord, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: get: not found", op)
		}
		return nil, fmt.Errorf("%s: get: %w", op, err)
	}

	return &ord, nil
}

func (r *OrdersPsql) ProcessPayment(ctx context.Context, eventID string, orderID string, updateFn func(ord *Order) error) error {
	const op = "OrdersPsql.ProcessPayment"

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	query, args, err := r.bd.Select("id", "sku", "status", "code", "created_at").
		From("orders").
		Where(sq.Eq{"id": orderID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build select query: %w", op, err)
	}

	var ord Order
	if err := tx.GetContext(ctx, &ord, query, args...); err != nil {
		return fmt.Errorf("%s: get order: %w", op, err)
	}

	insertEvQuery, evArgs, err := r.bd.Insert("payment_events").
		Columns("event_id", "order_id", "status").
		Values(eventID, orderID, "received").
		Suffix("ON CONFLICT (event_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build insert event: %w", op, err)
	}

	res, err := tx.ExecContext(ctx, insertEvQuery, evArgs...)
	if err != nil {
		return fmt.Errorf("%s: exec insert event: %w", op, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: check rows affected: %w", op, err)
	}

	if affected == 0 {
		return tx.Commit()
	}

	if err := updateFn(&ord); err != nil {
		return fmt.Errorf("%s: updateFn error: %w", op, err)
	}

	updateQ, updateArgs, err := r.bd.Update("orders").
		Set("status", ord.Status).
		Set("code", ord.Code).
		Where(sq.Eq{"id": ord.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build update query: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, updateQ, updateArgs...); err != nil {
		return fmt.Errorf("%s: execute update: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit tx: %w", op, err)
	}

	return nil
}

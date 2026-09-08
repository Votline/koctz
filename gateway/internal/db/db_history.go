package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type HistoryPsql struct {
	log *zap.Logger
	db  *sqlx.DB
	bd  sq.StatementBuilderType
}

func NewHistoryPsql(log *zap.Logger) (HistoryRepository, error) {
	const op = "db_history.NewHistoryPsql"

	db, err := GetDB()
	if err != nil {
		return nil, fmt.Errorf("%s: get db: %w", op, err)
	}

	return &HistoryPsql{
		log: log,
		db:  db,
		bd:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

func (r *HistoryPsql) SaveEvent(ctx context.Context, event *OrderHistoryEvent) error {
	const op = "db_history.SaveEvent"

	query, args, err := r.bd.Insert("order_history").
		Columns("order_id", "status", "event_type", "amount", "created_at").
		Values(event.OrderID, event.Status, event.EventType, event.Amount, time.Now()).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	return nil
}

func (r *HistoryPsql) GetStateAt(ctx context.Context, orderID string, at time.Time) (*OrderHistoryEvent, error) {
	const op = "db_history.GetStateAt"

	at = at.Add(1 * time.Second)

	query, args, err := r.bd.Select("id", "order_id", "status", "event_type", "amount", "created_at").
		From("order_history").
		Where(sq.Eq{"order_id": orderID}).
		Where(sq.LtOrEq{"created_at": at}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}

	var event OrderHistoryEvent
	if err := r.db.GetContext(ctx, &event, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: state at this time not found", op)
		}
		return nil, fmt.Errorf("%s: get: %w", op, err)
	}

	return &event, nil
}

func (r *HistoryPsql) GetReconciliation(ctx context.Context, from, to time.Time) (paid, delivered, refunded float64, isBalanced bool, err error) {
	const op = "db_history.GetReconciliation"

	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN event_type = 'paid' THEN amount ELSE 0 END), 0) as paid,
			COALESCE(SUM(CASE WHEN event_type = 'delivered' THEN amount ELSE 0 END), 0) as delivered,
			COALESCE(SUM(CASE WHEN event_type = 'refunded' THEN amount ELSE 0 END), 0) as refunded
		FROM order_history
		WHERE created_at BETWEEN $1 AND $2
	`

	row := r.db.QueryRowContext(ctx, query, from, to)
	if err := row.Scan(&paid, &delivered, &refunded); err != nil {
		return 0, 0, 0, false, fmt.Errorf("%s: scan reconciliation: %w", op, err)
	}

	return paid, delivered, refunded, paid == (delivered + refunded), nil
}

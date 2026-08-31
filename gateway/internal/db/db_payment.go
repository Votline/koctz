package db

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type PaymentsPsql struct {
	log *zap.Logger
	db  *sqlx.DB
	bd  sq.StatementBuilderType
}

func NewPaymentsPsql(log *zap.Logger) (PaymentRepository, error) {
	const op = "db_payments.NewPaymentsPsql"

	db, err := GetDB()
	if err != nil {
		return nil, fmt.Errorf("%s: get db: %w", op, err)
	}

	return &PaymentsPsql{
		log: log,
		db:  db,
		bd:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

func (r *PaymentsPsql) RegisterEvent(ctx context.Context, eventID, orderID, status string) error {
	const op = "db_payments.RegisterEvent"

	query, args, err := r.bd.Insert("payment_events").
		Columns("event_id", "order_id", "status").
		Values(eventID, orderID, status).
		Suffix("ON CONFLICT (event_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key constraint") || strings.Contains(err.Error(), "23503") {
			return fmt.Errorf("%s: order not found", op)
		}
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: check rows affected: %w", op, err)
	}

	if rows == 0 {
		return fmt.Errorf("%s: no rows: event already processed", op)
	}

	return nil
}

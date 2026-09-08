package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: being tx: %w", op, err)
	}
	defer tx.Rollback()

	query, args, err := r.bd.Insert("orders").
		Columns("id", "status", "price", "created_at").
		Values(order.ID, order.Status, order.Price, order.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("%s: exec tx: %w", op, err)
	}

	if len(order.Items) > 0 {
		builder := r.bd.Insert("order_items").
			Columns("id", "order_id", "sku", "price", "status", "code", "created_at")

		for _, item := range order.Items {
			builder = builder.Values(item.ID, order.ID, item.SKU, item.Price, item.Status, item.Code, item.CreatedAt)
		}

		itemsQuery, itemsArgs, err := builder.ToSql()
		if err != nil {
			return fmt.Errorf("%s: build items query: %w", op, err)
		}

		if _, err = tx.ExecContext(ctx, itemsQuery, itemsArgs...); err != nil {
			return fmt.Errorf("%s: exec items: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit tx: %w", op, err)
	}

	return nil
}

func (r *OrdersPsql) GetOrderByID(ctx context.Context, id string) (*Order, error) {
	const op = "db_orders.GetOrderByID"

	query, args, err := r.bd.Select("id", "status", "price", "created_at").
		From("orders").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build order query: %w", op, err)
	}

	var ord Order
	if err := r.db.GetContext(ctx, &ord, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: get order: not found", op)
		}
		return nil, fmt.Errorf("%s: get order: %w", op, err)
	}

	itemsQuery, itemsArgs, err := r.bd.Select("id", "order_id", "sku", "price", "status", "COALESCE(code, '') as code", "created_at").
		From("order_items").
		Where(sq.Eq{"order_id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build items query: %w", op, err)
	}

	if err := r.db.SelectContext(ctx, &ord.Items, itemsQuery, itemsArgs...); err != nil {
		return nil, fmt.Errorf("%s: select items: %w", op, err)
	}

	return &ord, nil
}

func (r *OrdersPsql) ProcessPayment(ctx context.Context, eventID, orderID string, updateFn func(ord *Order) error) error {
	const op = "OrdersPsql.ProcessPayment"

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	query, args, err := r.bd.Select("id", "status", "price", "created_at").
		From("orders").
		Where(sq.Eq{"id": orderID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build select query: %w", op, err)
	}

	var ord Order
	if err := tx.GetContext(ctx, &ord, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return fmt.Errorf("%s: order not found", op)
		}
		return fmt.Errorf("%s: get order: %w", op, err)
	}

	itemsQuery, itemsArgs, err := r.bd.Select("id", "order_id", "sku", "price", "status", "COALESCE(code, '') AS code", "created_at").
		From("order_items").
		Where(sq.Eq{"order_id": orderID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build select items query: %w", op, err)
	}

	if err := tx.SelectContext(ctx, &ord.Items, itemsQuery, itemsArgs...); err != nil {
		return fmt.Errorf("%s: get items: %w", op, err)
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
		return fmt.Errorf("%s: event already processed", op)
	}

	if err := updateFn(&ord); err != nil {
		return fmt.Errorf("%s: updateFn error: %w", op, err)
	}

	updateQ, updateArgs, err := r.bd.Update("orders").
		Set("status", ord.Status).
		Set("price", ord.Price).
		Where(sq.Eq{"id": ord.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build update query: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, updateQ, updateArgs...); err != nil {
		return fmt.Errorf("%s: execute update: %w", op, err)
	}

	for _, item := range ord.Items {
		itemQ, itemArgs, err := r.bd.Update("order_items").
			Set("status", item.Status).
			Set("code", item.Code).
			Set("price", item.Price).
			Where(sq.Eq{"id": item.ID}).
			ToSql()
		if err != nil {
			return fmt.Errorf("%s: build item update query: %w", op, err)
		}

		if _, err := tx.ExecContext(ctx, itemQ, itemArgs...); err != nil {
			return fmt.Errorf("%s: execute item update: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit tx: %w", op, err)
	}

	return nil
}

func (r *OrdersPsql) GetPendingOrders(ctx context.Context, limit int) ([]Order, error) {
	const op = "db_orders.GetPendingOrders"

	query, args, err := r.bd.Select("id", "status", "price", "created_at").
		From("orders").
		Where(sq.Eq{"status": "paid"}).
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}

	var pending []Order
	if err := r.db.SelectContext(ctx, &pending, query, args...); err != nil {
		return nil, fmt.Errorf("%s: select: %w", op, err)
	}

	if len(pending) == 0 {
		return pending, nil
	}

	orderIDs := make([]string, len(pending))
	orderMap := make(map[string]*Order, len(pending))
	for i := range pending {
		orderIDs[i] = pending[i].ID
		orderMap[pending[i].ID] = &pending[i]
	}

	itemsQuery, itemsArgs, err := r.bd.Select("id", "order_id", "sku", "price", "status", "COALESCE(code, '') as code", "created_at").
		From("order_items").
		Where(sq.Eq{"order_id": orderIDs}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build items query: %w", op, err)
	}

	var items []OrderItem
	if err := r.db.SelectContext(ctx, &items, itemsQuery, itemsArgs...); err != nil {
		return nil, fmt.Errorf("%s: select items: %w", op, err)
	}

	for _, item := range items {
		if ord, exists := orderMap[item.OrderID]; exists {
			ord.Items = append(ord.Items, item)
		}
	}

	return pending, nil
}

func (r *OrdersPsql) UpdateOrderStatus(ctx context.Context, itemID, status, code string) error {
	const op = "db_orders.UpdateOrderStatus"

	query, args, err := r.bd.Update("order_items").
		Set("status", status).
		Set("code", code).
		Where(sq.Eq{"id": itemID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	return nil
}

func (r *OrdersPsql) UpdateOrderDelivery(ctx context.Context, ord *Order) error {
	const op = "OrdersPsql.UpdateOrderDelivery"

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer tx.Rollback()

	updateQ, updateArgs, err := r.bd.Update("orders").
		Set("status", ord.Status).
		Set("price", ord.Price).
		Where(sq.Eq{"id": ord.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: build update query: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, updateQ, updateArgs...); err != nil {
		return fmt.Errorf("%s: execute update: %w", op, err)
	}

	for _, item := range ord.Items {
		itemQ, itemArgs, err := r.bd.Update("order_items").
			Set("status", item.Status).
			Set("code", item.Code).
			Set("price", item.Price).
			Where(sq.Eq{"id": item.ID}).
			ToSql()
		if err != nil {
			return fmt.Errorf("%s: build item update query: %w", op, err)
		}

		if _, err := tx.ExecContext(ctx, itemQ, itemArgs...); err != nil {
			return fmt.Errorf("%s: execute item update: %w", op, err)
		}
	}

	return tx.Commit()
}

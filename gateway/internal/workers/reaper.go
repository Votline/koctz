// Package workers provides background workers
package workers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"koctz/internal/db"
	supplierclient "koctz/internal/suppliersclient"

	"go.uber.org/zap"
)

type Worker interface {
	Start()
	Stop()
}

type Reaper struct {
	odb      db.OrdersRepository
	splc     supplierclient.Client
	log      *zap.Logger
	interval time.Duration
	stopCh   chan struct{}
}

func NewReaper(splc supplierclient.Client, log *zap.Logger, interval time.Duration) (Worker, error) {
	const op = "workers.NewReaper"

	odb, err := db.NewOrdersPsql(log)
	if err != nil {
		return nil, fmt.Errorf("%s: get orders db: %w", op, err)
	}

	return &Reaper{
		odb:      odb,
		splc:     splc,
		log:      log,
		interval: interval,
		stopCh:   make(chan struct{}),
	}, nil
}

func (r *Reaper) Start() {
	ticker := time.NewTicker(r.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				r.processPendingOrders()
			case <-r.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (r *Reaper) Stop() {
	close(r.stopCh)
}

func (r *Reaper) processPendingOrders() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orders, err := r.odb.GetPendingOrders(ctx, 50)
	if err != nil {
		r.log.Error("reaper: failed to get pending orders", zap.Error(err))
		return
	}

	for _, ord := range orders {
		for _, item := range ord.Items {
			if item.Status == "delivered" {
				continue
			}

			reqID := fmt.Sprintf("reaper-%s-%s", ord.ID, item.ID)
			code, err := r.splc.IssueKey(ctx, item.SKU, reqID)
			if err != nil {
				status := "delivery_failed"
				if strings.Contains(err.Error(), "out of stock") {
					status = "out_of_stock"
				}
				_ = r.odb.UpdateOrderStatus(ctx, item.ID, status, "")
				continue
			}

			if err := r.odb.UpdateOrderStatus(ctx, item.ID, "delivered", code); err != nil {
				r.log.Error(
					"reaper: failed to update delivered status",
					zap.String("order_id", ord.ID),
					zap.String("item_id", item.ID),
					zap.Error(err),
				)
			} else {
				r.log.Info(
					"reaper: successfully retried and delivered order item",
					zap.String("order_id", ord.ID),
					zap.String("item_id", item.ID),
				)
			}
		}
	}
}

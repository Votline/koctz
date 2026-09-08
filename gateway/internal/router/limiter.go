package router

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type QueueLimiter struct {
	highChan  chan http.Handler
	lowChan   chan http.Handler
	ticker    *time.Ticker
	log       *zap.Logger
	inQueue   int64
	processed int64
}

func NewQueueLimiter(log *zap.Logger, rps int) *QueueLimiter {
	interval := time.Minute / time.Duration(rps)
	if rps <= 0 {
		interval = 100 * time.Millisecond
	}

	ql := &QueueLimiter{
		highChan: make(chan http.Handler, 500),
		lowChan:  make(chan http.Handler, 500),
		ticker:   time.NewTicker(interval),
		log:      log,
	}

	go ql.startWorker()
	return ql
}

func (ql *QueueLimiter) startWorker() {
	for range ql.ticker.C {
		var h http.Handler

		select {
		case h = <-ql.highChan:
		default:
			select {
			case h = <-ql.highChan:
			case h = <-ql.lowChan:
			default:
				continue
			}
		}

		atomic.AddInt64(&ql.inQueue, -1)
		atomic.AddInt64(&ql.processed, 1)

		if h != nil {
			h.ServeHTTP(nil, nil)
		}
	}
}

func (ql *QueueLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/queue/stats" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(ql.getStatsJSON()))
			return
		}

		if !strings.HasPrefix(r.URL.Path, "/api/orders") && !strings.HasPrefix(r.URL.Path, "/webhook/payment") {
			next.ServeHTTP(w, r)
			return
		}

		done := make(chan struct{})
		wrapped := http.HandlerFunc(func(ww http.ResponseWriter, rr *http.Request) {
			next.ServeHTTP(w, r)
			close(done)
		})

		atomic.AddInt64(&ql.inQueue, 1)

		if strings.HasPrefix(r.URL.Path, "/webhook/payment") {
			ql.highChan <- wrapped
		} else {
			ql.lowChan <- wrapped
		}

		<-done
	})
}

func (ql *QueueLimiter) getStatsJSON() string {
	inQ := atomic.LoadInt64(&ql.inQueue)
	proc := atomic.LoadInt64(&ql.processed)
	return fmt.Sprintf(`{"in_queue":%d,"processed":%d}`, inQ, proc)
}

package observability

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type DBPoolCollector struct {
	db *pgxpool.Pool

	acquiredConns          *prometheus.Desc
	idleConns              *prometheus.Desc
	totalConns             *prometheus.Desc
	constructingConns      *prometheus.Desc
	acquireCount           *prometheus.Desc
	canceledAcquireCount   *prometheus.Desc
	emptyAcquireCount      *prometheus.Desc
	acquireDurationSeconds *prometheus.Desc
	waitDurationSeconds    *prometheus.Desc
	newConnsCount          *prometheus.Desc
	maxConns               *prometheus.Desc
}

func NewDBPoolCollector(db *pgxpool.Pool) *DBPoolCollector {
	return &DBPoolCollector{
		db: db,
		acquiredConns: prometheus.NewDesc(
			"marketplace_db_pool_acquired_connections",
			"Number of acquired database connections.",
			nil, nil,
		),
		idleConns: prometheus.NewDesc(
			"marketplace_db_pool_idle_connections",
			"Number of idle database connections.",
			nil, nil,
		),
		totalConns: prometheus.NewDesc(
			"marketplace_db_pool_total_connections",
			"Total number of database connections in the pool.",
			nil, nil,
		),
		constructingConns: prometheus.NewDesc(
			"marketplace_db_pool_constructing_connections",
			"Number of database connections currently being established.",
			nil, nil,
		),
		acquireCount: prometheus.NewDesc(
			"marketplace_db_pool_acquire_total",
			"Total number of successful connection acquires.",
			nil, nil,
		),
		canceledAcquireCount: prometheus.NewDesc(
			"marketplace_db_pool_acquire_canceled_total",
			"Total number of canceled connection acquires.",
			nil, nil,
		),
		emptyAcquireCount: prometheus.NewDesc(
			"marketplace_db_pool_acquire_empty_total",
			"Total number of acquires that waited for a connection.",
			nil, nil,
		),
		acquireDurationSeconds: prometheus.NewDesc(
			"marketplace_db_pool_acquire_duration_seconds_total",
			"Total time spent acquiring connections.",
			nil, nil,
		),
		waitDurationSeconds: prometheus.NewDesc(
			"marketplace_db_pool_empty_wait_seconds_total",
			"Total time spent waiting for a connection from an empty pool.",
			nil, nil,
		),
		newConnsCount: prometheus.NewDesc(
			"marketplace_db_pool_new_connections_total",
			"Total number of newly created database connections.",
			nil, nil,
		),
		maxConns: prometheus.NewDesc(
			"marketplace_db_pool_max_connections",
			"Maximum configured database connections.",
			nil, nil,
		),
	}
}

func (c *DBPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.totalConns
	ch <- c.constructingConns
	ch <- c.acquireCount
	ch <- c.canceledAcquireCount
	ch <- c.emptyAcquireCount
	ch <- c.acquireDurationSeconds
	ch <- c.waitDurationSeconds
	ch <- c.newConnsCount
	ch <- c.maxConns
}

func (c *DBPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c == nil || c.db == nil {
		return
	}

	stats := c.db.Stat()
	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(stats.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stats.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stats.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.constructingConns, prometheus.GaugeValue, float64(stats.ConstructingConns()))
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(stats.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.canceledAcquireCount, prometheus.CounterValue, float64(stats.CanceledAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.emptyAcquireCount, prometheus.CounterValue, float64(stats.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.acquireDurationSeconds, prometheus.CounterValue, stats.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(c.waitDurationSeconds, prometheus.CounterValue, stats.EmptyAcquireWaitTime().Seconds())
	ch <- prometheus.MustNewConstMetric(c.newConnsCount, prometheus.CounterValue, float64(stats.NewConnsCount()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(stats.MaxConns()))
}

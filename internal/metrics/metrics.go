// Package metrics expõe ao Prometheus o que aconteceu com cada mensagem da fila.
package metrics

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Messages conta as mensagens por desfecho: notified, requeued ou dropped.
	Messages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fiapx_notification_messages_total",
		Help: "Mensagens da fila de falhas por desfecho",
	}, []string{"outcome"})

	// SendDuration mede quanto tempo o servidor SMTP leva para aceitar a mensagem.
	SendDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "fiapx_notification_send_duration_seconds",
		Help:    "Duração do envio ao servidor SMTP",
		Buckets: prometheus.DefBuckets,
	})

	// Connected indica se o consumidor está ligado ao broker.
	Connected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fiapx_notification_broker_connected",
		Help: "1 quando o consumidor está conectado ao RabbitMQ",
	})
)

// Serve sobe o endpoint /metrics e um /health para as probes. Não bloqueia.
func Serve(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("servidor de métricas encerrado: %v", err)
		}
	}()
	log.Printf("métricas em %s/metrics", addr)
}

// Observe registra o tempo gasto em uma tentativa de envio.
func Observe(started time.Time) {
	SendDuration.Observe(time.Since(started).Seconds())
}

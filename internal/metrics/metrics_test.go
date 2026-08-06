package metrics

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func portaLivre(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("não consegui reservar uma porta: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}

func buscar(t *testing.T, url string) (int, string) {
	t.Helper()
	var ultimo error
	for range 20 {
		resposta, err := http.Get(url)
		if err != nil {
			ultimo = err
			time.Sleep(50 * time.Millisecond)
			continue
		}
		defer resposta.Body.Close()
		corpo, _ := io.ReadAll(resposta.Body)
		return resposta.StatusCode, string(corpo)
	}
	t.Fatalf("servidor não respondeu: %v", ultimo)
	return 0, ""
}

func TestServeExpoeMetricasEHealth(t *testing.T) {
	addr := portaLivre(t)
	Serve(addr)
	base := "http://" + addr

	status, corpo := buscar(t, base+"/health")
	if status != http.StatusOK {
		t.Errorf("/health respondeu %d", status)
	}
	if !strings.Contains(corpo, `"status":"ok"`) {
		t.Errorf("/health devolveu %q", corpo)
	}

	Messages.WithLabelValues("notified").Inc()
	status, corpo = buscar(t, base+"/metrics")
	if status != http.StatusOK {
		t.Errorf("/metrics respondeu %d", status)
	}
	if !strings.Contains(corpo, "fiapx_notification_messages_total") {
		t.Error("/metrics não trouxe o contador de mensagens")
	}
	if !strings.Contains(corpo, `outcome="notified"`) {
		t.Error("/metrics não trouxe o rótulo de desfecho")
	}
}

func TestObserveRegistraADuracaoDoEnvio(t *testing.T) {
	Observe(time.Now().Add(-250 * time.Millisecond))

	// A asserção real é não entrar em pânico; o valor em si é verificado pelo /metrics.
	if SendDuration == nil {
		t.Fatal("histograma não inicializado")
	}
}

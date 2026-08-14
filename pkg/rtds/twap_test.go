package rtds

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestTwapTopic(t *testing.T) {
	topic, err := twapTopic(TWAPWindow30)
	if err != nil || topic != CryptoPriceTWAPThirty {
		t.Fatalf("expected thirty topic, got %s err=%v", topic, err)
	}
	topic, err = twapTopic(TWAPWindow60)
	if err != nil || topic != CryptoPriceTWAPSixty {
		t.Fatalf("expected sixty topic, got %s err=%v", topic, err)
	}
	if _, err := twapTopic(15); err == nil {
		t.Fatal("expected error for unsupported window")
	}
}

func TestCompactSymbolFilter(t *testing.T) {
	got := compactSymbolFilter(" BTC/USD ")
	if got != `{"symbol":"btc/usd"}` {
		t.Fatalf("expected compact lowercase filter, got %s", got)
	}
	if compactSymbolFilter("  ") != "" {
		t.Fatal("expected empty filter for blank symbol")
	}
}

func TestTWAPPriceEventUnmarshalAndExactValue(t *testing.T) {
	raw := `{"symbol":"btc/usd","value":65000.5,"full_accuracy_value":"65000500000000000000000","timestamp":1785178800000,"window_s":30}`
	var ev TWAPPriceEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if ev.Symbol != "btc/usd" || ev.WindowS != 30 || ev.Timestamp != 1785178800000 {
		t.Fatalf("unexpected event: %+v", ev)
	}
	exact, err := ev.ExactValue()
	if err != nil {
		t.Fatalf("ExactValue failed: %v", err)
	}
	if exact.String() != "65000.5" {
		t.Fatalf("expected 65000.5, got %s", exact.String())
	}

	fallback := TWAPPriceEvent{Value: ev.Value}
	got, err := fallback.ExactValue()
	if err != nil {
		t.Fatalf("empty ExactValue failed: %v", err)
	}
	if !got.Equal(ev.Value) {
		t.Fatalf("expected fallback to Value, got %s", got.String())
	}

	invalid := TWAPPriceEvent{FullAccuracyValue: "not-a-number"}
	if _, err := invalid.ExactValue(); err == nil {
		t.Fatal("expected error for invalid full_accuracy_value")
	}
}

func TestSubscribeTWAPPrices_InvalidWindow(t *testing.T) {
	c := newTestClient()
	if _, err := c.SubscribeTWAPPricesStream(context.Background(), 15, []string{"btc/usd"}); err == nil {
		t.Fatal("expected error for unsupported window")
	}
	if err := c.UnsubscribeTWAPPrices(context.Background(), 15); err == nil {
		t.Fatal("expected unsubscribe error for unsupported window")
	}
}

func TestSubscribeTWAPPrices_SendsCompactFilterAndMapsEvent(t *testing.T) {
	gotSub := make(chan []byte, 1)
	s := mockWSServer(t, func(c *websocket.Conn) {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			if string(msg) == "PING" {
				continue
			}
			select {
			case gotSub <- msg:
			default:
			}
			payload := `{"topic":"crypto_prices_twap_thirty","type":"update","timestamp":1785178800123,"payload":{"symbol":"btc/usd","value":65000.5,"full_accuracy_value":"65000500000000000000000","timestamp":1785178800000,"window_s":30}}`
			_ = c.WriteMessage(websocket.TextMessage, []byte(payload))
		}
	})
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	client, err := NewClientWithConfig(wsURL, ClientConfig{
		Reconnect:    false,
		PingInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	defer client.Close()

	stream, err := client.SubscribeTWAPPricesStream(context.Background(), TWAPWindow30, []string{"BTC/USD"})
	if err != nil {
		t.Fatalf("SubscribeTWAPPricesStream failed: %v", err)
	}
	defer stream.Close()

	select {
	case raw := <-gotSub:
		var req SubscriptionRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Fatalf("subscribe unmarshal failed: %v body=%s", err, raw)
		}
		if req.Action != SubscribeAction || len(req.Subscriptions) != 1 {
			t.Fatalf("unexpected subscribe request: %+v", req)
		}
		sub := req.Subscriptions[0]
		if sub.Topic != string(CryptoPriceTWAPThirty) || sub.MsgType != "update" {
			t.Fatalf("unexpected subscription: %+v", sub)
		}
		filter, ok := sub.Filters.(string)
		if !ok || filter != `{"symbol":"btc/usd"}` {
			t.Fatalf("expected compact string filter, got %#v", sub.Filters)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscribe frame")
	}

	select {
	case ev := <-stream.C:
		if ev.Symbol != "btc/usd" || ev.WindowS != 30 || ev.Topic != CryptoPriceTWAPThirty {
			t.Fatalf("unexpected mapped event: %+v", ev)
		}
		if ev.FullAccuracyValue != "65000500000000000000000" {
			t.Fatalf("unexpected full_accuracy_value: %s", ev.FullAccuracyValue)
		}
		if ev.MessageTimestamp != 1785178800123 {
			t.Fatalf("unexpected message timestamp: %d", ev.MessageTimestamp)
		}
	case err := <-stream.Err:
		t.Fatalf("unexpected stream error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for TWAP event")
	}
}

func TestSubscribeTWAPPrices_MultipleSymbolsOmitsFilter(t *testing.T) {
	gotSub := make(chan []byte, 1)
	s := mockWSServer(t, func(c *websocket.Conn) {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			if string(msg) == "PING" {
				continue
			}
			select {
			case gotSub <- msg:
			default:
			}
		}
	})
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	client, err := NewClientWithConfig(wsURL, ClientConfig{
		Reconnect:    false,
		PingInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	defer client.Close()

	if _, err := client.SubscribeTWAPPrices(context.Background(), TWAPWindow60, []string{"btc/usd", "eth/usd"}); err != nil {
		t.Fatalf("SubscribeTWAPPrices failed: %v", err)
	}

	select {
	case raw := <-gotSub:
		var req SubscriptionRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Fatalf("subscribe unmarshal failed: %v body=%s", err, raw)
		}
		if len(req.Subscriptions) != 1 {
			t.Fatalf("expected one subscription, got %+v", req)
		}
		sub := req.Subscriptions[0]
		if sub.Topic != string(CryptoPriceTWAPSixty) {
			t.Fatalf("expected sixty topic, got %s", sub.Topic)
		}
		if sub.Filters != nil {
			t.Fatalf("expected omitted filters, got %#v", sub.Filters)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscribe frame")
	}
}

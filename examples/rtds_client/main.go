package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/rtds"
)

func main() {
	// 1. Connect to RTDS (Real-Time Data Service)
	fmt.Println("Connecting to RTDS WebSocket...")
	client, err := rtds.NewClient("") // Use default ProdURL
	if err != nil {
		log.Fatalf("Failed to connect to RTDS: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 2. Subscribe to Crypto Prices
	// Symbols follow Binance pairs like "btcusdt", "ethusdt"
	symbols := []string{"btcusdt", "ethusdt"}
	fmt.Printf("Subscribing to Crypto Prices for: %v\n", symbols)

	priceCh, err := client.SubscribeCryptoPrices(ctx, symbols)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// 3. Subscribe to Chainlink TWAP prices (30s lookback).
	// Symbols are lowercase slash pairs such as "btc/usd". Omit symbols to receive every pair.
	twapSymbols := []string{"btc/usd"}
	fmt.Printf("Subscribing to TWAP Prices for: %v\n", twapSymbols)
	twapCh, err := client.SubscribeTWAPPrices(ctx, rtds.TWAPWindow30, twapSymbols)
	if err != nil {
		log.Fatalf("Failed to subscribe TWAP: %v", err)
	}

	// 4. Read Loop
	go func() {
		for event := range priceCh {
			fmt.Printf("[RTDS] %s Price: %s (ts=%d)\n", event.Symbol, event.Value.String(), event.Timestamp)
		}
	}()
	go func() {
		for event := range twapCh {
			exact, err := event.ExactValue()
			if err != nil {
				exact = event.Value
			}
			fmt.Printf("[RTDS TWAP %ds] %s Price: %s (ts=%d)\n", event.WindowS, event.Symbol, exact.String(), event.Timestamp)
		}
	}()

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	fmt.Println("Shutting down...")
}

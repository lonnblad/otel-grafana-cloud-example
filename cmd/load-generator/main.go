package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/lonnblad/basic-otel-grafana-cloud-example/config"
	"github.com/lonnblad/basic-otel-grafana-cloud-example/telemetry"
)

func main() {
	ctx := context.Background()

	//=== Setting up logging ===//
	slog.SetDefault(
		slog.New(
			telemetry.NewOtelHandler(
				slog.NewJSONHandler(os.Stdout, nil).
					WithAttrs([]slog.Attr{slog.String("environment", config.GetEnvironment())}).
					WithAttrs([]slog.Attr{slog.String("service_name", config.GetServiceName())}).
					WithAttrs([]slog.Attr{slog.String("service_version", config.GetServiceVersion())}),
			),
		),
	)

	slog.InfoContext(ctx, "Starting up")

	//=== Setting up Open Telemetry ===//
	res, err := telemetry.NewResource(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Couldn't create new resource", "error", err)
		return
	}

	meterProvider, err := telemetry.InitMeter(ctx, res)
	if err != nil {
		slog.ErrorContext(ctx, "Couldn't create a new meter provider", "error", err)
		return
	}

	defer func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "Couldn't shutdown meter provider", "error", err)
			return
		}
	}()

	traceProvider, err := telemetry.InitTracer(ctx, res)
	if err != nil {
		slog.ErrorContext(ctx, "Couldn't create a new tracer provider", "error", err)
		return
	}

	defer func() {
		if err := traceProvider.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "Couldn't shutdown meter provider", "error", err)
			return
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	//=== Run the load generator ===//
	go runLoadGenerator(ctx)

	select {
	case <-sigCh:
		_, cancel := context.WithCancel(ctx)
		cancel()

		slog.InfoContext(ctx, "Good bye")

		return
	}
}

func runLoadGenerator(ctx context.Context) {
	for {
		n, err := rand.Int(rand.Reader, big.NewInt(100))
		if err != nil {
			continue
		}

		fibonacci(ctx, n.Int64())

		select {
		case <-ctx.Done():
			return

		default:
			time.Sleep(time.Second)
		}
	}
}

func fibonacci(ctx context.Context, n int64) {
	ctx, span := otel.Tracer("fib").Start(ctx, "fibonacci")
	defer span.End()

	f, err := callFibonacciService(ctx, n)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		slog.LogAttrs(
			ctx, slog.LevelError, "Couldn't calculate Fibonacci",
			slog.Int64("n", n), slog.Any("error", err),
		)

		return
	}

	slog.LogAttrs(
		ctx, slog.LevelInfo, "Calculated Fibonacci",
		slog.Int64("n", n), slog.Int64("fib", f),
	)
}

func callFibonacciService(ctx context.Context, n int64) (_ int64, err error) {
	ctx, span := otel.Tracer("client").Start(ctx, "call fibonacci-service")
	defer span.End()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	url := config.GetFibonacciServiceUrl() + "/calculate"

	span.SetAttributes(attribute.Int64("n", n))

	body := strings.NewReader(fmt.Sprintf(`{"n": %d}`, n))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		err = fmt.Errorf("couldn't create a request to the fibonacci-service: %w", err)

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return
	}

	req.Header.Add("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("couldn't call the fibonacci-service: %w", err)

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("the fibonacci-service returned a non 200 response: %d", resp.StatusCode)

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return
	}

	var respBody struct {
		F int64 `json:"f"`
	}

	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		err = fmt.Errorf("couldn't decode the fibonacci-service response: %w", err)

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return
	}

	span.SetAttributes(attribute.Int64("f", respBody.F))
	span.SetStatus(codes.Ok, "")

	return respBody.F, nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lonnblad/basic-otel-grafana-cloud-example/config"
	"github.com/lonnblad/basic-otel-grafana-cloud-example/telemetry"
)

func main() {
	ctx := context.Background()

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

	otelHandler := otelhttp.NewHandler(calculateHandlerFunc(), "POST :: /calculate")
	http.Handle("/calculate", otelHandler)

	server := http.Server{Addr: ":" + config.GetRestPort(), Handler: nil}

	go func() {
		err = server.ListenAndServe()
	}()

	<-sigCh
	ctx, cancel := context.WithTimeout(ctx, config.GetShutdownTimeout())
	defer cancel()

	server.Shutdown(ctx)
	slog.InfoContext(ctx, "Good bye")
}

func calculateHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		slog.InfoContext(ctx, "Calculate Fibonacci started")

		var span trace.Span
		ctx, span = otel.Tracer("server").Start(ctx, "calculateHandler")
		defer span.End()

		defer req.Body.Close()

		var reqBody struct {
			N int64 `json:"n"`
		}

		span.AddEvent("parsing request")
		err := json.NewDecoder(req.Body).Decode(&reqBody)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		span.AddEvent("calculate fibonacci")

		f, err := func(ctx context.Context) (_ int64, err error) {
			ctx, span = otel.Tracer("server").Start(ctx, "fibonacci")
			defer span.End()

			f, err := fibonacci(reqBody.N)
			if err != nil {
				return
			}

			return f, nil
		}(ctx)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		span.SetAttributes(attribute.Int64("f", f))
		span.SetStatus(codes.Ok, "")

		span.AddEvent("writing response")

		_, err = fmt.Fprintf(w, `{"f": %d}`, f)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}
}

// Fibonacci returns the n-th Fibonacci number. An error is returned if the
// input is above 90.
func fibonacci(n int64) (int64, error) {
	if n <= 1 {
		return n, nil
	}

	if n > 90 {
		return 0, fmt.Errorf("unsupported Fibonacci number %d: too large", n)
	}

	var n2, n1 int64 = 0, 1
	for i := int64(2); i < n; i++ {
		n2, n1 = n1, n1+n2
	}

	return n2 + n1, nil
}

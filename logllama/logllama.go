package logllama

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
)

type tracingHandler struct {
	slog.Handler
	data map[any]any
}

type spanIDKey struct{}

func (t *tracingHandler) Handle(ctx context.Context, r slog.Record) error {
	if v := ctx.Value(spanIDKey{}); v != nil {
		if spanId, ok := v.(string); ok {
			r.AddAttrs(slog.String("span_id", spanId))
		}
	}
	// add other data attrs available on ctx
	for k := range t.data {
		if v := ctx.Value(k); v != nil {
			key := fmt.Sprint(k)
			value := fmt.Sprint(v)
			r.AddAttrs(slog.String(key, value))
		}
	}

	return t.Handler.Handle(ctx, r)
}

func Setup(data map[any]any) func(next http.Handler) http.Handler {
	slog.SetDefault(slog.New(&tracingHandler{
		Handler: slog.NewJSONHandler(os.Stdout, nil),
		data:    data,
	}))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// add base span identifier
			spanID := randID()
			var ctx context.Context
			ctx = context.WithValue(r.Context(), spanIDKey{}, spanID)
			// add other provided data
			for k, v := range data {
				ctx = context.WithValue(ctx, k, v)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var alphanum = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func randID() string {
	b := make([]rune, 15)

	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphanum))))
		b[i] = alphanum[idx.Int64()]
	}
	return string(b)
}

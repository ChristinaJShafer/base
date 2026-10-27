package span

import (
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"go.opentelemetry.io/otel/attribute"
)

type logValueFunc func() slog.Value

func (f logValueFunc) LogValue() slog.Value { return f() }

func TestAppendSlogAttrs(t *testing.T) {
	t.Parallel()

	timestamp := time.Unix(42, 123)
	tests := []struct {
		name   string
		prefix string
		attrs  []slog.Attr
		want   []attribute.KeyValue
	}{
		{
			name: "unsigned boundaries",
			attrs: []slog.Attr{
				slog.Uint64("zero", 0),
				slog.Uint64("max", math.MaxInt64),
				slog.Uint64("overflow", uint64(math.MaxInt64)+1),
				slog.Uint64("largest", math.MaxUint64),
			},
			want: []attribute.KeyValue{
				attribute.Int64("zero", 0),
				attribute.Int64("max", math.MaxInt64),
				attribute.String("overflow", "9223372036854775808"),
				attribute.String("largest", "18446744073709551615"),
			},
		},
		{
			name:   "nested and unnamed groups keep siblings independent",
			prefix: "rpc.",
			attrs: []slog.Attr{
				slog.Group("target",
					slog.Group("node", slog.String("name", "worker")),
					slog.Group("", slog.Bool("ready", true)),
					slog.Int64("count", 2),
				),
				slog.Group("", slog.String("method", "Submit")),
				slog.String("service", "medbay"),
			},
			want: []attribute.KeyValue{
				attribute.String("rpc.target.node.name", "worker"),
				attribute.Bool("rpc.target.ready", true),
				attribute.Int64("rpc.target.count", 2),
				attribute.String("rpc.method", "Submit"),
				attribute.String("rpc.service", "medbay"),
			},
		},
		{
			name: "resolve chained values and groups",
			attrs: []slog.Attr{
				slog.Any("value", logValueFunc(func() slog.Value {
					return slog.AnyValue(logValueFunc(func() slog.Value {
						return slog.StringValue("resolved")
					}))
				})),
				slog.Any("target", logValueFunc(func() slog.Value {
					return slog.GroupValue(slog.String("node", "worker"))
				})),
			},
			want: []attribute.KeyValue{
				attribute.String("value", "resolved"),
				attribute.String("target.node", "worker"),
			},
		},
		{
			name: "exact duration and time suffixes",
			attrs: []slog.Attr{
				slog.Duration("latency", -time.Second),
				slog.Duration("latency_ns", time.Second),
				slog.Duration("runs", time.Second),
				slog.Time("start", timestamp),
				slog.Time("start_unix_ns", timestamp),
				slog.Time("notunix_ns", timestamp),
			},
			want: []attribute.KeyValue{
				attribute.Int64("latency_ns", -int64(time.Second)),
				attribute.Int64("latency_ns", int64(time.Second)),
				attribute.Int64("runs_ns", int64(time.Second)),
				attribute.Int64("start_unix_ns", timestamp.UnixNano()),
				attribute.Int64("start_unix_ns", timestamp.UnixNano()),
				attribute.Int64("notunix_ns_unix_ns", timestamp.UnixNano()),
			},
		},
		{
			name:   "skip empty keys and arbitrary objects",
			prefix: "rpc",
			attrs: []slog.Attr{
				{},
				slog.String("", "ignored"),
				slog.Group("empty"),
				slog.Any("object", struct{ X int }{X: 1}),
				slog.Any("slice", []string{"ignored"}),
				slog.Any("nil", nil),
				slog.Any("resolvedObject", logValueFunc(func() slog.Value {
					return slog.AnyValue(struct{ X int }{X: 1})
				})),
			},
		},
		{
			name: "no attrs",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			existing := attribute.String("existing", "kept")
			dst := make([]attribute.KeyValue, 1, 1+len(test.want))
			dst[0] = existing
			want := append([]attribute.KeyValue{existing}, test.want...)
			got := AppendSlogAttrs(dst, test.prefix, test.attrs...)
			if diff := pretty.Compare(want, got); diff != "" {
				t.Errorf("AppendSlogAttrs: -want/+got:\n%s", diff)
			}
		})
	}
}

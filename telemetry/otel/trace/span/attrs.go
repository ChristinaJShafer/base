package span

import (
	"log/slog"
	"math"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

// AppendSlogAttrs appends supported slog attributes to dst, preserving their order.
// Prefix is a dotted namespace; trailing dots are ignored. Groups are flattened
// into dotted keys, with unnamed groups inlined, and LogValuer values are resolved.
// Unsigned integers above math.MaxInt64 are stored as decimal strings.
// Durations and timestamps are stored as int64 nanoseconds, with _ns and _unix_ns
// suffixes respectively (unless already present). Empty keys and unsupported
// arbitrary objects are skipped.
func AppendSlogAttrs(dst []attribute.KeyValue, prefix string, attrs ...slog.Attr) []attribute.KeyValue {
	prefix = strings.TrimRight(prefix, ".")
	for _, attr := range attrs {
		value := attr.Value.Resolve()
		key := attr.Key
		if key != "" && prefix != "" {
			key = prefix + "." + key
		}
		if value.Kind() == slog.KindGroup {
			if attr.Key == "" {
				key = prefix
			}
			dst = AppendSlogAttrs(dst, key, value.Group()...)
			continue
		}
		if attr.Key == "" {
			continue
		}

		switch value.Kind() {
		case slog.KindBool:
			dst = append(dst, attribute.Bool(key, value.Bool()))
		case slog.KindInt64:
			dst = append(dst, attribute.Int64(key, value.Int64()))
		case slog.KindUint64:
			n := value.Uint64()
			if n <= math.MaxInt64 {
				dst = append(dst, attribute.Int64(key, int64(n)))
			} else {
				dst = append(dst, attribute.String(key, strconv.FormatUint(n, 10)))
			}
		case slog.KindFloat64:
			dst = append(dst, attribute.Float64(key, value.Float64()))
		case slog.KindString:
			dst = append(dst, attribute.String(key, value.String()))
		case slog.KindDuration:
			if !strings.HasSuffix(key, "_ns") {
				key += "_ns"
			}
			dst = append(dst, attribute.Int64(key, int64(value.Duration())))
		case slog.KindTime:
			if !strings.HasSuffix(key, "_unix_ns") {
				key += "_unix_ns"
			}
			dst = append(dst, attribute.Int64(key, value.Time().UnixNano()))
		}
	}
	return dst
}

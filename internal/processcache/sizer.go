package processcache

import "time"

const itemOverhead int64 = 64

type DefaultSizer struct{}

func (DefaultSizer) SizeOf(key string, value any) int64 {
	size := int64(len(key)) + itemOverhead
	switch v := value.(type) {
	case nil:
		return size
	case string:
		return size + int64(len(v))
	case []byte:
		return size + int64(len(v))
	case bool:
		return size + 1
	case int, uint, uintptr:
		return size + 8
	case int8, uint8:
		return size + 1
	case int16, uint16:
		return size + 2
	case int32, uint32, float32:
		return size + 4
	case int64, uint64, float64, complex64:
		return size + 8
	case complex128:
		return size + 16
	case time.Time:
		return size + 24
	case time.Duration:
		return size + 8
	default:
		return size + 128
	}
}

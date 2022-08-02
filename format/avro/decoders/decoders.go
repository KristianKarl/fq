package decoders

import (
	"fmt"

	"github.com/wader/fq/format/avro/schema"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/scalar"
)

type DecodeFn func(string, *decode.D) any

func DecodeFnForSchema(s schema.SimplifiedSchema) (DecodeFn, error) {
	var sms []scalar.Mapper
	mapper := logicalMapperForSchema(s)
	if mapper != nil {
		sms = append(sms, mapper)
	}

	switch s.Type {
	case schema.ARRAY:
		return decodeArrayFn(s)
	case schema.BOOLEAN:
		return decodeBoolFn(sms...)
	case schema.BYTES:
		return decodeBytesFn(sms...)
	case schema.DOUBLE:
		return decodeDoubleFn(sms...)
	case schema.ENUM:
		return decodeEnumFn(s, sms...)
	case schema.FIXED:
		return decodeFixedFn(s, sms...)
	case schema.FLOAT:
		return decodeFloatFn(sms...)
	case schema.INT:
		return decodeIntFn(sms...)
	case schema.LONG:
		return decodeLongFn(sms...)
	case schema.NULL:
		return decodeNullFn(sms...)
	case schema.RECORD:
		return decodeRecordFn(s)
	case schema.STRING:
		return decodeStringFn(s, sms...)
	case schema.UNION:
		return decodeUnionFn(s)
	case schema.MAP:
		return decodeMapFn(s)
	default:
		return nil, fmt.Errorf("unknown type: %s", s.Type)
	}
}

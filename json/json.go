package json

import (
	"io"

	"github.com/creasty/defaults"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Encoder struct {
	*jsoniter.Encoder
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		Encoder: jsoniter.NewEncoder(w),
	}
}

// Encode 覆盖嵌入的 Encode 方法，添加 defaults.Set 逻辑
func (e *Encoder) Encode(v any) error {
	if err := defaults.Set(v); err != nil {
		return err
	}
	return e.Encoder.Encode(v)
}

type Decoder struct {
	*jsoniter.Decoder
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		Decoder: jsoniter.NewDecoder(r),
	}
}

// Decode 覆盖嵌入的 Decode 方法，添加 defaults.Set 逻辑
func (d *Decoder) Decode(v any) error {
	if err := defaults.Set(v); err != nil {
		return err
	}
	return d.Decoder.Decode(v)
}

func Marshal(v any) ([]byte, error) {
	if err := defaults.Set(v); err != nil {
		return nil, err
	}
	return jsoniter.Marshal(v)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	if err := defaults.Set(v); err != nil {
		return nil, err
	}
	return jsoniter.MarshalIndent(v, prefix, indent)
}

func MarshalToString(v any) (string, error) {
	if err := defaults.Set(v); err != nil {
		return "", err
	}
	return jsoniter.MarshalToString(v)
}

func Unmarshal(data []byte, v any) error {
	if err := defaults.Set(v); err != nil {
		return err
	}
	return jsoniter.Unmarshal(data, v)
}

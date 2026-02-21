package json

type EncoderInterface interface {
	Encode(any) error
}

type DecoderInterface interface {
	Decode(any) error
}

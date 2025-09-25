package grpcjson

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

// Codec implements gRPC encoding.Codec using JSON so we can marshal/unmarshal
// plain Go structs without generated protobuf types.
type Codec struct{}

func (Codec) Name() string { return "json" }

func (Codec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (Codec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Register registers the JSON codec globally so client and server can negotiate it.
func Register() {
	encoding.RegisterCodec(Codec{})
}

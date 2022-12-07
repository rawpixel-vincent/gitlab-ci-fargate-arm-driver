package encoding

import (
	stdJSON "encoding/json"
	"io"
)

type json struct{}

func NewJSON() Encoder {
	return new(json)
}

func (j *json) Decode(source io.Reader, target interface{}) error {
	return stdJSON.
		NewDecoder(source).
		Decode(target)
}

func (j *json) Encode(source interface{}, target io.Writer) error {
	return stdJSON.
		NewEncoder(target).
		Encode(source)
}

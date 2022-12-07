package encoding

import (
	"io"
)

type Encoder interface {
	Decode(source io.Reader, target interface{}) error
	Encode(source interface{}, target io.Writer) error
}

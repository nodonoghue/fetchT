package fetcht

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

var (
	JSONDecoder Decoder = jsonDecoder{}
	XMLDecoder  Decoder = xmlDecoder{}
)

// Decoder is an interface that can be used to implement your own decoder.
// application/json, application/xml, text/xml are included, but can be
// overridden with your own implementation
type Decoder interface {
	Decode(r io.Reader, v any) error
}

type jsonDecoder struct{}

func (d jsonDecoder) Decode(r io.Reader, v any) error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return err
	}

	return nil
}

type xmlDecoder struct{}

func (x xmlDecoder) Decode(r io.Reader, v any) error {
	if err := xml.NewDecoder(r).Decode(v); err != nil {
		return err
	}

	return nil
}

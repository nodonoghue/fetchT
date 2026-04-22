package fetcht

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/google/go-querystring/query"
)

var (
	JSONDecoder Decoder = jsonDecoder{}
	XMLDecoder  Decoder = xmlDecoder{}

	JSONEncoder      Encoder = jsonEncoder{}
	XMLEncoder       Encoder = xmlEncoder{}
	FormEncoder      Encoder = formEncoder{}
	MultipartEncoder Encoder = multipartEncoder{}
)

var defaultDecoders = map[string]Decoder{
	"application/json": JSONDecoder,
	"application/xml":  XMLDecoder,
	"text/xml":         XMLDecoder,
}

// Decoder is an interface that can be used to implement your own decoder.
// application/json, application/xml, text/xml are included, but can be
// overridden with your own implementation
type Decoder interface {
	Decode(r io.Reader, v any) error
}

type jsonDecoder struct{}

func (d jsonDecoder) Decode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

type xmlDecoder struct{}

func (x xmlDecoder) Decode(r io.Reader, v any) error {
	return xml.NewDecoder(r).Decode(v)
}

// Encoder is an interface that can be used to implement your own encoder.
// application/json, application/xml; charset=utf-8, application/x-www-form-urlencoded,
// and multipart/form-data are included.  Others can be implemented and passed into
// NewClient()
type Encoder interface {
	Encode(v any) (io.Reader, string, error)
}

type jsonEncoder struct{}

func (e jsonEncoder) Encode(v any) (io.Reader, string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(b), "application/json", nil
}

type formEncoder struct{}

func (e formEncoder) Encode(v any) (io.Reader, string, error) {
	values, err := query.Values(v)
	if err != nil {
		return nil, "", err
	}
	return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", nil
}

type xmlEncoder struct{}

func (e xmlEncoder) Encode(v any) (io.Reader, string, error) {
	x, err := xml.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(x), "application/xml; charset=utf-8", nil
}

type FilePart struct {
	Reader io.Reader
	Name   string
}
type MultipartForm struct {
	Fields map[string]string
	Files  map[string]FilePart
}
type multipartEncoder struct{}

func (e multipartEncoder) Encode(v any) (bodyReader io.Reader, contentType string, err error) {
	form, ok := v.(MultipartForm)
	if !ok {
		return nil, "", fmt.Errorf("MultipartEncoder expects MultipartForm, got %T", v)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			if err == nil {
				err = fmt.Errorf("MultipartEncoder failed to close multipart writer: %v", closeErr)
			} else {
				err = fmt.Errorf("%w, failed to close multipart writer: %v", err, closeErr)
			}
		}
	}()

	for key, val := range form.Fields {
		if err = writer.WriteField(key, val); err != nil {
			return
		}
	}

	for fieldName, filePart := range form.Files {
		var part io.Writer
		part, err = writer.CreateFormFile(fieldName, filePart.Name)
		if err != nil {
			return
		}
		if _, err = io.Copy(part, filePart.Reader); err != nil {
			return
		}
	}

	bodyReader = &buf
	contentType = writer.FormDataContentType()

	return
}

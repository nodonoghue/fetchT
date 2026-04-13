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
	JSON      Encoder = jsonEncoder{}
	XML       Encoder = xmlEncoder{}
	Form      Encoder = formEncoder{}
	Multipart Encoder = multipartEncoder{}
)

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

type MultipartForm struct {
	Fields map[string]string
	Files  map[string]io.Reader
}
type multipartEncoder struct{}

func (e multipartEncoder) Encode(v any) (io.Reader, string, error) {
	form, ok := v.(MultipartForm)
	if !ok {
		return nil, "", fmt.Errorf("MultipartEncoder expects MultipartForm, got %T", v)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, val := range form.Fields {
		writer.WriteField(key, val)
	}

	for fieldName, reader := range form.Files {
		part, err := writer.CreateFormFile(fieldName, fieldName)
		if err != nil {
			return nil, "", err
		}
		io.Copy(part, reader)
	}

	writer.Close()
	return &buf, writer.FormDataContentType(), nil
}

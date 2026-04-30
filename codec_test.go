package fetcht

import (
	"io"
	"strings"
	"testing"
)

type testStruct struct {
	Name string `json:"name" xml:"name" url:"name"`
	Age  int    `json:"age" xml:"age" url:"age"`
}

func TestJSONDecoder(t *testing.T) {
	input := `{"name":"John","age":30}`
	r := strings.NewReader(input)
	var v testStruct
	err := JSONDecoder.Decode(r, &v)
	if err != nil {
		t.Fatalf("JSONDecoder.Decode failed: %v", err)
	}
	if v.Name != "John" || v.Age != 30 {
		t.Errorf("JSONDecoder.Decode got %+v, want {Name:John Age:30}", v)
	}
}

func TestXMLDecoder(t *testing.T) {
	input := `<testStruct><name>John</name><age>30</age></testStruct>`
	r := strings.NewReader(input)
	var v testStruct
	err := XMLDecoder.Decode(r, &v)
	if err != nil {
		t.Fatalf("XMLDecoder.Decode failed: %v", err)
	}
	if v.Name != "John" || v.Age != 30 {
		t.Errorf("XMLDecoder.Decode got %+v, want {Name:John Age:30}", v)
	}
}

func TestJSONEncoder(t *testing.T) {
	v := testStruct{Name: "John", Age: 30}
	r, contentType, err := JSONEncoder.Encode(v)
	if err != nil {
		t.Fatalf("JSONEncoder.Encode failed: %v", err)
	}
	if contentType != "application/json" {
		t.Errorf("JSONEncoder.Encode got contentType %q, want application/json", contentType)
	}
	body, _ := io.ReadAll(r)
	expected := `{"name":"John","age":30}`
	if string(body) != expected {
		t.Errorf("JSONEncoder.Encode got body %q, want %q", string(body), expected)
	}
}

func TestXMLEncoder(t *testing.T) {
	v := testStruct{Name: "John", Age: 30}
	r, contentType, err := XMLEncoder.Encode(v)
	if err != nil {
		t.Fatalf("XMLEncoder.Encode failed: %v", err)
	}
	if contentType != "application/xml; charset=utf-8" {
		t.Errorf("XMLEncoder.Encode got contentType %q, want application/xml; charset=utf-8", contentType)
	}
	body, _ := io.ReadAll(r)
	// XML output might include header or different spacing, but xml.Marshal usually doesn't include header by default
	expected := `<testStruct><name>John</name><age>30</age></testStruct>`
	if string(body) != expected {
		t.Errorf("XMLEncoder.Encode got body %q, want %q", string(body), expected)
	}
}

func TestFormEncoder(t *testing.T) {
	v := testStruct{Name: "John", Age: 30}
	r, contentType, err := FormEncoder.Encode(v)
	if err != nil {
		t.Fatalf("FormEncoder.Encode failed: %v", err)
	}
	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("FormEncoder.Encode got contentType %q, want application/x-www-form-urlencoded", contentType)
	}
	body, _ := io.ReadAll(r)
	// Order might vary, but query.Values usually maintains order or is deterministic
	expected := "age=30&name=John"
	if string(body) != expected && string(body) != "name=John&age=30" {
		t.Errorf("FormEncoder.Encode got body %q, want %q", string(body), expected)
	}
}

func TestMultipartEncoder(t *testing.T) {
	form := MultipartForm{
		Fields: map[string]string{
			"foo": "bar",
		},
		Files: map[string]FilePart{
			"file1": {
				Reader: strings.NewReader("file content"),
				Name:   "test.txt",
			},
		},
	}
	r, contentType, err := MultipartEncoder.Encode(form)
	if err != nil {
		t.Fatalf("MultipartEncoder.Encode failed: %v", err)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Errorf("MultipartEncoder.Encode got contentType %q, want prefix multipart/form-data; boundary=", contentType)
	}

	body, _ := io.ReadAll(r)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, `name="foo"`) || !strings.Contains(bodyStr, "bar") {
		t.Errorf("MultipartEncoder.Encode body missing field foo: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `name="file1"`) || !strings.Contains(bodyStr, `filename="test.txt"`) || !strings.Contains(bodyStr, "file content") {
		t.Errorf("MultipartEncoder.Encode body missing file part: %s", bodyStr)
	}
}

func TestMultipartEncoder_Error(t *testing.T) {
	_, _, err := MultipartEncoder.Encode("not a form")
	if err == nil {
		t.Error("MultipartEncoder.Encode should fail for non-MultipartForm input")
	}
}

// Package humatest provides testing utilities for Huma services. It is based on
// the standard library `http.Request` & `http.ResponseWriter` types.
package humatest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"reflect"
	"strings"
	"testing/iotest"

	"github.com/ross96D/huma"
	"github.com/ross96D/huma/adapters/humaflow"
	"github.com/ross96D/huma/adapters/humaflow/flow"
)

// TB is a subset of the `testing.TB` interface used by the test API and
// implemented by the `*testing.T` and `*testing.B` structs.
type TB interface {
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
}

// NewContext creates a new test context from an HTTP request and response.
func NewContext(op *huma.Operation, r *http.Request, w http.ResponseWriter) huma.Context {
	return humaflow.NewContext(op, r, w)
}

// NewAdapter creates a new test adapter from a router.
func NewAdapter() huma.Adapter {
	return humaflow.NewAdapter(flow.New())
}

// TestAPI is a `huma.API` with additional methods specifically for testing.
type TestAPI interface {
	huma.API

	// Do a request against the API. Args, if provided, should be string headers
	// like `Content-Type: application/json`, an `io.Reader` for the request
	// body, or a slice/map/struct which will be serialized to JSON and sent
	// as the request body. Anything else will panic.
	Do(method, path string, args ...any) *httptest.ResponseRecorder

	// Get performs a GET request against the API. Args, if provided, should be
	// string headers like `Content-Type: application/json`, an `io.Reader`
	// for the request body, or a slice/map/struct which will be serialized to
	// JSON and sent as the request body. Anything else will panic.
	//
	// 	// Make a GET request
	// 	api.Get("/foo")
	//
	// 	// Make a GET request with a custom header.
	// 	api.Get("/foo", "X-My-Header: my-value")
	Get(path string, args ...any) *httptest.ResponseRecorder

	// Post performs a POST request against the API. Args, if provided, should be
	// string headers like `Content-Type: application/json`, an `io.Reader`
	// for the request body, or a slice/map/struct which will be serialized to
	// JSON and sent as the request body. Anything else will panic.
	//
	// 	// Make a POST request
	// 	api.Post("/foo", bytes.NewReader(`{"foo": "bar"}`))
	//
	// 	// Make a POST request with a custom header.
	// 	api.Post("/foo", "X-My-Header: my-value", MyBody{Foo: "bar"})
	Post(path string, args ...any) *httptest.ResponseRecorder

	// Put performs a PUT request against the API. Args, if provided, should be
	// string headers like `Content-Type: application/json`, an `io.Reader`
	// for the request body, or a slice/map/struct which will be serialized to
	// JSON and sent as the request body. Anything else will panic.
	//
	// 	// Make a PUT request
	// 	api.Put("/foo", bytes.NewReader(`{"foo": "bar"}`))
	//
	// 	// Make a PUT request with a custom header.
	// 	api.Put("/foo", "X-My-Header: my-value", MyBody{Foo: "bar"})
	Put(path string, args ...any) *httptest.ResponseRecorder

	// Patch performs a PATCH request against the API. Args, if provided, should
	// be string headers like `Content-Type: application/json`, an `io.Reader`
	// for the request body, or a slice/map/struct which will be serialized to
	// JSON and sent as the request body. Anything else will panic.
	//
	// 	// Make a PATCH request
	// 	api.Patch("/foo", bytes.NewReader(`{"foo": "bar"}`))
	//
	// 	// Make a PATCH request with a custom header.
	// 	api.Patch("/foo", "X-My-Header: my-value", MyBody{Foo: "bar"})
	Patch(path string, args ...any) *httptest.ResponseRecorder

	// Delete performs a DELETE request against the API. Args, if provided, should
	// be string headers like `Content-Type: application/json`, an `io.Reader`
	// for the request body, or a slice/map/struct which will be serialized to
	// JSON and sent as the request body. Anything else will panic.
	//
	// 	// Make a DELETE request
	// 	api.Delete("/foo")
	//
	// 	// Make a DELETE request with a custom header.
	// 	api.Delete("/foo", "X-My-Header: my-value")
	Delete(path string, args ...any) *httptest.ResponseRecorder
}

type testAPI struct {
	huma.API
	tb TB
}

func (a *testAPI) Do(method, path string, args ...any) *httptest.ResponseRecorder {
	a.tb.Helper()
	var b io.Reader
	isJSON := false
	for _, arg := range args {
		kind := reflect.Indirect(reflect.ValueOf(arg)).Kind()
		if reader, ok := arg.(io.Reader); ok {
			b = reader
			break
		} else if _, ok := arg.(string); ok {
			// do nothing
		} else if kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice {
			encoded, err := json.Marshal(arg)
			if err != nil {
				panic(err)
			}
			b = bytes.NewReader(encoded)
			isJSON = true
		} else {
			panic("unsupported argument type, expected string header or io.Reader/slice/map/struct body")
		}
	}

	req, _ := http.NewRequest(method, path, b)
	req.RequestURI = path
	req.RemoteAddr = "127.0.0.1:12345"
	if isJSON {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, arg := range args {
		if s, ok := arg.(string); ok {
			parts := strings.Split(s, ":")
			req.Header.Set(parts[0], strings.TrimSpace(strings.Join(parts[1:], ":")))

			if strings.ToLower(parts[0]) == "host" {
				req.Host = strings.TrimSpace(parts[1])
			}
		}
	}
	resp := httptest.NewRecorder()

	bytes, _ := DumpRequest(req)
	a.tb.Log("Making request:\n" + strings.TrimSpace(string(bytes)))

	a.Adapter().ServeHTTP(resp, req)

	bytes, _ = DumpResponse(resp.Result())
	a.tb.Log("Got response:\n" + strings.TrimSpace(string(bytes)))

	return resp
}

func (a *testAPI) Get(path string, args ...any) *httptest.ResponseRecorder {
	a.tb.Helper()
	return a.Do(http.MethodGet, path, args...)
}

func (a *testAPI) Post(path string, args ...any) *httptest.ResponseRecorder {
	a.tb.Helper()
	return a.Do(http.MethodPost, path, args...)
}

func (a *testAPI) Put(path string, args ...any) *httptest.ResponseRecorder {
	a.tb.Helper()
	return a.Do(http.MethodPut, path, args...)
}

func (a *testAPI) Patch(path string, args ...any) *httptest.ResponseRecorder {
	a.tb.Helper()
	return a.Do(http.MethodPatch, path, args...)
}

func (a *testAPI) Delete(path string, args ...any) *httptest.ResponseRecorder {
	a.tb.Helper()
	return a.Do(http.MethodDelete, path, args...)
}

// Wrap returns a `TestAPI` wrapping the given API.
func Wrap(tb TB, api huma.API) TestAPI {
	return &testAPI{api, tb}
}

// New creates a new router and test API, making it easy to register operations
// and perform requests against them. Optionally takes a configuration object
// to customize how the API is created. If no configuration is provided then
// a simple default configuration supporting `application/json` is used.
func New(tb TB, configs ...huma.Config) (http.Handler, TestAPI) {
	for _, config := range configs {
		if config.OpenAPI == nil {
			panic("custom huma.Config structs must specify a value for OpenAPI")
		}
	}
	if len(configs) == 0 {
		configs = append(configs, huma.Config{
			OpenAPI: &huma.OpenAPI{
				Info: &huma.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
			Formats: map[string]huma.Format{
				"application/json": huma.DefaultJSONFormat,
				"json":             huma.DefaultJSONFormat,
			},
			DefaultFormat: "application/json",
		})
	}
	r := flow.New()
	return r, Wrap(tb, humaflow.New(r, configs[0]))
}

func dumpBody(body io.ReadCloser, buf *bytes.Buffer) (io.ReadCloser, error) {
	if body == nil {
		return nil, nil
	}

	b, err := io.ReadAll(body)
	if err != nil {
		return io.NopCloser(iotest.ErrReader(err)), err
	}
	body.Close()
	if strings.Contains(buf.String(), "json") {
		json.Indent(buf, b, "", "  ")
	} else {
		buf.Write(b)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// DumpRequest returns a string representation of an HTTP request, automatically
// pretty printing JSON bodies for readability.
func DumpRequest(req *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	b, err := httputil.DumpRequest(req, false)

	if err == nil {
		buf.Write(b)
		req.Body, err = dumpBody(req.Body, &buf)
	}

	return buf.Bytes(), err
}

// DumpResponse returns a string representation of an HTTP response,
// automatically pretty printing JSON bodies for readability.
func DumpResponse(resp *http.Response) ([]byte, error) {
	var buf bytes.Buffer
	b, err := httputil.DumpResponse(resp, false)

	if err == nil {
		buf.Write(b)
		resp.Body, err = dumpBody(resp.Body, &buf)
	}

	return buf.Bytes(), err
}

// PrintRequest prints a string representation of an HTTP request to stdout,
// automatically pretty printing JSON bodies for readability.
func PrintRequest(req *http.Request) {
	b, _ := DumpRequest(req)
	// Turn `/r/n` into `/n` for more straightforward output that is also
	// compatible with Go's testable examples.
	b = bytes.ReplaceAll(b, []byte("\r"), []byte(""))
	fmt.Println(string(b))
}

// PrintResponse prints a string representation of an HTTP response to stdout,
// automatically pretty printing JSON bodies for readability.
func PrintResponse(resp *http.Response) {
	b, _ := DumpResponse(resp)
	// Turn `/r/n` into `/n` for more straightforward output that is also
	// compatible with Go's testable examples.
	b = bytes.ReplaceAll(b, []byte("\r"), []byte(""))
	fmt.Println(string(b))
}

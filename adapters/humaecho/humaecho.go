package humaecho

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ross96D/huma"
)

// MultipartMaxMemory is the maximum memory to use when parsing multipart
// form data.
var MultipartMaxMemory int64 = 8 * 1024

type echoCtx struct {
	op     *huma.Operation
	orig   echo.Context
	status int
}

// check that echoCtx implements huma.Context
var _ huma.Context = &echoCtx{}

func (c *echoCtx) Operation() *huma.Operation {
	return c.op
}

func (c *echoCtx) Context() context.Context {
	return c.orig.Request().Context()
}

func (c *echoCtx) Method() string {
	return c.orig.Request().Method
}

func (c *echoCtx) Host() string {
	return c.orig.Request().Host
}

func (c *echoCtx) URL() url.URL {
	return *c.orig.Request().URL
}

func (c *echoCtx) Param(name string) string {
	return c.orig.Param(name)
}

func (c *echoCtx) Query(name string) string {
	return c.orig.QueryParam(name)
}

func (c *echoCtx) Header(name string) string {
	return c.orig.Request().Header.Get(name)
}

func (c *echoCtx) EachHeader(cb func(name, value string)) {
	for name, values := range c.orig.Request().Header {
		for _, value := range values {
			cb(name, value)
		}
	}
}

func (c *echoCtx) BodyReader() io.Reader {
	return c.orig.Request().Body
}

func (c *echoCtx) GetMultipartForm() (*multipart.Form, error) {
	err := c.orig.Request().ParseMultipartForm(MultipartMaxMemory)
	return c.orig.Request().MultipartForm, err
}

func (c *echoCtx) SetReadDeadline(deadline time.Time) error {
	return huma.SetReadDeadline(c.orig.Response(), deadline)
}

func (c *echoCtx) SetStatus(code int) {
	c.status = code
	c.orig.Response().WriteHeader(code)
}

func (c *echoCtx) Status() int {
	return c.status
}

func (c *echoCtx) AppendHeader(name, value string) {
	c.orig.Response().Header().Add(name, value)
}

func (c *echoCtx) SetHeader(name, value string) {
	c.orig.Response().Header().Set(name, value)
}

func (c *echoCtx) BodyWriter() io.Writer {
	return c.orig.Response()
}

type router interface {
	Add(method, path string, handler echo.HandlerFunc, middlewares ...echo.MiddlewareFunc) *echo.Route
}

type echoAdapter struct {
	http.Handler
	router router
}

func (a *echoAdapter) Handle(op *huma.Operation, handler func(huma.Context)) {
	// Convert {param} to :param
	path := op.Path
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	a.router.Add(op.Method, path, func(c echo.Context) error {
		ctx := &echoCtx{op: op, orig: c}
		handler(ctx)
		return nil
	})
}

func New(r *echo.Echo, config huma.Config) huma.API {
	return huma.NewAPI(config, &echoAdapter{Handler: r, router: r})
}

// NewWithGroup creates a new Huma API using the provided Echo router and group,
// letting you mount the API at a sub-path. Can be used in combination with
// the `OpenAPI.Servers` field to set the correct base URL for the API / docs
// / schemas / etc.
func NewWithGroup(r *echo.Echo, g *echo.Group, config huma.Config) huma.API {
	return huma.NewAPI(config, &echoAdapter{Handler: r, router: g})
}

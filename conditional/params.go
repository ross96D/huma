package conditional

import (
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma"
)

// trimETag removes the quotes and `W/` prefix for incoming ETag values to
// make comparisons easier.
func trimETag(value string) string {
	if strings.HasPrefix(value, "W/") && len(value) > 2 {
		value = value[2:]
	}
	return strings.Trim(value, "\"")
}

// Params allow clients to send ETags or times to make a read or
// write conditional based on the state of the resource on the server, e.g.
// when it was last modified. This is useful for determining when a cache
// should be updated or to prevent multiple writers from overwriting each
// other's changes.
type Params struct {
	IfMatch           []string  `header:"If-Match" doc:"Succeeds if the server's resource matches one of the passed values."`
	IfNoneMatch       []string  `header:"If-None-Match" doc:"Succeeds if the server's resource matches none of the passed values. On writes, the special value * may be used to match any existing value."`
	IfModifiedSince   time.Time `header:"If-Modified-Since" doc:"Succeeds if the server's resource date is more recent than the passed date."`
	IfUnmodifiedSince time.Time `header:"If-Unmodified-Since" doc:"Succeeds if the server's resource date is older or the same as the passed date."`

	// isWrite tracks whether we should emit errors vs. a 304 Not Modified from
	// the `PreconditionFailed` method.
	isWrite bool
}

func (p *Params) Resolve(ctx huma.Context, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		p.isWrite = true
	}
}

// HasConditionalParams returns true if any conditional request headers have
// been set on the incoming request.
func (p *Params) HasConditionalParams() bool {
	return len(p.IfMatch) > 0 || len(p.IfNoneMatch) > 0 || !p.IfModifiedSince.IsZero() || !p.IfUnmodifiedSince.IsZero()
}

// PreconditionFailed returns false if no conditional headers are present, or if
// the values passed fail based on the conditional read/write rules. See also:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Conditional_requests.
// This method assumes there is some fast/efficient way to get a resource's
// current ETag and/or last-modified time before it is run.
func (p *Params) PreconditionFailed(ctx huma.Context, etag string, modified time.Time) bool {
	failed := false

	foundMsg := "found no existing resource"
	if etag != "" {
		foundMsg = "found resource with ETag " + etag
	}

	// If-None-Match fails on the first match. The `*` is a special case meaning
	// to match any existing value.
	for _, match := range p.IfNoneMatch {
		trimmed := trimETag(match)
		if trimmed == etag || (trimmed == "*" && etag != "") {
			// We matched an existing resource, abort!
			if p.isWrite {
				ctx.AddError(&huma.ErrorDetail{
					Message:  "If-None-Match: " + match + " precondition failed, " + foundMsg,
					Location: "request.headers.If-None-Match",
					Value:    match,
				})
			}
			failed = true
		}
	}

	// If-Match fails if none of the passed ETags matches the current resource.
	if len(p.IfMatch) > 0 {
		found := false
		for _, match := range p.IfMatch {
			if trimETag(match) == etag {
				found = true
				break
			}
		}

		if !found {
			// We did not match the expected resource, abort!
			if p.isWrite {
				ctx.AddError(&huma.ErrorDetail{
					Message:  "If-Match precondition failed, " + foundMsg,
					Location: "request.headers.If-Match",
					Value:    p.IfMatch,
				})
			}
			failed = true
		}
	}

	if !p.IfModifiedSince.IsZero() && !modified.After(p.IfModifiedSince) {
		// Resource was modified *before* the date that was passed, abort!
		if p.isWrite {
			ctx.AddError(&huma.ErrorDetail{
				Message:  "If-Modified-Since: " + p.IfModifiedSince.Format(http.TimeFormat) + " precondition failed, resource was modified at " + modified.Format(http.TimeFormat),
				Location: "request.headers.If-Modified-Since",
				Value:    p.IfModifiedSince.Format(http.TimeFormat),
			})
		}
		failed = true
	}

	if !p.IfUnmodifiedSince.IsZero() && modified.After(p.IfUnmodifiedSince) {
		// Resource was modified *after* the date that was passed, abort!
		if p.isWrite {
			ctx.AddError(&huma.ErrorDetail{
				Message:  "If-Unmodified-Since: " + p.IfUnmodifiedSince.Format(http.TimeFormat) + " precondition failed, resource was modified at " + modified.Format(http.TimeFormat),
				Location: "request.headers.If-Unmodified-Since",
				Value:    p.IfUnmodifiedSince.Format(http.TimeFormat),
			})
		}
		failed = true
	}

	if failed {
		if p.isWrite {
			ctx.WriteError(http.StatusPreconditionFailed, http.StatusText(http.StatusPreconditionFailed))
		} else {
			ctx.WriteHeader(http.StatusNotModified)
		}

		return true
	}

	return false
}
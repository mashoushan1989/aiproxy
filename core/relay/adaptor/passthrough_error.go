package adaptor

import (
	"fmt"
	"net/http"
)

// Aiproxy-injected response headers and error type identifiers, used wherever
// aiproxy needs to make its presence visible to the client without disturbing
// the upstream response schema.
const (
	// HeaderAiproxyRequestID exposes aiproxy's own request id on every response.
	// Passthrough adaptors forward upstream headers verbatim, which can overwrite
	// X-Request-Id with the upstream value; this dedicated header guarantees a
	// stable correlation id regardless of upstream behavior.
	HeaderAiproxyRequestID = "X-Aiproxy-Request-Id"
	// HeaderAiproxyRetryCount tells the client which attempt produced the
	// response. "0" means the first attempt succeeded or returned an error
	// the client received as-is.
	HeaderAiproxyRetryCount = "X-Aiproxy-Retry-Count"
	// HeaderAiproxyBodyTruncated signals that the upstream error body exceeded
	// errorBodyMaxBytes and was truncated before forwarding. Body is still
	// byte-exact up to the cap, but clients should treat JSON parse failures as
	// "too large" rather than "upstream sent malformed JSON".
	HeaderAiproxyBodyTruncated = "X-Aiproxy-Body-Truncated"
	// ErrorTypeAiproxyTimeout marks errors synthesized by aiproxy when its
	// own safety-cap timeout fires before the upstream responds. Lets clients
	// distinguish aiproxy bookkeeping from real upstream timeouts.
	ErrorTypeAiproxyTimeout = "aiproxy_timeout"
)

// PassthroughError carries an upstream error response verbatim (status code,
// headers, body) so the client receives the byte-exact response from the
// upstream provider. Use this in passthrough adaptors (PPIO, Novita, etc.)
// instead of WrapperError to honor the extreme-passthrough principle: aiproxy
// must not rewrite or wrap upstream errors, lest SDKs that rely on the
// upstream's error schema misclassify the failure.
//
// Buffering (rather than streaming the error body directly to c.Writer) is
// intentional: aiproxy may retry to another channel, and we must not write
// any bytes to the client until we know which attempt is final.
type PassthroughError struct {
	statusCode int
	header     http.Header
	body       []byte
	truncated  bool
}

// NewPassthroughError returns an Error that, when handed to the relay error
// writer, will be forwarded to the client as the verbatim upstream response.
// header and body must already be fully read from the upstream response;
// truncated signals that body was cut off at the adaptor's read cap so WriteTo
// can add X-Aiproxy-Body-Truncated for client-side diagnostics.
func NewPassthroughError(statusCode int, header http.Header, body []byte, truncated bool) Error {
	return &PassthroughError{
		statusCode: statusCode,
		header:     header,
		body:       body,
		truncated:  truncated,
	}
}

func (e *PassthroughError) StatusCode() int { return e.statusCode }

// WriteTo forwards the captured upstream response (status, headers minus
// hop-by-hop, body) to the client. Encapsulates the filter loop so callers
// do not reach into UpstreamHeader/UpstreamBody themselves.
func (e *PassthroughError) WriteTo(w http.ResponseWriter) {
	for k, vs := range e.header {
		if HopByHopHeaders[k] {
			continue
		}

		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}

	if e.truncated {
		w.Header().Set(HeaderAiproxyBodyTruncated, "true")
	}

	w.WriteHeader(e.statusCode)
	_, _ = w.Write(e.body)
}

// UpstreamHeader returns the headers captured from the upstream response.
// The relay error writer forwards these to the client (minus hop-by-hop).
func (e *PassthroughError) UpstreamHeader() http.Header { return e.header }

// UpstreamBody returns the verbatim upstream error body bytes.
func (e *PassthroughError) UpstreamBody() []byte { return e.body }

// MarshalJSON returns the raw upstream body. This is what gets logged into
// the request log when the relay layer needs a JSON form of the error;
// keeping it byte-identical means logs match what the client saw.
func (e *PassthroughError) MarshalJSON() ([]byte, error) {
	if len(e.body) == 0 {
		return []byte("{}"), nil
	}

	return e.body, nil
}

func (e *PassthroughError) Error() string {
	return fmt.Sprintf("upstream error: status=%d body_len=%d", e.statusCode, len(e.body))
}

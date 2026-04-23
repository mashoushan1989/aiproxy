package adaptor

import (
	"fmt"

	"github.com/bytedance/sonic"
)

// HopByHopHeaders lists HTTP/1.1 hop-by-hop headers (RFC 7230 §6.1) that must
// not be forwarded between client and upstream when proxying. Shared by the
// passthrough adaptor (request → upstream) and the relay error writer
// (upstream error → client) to keep filtering logic consistent.
//
// Keys are in http.CanonicalHeaderKey form to match http.Header lookup.
var HopByHopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailers":            true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

type BasicError[T any] struct {
	error      T
	statusCode int
}

func (e BasicError[T]) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(e.error)
}

func (e BasicError[T]) StatusCode() int {
	return e.statusCode
}

func (e BasicError[T]) Error() string {
	return fmt.Sprintf("status code: %d, error: %v", e.statusCode, e.error)
}

func NewError[T any](statusCode int, err T) Error {
	return BasicError[T]{
		error:      err,
		statusCode: statusCode,
	}
}

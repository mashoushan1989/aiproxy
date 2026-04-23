package passthrough

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

// AnthropicAdaptor specializes the generic passthrough.Adaptor for Anthropic-
// protocol upstreams (api.anthropic.com, PPIO /anthropic, Novita /anthropic/v1).
//
// Composed (not registered) inside anthropic.Adaptor; activated when the
// channel config has pure_passthrough=true. Keeping it unregistered avoids a
// new channel type number + database migration.
//
// Overrides three pieces vs the base:
//   - SupportMode/NativeMode: only mode.Anthropic.
//   - SetupRequestHeader: X-Api-Key (Anthropic auth scheme), not Bearer.
//   - DoResponse: dual-buffer SSE scan via DoAnthropicPassthrough.
//
// Inherits ConvertRequest, GetRequestURL, DoRequest unchanged — their
// header-blacklist forwarding, stripV1 + query preservation, and verbatim body
// semantics are already correct for Anthropic.
type AnthropicAdaptor struct {
	Adaptor
}

func (a *AnthropicAdaptor) SupportMode(m mode.Mode) bool {
	return m == mode.Anthropic
}

// NativeMode overrides the base (which returns false for Anthropic to
// deprioritize generic passthrough versus the dedicated anthropic adaptor) —
// this adaptor IS the Anthropic-channel path.
func (a *AnthropicAdaptor) NativeMode(m mode.Mode) bool {
	return m == mode.Anthropic
}

// SetupRequestHeader installs Anthropic's X-Api-Key auth. Base sets
// Authorization: Bearer which Anthropic and PPIO/Novita's compat endpoints reject.
func (a *AnthropicAdaptor) SetupRequestHeader(
	m *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	req.Header.Set("X-Api-Key", m.Channel.Key)

	return nil
}

func (a *AnthropicAdaptor) DoResponse(
	m *meta.Meta,
	_ adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	return DoAnthropicPassthrough(m, c, resp)
}

package middleware

import "github.com/labring/aiproxy/core/model"

// EnterprisePriceResolver is set by enterprise build tag to apply request-time
// commercial pricing policies after group-level model price overrides.
var EnterprisePriceResolver func(group model.GroupCache, requestModel string, fallback model.Price) (model.Price, error)

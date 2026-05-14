package controller

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

func TestResolveRequestPriceAppliesEnterpriseResolver(t *testing.T) {
	originalResolver := middleware.EnterprisePriceResolver
	t.Cleanup(func() {
		middleware.EnterprisePriceResolver = originalResolver
	})

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.Group, model.GroupCache{ID: "engineering"})

	basePrice := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.03),
		InputPriceUnit: model.ZeroNullInt64(1),
	}
	promotedPrice := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.01),
		InputPriceUnit: model.ZeroNullInt64(1),
	}

	called := false
	middleware.EnterprisePriceResolver = func(group model.GroupCache, requestModel string, fallback model.Price) (model.Price, error) {
		called = true
		if group.ID != "engineering" {
			t.Fatalf("group = %q, want engineering", group.ID)
		}
		if requestModel != "pa/gpt-5.5" {
			t.Fatalf("model = %q, want pa/gpt-5.5", requestModel)
		}
		if fallback.InputPrice != basePrice.InputPrice {
			t.Fatalf("fallback price = %v, want %v", fallback.InputPrice, basePrice.InputPrice)
		}
		return promotedPrice, nil
	}

	got, err := resolveRequestPrice(c, model.ModelConfig{Model: "pa/gpt-5.5"}, func(*gin.Context, model.ModelConfig) (model.Price, error) {
		return basePrice, nil
	})
	if err != nil {
		t.Fatalf("resolve price: %v", err)
	}
	if !called {
		t.Fatalf("enterprise price resolver was not called")
	}
	if got.InputPrice != promotedPrice.InputPrice {
		t.Fatalf("price = %v, want %v", got.InputPrice, promotedPrice.InputPrice)
	}
}

func TestResolveRequestPriceKeepsGroupOverrideBeforeEnterpriseResolver(t *testing.T) {
	originalResolver := middleware.EnterprisePriceResolver
	t.Cleanup(func() {
		middleware.EnterprisePriceResolver = originalResolver
	})

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	overridePrice := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.005),
		InputPriceUnit: model.ZeroNullInt64(1),
	}
	c.Set(middleware.Group, model.GroupCache{
		ID: "engineering",
		ModelConfigs: map[string]model.GroupModelConfig{
			"pa/gpt-5.5": {
				Model:         "pa/gpt-5.5",
				OverridePrice: true,
				Price:         overridePrice,
			},
		},
	})

	middleware.EnterprisePriceResolver = func(model.GroupCache, string, model.Price) (model.Price, error) {
		t.Fatalf("enterprise resolver should not be called when group price override exists")
		return model.Price{}, nil
	}

	endpointPrice := model.Price{
		InputPrice:     model.ZeroNullFloat64(0.004),
		InputPriceUnit: model.ZeroNullInt64(1),
	}
	got, err := resolveRequestPrice(c, model.ModelConfig{
		Model: "pa/gpt-5.5",
		Price: overridePrice,
	}, func(*gin.Context, model.ModelConfig) (model.Price, error) {
		return endpointPrice, nil
	})
	if err != nil {
		t.Fatalf("resolve price: %v", err)
	}
	if got.InputPrice != endpointPrice.InputPrice {
		t.Fatalf("price = %v, want endpoint-specific group override %v", got.InputPrice, endpointPrice.InputPrice)
	}
}

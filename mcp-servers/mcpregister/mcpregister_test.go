package mcpregister

import (
	"testing"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

func TestDefaultRegistryExcludesAGPLAcademicSearch(t *testing.T) {
	registered := mcpservers.Servers()

	if _, ok := registered["academic-search"]; ok {
		t.Fatal("academic-search must not be included in the default registry")
	}

	if _, ok := registered["web-search"]; !ok {
		t.Fatal("web-search should remain available as the academic search alternative")
	}
}

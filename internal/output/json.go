package output

import (
	"encoding/json"
	"io"

	"github.com/faultlens/faultlens/internal/engine"
)

// RenderJSON writes the full analysis result as a stable, tool-friendly JSON
// document. The schema is fixed by the struct tags on engine.Result and its
// nested types, so GitHub Actions and other consumers can rely on it.
func RenderJSON(w io.Writer, res *engine.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

package output

import "fmt"

type Format string

const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
)

func (f Format) Validate() error {
	switch f {
	case FormatHuman, FormatJSON:
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", f)
	}
}

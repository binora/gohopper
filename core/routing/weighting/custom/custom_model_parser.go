package custom

// ModelParser mirrors GraphHopper's CustomModelParser boundary.
// Full DSL parity implementation is pending.
type ModelParser struct{}

func NewModelParser() *ModelParser {
	return &ModelParser{}
}

package ch

// PreparationHandler is the CH preparation entry point mirroring GraphHopper's CHPreparationHandler.
// Implementation will move from placeholder to GH11-compatible contraction preprocessing.
type PreparationHandler struct{}

func NewPreparationHandler() *PreparationHandler {
	return &PreparationHandler{}
}

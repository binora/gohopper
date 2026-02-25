package lm

// PreparationHandler is the LM preparation entry point mirroring GraphHopper's LMPreparationHandler.
// Implementation will move from placeholder to GH11-compatible landmarks preprocessing.
type PreparationHandler struct{}

func NewPreparationHandler() *PreparationHandler {
	return &PreparationHandler{}
}

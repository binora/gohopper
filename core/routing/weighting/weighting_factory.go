package weighting

import "gohopper/core/config"

// WeightingFactory creates Weighting instances from profile configuration.
type WeightingFactory interface {
	// CreateWeighting creates a Weighting for the given profile.
	// hints provides additional request-level parameters.
	// disableTurnCosts can be used to explicitly create the weighting without turn costs,
	// e.g. for node-based graph traversal like LM preparation.
	CreateWeighting(profile config.Profile, hints map[string]any, disableTurnCosts bool) Weighting
}

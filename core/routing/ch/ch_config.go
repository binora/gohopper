package ch

import (
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
)

// CHConfig is a configuration container for CH preparation.
type CHConfig struct {
	name      string
	weighting weighting.Weighting
	edgeBased bool
}

func NewCHConfigNodeBased(name string, w weighting.Weighting) *CHConfig {
	return &CHConfig{name: name, weighting: w, edgeBased: false}
}

func NewCHConfigEdgeBased(name string, w weighting.Weighting) *CHConfig {
	return &CHConfig{name: name, weighting: w, edgeBased: true}
}

func (c *CHConfig) GetName() string                   { return c.name }
func (c *CHConfig) GetWeighting() weighting.Weighting { return c.weighting }
func (c *CHConfig) IsEdgeBased() bool                 { return c.edgeBased }

func (c *CHConfig) GetTraversalMode() routingutil.TraversalMode {
	if c.edgeBased {
		return routingutil.EdgeBased
	}
	return routingutil.NodeBased
}

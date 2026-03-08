package webapi

import "slices"

// CustomModel defines speed and priority adjustments for custom routing.
type CustomModel struct {
	DistanceInfluence *float64    `json:"distance_influence,omitempty"`
	HeadingPenalty    *float64    `json:"heading_penalty,omitempty"`
	Speed             []Statement `json:"speed,omitempty"`
	Priority          []Statement `json:"priority,omitempty"`
}

func NewCustomModel() *CustomModel {
	return &CustomModel{}
}

func (cm *CustomModel) AddToSpeed(st Statement) *CustomModel {
	cm.Speed = append(cm.Speed, st)
	return cm
}

func (cm *CustomModel) AddToPriority(st Statement) *CustomModel {
	cm.Priority = append(cm.Priority, st)
	return cm
}

func (cm *CustomModel) SetDistanceInfluence(v float64) *CustomModel {
	cm.DistanceInfluence = &v
	return cm
}

func (cm *CustomModel) SetHeadingPenalty(v float64) *CustomModel {
	cm.HeadingPenalty = &v
	return cm
}

func MergeCustomModels(base, query *CustomModel) *CustomModel {
	if query == nil {
		return copyCustomModel(base)
	}
	merged := copyCustomModel(base)
	if query.DistanceInfluence != nil {
		merged.DistanceInfluence = query.DistanceInfluence
	}
	if query.HeadingPenalty != nil {
		merged.HeadingPenalty = query.HeadingPenalty
	}
	merged.Speed = append(merged.Speed, query.Speed...)
	merged.Priority = append(merged.Priority, query.Priority...)
	return merged
}

func copyCustomModel(cm *CustomModel) *CustomModel {
	c := &CustomModel{
		Speed:    slices.Clone(cm.Speed),
		Priority: slices.Clone(cm.Priority),
	}
	if cm.DistanceInfluence != nil {
		v := *cm.DistanceInfluence
		c.DistanceInfluence = &v
	}
	if cm.HeadingPenalty != nil {
		v := *cm.HeadingPenalty
		c.HeadingPenalty = &v
	}
	return c
}

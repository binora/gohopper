package webapi

type ResponsePath struct {
	Distance         float64          `json:"distance"`
	Time             int64            `json:"time"`
	Points           any              `json:"points,omitempty"`
	PointsEncoded    bool             `json:"points_encoded"`
	BBox             [4]float64       `json:"bbox,omitempty"`
	SnappedWaypoints any              `json:"snapped_waypoints,omitempty"`
	Instructions     []Instruction    `json:"instructions,omitempty"`
	Details          map[string]any   `json:"details,omitempty"`
	Ascend           float64          `json:"ascend,omitempty"`
	Descend          float64          `json:"descend,omitempty"`
	Weight           float64          `json:"weight,omitempty"`
	Description      []string         `json:"description,omitempty"`
	PointsOrder      []int            `json:"points_order,omitempty"`
	Legs             []map[string]any `json:"legs,omitempty"`
}

type Instruction struct {
	Text      string    `json:"text"`
	Street    string    `json:"street_name,omitempty"`
	Distance  float64   `json:"distance"`
	Time      int64     `json:"time"`
	Interval  [2]int    `json:"interval"`
	Sign      int       `json:"sign"`
	Exit      *int      `json:"exit_number,omitempty"`
	Exited    *bool     `json:"exited,omitempty"`
	TurnAngle *float64  `json:"turn_angle,omitempty"`
	ExtraInfo []float64 `json:"-"`
}

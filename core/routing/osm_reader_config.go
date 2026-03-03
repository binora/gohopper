package routing

import "math"

// OSMReaderConfig holds configuration for OSM import.
type OSMReaderConfig struct {
	IgnoredHighways                []string
	ParseWayNames                  bool
	PreferredLanguage              string
	MaxWayPointDistance             float64
	ElevationMaxWayPointDistance    float64
	SmoothElevation                string
	SmoothElevationAverageWindowSize float64
	RamerElevationSmoothingMax     int
	LongEdgeSamplingDistance       float64
	WorkerThreads                  int
	DefaultElevation               float64
}

func NewOSMReaderConfig() OSMReaderConfig {
	return OSMReaderConfig{
		ParseWayNames:                  true,
		MaxWayPointDistance:             0.5,
		ElevationMaxWayPointDistance:    math.MaxFloat64,
		SmoothElevationAverageWindowSize: 150.0,
		RamerElevationSmoothingMax:     5,
		LongEdgeSamplingDistance:       math.MaxFloat64,
		WorkerThreads:                  2,
	}
}

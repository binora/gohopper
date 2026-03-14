package ch

// OnFlyStatisticsCalculator computes running mean and variance using Welford's algorithm.
type OnFlyStatisticsCalculator struct {
	count          int64
	mean           float64
	varianceHelper float64
}

func (c *OnFlyStatisticsCalculator) AddObservation(value int64) {
	c.count++
	delta := float64(value) - c.mean
	c.mean += delta / float64(c.count)
	newDelta := float64(value) - c.mean
	c.varianceHelper += delta * newDelta
}

func (c *OnFlyStatisticsCalculator) GetCount() int64    { return c.count }
func (c *OnFlyStatisticsCalculator) GetMean() float64    { return c.mean }
func (c *OnFlyStatisticsCalculator) GetVariance() float64 { return c.varianceHelper / float64(c.count) }

func (c *OnFlyStatisticsCalculator) Reset() {
	c.count = 0
	c.mean = 0
	c.varianceHelper = 0
}

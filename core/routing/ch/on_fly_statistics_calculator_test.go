package ch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnFlyStatisticsCalculator(t *testing.T) {
	calc := &OnFlyStatisticsCalculator{}
	calc.AddObservation(5)
	calc.AddObservation(7)
	calc.AddObservation(10)
	calc.AddObservation(12)
	calc.AddObservation(17)
	assert.InDelta(t, 10.2, calc.GetMean(), 1e-6)
	assert.InDelta(t, 17.36, calc.GetVariance(), 1e-6)
}

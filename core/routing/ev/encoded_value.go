package ev

// EncodedValue defines how to store and read values from a list of integers.
type EncodedValue interface {
	// Init allocates bit space and returns the number of used bits.
	Init(cfg *InitializerConfig) int
	GetName() string
	IsStoreTwoDirections() bool
}

// InitializerConfig tracks bit allocation across multiple EncodedValues.
type InitializerConfig struct {
	DataIndex int
	Shift     int
	NextShift int
	BitMask   int32
}

func NewInitializerConfig() *InitializerConfig {
	return &InitializerConfig{
		DataIndex: -1,
		Shift:     32,
		NextShift: 32,
	}
}

// Next allocates space for usedBits, updating Shift and DataIndex.
func (c *InitializerConfig) Next(usedBits int) {
	c.Shift = c.NextShift
	if (c.Shift-1+usedBits)/32 > (c.Shift-1)/32 {
		c.DataIndex++
		c.Shift = 0
	}
	c.BitMask = int32((int64(1) << usedBits) - 1)
	c.BitMask <<= c.Shift
	c.NextShift = c.Shift + usedBits
}

func (c *InitializerConfig) requiredBits() int {
	return c.DataIndex*32 + c.NextShift
}

// GetRequiredInts returns the number of int32 slots needed.
func (c *InitializerConfig) GetRequiredInts() int {
	return (c.requiredBits() + 31) / 32
}

// GetRequiredBytes returns the number of bytes needed.
func (c *InitializerConfig) GetRequiredBytes() int {
	return (c.requiredBits() + 7) / 8
}

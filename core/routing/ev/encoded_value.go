package ev

// EncodedValue defines how to store and read values from a list of integers.
type EncodedValue interface {
	// Init sets the dataIndex and shift of this EncodedValue and
	// advances the config via InitializerConfig.Next.
	// Returns the number of used bits.
	Init(init *InitializerConfig) int

	// GetName returns the hierarchical name (e.g. "vehicle.type").
	GetName() string

	// IsStoreTwoDirections returns true if this EncodedValue stores
	// a separate value for the reverse direction.
	IsStoreTwoDirections() bool
}

// InitializerConfig tracks bit allocation across multiple EncodedValues.
type InitializerConfig struct {
	DataIndex int
	Shift     int
	NextShift int
	BitMask   int32
}

// NewInitializerConfig creates a config with the same initial state as Java's.
func NewInitializerConfig() *InitializerConfig {
	return &InitializerConfig{
		DataIndex: -1,
		Shift:     32,
		NextShift: 32,
		BitMask:   0,
	}
}

// Next allocates space for the given number of bits, updating Shift and DataIndex.
func (c *InitializerConfig) Next(usedBits int) {
	c.Shift = c.NextShift
	if (c.Shift-1+usedBits)/32 > (c.Shift-1)/32 {
		c.DataIndex++
		c.Shift = 0
	}
	// Use int64 so the shift works when usedBits == 32.
	c.BitMask = int32((int64(1) << usedBits) - 1)
	c.BitMask <<= c.Shift
	c.NextShift = c.Shift + usedBits
}

// requiredBits returns the total number of bits allocated so far.
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

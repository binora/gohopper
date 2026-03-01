package util

// TransportationMode defines disjoint ways of transportation used to create
// and populate encoded values from a data source like OpenStreetMap.
type TransportationMode int

const (
	TransportationModeOther      TransportationMode = iota
	TransportationModeFoot
	TransportationModeVehicle
	TransportationModeBike
	TransportationModeCar
	TransportationModeMotorcycle
	TransportationModeHGV
	TransportationModePSV
	TransportationModeBus
	TransportationModeHOV
)

func (t TransportationMode) IsMotorVehicle() bool {
	switch t {
	case TransportationModeCar, TransportationModeMotorcycle,
		TransportationModeHGV, TransportationModePSV,
		TransportationModeBus, TransportationModeHOV:
		return true
	default:
		return false
	}
}

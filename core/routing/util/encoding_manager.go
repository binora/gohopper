package util

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gohopper/core/routing/ev"
	ghutil "gohopper/core/util"
)

// PropertyStore is the interface used by EncodingManager for reading/writing properties.
// Implemented by storage.StorableProperties.
type PropertyStore interface {
	Get(key string) string
	Put(key string, val any)
	ContainsVersion() bool
}

// EncodingManager manages encoded values for edges and turn costs.
type EncodingManager struct {
	encodedValues     []ev.EncodedValue
	encodedValueIndex map[string]ev.EncodedValue
	turnEncodedValues []ev.EncodedValue
	turnEVIndex       map[string]ev.EncodedValue
	BytesForFlags     int
	IntsForTurnCostFlags int
}

var _ ev.EncodedValueLookup = (*EncodingManager)(nil)

func newEncodingManager() *EncodingManager {
	return &EncodingManager{
		encodedValueIndex: make(map[string]ev.EncodedValue),
		turnEVIndex:       make(map[string]ev.EncodedValue),
	}
}

func Start() *Builder {
	return &Builder{
		edgeConfig:     ev.NewInitializerConfig(),
		turnCostConfig: ev.NewInitializerConfig(),
		em:             newEncodingManager(),
	}
}

// Builder accumulates encoded values and produces an EncodingManager.
// Use Start() to create a new Builder.
type Builder struct {
	edgeConfig     *ev.InitializerConfig
	turnCostConfig *ev.InitializerConfig
	em             *EncodingManager
}

func (b *Builder) Add(encodedValue ev.EncodedValue) *Builder {
	b.checkNotBuiltAlready()
	if b.em.HasEncodedValue(encodedValue.GetName()) {
		panic("EncodedValue already exists: " + encodedValue.GetName())
	}
	if b.em.HasTurnEncodedValue(encodedValue.GetName()) {
		panic("Already defined as 'turn'-EncodedValue: " + encodedValue.GetName())
	}
	encodedValue.Init(b.edgeConfig)
	b.em.encodedValues = append(b.em.encodedValues, encodedValue)
	b.em.encodedValueIndex[encodedValue.GetName()] = encodedValue
	return b
}

func (b *Builder) AddTurnCostEncodedValue(turnCostEnc ev.EncodedValue) *Builder {
	b.checkNotBuiltAlready()
	if b.em.HasTurnEncodedValue(turnCostEnc.GetName()) {
		panic("Already defined: " + turnCostEnc.GetName())
	}
	if b.em.HasEncodedValue(turnCostEnc.GetName()) {
		panic("Already defined as EncodedValue: " + turnCostEnc.GetName())
	}
	turnCostEnc.Init(b.turnCostConfig)
	b.em.turnEncodedValues = append(b.em.turnEncodedValues, turnCostEnc)
	b.em.turnEVIndex[turnCostEnc.GetName()] = turnCostEnc
	return b
}

func (b *Builder) checkNotBuiltAlready() {
	if b.em == nil {
		panic("Cannot call method after Builder.Build() was called")
	}
}

func (b *Builder) Build() *EncodingManager {
	b.checkNotBuiltAlready()
	b.em.BytesForFlags = b.edgeConfig.GetRequiredBytes()
	b.em.IntsForTurnCostFlags = b.turnCostConfig.GetRequiredInts()
	result := b.em
	b.em = nil
	return result
}

func PutEncodingManagerIntoProperties(em *EncodingManager, props PropertyStore) {
	props.Put("graph.em.version", ghutil.VersionEM)
	props.Put("graph.em.bytes_for_flags", em.BytesForFlags)
	props.Put("graph.em.ints_for_turn_cost_flags", em.IntsForTurnCostFlags)
	props.Put("graph.encoded_values", em.ToEncodedValuesAsString())
	props.Put("graph.turn_encoded_values", em.ToTurnEncodedValuesAsString())
}

func FromProperties(props PropertyStore) *EncodingManager {
	if props.ContainsVersion() {
		panic("The GraphHopper file format is not compatible with the data you are " +
			"trying to load. You either need to use an older version of GraphHopper or run a new import")
	}

	versionStr := props.Get("graph.em.version")
	if versionStr == "" || versionStr != strconv.Itoa(ghutil.VersionEM) {
		stored := versionStr
		if stored == "" {
			stored = "missing"
		}
		panic(fmt.Sprintf("Incompatible encoding version. You need to use the same GraphHopper version "+
			"you used to import the graph. Stored encoding version: %s, used encoding version: %d",
			stored, ghutil.VersionEM))
	}

	encodedValueStr := props.Get("graph.encoded_values")
	evList := deserializeEncodedValueList(encodedValueStr)
	encodedValues := make([]ev.EncodedValue, 0, len(evList))
	encodedValueIndex := make(map[string]ev.EncodedValue, len(evList))
	for _, s := range evList {
		encodedValue, err := ev.DeserializeEncodedValue(s)
		if err != nil {
			panic(fmt.Sprintf("could not deserialize encoded value: %v", err))
		}
		if _, exists := encodedValueIndex[encodedValue.GetName()]; exists {
			panic("Duplicate encoded value name: " + encodedValue.GetName() + " in: graph.encoded_values=" + encodedValueStr)
		}
		encodedValues = append(encodedValues, encodedValue)
		encodedValueIndex[encodedValue.GetName()] = encodedValue
	}

	turnEncodedValueStr := props.Get("graph.turn_encoded_values")
	tevList := deserializeEncodedValueList(turnEncodedValueStr)
	turnEncodedValues := make([]ev.EncodedValue, 0, len(tevList))
	turnEVIndex := make(map[string]ev.EncodedValue, len(tevList))
	for _, s := range tevList {
		encodedValue, err := ev.DeserializeEncodedValue(s)
		if err != nil {
			panic(fmt.Sprintf("could not deserialize turn encoded value: %v", err))
		}
		if _, exists := turnEVIndex[encodedValue.GetName()]; exists {
			panic("Duplicate turn encoded value name: " + encodedValue.GetName() + " in: graph.turn_encoded_values=" + turnEncodedValueStr)
		}
		turnEncodedValues = append(turnEncodedValues, encodedValue)
		turnEVIndex[encodedValue.GetName()] = encodedValue
	}

	return &EncodingManager{
		encodedValues:        encodedValues,
		encodedValueIndex:    encodedValueIndex,
		turnEncodedValues:    turnEncodedValues,
		turnEVIndex:          turnEVIndex,
		BytesForFlags:        getIntegerProperty(props, "graph.em.bytes_for_flags"),
		IntsForTurnCostFlags: getIntegerProperty(props, "graph.em.ints_for_turn_cost_flags"),
	}
}

func getIntegerProperty(props PropertyStore, key string) int {
	s := props.Get(key)
	if s == "" {
		panic("Missing EncodingManager property: '" + key + "'")
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("invalid integer for property '%s': %s", key, s))
	}
	return v
}

func deserializeEncodedValueList(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(s), &list); err != nil {
		panic(fmt.Sprintf("could not deserialize encoded value list: %v", err))
	}
	return list
}

func (em *EncodingManager) HasEncodedValue(key string) bool {
	_, ok := em.encodedValueIndex[key]
	return ok
}

func (em *EncodingManager) HasTurnEncodedValue(key string) bool {
	_, ok := em.turnEVIndex[key]
	return ok
}

func (em *EncodingManager) GetEncodedValues() []ev.EncodedValue {
	out := make([]ev.EncodedValue, len(em.encodedValues))
	copy(out, em.encodedValues)
	return out
}

func (em *EncodingManager) GetEncodedValue(key string) ev.EncodedValue {
	v, ok := em.encodedValueIndex[key]
	if !ok {
		panic(fmt.Sprintf("Cannot find EncodedValue '%s' in collection: %s", key, em.encodedValueKeys()))
	}
	return v
}

func (em *EncodingManager) GetBooleanEncodedValue(key string) ev.BooleanEncodedValue {
	return em.GetEncodedValue(key).(ev.BooleanEncodedValue)
}

func (em *EncodingManager) GetIntEncodedValue(key string) ev.IntEncodedValue {
	return em.GetEncodedValue(key).(ev.IntEncodedValue)
}

func (em *EncodingManager) GetDecimalEncodedValue(key string) ev.DecimalEncodedValue {
	return em.GetEncodedValue(key).(ev.DecimalEncodedValue)
}

func (em *EncodingManager) GetStringEncodedValue(key string) *ev.StringEncodedValue {
	return em.GetEncodedValue(key).(*ev.StringEncodedValue)
}

func (em *EncodingManager) GetTurnEncodedValues() []ev.EncodedValue {
	out := make([]ev.EncodedValue, len(em.turnEncodedValues))
	copy(out, em.turnEncodedValues)
	return out
}

func (em *EncodingManager) GetTurnEncodedValue(key string) ev.EncodedValue {
	v, ok := em.turnEVIndex[key]
	if !ok {
		panic(fmt.Sprintf("Cannot find Turn-EncodedValue '%s' in collection: %s", key, em.turnEncodedValueKeys()))
	}
	return v
}

func (em *EncodingManager) GetTurnBooleanEncodedValue(key string) ev.BooleanEncodedValue {
	return em.GetTurnEncodedValue(key).(ev.BooleanEncodedValue)
}

func (em *EncodingManager) GetTurnDecimalEncodedValue(key string) ev.DecimalEncodedValue {
	return em.GetTurnEncodedValue(key).(ev.DecimalEncodedValue)
}

func (em *EncodingManager) GetTurnIntEncodedValue(key string) ev.IntEncodedValue {
	return em.GetTurnEncodedValue(key).(ev.IntEncodedValue)
}

func (em *EncodingManager) NeedsTurnCostsSupport() bool {
	return em.IntsForTurnCostFlags > 0
}

func (em *EncodingManager) GetVehicles() []string {
	var vehicles []string
	for _, e := range em.encodedValues {
		name := e.GetName()
		if !strings.HasSuffix(name, "_access") {
			continue
		}
		prefix := strings.TrimSuffix(name, "_access")
		speedKey := ev.VehicleSpeedKey(prefix)
		for _, e2 := range em.encodedValues {
			if strings.Contains(e2.GetName(), speedKey) {
				vehicles = append(vehicles, prefix)
				break
			}
		}
	}
	return vehicles
}

func (em *EncodingManager) ToEncodedValuesAsString() string {
	return serializeEVList(em.encodedValues)
}

func (em *EncodingManager) ToTurnEncodedValuesAsString() string {
	return serializeEVList(em.turnEncodedValues)
}

func serializeEVList(evs []ev.EncodedValue) string {
	strs := make([]string, len(evs))
	for i, e := range evs {
		s, err := ev.SerializeEncodedValue(e)
		if err != nil {
			panic(fmt.Sprintf("could not serialize encoded value %s: %v", e.GetName(), err))
		}
		strs[i] = s
	}
	data, err := json.Marshal(strs)
	if err != nil {
		panic(fmt.Sprintf("could not marshal encoded value list: %v", err))
	}
	return string(data)
}

func (em *EncodingManager) String() string {
	return strings.Join(em.GetVehicles(), ",")
}

func (em *EncodingManager) encodedValueKeys() string {
	keys := make([]string, 0, len(em.encodedValueIndex))
	for k := range em.encodedValueIndex {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}

func (em *EncodingManager) turnEncodedValueKeys() string {
	keys := make([]string, 0, len(em.turnEVIndex))
	for k := range em.turnEVIndex {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}

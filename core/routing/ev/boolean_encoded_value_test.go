package ev

import "testing"

func TestBit(t *testing.T) {
	config := NewInitializerConfig()
	intProp := NewIntEncodedValueImpl("somevalue", 5, false)
	intProp.Init(config)

	boolEV := NewSimpleBooleanEncodedValueDir("access", false)
	boolEV.Init(config)
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	boolEV.SetBool(false, edgeID, edgeIntAccess, false)
	if boolEV.GetBool(false, edgeID, edgeIntAccess) {
		t.Fatal("expected false")
	}
	boolEV.SetBool(false, edgeID, edgeIntAccess, true)
	if !boolEV.GetBool(false, edgeID, edgeIntAccess) {
		t.Fatal("expected true")
	}
}

func TestBitDirected(t *testing.T) {
	config := NewInitializerConfig()
	boolEV := NewSimpleBooleanEncodedValueDir("access", true)
	boolEV.Init(config)
	edgeIntAccess := NewArrayEdgeIntAccess(1)
	edgeID := 0

	boolEV.SetBool(false, edgeID, edgeIntAccess, false)
	boolEV.SetBool(true, edgeID, edgeIntAccess, true)

	if boolEV.GetBool(false, edgeID, edgeIntAccess) {
		t.Fatal("expected false for forward")
	}
	if !boolEV.GetBool(true, edgeID, edgeIntAccess) {
		t.Fatal("expected true for reverse")
	}
}

package util

import (
	"math"
	"strings"
	"testing"
)

var points1 = "[[11.571499218899739,49.945605917549265],[11.571664621792689,49.94570668665409],[11.571787742639804,49.94578156499077],[11.572065649302282,49.94590338198625],[11.572209445511016,49.94595944760649],[11.57229438213172,49.94598850487147]," +
	"[11.573315297960832,49.946237913062525],[11.57367665112786,49.946338495902836],[11.573895511937787,49.94641784458796],[11.574013417378367,49.94646347939514],[11.574228180368875,49.94654916107392],[11.574703899950622,49.94677509993557]," +
	"[11.575003599561832,49.946924670344394],[11.575434615658997,49.94711838544425],[11.575559971680342,49.94716010869652],[11.57563783024932,49.947186185729194],[11.57609697228887,49.94727875919518],[11.57656188852851,49.947290121330845]," +
	"[11.576840167720023,49.94727782787258],[11.576961425921949,49.94725827009808],[11.577226852861648,49.947215242994176],[11.577394863457863,49.94717668623872],[11.577511092517772,49.94715005041249],[11.577635517216523,49.947112238715114]," +
	"[11.577917149169382,49.94702655703634],[11.577969116970207,49.947010724552214],[11.578816061738493,49.94673523932849],[11.579533552666014,49.94648974269233],[11.580073719771365,49.946299007824784],[11.580253092503245,49.946237913062525]," +
	"[11.580604946179799,49.94608871518274],[11.580740546749693,49.94603041438826]]"

var points2 = "[[9.961074440801317,50.203764443183644],[9.96106605889796,50.20365789987872],[9.960999562464645,50.20318963087774],[9.96094144793469,50.202952888673984],[9.96223002587773,50.20267889356641],[9.962200968612752,50.20262022024289]," +
	"[9.961859918278305,50.201853928011374],[9.961668810881722,50.20138565901039],[9.96216874485095,50.20128507617008],[9.961953795595925,50.20088553877664],[9.961899033827313,50.200686794534775],[9.961716680863127,50.20014066696481],[9.961588158344957,50.199798499043254]]"

func TestParse2DJSON(t *testing.T) {
	pl := NewPointList(0, false)
	pl.Parse2DJSON("[[11.571499218899739,49.945605917549265],[11.571664621792689,49.94570668665409]]")
	assertNear(t, 49.945605917549265, pl.GetLat(0), 1e-6)
	assertNear(t, 11.571499218899739, pl.GetLon(0), 1e-6)
	assertNear(t, 49.94570668665409, pl.GetLat(1), 1e-6)
	assertNear(t, 11.571664621792689, pl.GetLon(1), 1e-6)
}

func TestPathSimplify(t *testing.T) {
	pl := NewPointList(0, false)
	pl.Parse2DJSON(points1)
	if pl.Size() != 32 {
		t.Fatalf("expected 32 points, got %d", pl.Size())
	}
	NewRamerDouglasPeucker().SetMaxDistance(.5).Simplify(pl)
	if pl.Size() != 20 {
		t.Fatalf("expected 20 points after simplify, got %d", pl.Size())
	}
}

func TestSimplifyCheckPointCount(t *testing.T) {
	pl := NewPointList(0, false)
	pl.Parse2DJSON(points1)
	rdp := NewRamerDouglasPeucker().SetMaxDistance(.5)
	if pl.Size() != 32 {
		t.Fatalf("expected 32, got %d", pl.Size())
	}
	rdp.Simplify(pl)
	if pl.Size() != 20 {
		t.Fatalf("expected 20, got %d", pl.Size())
	}
	if strings.Contains(pl.String(), "NaN") {
		t.Fatalf("PointList contains NaN: %s", pl.String())
	}

	pl.Clear()
	pl.Parse2DJSON(points1)
	rdp.SimplifyFromTo(pl, 0, pl.Size()-1)
	if pl.Size() != 20 {
		t.Fatalf("expected 20 after SimplifyFromTo, got %d", pl.Size())
	}

	pl.Clear()
	pl.Parse2DJSON(points1)
	removed1 := rdp.Simplify(pl.Copy(10, 20))

	pl.Clear()
	pl.Parse2DJSON(points1)
	removed2 := rdp.SimplifyFromTo(pl, 10, 19)

	if removed1 != removed2 {
		t.Fatalf("removed1 %d != removed2 %d", removed1, removed2)
	}
}

func TestSimplifyCheckPointOrder(t *testing.T) {
	pl := NewPointList(0, false)
	pl.Parse2DJSON(points2)
	if pl.Size() != 13 {
		t.Fatalf("expected 13, got %d", pl.Size())
	}
	NewRamerDouglasPeucker().SetMaxDistance(.5).Simplify(pl)
	if pl.Size() != 11 {
		t.Fatalf("expected 11, got %d", pl.Size())
	}
	if strings.Contains(pl.String(), "NaN") {
		t.Fatalf("PointList contains NaN: %s", pl.String())
	}
	expected := "(50.203764443183644,9.961074440801317), (50.20318963087774,9.960999562464645), (50.202952888673984,9.96094144793469), (50.20267889356641,9.96223002587773), (50.201853928011374,9.961859918278305), " +
		"(50.20138565901039,9.961668810881722), (50.20128507617008,9.96216874485095), (50.20088553877664,9.961953795595925), (50.200686794534775,9.961899033827313), (50.20014066696481,9.961716680863127), (50.199798499043254,9.961588158344957)"
	if pl.String() != expected {
		t.Fatalf("mismatch:\ngot:  %s\nwant: %s", pl.String(), expected)
	}
}

func TestRemoveNaN(t *testing.T) {
	pl := NewPointList(10, true)
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(1, 1, 1)
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(5, 5, 5)
	pl.Add3D(6, 6, 6)
	pl.Add3D(7, 7, 7)
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(8, 8, 8)
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(9, 9, 9)
	pl.Add3D(10, 10, 10)
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())
	pl.Add3D(14, 14, 14)
	pl.Add3D(math.NaN(), math.NaN(), math.NaN())

	RemoveNaN(pl)
	// doing it again should be no problem
	RemoveNaN(pl)
	RemoveNaN(pl)
	if pl.Size() != 8 {
		t.Fatalf("expected 8, got %d", pl.Size())
	}
	expected := []int{1, 5, 6, 7, 8, 9, 10, 14}
	for i := 0; i < pl.Size(); i++ {
		assertNear(t, pl.GetLat(i), pl.GetEle(i), 1e-6)
		assertNear(t, pl.GetLon(i), pl.GetEle(i), 1e-6)
		if int(pl.GetLat(i)) != expected[i] {
			t.Fatalf("index %d: expected %d, got %v", i, expected[i], pl.GetLat(i))
		}
	}
}

func Test3DPathSimplify(t *testing.T) {
	pl := NewPointList(5, true)
	pl.Add3D(0, 0, 0)
	pl.Add3D(0.01, 0, 10)
	pl.Add3D(0.02, 0, 20)
	pl.Add3D(0.03, 0, 30)
	pl.Add3D(0.04, 0, 50)
	NewRamerDouglasPeucker().SetMaxDistance(1).SetElevationMaxDistance(1).Simplify(pl)
	expected := "(0,0,0), (0.03,0,30), (0.04,0,50)"
	if pl.String() != expected {
		t.Fatalf("got: %s\nwant: %s", pl.String(), expected)
	}
}

func Test3DPathSimplifyElevationDisabled(t *testing.T) {
	pl := NewPointList(5, true)
	pl.Add3D(0, 0, 0)
	pl.Add3D(0.03, 0, 30)
	pl.Add3D(0.04, 0, 50)
	NewRamerDouglasPeucker().SetMaxDistance(1).SetElevationMaxDistance(math.MaxFloat64).Simplify(pl)
	expected := "(0,0,0), (0.04,0,50)"
	if pl.String() != expected {
		t.Fatalf("got: %s\nwant: %s", pl.String(), expected)
	}
}

func Test3DPathSimplifyElevationMaxDistFive(t *testing.T) {
	pl := NewPointList(5, true)
	pl.Add3D(0, 0, 0)
	pl.Add3D(0.01, 0, 14)
	pl.Add3D(0.02, 0, 20)
	pl.Add3D(0.03, 0, 30)
	pl.Add3D(0.04, 0, 50)
	NewRamerDouglasPeucker().SetMaxDistance(1).SetElevationMaxDistance(5).Simplify(pl)
	expected := "(0,0,0), (0.03,0,30), (0.04,0,50)"
	if pl.String() != expected {
		t.Fatalf("got: %s\nwant: %s", pl.String(), expected)
	}
}

func Test3DPathSimplifyWithMissingElevation(t *testing.T) {
	pl := NewPointList(5, true)
	pl.Add3D(0, 0, 0)
	pl.Add3D(0, 0.5, math.NaN())
	pl.Add3D(0, 1, 14)
	pl.Add3D(1, 1, 20)
	NewRamerDouglasPeucker().SetMaxDistance(1).SetElevationMaxDistance(1).Simplify(pl)
	expected := "(0,0,0), (0,1,14), (1,1,20)"
	if pl.String() != expected {
		t.Fatalf("got: %s\nwant: %s", pl.String(), expected)
	}
}

func Test3DSimplifyStartEndSame(t *testing.T) {
	pl := NewPointList(3, true)
	pl.Add3D(0, 0, 0)
	pl.Add3D(0.03, 0, 30)
	pl.Add3D(0, 0, 0)
	NewRamerDouglasPeucker().SetMaxDistance(1).SetElevationMaxDistance(1).Simplify(pl)
	expected := "(0,0,0), (0.03,0,30), (0,0,0)"
	if pl.String() != expected {
		t.Fatalf("got: %s\nwant: %s", pl.String(), expected)
	}
}

func Test2DSimplifyStartEndSame(t *testing.T) {
	pl := NewPointList(3, false)
	pl.Add(0, 0)
	pl.Add(0.03, 0)
	pl.Add(0, 0)
	NewRamerDouglasPeucker().SetMaxDistance(1).SetElevationMaxDistance(1).Simplify(pl)
	expected := "(0,0), (0.03,0), (0,0)"
	if pl.String() != expected {
		t.Fatalf("got: %s\nwant: %s", pl.String(), expected)
	}
}

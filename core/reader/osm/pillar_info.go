package osm

import (
	"math"

	"gohopper/core/storage"
	"gohopper/core/util"
)

const (
	latOffset = 0
	lonOffset = 4
	eleOffset = 8
)

// PillarInfo stores temporary lat/lon/ele coordinates for pillar nodes during import.
// Backed by a DataAccess file that is removed after import.
type PillarInfo struct {
	enabled3D    bool
	da           storage.DataAccess
	dir          storage.Directory
	rowSizeBytes int
}

func NewPillarInfo(enabled3D bool, dir storage.Directory) *PillarInfo {
	dim := 2
	if enabled3D {
		dim = 3
	}
	da := dir.Create("tmp_pillar_info").Create(100)
	return &PillarInfo{
		enabled3D:    enabled3D,
		da:           da,
		dir:          dir,
		rowSizeBytes: dim * 4,
	}
}

func (p *PillarInfo) Is3D() bool { return p.enabled3D }

func (p *PillarInfo) Dimension() int {
	if p.enabled3D {
		return 3
	}
	return 2
}

func (p *PillarInfo) bytePos(nodeID int64) int64 {
	return nodeID * int64(p.rowSizeBytes)
}

func (p *PillarInfo) ensureNode(nodeID int64) {
	pos := p.bytePos(nodeID)
	p.da.EnsureCapacity(pos + int64(p.rowSizeBytes))
}

func (p *PillarInfo) SetNode(nodeID int64, lat, lon, ele float64) {
	p.ensureNode(nodeID)
	pos := p.bytePos(nodeID)
	p.da.SetInt(pos+latOffset, util.DegreeToInt(lat))
	p.da.SetInt(pos+lonOffset, util.DegreeToInt(lon))
	if p.enabled3D {
		p.da.SetInt(pos+eleOffset, int32(util.EleToUInt(ele)))
	}
}

func (p *PillarInfo) GetLat(id int64) float64 {
	return util.IntToDegree(p.da.GetInt(p.bytePos(id) + latOffset))
}

func (p *PillarInfo) GetLon(id int64) float64 {
	return util.IntToDegree(p.da.GetInt(p.bytePos(id) + lonOffset))
}

func (p *PillarInfo) GetEle(id int64) float64 {
	if !p.enabled3D {
		return math.NaN()
	}
	return util.UIntToEle(int(p.da.GetInt(p.bytePos(id) + eleOffset)))
}

// Clear removes the temporary DataAccess file.
func (p *PillarInfo) Clear() {
	p.dir.Remove(p.da.Name())
}

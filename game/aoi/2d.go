package aoi

import (
	"github.com/kercylan98/minotaur/utils/geometry"
	"github.com/kercylan98/minotaur/utils/hash"
	"math"
	"sync"
)

func NewTwoDimensional[E TwoDimensionalEntity](width, height, areaWidth, areaHeight int) *TwoDimensional[E] {
	aoi := &TwoDimensional[E]{
		event:  new(event[E]),
		width:  float64(width),
		height: float64(height),
		focus:  map[int64]map[int64]E{},
	}
	aoi.SetAreaSize(areaWidth, areaHeight)
	return aoi
}

type TwoDimensional[E TwoDimensionalEntity] struct {
	*event[E]
	rw               sync.RWMutex
	width            float64
	height           float64
	areaWidth        float64
	areaHeight       float64
	areaWidthLimit   int
	areaHeightLimit  int
	areas            [][]map[int64]E
	focus            map[int64]map[int64]E
	repartitionQueue []func()
}

func (slf *TwoDimensional[E]) AddEntity(entity E) {
	slf.rw.Lock()
	slf.addEntity(entity)
	slf.rw.Unlock()
}

func (slf *TwoDimensional[E]) DeleteEntity(entity E) {
	slf.rw.Lock()
	slf.deleteEntity(entity)
	slf.rw.Unlock()
}

func (slf *TwoDimensional[E]) Refresh(entity E) {
	slf.rw.Lock()
	defer slf.rw.Unlock()
	slf.refresh(entity)
}

func (slf *TwoDimensional[E]) GetFocus(guid int64) map[int64]E {
	slf.rw.RLock()
	defer slf.rw.RUnlock()
	return hash.Copy(slf.focus[guid])
}

func (slf *TwoDimensional[E]) SetSize(width, height int) {
	fw, fh := float64(width), float64(height)
	if fw == slf.width && fh == slf.height {
		return
	}
	slf.rw.Lock()
	defer slf.rw.Unlock()
	slf.width = fw
	slf.height = fh
	slf.setAreaSize(int(slf.areaWidth), int(slf.areaHeight))
}

func (slf *TwoDimensional[E]) SetAreaSize(width, height int) {
	fw, fh := float64(width), float64(height)
	if fw == slf.areaWidth && fh == slf.areaHeight {
		return
	}
	slf.rw.Lock()
	defer slf.rw.Unlock()
	slf.setAreaSize(width, height)
}

func (slf *TwoDimensional[E]) setAreaSize(width, height int) {

	// 旧分区备份
	var oldAreas = make([][]map[int64]E, len(slf.areas))
	for w := 0; w < len(slf.areas); w++ {
		hs := slf.areas[w]
		ohs := make([]map[int64]E, len(hs))
		for h := 0; h < len(hs); h++ {
			es := map[int64]E{}
			for g, e := range hs[h] {
				es[g] = e
			}
			ohs[h] = es
		}
		oldAreas[w] = ohs
	}

	// 清理分区
	for i := 0; i < len(oldAreas); i++ {
		area := slf.areas[i]
		for a := 0; a < len(area); a++ {
			entities := area[a]
			for _, entity := range entities {
				slf.deleteEntity(entity)
			}
		}
	}

	// 生成区域
	slf.areaWidth = float64(width)
	slf.areaHeight = float64(height)
	slf.areaWidthLimit = int(math.Ceil(slf.width / slf.areaWidth))
	slf.areaHeightLimit = int(math.Ceil(slf.height / slf.areaHeight))
	areas := make([][]map[int64]E, slf.areaWidthLimit+1)
	for i := 0; i < len(areas); i++ {
		entities := make([]map[int64]E, slf.areaHeightLimit+1)
		for e := 0; e < len(entities); e++ {
			entities[e] = map[int64]E{}
		}
		areas[i] = entities
	}
	slf.areas = areas

	// 重新分区
	for i := 0; i < len(oldAreas); i++ {
		area := oldAreas[i]
		for a := 0; a < len(area); a++ {
			entities := area[a]
			for _, entity := range entities {
				slf.addEntity(entity)
			}
		}
	}
}

func (slf *TwoDimensional[E]) addEntity(entity E) {
	x, y := entity.GetPosition()
	widthArea := int(x / slf.areaWidth)
	heightArea := int(y / slf.areaHeight)
	guid := entity.GetGuid()
	slf.areas[widthArea][heightArea][guid] = entity
	focus := map[int64]E{}
	slf.focus[guid] = focus
	slf.rangeVisionAreaEntities(entity, func(eg int64, e E) {
		focus[eg] = e
		slf.OnEntityJoinVisionEvent(entity, e)
		slf.refresh(e)
	})
}

func (slf *TwoDimensional[E]) refresh(entity E) {
	x, y := entity.GetPosition()
	vision := entity.GetVision()
	guid := entity.GetGuid()
	focus := slf.focus[guid]
	for eg, e := range focus {
		ex, ey := e.GetPosition()
		if geometry.CalcDistanceWithCoordinate(x, y, ex, ey) > vision {
			delete(focus, eg)
			delete(slf.focus[eg], guid)
		}
	}

	slf.rangeVisionAreaEntities(entity, func(guid int64, e E) {
		if _, exist := focus[guid]; !exist {
			focus[guid] = e
			slf.OnEntityJoinVisionEvent(entity, e)
		}
	})
}

func (slf *TwoDimensional[E]) rangeVisionAreaEntities(entity E, handle func(guid int64, entity E)) {
	x, y := entity.GetPosition()
	widthArea := int(x / slf.areaWidth)
	heightArea := int(y / slf.areaHeight)
	vision := entity.GetVision()
	widthSpan := int(math.Ceil(vision / slf.areaWidth))
	heightSpan := int(math.Ceil(vision / slf.areaHeight))
	guid := entity.GetGuid()

	sw := widthArea - widthSpan
	if sw < 0 {
		sw = 0
	} else if sw > slf.areaWidthLimit {
		sw = slf.areaWidthLimit
	}
	ew := widthArea - widthSpan
	if ew < sw {
		ew = sw
	} else if ew > slf.areaWidthLimit {
		ew = slf.areaWidthLimit
	}
	for w := sw; w < ew; w++ {
		sh := heightArea - heightSpan
		if sh < 0 {
			sh = 0
		} else if sh > slf.areaHeightLimit {
			sh = slf.areaHeightLimit
		}
		eh := widthArea - widthSpan
		if eh < sh {
			eh = sh
		} else if eh > slf.areaHeightLimit {
			eh = slf.areaHeightLimit
		}
		for h := sh; h < eh; h++ {
			var areaX, areaY float64
			if w < widthArea {
				tempW := w + 1
				areaX = float64(tempW * int(slf.areaWidth))
			} else if w > widthArea {
				areaX = float64(w * int(slf.areaWidth))
			} else {
				areaX = x
			}
			if h < heightArea {
				tempH := h + 1
				areaY = float64(tempH * int(slf.areaHeight))
			} else if h > heightArea {
				areaY = float64(h * int(slf.areaHeight))
			} else {
				areaY = y
			}
			areaDistance := geometry.CalcDistanceWithCoordinate(x, y, areaX, areaY)
			if areaDistance <= vision {
				for eg, e := range slf.areas[w][h] {
					if eg == guid {
						continue
					}
					if ex, ey := e.GetPosition(); geometry.CalcDistanceWithCoordinate(x, y, ex, ey) > vision {
						continue
					}
					handle(eg, e)
				}
			}
		}
	}
}

func (slf *TwoDimensional[E]) deleteEntity(entity E) {
	x, y := entity.GetPosition()
	widthArea := int(x / slf.areaWidth)
	heightArea := int(y / slf.areaHeight)
	guid := entity.GetGuid()
	focus := slf.focus[guid]
	for g, e := range focus {
		slf.OnEntityLeaveVisionEvent(entity, e)
		slf.OnEntityLeaveVisionEvent(e, entity)
		delete(slf.focus[g], guid)
	}
	delete(slf.focus, guid)
	delete(slf.areas[widthArea][heightArea], guid)
}

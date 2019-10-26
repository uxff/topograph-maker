package main

import "log"

type MapElem struct {
	X, Y int
	Val  int
}

type PosMap struct {
	//proto: map[int posX, int posY]int Id
	w    int
	h    int
	data []MapElem
}

func (m *PosMap) Init(width, height int) {
	m.w, m.h = width, height
	m.data = make([]MapElem, 0)
}

func (m *PosMap) AddElem(e *MapElem) {
	m.data = append(m.data, *e)
}

func (m *PosMap) Search(x, y int, radius int) []*MapElem {
	matched := make([]*MapElem, 0)

	for ei := range m.data {
		if (m.data[ei].X-x)*(m.data[ei].X-x)+(m.data[ei].X-x)*(m.data[ei].X-x) < radius*radius {
			matched = append(matched, &m.data[ei])
		}
	}

	return nil
}

func (m *PosMap) ElemOnMove(e *MapElem, xStep, yStep int) {
	// todo reindex elem in m.data
}

type DeepPosMap struct {
	//proto: map[int posX, int posY]int Id
	w int
	h int

	wDivBy int
	hDivBy int

	data [][]MapElem
}

func (m *DeepPosMap) Init(width, height int) {
	m.w, m.h = width, height
	m.wDivBy = 16
	m.hDivBy = 16
	m.data = make([][]MapElem, m.wDivBy*m.hDivBy)
}

func (m *DeepPosMap) AddElem(e *MapElem) {
	divIdx := (e.Y/m.hDivBy)*m.wDivBy + e.X/m.wDivBy
	if divIdx > m.wDivBy*m.hDivBy {
		log.Printf(" divIdx(%d) out of bound", divIdx)
		return
	}
	m.data[divIdx] = append(m.data[divIdx], *e)
}

func (m *DeepPosMap) Search(x, y int, radius int) []*MapElem {
	matched := make([]*MapElem, 0)
	divIdx := (y/m.hDivBy)*m.wDivBy + x/m.wDivBy

	// todo divIdx周围8个区都要查找
	for ei := range m.data[divIdx] {
		if (m.data[divIdx][ei].X-x)*(m.data[divIdx][ei].X-x)+(m.data[divIdx][ei].X-x)*(m.data[divIdx][ei].X-x) < radius*radius {
			matched = append(matched, &m.data[divIdx][ei])
		}
	}

	return matched
}

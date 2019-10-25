package main

type MapElem struct {
	X, Y int
	Val  int
}

type PosMap struct {
	//proto: map[int posX, int posY]int Id

}

func (m *PosMap) AddElem(e *MapElem) {

}

func (m *PosMap) Search(e *MapElem, radius int) []*MapElem {
	return nil
}

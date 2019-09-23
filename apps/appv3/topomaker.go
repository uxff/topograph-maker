/*
	usage: time ./topomaker -w 800 -h 800 -hill 200 -hill-wide 200 -ridge 2 -ridge-wide 50 -times 1000 -dropnum 100 -zoom 5
	./topomaker --zoom 3 -h 500 -w 500 --hill 0 --ridge 50 --ridge-len 50 --ridge-wide 50 --dropnum 100 --times 1000
	./topomaker --zoom 3 -h 500 -w 500 --hill 200 --hill-wide 100 --ridge 15 --ridge-len 40 --ridge-wide 50 --dropnum 0 --times 1 --color-tpl-step 7
	./topomaker --zoom 3 -h 500 -w 500 --hill 200 --hill-wide 150 --ridge 15 --ridge-len 40 --ridge-wide 20 --dropnum 0 --times 1 --color-tpl-step 3
	./topomaker --zoom 1 -h 500 -w 500 --hill 200 --hill-wide 100 --ridge 20 --ridge-len 50 --ridge-wide 20 --dropnum 0 --times 1 --color-tpl-step 10
	# excellent
	./topomaker --zoom 1 -h 1000 -w 1000 --hill 1000 --hill-wide 150 --ridge 40 --ridge-len 30 --ridge-wide 40 --dropnum 0 --times 1000 --color-tpl-step 10 # excellent
    todo: table lize with http server
*/
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/uxff/topograph-maker/drawer"
)

const colorTplFile = "./image/color-tpl2.png"

// 方案二(un done) 计算好地形场向量 没有流量向量
// 随机撒水珠 水滴会移动 移动的时候会动态影响周边的其他水滴
// 不断撒水滴 看看水滴运动趋势
// 将借助canvas+js+websocket实现
// 使用水滴滚动
type Droplet struct {
	x         float32
	y         float32
	fallPower int     // 落差能量
	vx        float32 // 滑行速度
	vy        float32
	hisway    []int
}

// 将变成固定不移动 记录场 被流动水雕刻
type WaterDot struct {
	x      float32 // 将不变化 =Topomap[x,y] +(0.5, 0.5)
	y      float32
	xPower float32 // x方向速度增益 场强度 v2 根据地形得出 初始化后不变(地形改变则会变) 基于Atan2 范围(-1,1)
	yPower float32 // y方向速度增益 场强度 v2 根据地形得出 初始化后不变(地形改变则会变)
	h      int     // 积水高度，产生积水不参与流动，流动停止 // v2将由水滴实体代替该变量
	q      int     // 流量 0=无 历史流量    //
	//xPowerQ float32 // v2 根据周围流量算出 每次update变化 基于Atan2 范围(-1,1)
	//yPowerQ float32 // v2
}
type Topomap struct {
	data   []uint8 // 对应坐标只保存高度
	width  int
	height int
	events chan *ErodeEvent
	evtIdx int
}
type WaterMap struct {
	data   []WaterDot // slice的idx不再是pos
	width  int
	height int
	events chan *ErodeEvent
	evtIdx int
}

// 先处理场向量 // 再注水流动

// 预先处理每个点的场向量  只计算地势的影响，不考虑流量的影响
// 假设每个点都有一个场，计算出这个场的方向
// 启动只执行1次
// @param Topomap m is basic topomap
// @param int ring 表示计算到几环 默认2环
func (w *WaterMap) AssignVector(m *Topomap, ring int) {
	for idx, curDot := range w.data {
		var xPower, yPower int // xPower, yPower 单位为1
		// 2nd ring
		_, lowestPos := curDot.getLowestNeighbors(curDot.getNeighbors(w), m)
		for _, neiPos := range lowestPos {
			xPower += 4 * (neiPos.x - int(idx%w.width))
			yPower += 4 * (neiPos.y - int(idx/w.width))
		}

		// 3rd ring. done 三环的影响力是二环的1/4
		if ring >= 3 {
			_, lowestPos = curDot.getLowestNeighbors(curDot.get3rdNeighbors(w), m)
			for _, neiPos := range lowestPos {
				xPower += neiPos.x - int(idx%w.width)
				yPower += neiPos.y - int(idx/w.width)
			}
		}

		// 四环 四环影响力是二环的1/16 暂不实现4环

		if xPower != 0 || yPower != 0 {
			thedir := math.Atan2(float64(yPower), float64(xPower))
			w.data[idx].xPower, w.data[idx].yPower = float32(math.Cos(thedir)), float32(math.Sin(thedir))
		}
	}
}

// 按照周围流量更新场向量
// powerRate 一般指定小于1 比如0.1
func (w *WaterMap) UpdateVectorByQuantity(m *Topomap, ring int, powerRate float32) {
	for idx, curDot := range w.data {
		//go func() {
		var xPower, yPower int // xPower, yPower 单位为1
		// 2nd ring
		_, mostQuanPos := curDot.getPostQuanNeighbors(curDot.getNeighbors(w), w)
		for _, neiPos := range mostQuanPos {
			xPower += 4 * (neiPos.x - int(idx%w.width))
			yPower += 4 * (neiPos.y - int(idx/w.width))
		}

		// 3rd ring. done 三环的影响力是二环的1/4
		if ring >= 3 {
			_, mostQuanPos = curDot.getPostQuanNeighbors(curDot.get3rdNeighbors(w), w)
			for _, neiPos := range mostQuanPos {
				xPower += neiPos.x - int(idx%w.width)
				yPower += neiPos.y - int(idx/w.width)
			}
		}

		//if xPower != 0 || yPower != 0 {
		//	thedir := math.Atan2(float64(yPower), float64(xPower))
		//	w.data[idx].xPowerQ, w.data[idx].yPowerQ = float32(math.Cos(thedir))*powerRate, float32(math.Sin(thedir))*powerRate
		//}
		//}() //可以不等待 //使用go反而慢
	}
}

func DropletsMove(times int, drops []*Droplet, m *Topomap, w *WaterMap) []*Droplet {
	for i := 1; i <= times; i++ {
		wg := &sync.WaitGroup{}
		for _, d := range drops {
			wg.Add(1)
			go func(d *Droplet, step int) {
				d.Move(m, w, drops, step)
				wg.Done()
			}(d, i)
		}

		wg.Wait()
	}
	return drops
}

// 将坐标是0的清理出数组
func ClearDroplets(drops []*Droplet) []*Droplet {
	newDrops := make([]*Droplet, 0)
	for idx, d := range drops {
		if d.vx == 0 && d.vy == 0 {
			log.Printf("clear:[%d]=%+v", idx, *d)
			//drops = append(drops[:idx], drops[idx+1:]...)//panic
			//idx--
			continue
		}
		newDrops = append(newDrops, d)
	}
	return newDrops
}

func MakeDroplet(w *WaterMap) *Droplet {
	idx := rand.Int() % len(w.data)
	d := Droplet{
		x:         float32(idx%w.width) + 0.5,
		y:         float32(idx/w.width) + 0.5,
		hisway:    []int{idx},
		fallPower: 2,
	}

	w.data[idx].h++

	thedir := rand.Float64() * math.Pi * 2
	d.vx, d.vy = float32(math.Cos(thedir))/2, float32(math.Sin(thedir))/2
	return &d
}

// 改变地形的事件 异步化 不并行处理 防止并行修改计算错误
type ErodeEvent struct {
	oldIdx int
	newIdx int
	m      *Topomap
	w      *WaterMap
	drop   *Droplet
}

// drop(readonly) change the watermap
func (w *WaterMap) EmitErodeEvents(oldIdx, newIdx int, m *Topomap, drop *Droplet) {
	w.events <- &ErodeEvent{oldIdx: oldIdx, newIdx: newIdx, m: m, w: w, drop: drop}
}

func (m *Topomap) EmitErodeEvents(oldIdx, newIdx int, w *WaterMap, drop *Droplet) {
	m.events <- &ErodeEvent{oldIdx: oldIdx, newIdx: newIdx, m: m, w: w, drop: drop}
}

func (m *Topomap) eroding() {
	for {
		select {
		case e := <-m.events:
			m.evtIdx++
			// 50% 的几率将
			if m.data[e.oldIdx] > 0 {
				m.data[e.oldIdx]--
			}
			// todo 这里有bug
			neis := e.w.data[e.oldIdx].getNeighbors(e.w)
			for nei := range neis {
				if neis[nei].y >= m.height || neis[nei].x >= m.width {
					continue
				}
				if m.data[neis[nei].y*m.width+neis[nei].x] > 0 {
					m.data[neis[nei].y*m.width+neis[nei].x]--
				}
			}
		}
	}
}

func (w *WaterMap) eroding() {
	//evtIdx := 0
	for {
		select {
		case e := <-w.events:
			w.evtIdx++
			w.data[e.newIdx].h++
			w.data[e.oldIdx].h--
			log.Printf("eroding watermap: oldIdx:%d newIdx:%d", e.oldIdx, e.newIdx)

			if e.oldIdx != e.newIdx {
				log.Printf("will erode topomap: oldIdx:%d newIdx:%d", e.oldIdx, e.newIdx)
				go e.m.EmitErodeEvents(e.oldIdx, e.newIdx, w, e.drop)
				//w.data[e.oldIdx].q -= 3
			}

			if w.evtIdx%100 == 0 {
				w.UpdateVectorByQuantity(e.m, 2, 0.2)
				//drops = ClearDroplets(drops)
				//log.Printf("drops cleard len=%d", len(drops))
			}
		}
	}
}

// 只更新WaterMap中的场 不更新里面的坐标
func (d *Droplet) Move(m *Topomap, w *WaterMap, drops []*Droplet, step int) {
	oldIdx := int(d.x) + int(d.y)*w.width
	if oldIdx > len(w.data) || oldIdx < 0 {
		log.Printf("oldIdx(%d) out of w.data. stop it.", oldIdx)
		return
	}

	d.vx, d.vy = d.vx+w.data[oldIdx].xPower, d.vy+w.data[oldIdx].yPower
	w.data[oldIdx].q++ // 流出，才算流量

	// 地形场加速
	//d.vx, d.vy = (d.vx+w.data[oldIdx].xPower)/2, (d.vy+w.data[oldIdx].yPower)/2

	// 距离平方在2以内的WaterDot有吸引力 // todo 使用分层数组
	for i := 0; i < len(drops); i += 4 {
		di := drops[i+((step)%4)] // 几率变成4分之1 但是不会重复，会轮询
		distSquare := (di.x-d.x)*(di.x-d.x) + (di.y-d.y)*(di.y-d.y)
		if distSquare < 9.0 {
			// di 是在范围sqrt(8)以内的水滴
			d.CloseTo(di, distSquare) // 靠近
		}
	}

	// 没有场 可撒欢
	if w.data[oldIdx].xPower == 0 && w.data[oldIdx].yPower == 0 {
		// 自己生速度 比较浪
		d.GenVeloByFallPower(m, w)
		//log.Printf("no field power, after play self(x=%f,y=%f, vx,vy=%f,%f) fallPower=%d", d.x, d.y, d.vx, d.vy, d.fallPower)
		//return
	}

	// 将超出的速度限制成标准速度
	d.ReduceSpeed()

	// 场速度与自身速度的平均值
	tmpX := d.x + d.vx // todo:精度损失风险
	tmpY := d.y + d.vy

	// 越界判断
	if int(tmpX) < 0 || int(tmpX) > w.width-1 || int(tmpY) < 0 || int(tmpY) > w.height-1 {
		log.Printf("droplet move out of bound(x=%f,y=%f). stop move.", tmpX, tmpY)
		return
	}

	newIdx := int(tmpX) + int(tmpY)*w.width
	// 无力场，待在原地
	if newIdx == oldIdx {
		//log.Printf("no field power. stay here.")
		//w.data[oldIdx].q++ // 流出，才算流量
		return
	}

	if newIdx >= w.width*w.height {
		log.Printf("newIdx(%d) out of data range, ignore", newIdx)
		return
	}

	//d.hisway = append(d.hisway, newIdx)

	// 不跑到高处
	if w.data[newIdx].h+int(m.data[newIdx]) > w.data[oldIdx].h+int(m.data[oldIdx]) {
		log.Printf("pos(%8d to %8d) is too high, stop, h:%d/%d", oldIdx, newIdx, w.data[oldIdx].h, w.data[newIdx].h)
		return
	}

	d.x, d.y = tmpX, tmpY
	d.fallPower += int(m.data[oldIdx]-m.data[newIdx]) * 10

	//w.data[oldIdx].h--
	//w.data[newIdx].h++
	go w.EmitErodeEvents(oldIdx, newIdx, m, d)
}

// 根据落差能量移动 包括位置浮动和速度浮动 只更改droplet
func (d *Droplet) GenVeloByFallPower(m *Topomap, w *WaterMap) {
	if d.fallPower > 0 {
		tmpRoll := rand.Float32()
		// 小于一定的几率才执行方向浮动
		if tmpRoll < 0.5 {
			// 要和PI有关系 否则都向右面走
			tmpDir := (rand.Float64() - rand.Float64()) * math.Pi * 2
			fx, fy := float32(math.Cos(tmpDir)), float32(math.Sin(tmpDir))
			d.vx, d.vy = d.vx+fx/4.0, d.vy+fy/4.0
		}

		// 在一定几率下 位移浮动
		if tmpRoll < 0.5 {
			//tmpDir := (rand.Float64() - rand.Float64()) * math.Pi * 2
			fx, fy := rand.Float32()-tmpRoll, rand.Float32()-tmpRoll //float32(math.Cos(tmpDir)), float32(math.Sin(tmpDir))
			d.x, d.y = d.x+fx/1.0, d.y+fy/1.0
		}

		//log.Printf("drop playself fx,fy=%f,%f", fx, fy)
		d.fallPower--
	}
}

func (m *Topomap) Init(width int, height int) {
	m.data = make([]uint8, width*height)
	m.width = width
	m.height = height
	m.events = make(chan *ErodeEvent, 1000)
	go m.eroding()
}
func (w *WaterMap) Init(width int, height int) {
	w.data = make([]WaterDot, width*height)
	w.width = width
	w.height = height
	// 把点的实际基点摆在中间
	for x := 0; x < w.width; x++ {
		for y := 0; y < w.height; y++ {
			w.data[x+y*w.width].x = float32(x) + 0.5
			w.data[x+y*w.width].y = float32(y) + 0.5
		}
	}
	w.events = make(chan *ErodeEvent, 1000)
	go w.eroding()
}

func (w *WaterMap) SumH() int {
	h := 0
	for idx := range w.data {
		h += w.data[idx].h
	}
	return h
}

type Hill struct {
	x       int
	y       int
	r       int
	h       int
	tiltDir float64 // 倾斜方向
	tiltLen int     // 倾斜长度
}

/*返回颜色数组，下标越大颜色海拔越高*/
func colorTpl(colorTplFile string, colorTplStep int) []color.Color {
	var colorTplFileIo, _ = os.Open(colorTplFile)
	defer colorTplFileIo.Close()
	var colorTplPng, err = png.Decode(colorTplFileIo)

	if err != nil {
		log.Println("png.decode err when read colorTpl:", err)
		return nil
	}

	// 从colorTplStep以上的部分取
	theLen := colorTplPng.Bounds().Dy() - colorTplStep
	cs := make([]color.Color, theLen)
	for i := 0; i < theLen; i++ {
		cs[i] = colorTplPng.At(0, (theLen-i-1)-colorTplStep)
	}
	return cs
}

func lineTo(img *image.RGBA, startX, startY, destX, destY int, lineColor, startColor color.Color, scale float64) {
	distM := math.Sqrt(float64((startX-destX)*(startX-destX) + (startY-destY)*(startY-destY)))
	var i float64
	for i = 0; i < distM*scale; i++ {
		img.Set(startX+int(i/distM*float64(destX-startX)), startY+int(i/distM*float64(destY-startY)), lineColor)
	}
	// 线段最后一点 绘制成始发地地形的颜色 startColor
	if startX != destX && startY != destY {
		img.Set(startX+int(i/distM*float64(destX-startX)), startY+int(i/distM*float64(destY-startY)), startColor)
	}
}

// 此函数固定返回本坐标周边2环8个边界点，可能包含超出地图边界的点
func (d *WaterDot) getNeighbors(w *WaterMap) []struct{ x, y int } {
	pos := make([]struct{ x, y int }, 8)
	pos[0].x, pos[0].y = int(d.x+1), int(d.y+0)
	pos[1].x, pos[1].y = int(d.x+1), int(d.y-1)
	pos[2].x, pos[2].y = int(d.x+0), int(d.y-1)
	pos[3].x, pos[3].y = int(d.x-1), int(d.y-1)
	pos[4].x, pos[4].y = int(d.x-1), int(d.y+0)
	pos[5].x, pos[5].y = int(d.x-1), int(d.y+1)
	pos[6].x, pos[6].y = int(d.x+0), int(d.y+1)
	pos[7].x, pos[7].y = int(d.x+1), int(d.y+1)
	return pos
}

// 此函数固定返回本点周边3环12个边界点，可能包含超出地图边界的点
func (d *WaterDot) get3rdNeighbors(w *WaterMap) []struct{ x, y int } {
	pos := make([]struct{ x, y int }, 12)
	// right
	pos[0].x, pos[0].y = int(d.x+2), int(d.y)
	pos[1].x, pos[1].y = int(d.x+2), int(d.y-1)
	// top
	pos[2].x, pos[2].y = int(d.x+1), int(d.y-2)
	pos[3].x, pos[3].y = int(d.x), int(d.y-2)
	pos[4].x, pos[4].y = int(d.x-1), int(d.y-2)
	// left
	pos[5].x, pos[5].y = int(d.x-2), int(d.y-1)
	pos[6].x, pos[6].y = int(d.x-2), int(d.y)
	pos[7].x, pos[7].y = int(d.x-2), int(d.y+1)
	// bottom
	pos[8].x, pos[8].y = int(d.x-1), int(d.y+2)
	pos[9].x, pos[9].y = int(d.x), int(d.y+2)
	pos[10].x, pos[10].y = int(d.x+1), int(d.y+2)
	// right
	pos[11].x, pos[11].y = int(d.x+2), int(d.y+1)
	return pos
}

/*获取周围最低的点 最低点集合数组中随机取一个 返回安全的坐标，不在地图外*/
func (d *WaterDot) getLowestNeighbors(arrNei []struct{ x, y int }, m *Topomap) (lowestLevel int, lowestPos []struct{ x, y int }) {
	// 原理： highMap[high] = []struct{int,int}
	highMap := make(map[int][]struct{ x, y int }, 8)
	for _, nei := range arrNei {
		if nei.x < 0 || nei.x > m.width-1 || nei.y < 0 || nei.y > m.height-1 {
			// 超出地图边界的点
			continue
		}
		// 邻居的高度 todo: 有BUG 此处不能加本地的水位 要加邻居的水位
		high := int(m.data[nei.x+nei.y*m.width]) + d.h
		if len(highMap[int(high)]) == 0 {
			highMap[high] = []struct{ x, y int }{{nei.x, nei.y}} //make([]struct{ x, y int }, 1)
			//highMap[int(m.data[nei.x+nei.y*w.width])][0].x, highMap[int(m.data[nei.x+nei.y*w.width])][0].y = nei.x, nei.y
		} else {
			highMap[high] = append(highMap[high], struct{ x, y int }{nei.x, nei.y})
		}
	}
	lowestLevel = 1000
	for k, _ := range highMap {
		if k < lowestLevel {
			lowestLevel = k
		}
	}
	//log.Println("lowest,highMap,count(highMap),d=", lowest, highMap, len(highMap), *d)
	if len(highMap[lowestLevel]) == 0 {
		log.Printf("how is can be zero?")
		return lowestLevel, nil
	}
	return lowestLevel, highMap[lowestLevel]
}

/*获取周围流量最大的点 流量最大的点集合数组中随机取一个 返回安全的坐标，不在地图外*/
func (d *WaterDot) getPostQuanNeighbors(arrNei []struct{ x, y int }, w *WaterMap) (mostQuanLevel int, poses []struct{ x, y int }) {
	// 原理： highMap[quantity] = []struct{int,int}
	highMap := make(map[int][]struct{ x, y int }, 8)
	for _, nei := range arrNei {
		if nei.x < 0 || nei.x > w.width-1 || nei.y < 0 || nei.y > w.height-1 {
			// 超出地图边界的点
			continue
		}
		high := int(w.data[nei.x+nei.y*w.width].q)
		if len(highMap[int(high)]) == 0 {
			highMap[high] = []struct{ x, y int }{{nei.x, nei.y}} //make([]struct{ x, y int }, 1)
			//highMap[int(m.data[nei.x+nei.y*w.width])][0].x, highMap[int(m.data[nei.x+nei.y*w.width])][0].y = nei.x, nei.y
		} else {
			highMap[high] = append(highMap[high], struct{ x, y int }{nei.x, nei.y})
		}
	}
	mostQuanLevel = 0
	for k, _ := range highMap {
		if k > mostQuanLevel {
			mostQuanLevel = k
		}
	}
	//log.Println("lowest,highMap,count(highMap),d=", lowest, highMap, len(highMap), *d)
	if len(highMap[mostQuanLevel]) == 0 {
		log.Printf("how is can be zero?")
		return mostQuanLevel, nil
	}
	return mostQuanLevel, highMap[mostQuanLevel]
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	var width, height int = 500, 500
	flag.IntVar(&width, "w", width, "width of map")
	flag.IntVar(&height, "h", width, "height of map")
	var outname = flag.String("out", "testmap", "image filename of output")
	var outdir = flag.String("outdir", "output", "out put dir")
	var nHills = flag.Int("hill", 100, "hill number for making rand topo by hill")
	var hillWide = flag.Int("hill-wide", 100, "hill wide for making rand topo by hill")
	var bShowMap = flag.Bool("print", false, "print map for debug")
	var nRidge = flag.Int("ridge", 1, "num of ridges for making ridges")
	var ridgeWide = flag.Int("ridge-wide", 50, "ridge wide for making ridge each")
	//var ridgeStep = flag.Int("ridge-step", 8, "ridge step for making ridge each")
	var ridgeLen = flag.Int("ridge-len", 100, "ridge length when making ridge each")
	var dropNum = flag.Int("dropnum", 100, "number of drops")
	var times = flag.Int("times", 1000, "update times")
	var zoom = flag.Int("zoom", 1, "zoom of out put image")
	var addr = flag.String("addr", "", "addr of http server to listen and to show img on html(deprecated)")
	var riverArrowScale = flag.Float64("river-arrow-scale", 0.8, "river arrow scale")
	var drawFlag = flag.Int("draw-flag", 0, "draw flag: 1=draw filed vecor in topomap 2=draw hisway of droplet")
	var colorTplStep = flag.Int("color-tpl-step", 0, "color tpl file step line, will igore there step in tpl")

	flag.Parse()

	var m Topomap
	var w WaterMap

	// 初始化 watermap topomap
	w.Init(width, height)
	m.Init(width, height)

	if _, derr := os.Open(*outdir); derr != nil {
		log.Println("output dir seems not exist:", *outdir, derr)
		if cerr := os.Mkdir(*outdir, os.ModePerm); cerr != nil {
			log.Println("os.mkdir:", *outdir, cerr)
		}
	}

	log.Printf("will make hills(n:%d,wide:%d)", *nHills, *hillWide)
	// 随机n个圆圈 累加抬高 输出到m中
	hills := MakeHills(width, height, *hillWide, *nHills)

	log.Printf("will make ridge(n:%d, wide:%d)", *ridgeLen, *ridgeWide)
	// 转换痕迹为ridge 为每个环分配随机半径 输出到m中
	var ridgeHills []Hill
	for ri := 0; ri < *nRidge; ri++ {
		ridgeHills = append(ridgeHills, MakeRidge(*ridgeLen, *ridgeWide, width, height)...)
	}
	//log.Println("ridgeHills=", ridgeHills)

	log.Printf("will fill hills and ridges to TopoMap")

	// 生成地图 制造地形 将上面生成的ridge和hills输出到m上
	var tmpColor, maxColor float32 = 1, 1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tmpColor = 0
			// 收集ridgeHills产生的altitude
			for _, r := range ridgeHills {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				rn := (r.r)
				if distM <= r.r*r.r {
					//tmpColor++
					tmpColor += float32(r.h) - float32(float64(r.h)*math.Sqrt(math.Sqrt(float64(distM)/float64((rn*rn)))))
					//tmpColor += float32(r.h) - float32(float64(r.h)*(math.Sqrt(float64(distM)/float64((rn*rn))))) //todo test 效果不好
					//tmpColor += float32(distM) / float32(r.r*r.r) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//log.Println("color fill x,y,r,c=", x, y, r, tmpColor)
				}
			}
			// 收集hills产生的attitude
			for _, r := range hills {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				//rn := float64(r.tiltLen)*math.Sin(r.tiltDir-math.Atan2(float64(y), float64(x))) + float64(r.r)	// 尝试倾斜地图中的圆环 尝试失败
				rn := (r.r)
				if distM <= int(rn*rn) {
					// 产生的ring中间隆起
					tmpColor += float32(r.h) - float32(float64(r.h)*math.Sqrt(math.Sqrt(float64(distM)/float64((rn*rn)))))
					//tmpColor += float32(distM) / float32(rn*rn) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//log.Println("color fill x,y,r,c=", x, y, r, tmpColor, "r=", r)
				}
			}

			if tmpColor < 0 {
				tmpColor = 0
			}
			m.data[x+y*width] = uint8(tmpColor) //+ uint8(rand.Int()%2) //int8(width - x)
		}
	}
	log.Printf("will make drops, maxColor=%f", maxColor)
	maxColor *= 1.2

	w.AssignVector(&m, 3)

	// 生成一组随机*Droplet
	drops := make([]*Droplet, *dropNum)
	for di := 0; di < *dropNum; di++ {
		drops[di] = MakeDroplet(&w)
	}

	log.Printf("will move drops")
	drops = DropletsMove(*times, drops, &m, &w)
	log.Printf("update drops done. times=%d num drops=%d->%d", *times, *dropNum, len(drops))

	log.Printf("will draw to image")
	// then draw
	img := image.NewRGBA(image.Rect(0, 0, width**zoom, height**zoom))

	DrawToImg(img, &m, &w, maxColor, *zoom, *riverArrowScale, drops, *drawFlag, *colorTplStep)

	wgm := sync.WaitGroup{}
	if *addr != "" {
		wgm.Add(2)
		go func() { drawer.StartHtmlDrawer(*addr); wgm.Done() }()
		go func() { DrawToHtml(&w, &m); wgm.Done() }()
		log.Printf("drow to html ok, open host(%s) and view", *addr)
	}

	// 输出图片文件
	wgm.Add(1)
	go func() {
		ImgToFile(fmt.Sprintf("%s/%s-%s.png", *outdir, *outname, time.Now().Format("20060102150405")), img, "png")
		wgm.Done()
	}()

	// 如果需要控制台打印地形
	if *bShowMap {
		wgm.Add(1)
		go func() { DrawToConsole(&m); wgm.Done() }()
	}
	wgm.Wait()
	log.Println("done w,h=", width, height, "maxColor=", maxColor, "nHills=", *nHills, "nRidge=", *nRidge, "ridgelen=", *ridgeLen)
	for di, d := range drops {
		log.Printf("[%d]=%+v", di, *d)
	}

	log.Printf("waterMap.sum(h)=%d events=%d", w.SumH(), w.evtIdx)
}

const (
	DrawFlagField  = iota << 1
	DrawFlagHisway = iota
)

func DrawToImg(img *image.RGBA, m *Topomap, w *WaterMap, maxColor float32, zoom int, riverArrowScale float64, drops []*Droplet, drawFlag int, colorTplStep int) {
	height := m.height
	width := m.width
	var tmpColor float32 = 1

	// 获取颜色模板
	cs := colorTpl(colorTplFile, colorTplStep)
	cslen := len(cs) - 1
	// 地图背景地形绘制
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			//tmpColor = (float32(m.data[x+y*width]/2) - 0.001) * 2
			tmpColor = float32(m.data[x+y*width])
			//img.Set(x, y, color.RGBA{uint8(0xFF * tmpColor / maxColor), 0xFF, uint8(0xFF * tmpColor / maxColor), 0xFF})
			// 比例上色
			//img.Set(x, y, cs[int(float32(cslen)*(1.0-tmpColor/maxColor))])
			// 按值上色
			//img.Set(x, y, cs[cslen-int(tmpColor)])
			// 放大
			for zix := 0; zix < zoom; zix++ {
				for ziy := 0; ziy < zoom; ziy++ {
					img.Set(x*zoom+zix, y*zoom+ziy, cs[int(float32(cslen)*(tmpColor/maxColor))])
				}
			}
		}
	}

	// 绘制WaterMap
	tmpLakeColor := color.RGBA{0, 0xa0, 0xE0, 0xFF}     // alpha=255 表示不透明 青色 有积水
	tmpLakeColor2 := color.RGBA{0x40, 0x72, 0xcb, 0xFF} //#b0d2eb    // blue-gray // 流量痕迹
	//tmpColor4 := color.RGBA{66, 50, 209, 0xFF}          //    // 蓝紫色 // 水滴
	tmpLakeColor3 := color.RGBA{0x99, 0xFF, 0xFF, 0xFF} //#亮蓝色    // 水滴痕迹
	tmpColor4 := color.RGBA{0x50, 0xd6, 0xFE, 0xFF}     //    // 青色 // 水滴最终位置
	//tmpColor5 := color.CMYK{100, 10, 10, 0}
	for _, dot := range w.data {
		// 绘制积水 点周围绘制
		if dot.h > 0 {
			//img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2, tmpLakeColor) // self
			//img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2+1, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2, tmpLakeColor)
			//img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2-1, tmpLakeColor)
			//img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2-1, tmpLakeColor)
			//img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2-1, tmpLakeColor)
			//img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2, tmpLakeColor)
			//img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2+1, tmpLakeColor)
			//img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2+1, tmpLakeColor)
		}
		if dot.q > 0 {
			img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2, tmpLakeColor2)
			//img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2-1, tmpLakeColor2)
			//img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2, tmpLakeColor2)
			//img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2+1, tmpColor5)
		}
	}

	// 绘制流动 在v2下相当于场
	if (drawFlag & DrawFlagField) > 0 {
		for di, dot := range w.data {
			// 绘制当前点 如果是源头 则绘制白色
			if dot.xPower != 0.0 || dot.yPower != 0.0 {
				// 计算相对比例尺的高度
				tmpLevel := int(m.data[di]) + dot.h
				tmpLevel = int(float32(cslen*tmpLevel) / maxColor)
				// 防止越界
				if tmpLevel >= len(cs) {
					tmpLevel = len(cs) - 1
				}
				if tmpLevel < 0 {
					tmpLevel = 0
				}

				// 绘制流动方向 考虑缩放
				tmpColor := cs[tmpLevel]
				lineTo(img, int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2, int(dot.x)*zoom+zoom/2+int(float32(zoom)*dot.xPower), int(dot.y)*zoom+zoom/2+int(float32(zoom)*dot.yPower), color.RGBA{0, 0, 0xFF, 0xFF}, tmpColor, riverArrowScale)
			}
		}
	}
	// 绘制droplets
	for _, drop := range drops {
		if drawFlag&DrawFlagHisway > 0 {
			for _, dxi := range drop.hisway {
				img.Set(int(dxi%width)*zoom+zoom/2, int(dxi/width)*zoom+zoom/2, tmpLakeColor3)
				//img.Set(int(dxi%width)*zoom+zoom/2+1, int(dxi/width)*zoom+zoom/2+1, tmpLakeColor3)
				//img.Set(int(dxi%width)*zoom+zoom/2+1, int(dxi/width)*zoom+zoom/2-1, tmpLakeColor3)
				//img.Set(int(dxi%width)*zoom+zoom/2-1, int(dxi/width)*zoom+zoom/2-1, tmpLakeColor3)
				//img.Set(int(dxi%width)*zoom+zoom/2-1, int(dxi/width)*zoom+zoom/2+1, tmpLakeColor3)
			}
		}
		img.Set(int(drop.x)*zoom+zoom/2, int(drop.y)*zoom+zoom/2, tmpColor4)
	}
	// 绘制颜色模板
	for i := 0; i < len(cs); i++ {
		c := cs[len(cs)-i-1]
		for wi := 0; wi < 5; wi++ {
			img.Set(wi, i, c)
		}
	}
}

func DrawToConsole(m *Topomap) {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ~!@#$%^&*-=_+()[]{}<>\\/;:,.???????????????????????????????????????"
	for di, dd := range m.data {
		fmt.Printf("%c", str[dd])
		if di%m.width == (m.width - 1) {
			fmt.Printf("\n")
		}
	}
}

func DrawToHtml(w *WaterMap, m *Topomap) {
	// todo: use svg
	footerHtml := "<table>"
	for wi := 0; wi < w.width; wi++ {
		footerHtml += "<tr>"
		for hi := 0; hi < w.height; hi++ {
			idx := hi*w.height + wi
			tdot := &w.data[hi*w.height+wi]
			footerHtml += fmt.Sprintf(`<td title="xpower=%f ypower=%f h=%d" style="width:1px;height:1px;background:rgb(0,%d,0)">&nbsp;&nbsp;</td>`,
				tdot.xPower, tdot.yPower, m.data[idx], m.data[idx]*10)
		}
		footerHtml += "</tr>"
	}
	footerHtml += "</table>"

	drawer.SetHomeDrawHandler(func(rw http.ResponseWriter) {
		rw.Write([]byte(footerHtml))
	})
	log.Printf("draw to html ok")
}

// ridgeLen=count(Hill)
/**
ridge = []Hill
ridgeLen = Hill 个数
ridgeWide = Hill wide
*/
func MakeRidge(ridgeLen, ridgeWide, mWidth, mHeight int) []Hill {
	ridgeHills := make([]Hill, ridgeLen)
	// toward as step
	//baseTowardX, baseTowardY := (rand.Int()%ridgeWide)-(rand.Int()%ridgeWide), (rand.Int()%ridgeWide)-(rand.Int()%ridgeWide)
	baseTowardX1, baseTowardY1 := randomDir()
	baseTowardX, baseTowardY := int(baseTowardX1*float32(ridgeWide)), int(baseTowardY1*float32(ridgeWide))
	log.Printf("baseTowardX,baseTowardY=%d,%d  sqare=%d", baseTowardX, baseTowardY, baseTowardX*baseTowardX+baseTowardY*baseTowardY)
	for ri := 0; ri < int(ridgeLen); ri++ {
		r := &ridgeHills[ri]
		// todo max height as const
		r.h = rand.Int()%(7) + 3
		if ri == 0 {
			// 第一个
			r.x, r.y, r.r = (rand.Int() % mWidth), (rand.Int() % mHeight), (rand.Int()%ridgeWide)/2+ridgeWide/2
		} else {
			// 其他 基础方向: ridgeHills[ri-1].x+baseTowardX 摆动:(rand.Int()%ridgeWide)/2-(rand.Int()%ridgeWide)/2
			waveX, waveY := 0, 0
			if baseTowardX != 0 {
				waveY = (rand.Int() % baseTowardX) - (rand.Int() % baseTowardX)
			}
			if baseTowardY != 0 {
				waveX = (rand.Int() % baseTowardY) - (rand.Int() % baseTowardY)
			}
			r.x, r.y, r.r = ridgeHills[ri-1].x+baseTowardX/2+waveX, ridgeHills[ri-1].y+baseTowardY/2+waveY, (rand.Int()%ridgeWide)/2+ridgeWide/2
		}
	}

	return ridgeHills
}

// todo 树枝型ridge
func MakeRidge2(startX, startY int, ridgeLen, ridgeWide, mWidth, mHeight int) []Hill {
	ridgeHills := make([]Hill, ridgeLen)
	baseTowardX, baseTowardY := (rand.Int()%mWidth-mWidth/2)/20, (rand.Int()%mHeight-mHeight/2)/20
	//incrX, incrY := (rand.Int()%ridgeWide)-ridgeWide/2, (rand.Int()%ridgeWide)-ridgeWide/2
	for ri := 0; ri < int(ridgeLen); ri++ {
		r := &ridgeHills[ri]
		if ri == 0 {
			// 第一个
			//r.x, r.y, r.r, r.h = (rand.Int() % mWidth), (rand.Int() % mHeight), (rand.Int()%(ridgeWide) + 1), (rand.Int()%(5) + 2)
			r.x, r.y, r.r, r.h = startX, startY, (rand.Int()%(ridgeWide) + 1), (rand.Int()%(5) + 2)
		} else {
			// 其他
			r.x, r.y, r.r, r.h = ridgeHills[ri-1].x+(rand.Int()%ridgeWide)-ridgeWide/2+baseTowardX, ridgeHills[ri-1].y+(rand.Int()%ridgeWide)-ridgeWide/2+baseTowardY, (rand.Int()%(ridgeWide) + 1), (rand.Int()%(5) + 2)
		}
	}

	return ridgeHills
}

func ImgToFile(outputFilePath string, img *image.RGBA, format string) {
	picFile2, err := os.Create(outputFilePath)
	if err != nil {
		log.Printf("when create file %s error:%v", outputFilePath, err)
		return
	}
	defer picFile2.Close()
	if err := png.Encode(picFile2, img); err != nil {
		log.Println("png.Encode error:", err)
	}
}

//func (m *Topomap)MarshalJSON() ([]byte, error) {
//	outputBuf := []byte{}
//	return outputBuf, nil
//}
//
//func (m *Topomap)UnmarshalJSON(inputBuf []byte) error {
//
//	return nil
//}

// todo save map, load map
//func SaveMap(m *Topomap, filepath string) error {
//	b, err := json.Marshal(m)
//	if err != nil {
//		return err
//	}
//
//	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR, os.ModePerm)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//
//	f.Write(b)
//
//	return nil
//}
//
//func LoadMap(filepath string) (*Topomap, error) {
//	return nil, nil
//}

const (
	// 水滴之间的吸引力度 类似于万有引力常量
	AttractPowerDecay = 0.25
)

// 会改变d的方向 即会改变 vx,vy 值
func (d *Droplet) CloseTo(target *Droplet, distSquare float32) {
	//
	distSquareRoot := math.Sqrt(float64(distSquare))
	d.vx = d.vx + (target.x-d.x)*float32(distSquareRoot)*AttractPowerDecay
	d.vy = d.vy + (target.y-d.y)*float32(distSquareRoot)*AttractPowerDecay
}

const (
	// 距离平方超过这个值 就会被等比例缩减速度 但是保持方向
	MinDistToReduce = 2
)

func (d *Droplet) ReduceSpeed() {
	vSquare := d.vx*d.vx + d.vy*d.vy
	if vSquare > MinDistToReduce {
		scale := float32(math.Sqrt(float64(vSquare / MinDistToReduce)))
		d.vx, d.vy = d.vx/scale, d.vy/scale
		//log.Printf("velo squashed, vSquare:%f", vSquare)
	}
}

// 返回随机方向
func randomDir() (x, y float32) {
	thedir := rand.Float64() * math.Pi * 2
	x, y = float32(math.Cos(thedir)), float32(math.Sin(thedir))
	return
}

func MakeHills(width, height, hillWide, num int) []Hill {
	// 随机n个圆圈 累加抬高 输出到m中
	widthEdge := width / 15
	heightEdge := height / 15
	hills := make([]Hill, num)
	for ri, _ := range hills {
		r := &hills[ri]
		// todo 地图边框附近不要去
		r.x, r.y = (rand.Int()%(width-widthEdge*2))+widthEdge, (rand.Int()%(height-heightEdge*2))+heightEdge
		// 倾斜度 todo tilt: 未生效
		r.tiltDir, r.tiltLen, r.r, r.h = rand.Float64()*math.Pi*2, (rand.Int()%20)+1, int(math.Sqrt(float64(rand.Int()%(hillWide*hillWide+1)))), (rand.Int()%(5) + 2)
		// todo max height as const
		if ri%3 == 1 {
			r.h *= -1
		}
		//log.Printf("ri=%d h=%d s=%d", ri, r.h, (ri%2)*2-1)
	}

	//log.Printf("wedge=%d hedge=%d hills=%v", widthEdge, heightEdge, hills)
	return hills
}

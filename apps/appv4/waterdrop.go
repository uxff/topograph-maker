package main

import (
	"log"
	"math"
	"math/rand"
	"sync"
)

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

	// 距离平方在2以内的WaterDot有吸引力 // todo 使用分层数组索引
	for i := 0; i < len(drops); i += 4 {
		di := drops[i+((step)%4)] // 几率变成4分之1 但是不会重复，会轮询
		distSquare := (di.x-d.x)*(di.x-d.x) + (di.y-d.y)*(di.y-d.y)
		if distSquare < DropsCohesiveDistSq {
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

// 会改变d的方向 即会改变 vx,vy 值
func (d *Droplet) CloseTo(target *Droplet, distSquare float32) {
	//
	distSquareRoot := math.Sqrt(float64(distSquare))
	d.vx = d.vx + (target.x-d.x)*float32(distSquareRoot)*AttractPowerDecay
	d.vy = d.vy + (target.y-d.y)*float32(distSquareRoot)*AttractPowerDecay
}

func (d *Droplet) ReduceSpeed() {
	vSquare := d.vx*d.vx + d.vy*d.vy
	if vSquare > MinDistToReduce {
		scale := float32(math.Sqrt(float64(vSquare / MinDistToReduce)))
		d.vx, d.vy = d.vx/scale, d.vy/scale
		//log.Printf("velo squashed, vSquare:%f", vSquare)
	}
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

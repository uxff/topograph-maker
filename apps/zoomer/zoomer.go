package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

type Dot struct {
	x int
	y int
}

func (d *Dot) ToFlatIdx(width int) int {
	return d.y*width + d.x
}

func (d *Dot) FromFlatIdx(idx, width int) *Dot {
	d.x = idx % width
	d.y = idx / width
	return d
}

func getNeighbors(d *Dot) []Dot {
	pos := make([]Dot, 8)
	pos[0].x, pos[0].y = int(d.x+1), int(d.y)
	pos[1].x, pos[1].y = int(d.x+1), int(d.y-1)
	pos[2].x, pos[2].y = int(d.x), int(d.y-1)
	pos[3].x, pos[3].y = int(d.x-1), int(d.y-1)
	pos[4].x, pos[4].y = int(d.x-1), int(d.y)
	pos[5].x, pos[5].y = int(d.x-1), int(d.y+1)
	pos[6].x, pos[6].y = int(d.x), int(d.y+1)
	pos[7].x, pos[7].y = int(d.x+1), int(d.y+1)
	return pos
}

func (d *Dot) GetNE() []Dot {
	pos := make([]Dot, 3)
	pos[0].x, pos[0].y = d.x, d.y-1
	pos[1].x, pos[1].y = d.x+1, d.y-1
	pos[2].x, pos[2].y = d.x+1, d.y
	return pos
}
func (d *Dot) GetSE() []Dot {
	pos := make([]Dot, 3)
	pos[0].x, pos[0].y = d.x+1, d.y
	pos[1].x, pos[1].y = d.x+1, d.y+1
	pos[2].x, pos[2].y = d.x, d.y+1
	return pos
}
func (d *Dot) GetSW() []Dot {
	pos := make([]Dot, 3)
	pos[0].x, pos[0].y = d.x, d.y+1
	pos[1].x, pos[1].y = d.x-1, d.y+1
	pos[2].x, pos[2].y = d.x-1, d.y
	return pos
}
func (d *Dot) GetNW() []Dot {
	pos := make([]Dot, 3)
	pos[0].x, pos[0].y = d.x-1, d.y
	pos[1].x, pos[1].y = d.x-1, d.y-1
	pos[2].x, pos[2].y = d.x, d.y-1
	return pos
}

func ForeachImgDot(imgIo image.Image, walkHandler func(o image.Image, x int, y int, c color.Color)) error {

	if walkHandler == nil {
		return fmt.Errorf("walkHandler is NULL")
	}

	//var pngIo, err = png.Decode(imgIo)

	//	if err != nil {
	//		fmt.Println("png.decode err when read colorTpl:", err)
	//		return err
	//	}

	width := imgIo.Bounds().Dx()
	height := imgIo.Bounds().Dy()

	for hi := 0; hi < height; hi++ {
		for wi := 0; wi < width; wi++ {
			walkHandler(imgIo, wi, hi, imgIo.At(wi, hi))
		}
	}

	return nil
}

func WalkTopo(o image.Image, x int, y int, c color.Color) {
	// todo: mark every dot
}

func ImgToFile(img image.Image, outputFilePath string) {
	picFile2, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("when create file %s error:%v\n", outputFilePath, err)
		return
	}
	defer picFile2.Close()
	if err := png.Encode(picFile2, img); err != nil {
		fmt.Println("png.Encode error:", err)
	}

}

func main() {

	oldTopoPath := ""
	newTopoPath := ""

	fOld, err := os.Open(oldTopoPath)
	if err != nil {
		fmt.Printf("open:%s failed:%v+\n", oldTopoPath, err)
		return
	}
	defer fOld.Close()

	imgOld, err := png.Decode(fOld)
	if err != nil {
		fmt.Printf("png.Decode:%s failed:%v+\n", oldTopoPath, err)
		return
	}

	width, height := imgOld.Bounds().Dx(), imgOld.Bounds().Dy()

	imgNew := image.NewRGBA(image.Rect(0, 0, width*2, height*2))

	dotCounter := 0

	ForeachImgDot(imgOld, func(o image.Image, x int, y int, c color.Color) {
		dotCounter++
		d := &Dot{x: x, y: y}
		var dots []*Dot
		// dotNE
		dots = d.GetNE()
		for _, do := range dots {
			if do.x < 0 || do.y < 0 || do.x >= width || do.y >= height {
				break
			}
		}
		// dotSE
		oldSEDots := getNeighbors()
		// dotSW
		// dotNW
	})

	ImgToFile(imgNew, newTopoPath)
	fmt.Printf("dotCounter:%d\n", dotCounter)

}

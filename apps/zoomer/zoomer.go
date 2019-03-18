package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"sync"
	"time"
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

func ZoomAllDots(imgIo image.Image, walkHandler func(o image.Image, x int, y int, c color.Color)) error {

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

	wg := &sync.WaitGroup{}

	for hi := 0; hi < height; hi++ {
		// use goroutine for each line
		wg.Add(1)
		go func(hi int) {
			for wi := 0; wi < width; wi++ {
				walkHandler(imgIo, wi, hi, imgIo.At(wi, hi))
			}
			wg.Done()
		}(hi)
	}
	wg.Wait()

	return nil
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

	inputPath := ""
	outputPath := ""

	flag.StringVar(&inputPath, "i", inputPath, "input path")
	flag.StringVar(&outputPath, "o", outputPath, "output path")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	if outputPath == "" {
		outputPath = "x2-" + inputPath
	}

	fOld, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("open:%s failed:%v+\n", inputPath, err)
		return
	}
	defer fOld.Close()

	imgOld, err := png.Decode(fOld)
	if err != nil {
		fmt.Printf("png.Decode:%s failed:%v+\n", inputPath, err)
		return
	}

	width, height := imgOld.Bounds().Dx(), imgOld.Bounds().Dy()

	imgNew := image.NewRGBA(image.Rect(0, 0, width*2, height*2))

	dotCounter := 0

	ZoomAllDots(imgOld, func(o image.Image, x int, y int, c color.Color) {
		dotCounter++
		d := &Dot{x: x, y: y}
		var sideDots []Dot
		// dotNE
		sideDots = d.GetNE()
		c = selColors(o, x, y, width, height, sideDots)
		imgNew.Set(x*2+1, y*2, c)
		// dotSE
		sideDots = d.GetSE()
		c = selColors(o, x, y, width, height, sideDots)
		imgNew.Set(x*2+1, y*2+1, c)
		// dotSW
		sideDots = d.GetSW()
		c = selColors(o, x, y, width, height, sideDots)
		imgNew.Set(x*2, y*2+1, c)
		// dotNW
		sideDots = d.GetNW()
		c = selColors(o, x, y, width, height, sideDots)
		imgNew.Set(x*2, y*2, c)
	})

	ImgToFile(imgNew, outputPath)
	fmt.Printf("dotCounter:%d\n", dotCounter)

}

func selColors(o image.Image, x, y int, width, height int, sideDots []Dot) (c color.Color) {
	// as origin
	//return o.At(x, y)
	// as random
	var colors []color.Color
	colorMap := make(map[int]color.Color)
	for _, do := range sideDots {
		if do.x < 0 || do.y < 0 || do.x >= width || do.y >= height {
			// out of bound
			continue
		}
		cSide := o.At(do.x, do.y)
		colors = append(colors, o.At(do.x, do.y))
		cR, cG, cB, cA := cSide.RGBA()
		colorMap[int((cR<<24)+(cG<<16)+(cB<<8)+(cA))] = cSide
	}
	c = o.At(x, y)
	colors = append(colors, c)
	colors = append(colors, c)
	colors = append(colors, c)
	colors = append(colors, c)
	colors = append(colors, c)
	colors = append(colors, c)

	if len(colors) == 0 {
		fmt.Printf("no color found from sides, use self x:%d y:%d\n", x, y)
	} else {
		// as random
		c = colors[rand.Int()%len(colors)]
		return
		// as side if side same
		if len(colorMap) == 1 {
			for _, cv := range colorMap {
				c = cv
				break
			}
		}
	}
	return c

}

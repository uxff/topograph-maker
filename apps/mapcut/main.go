/**
图片切割 从某个坐标裁剪出指定宽度来
usage:
./mapcut.exe -i USA_topo_en.jpg -o us2.png -x 75 -y 70 -x2 -75 -y2 -70
*/
package main

import (
	"flag"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
	"sync"
	//"github.com/disintegration/imaging"
)

func main() {

	inputFile := flag.String("i", "", "origin input image file, only read it, no write")
	outputFile := flag.String("o", "", "output image file")

	startX, startY := 0, 0
	flag.IntVar(&startX, "x", 0, "start x pos")
	flag.IntVar(&startY, "y", 0, "start y pos")

	toX, toY := 0, 0
	flag.IntVar(&toX, "x2", 0, "to x pos, will override width")
	flag.IntVar(&toY, "y2", 0, "to y pos, will override height")

	width, height := 0, 0
	flag.IntVar(&width, "w", -1, "width you select to output")
	flag.IntVar(&height, "h", -1, "height you select to output")

	var scale float64 = 1.0
	flag.Float64Var(&scale, "scale", scale, "default scale of output")

	flag.Parse()

	imgTplIo, err := os.Open(*inputFile)
	if err != nil {
		log.Printf("open imagefile %s error:%v", *inputFile, err)
		return
	}

	defer imgTplIo.Close()

	imgIn, fmt, err := image.Decode(imgTplIo)
	if err != nil {
		log.Printf("decode imagefile %s error:%v", *inputFile, err)
		return
	}

	log.Printf("fmt of imgTpl(%s)=%s", *inputFile, fmt)

	//imgOutput.ColorModel().Convert().RGBA()

	oriW := imgIn.Bounds().Dx()
	oriH := imgIn.Bounds().Dy()

	if width <= 0 || width >= oriW {
		width = oriW - startX
	}
	if height <= 0 || height >= oriH {
		height = oriH - startY
	}

	if toX > 0 {
		width = toX - startX
	} else if toX < 0 {
		width = oriW + toX - startX
	}
	if toY > 0 {
		height = toY - startY
	} else if toY < 0 {
		height = oriH + toY - startY
	}

	if width <= 0 {
		log.Printf("width %d 不合法，请重新检查", width)
		return
	}
	if height <= 0 {
		log.Printf("height %d 不合法，请重新检查", height)
		return
	}

	imgOut := image.NewRGBA(image.Rect(0, 0, width, height))

	wg := sync.WaitGroup{}
	wg.Add(width)
	for xi := 0; xi < width; xi++ {
		go func(xi int) {
			defer wg.Done()
			for yi := 0; yi < height; yi++ {
				imgOut.Set(xi, yi, imgIn.At(xi+startX, yi+startY))
			}
		}(xi)
	}

	wg.Wait()
	ImgToFile(*outputFile, imgOut, "")
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
	log.Printf("done: %s", outputFilePath)
}

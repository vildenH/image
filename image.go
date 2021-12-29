package main

import (
	"flag"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"image"
	"image/jpeg"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// 自动切换工作目录
func init() {
	pwd, _ := os.Getwd()
	fmt.Println("开始工作目录", pwd)
	// 程序所在目录
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	if pwd == execDir {
		fmt.Println("不需要切换工作目录")
		return
	}
	fmt.Println("切换工作目录到", execDir)
	if err := os.Chdir(execDir); err != nil {
		log.Fatal(err)
	}
	pwd, _ = os.Getwd()
	fmt.Println("切换后工作目录:", pwd)
}

var name = "SOMETHINGFORNOTHING"

var (
	dpi      = flag.Float64("dpi", 128, "screen resolution in Dots Per Inch")
	fontfile = flag.String("fontfile", "font.ttf", "filename of the ttf font")
	spacing  = flag.Float64("spacing", 1.5, "line spacing (e.g. 2 means double spaced)")
)

func getOrientation(reader io.Reader) string {
	x, err := exif.Decode(reader)
	if err != nil {
		return "1"
	}
	if x != nil {
		orient, err := x.Get(exif.Orientation)
		if err != nil {
			return "1"
		}
		if orient != nil {
			return orient.String()
		}
	}

	return "1"
}

//Decode is image.Decode handling orientation in EXIF tags if exists.
//Requires io.ReadSeeker instead of io.Reader.
func Decode(reader io.ReadSeeker) (image.Image, string, error) {
	img, fmt, err := image.Decode(reader)
	if err != nil {
		return img, fmt, err
	}
	reader.Seek(0, io.SeekStart)
	orientation := getOrientation(reader)
	switch orientation {
	case "1":
	case "2":
		img = imaging.FlipH(img)
	case "3":
		img = imaging.Rotate180(img)
	case "4":
		img = imaging.Rotate180(imaging.FlipH(img))
	case "5":
		img = imaging.Rotate270(imaging.FlipV(img))
	case "6":
		img = imaging.Rotate270(img)
	case "7":
		img = imaging.Rotate90(imaging.FlipV(img))
	case "8":
		img = imaging.Rotate90(img)
	}

	return img, fmt, err
}

func readFont() *truetype.Font {
	//读取字体
	fontBytes, err := ioutil.ReadFile(*fontfile)
	if err != nil {
		log.Println(err)
		return nil
	}
	//解析字体
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		return nil
	}

	return f
}

func showImage(img image.Image) {
	a := app.New()
	w := a.NewWindow("Images")

	canvasImg := canvas.NewImageFromImage(img)
	w.SetContent(canvasImg)
	w.Resize(fyne.NewSize(float32(img.Bounds().Max.X/10), float32(img.Bounds().Max.Y/10)))
	w.ShowAndRun()
}

func pre() []fs.FileInfo {
	files, _ := ioutil.ReadDir("./")

	path := "./waterMark"
	b, err := PathExists(path)
	if err != nil {
		fmt.Printf("PathExists(%s),err(%v)\n", path, err)
	}
	if b {
		fmt.Printf("path %s 存在\n", path)
	} else {
		fmt.Printf("path %s 不存在\n", path)
		err := os.Mkdir("waterMark", os.ModePerm)
		if err != nil {
			fmt.Printf("mkdir failed![%v]\n", err)
		} else {
			fmt.Printf("mkdir success!\n")
		}
	}
	return files
}

/*
   判断文件或文件夹是否存在
   如果返回的错误为nil,说明文件或文件夹存在
   如果返回的错误类型使用os.IsNotExist()判断为true,说明文件或文件夹不存在
   如果返回的错误为其它类型,则不确定是否在存在
*/
func PathExists(path string) (bool, error) {

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func addWaterMark(filename string) *image.NRGBA {
	imgFile, err := os.Open(filename)
	if err != nil {
		fmt.Println("打开图片出错")
		fmt.Println(err)
		os.Exit(-1)
	}
	defer imgFile.Close()

	img, _, err := Decode(imgFile)
	if err != nil {
		fmt.Println("把图片解码为结构体时出错")
		fmt.Println(img)
		os.Exit(-1)
	}

	// 初始化图片背景
	fg := image.White
	rgba := image.NewNRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

	//在图片上面添加文字
	c := freetype.NewContext()
	c.SetDPI(*dpi)

	//设置字体
	c.SetFont(readFont())
	//设置大小
	fontSize := float64(img.Bounds().Max.Y / 50)
	c.SetFontSize(fontSize)
	//设置边界
	c.SetClip(rgba.Bounds())
	//设置背景底图
	c.SetDst(rgba)
	//设置背景图
	c.SetSrc(fg)
	// 优化
	c.SetHinting(font.HintingFull)

	// 画文字
	i := img.Bounds().Max.X / 2
	fixed := c.PointToFixed(fontSize)
	nameSize := (utf8.RuneCountInString(name) / 3) * fixed.Ceil()
	pt := freetype.Pt(i-nameSize, img.Bounds().Max.Y/20)
	c.DrawString(name, pt)
	//showImage(rgba)

	return rgba
}

func saveFile(img image.Image, fileName string) {
	outputFile, err := os.Create("./waterMark/" + fileName)
	if err != nil {
		fmt.Println("打开图片出错")
		fmt.Println(err)
		os.Exit(-1)
	}
	defer outputFile.Close()
	jpeg.Encode(outputFile, img, &jpeg.Options{100})
}

func main() {
	//预处理,创建输出目录，读取文件名
	files := pre()
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.Name()), "jpg") || strings.HasSuffix(strings.ToLower(f.Name()), "jpeg") {
			fmt.Println("handle:" + f.Name())
			img := addWaterMark(f.Name())
			saveFile(img, f.Name())
		}
	}

}

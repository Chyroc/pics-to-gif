package main

import (
	"flag"
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/andybons/gogif"
	"gopkg.in/cheggaaa/pb.v1"
)

type File struct {
	Index int
	Name  string
}

type Image struct {
	Img *image.Paletted
	Err error
}

var folder string
var outFilename string
var gorou int

func ReadFileList(folder string) ([]string, error) {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, v := range files {
		names = append(names, folder+"/"+v.Name())
	}

	return names, nil
}

func ReadToGif(filename string) (*image.Paletted, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	simage, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := simage.Bounds()
	palettedImage := image.NewPaletted(bounds, nil)
	quantizer := gogif.MedianCutQuantizer{NumColor: 64}
	quantizer.Quantize(palettedImage, bounds, simage, image.ZP)

	return palettedImage, nil
}

func SveAsGif(filename string, g *gif.GIF) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return gif.EncodeAll(f, g)
}

func Run(folder, output string, p int) error {
	files, err := ReadFileList(folder)
	if err != nil {
		return err
	}

	fileCount := len(files)

	FilePb := pb.StartNew(fileCount).Prefix("File ")
	var fs = make(chan File, fileCount)
	var gs = make([]*image.Paletted, fileCount)
	var outGif = new(gif.GIF)

	for k, v := range files {
		FilePb.Increment()
		fs <- File{
			Index: k,
			Name:  v,
		}
	}
	FilePb.Finish()

	ReadImgPb := pb.StartNew(fileCount).Prefix("ReadImg ")
	var sw sync.WaitGroup
	for i := 0; i < p; i++ {
		sw.Add(1)
		go func() {
			defer sw.Done()
			for {
				select {
				case f := <-fs:
					img, err := ReadToGif(f.Name)
					if err != nil {
						panic(f.Name)
					}
					gs[f.Index] = img
					ReadImgPb.Increment()
				default:
					return
				}
			}
		}()
	}
	sw.Wait()
	ReadImgPb.Finish()

	AppendGIFPb := pb.StartNew(fileCount + 1).Prefix("AppendGIF ")
	for _, v := range gs {
		outGif.Image = append(outGif.Image, v)
		outGif.Delay = append(outGif.Delay, 0)
		AppendGIFPb.Increment()
		time.Sleep(time.Second)
	}
	err = SveAsGif(output, outGif)

	AppendGIFPb.Increment()
	AppendGIFPb.Finish()

	return err
}

func init() {
	if len(os.Args) == 1 || strings.ToUpper(os.Args[1]) == "-H" || strings.ToUpper(os.Args[1]) == "--HELP" {
		fmt.Printf(`NAME:
   pics-to-gif - 图片转GIF动图

USAGE:
   t -f folder [-o out_filename] [-p goroutine]

VERSION:
   v0.1.0

ARGS:
     -f   图片所在文件夹
     -o   输出GIF文件名
     -p   并发数

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
`)
		os.Exit(0)
	}

	flag.StringVar(&folder, "f", "", "")
	flag.StringVar(&outFilename, "o", "out.gif", "")
	flag.IntVar(&gorou, "p", 10, "")
	flag.Parse()

	if folder == "" || outFilename == "" {
		log.Fatal(fmt.Sprintf("-f or -o 不能为空"))
	}
	if gorou <= 0 {
		log.Fatal(fmt.Sprintf("-p 必须大于0"))
	}
}

func main() {
	err := Run(folder, outFilename, gorou)
	if err != nil {
		log.Fatal(fmt.Sprintf("Err [%s]", err))
	}
}

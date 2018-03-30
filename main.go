package main

import (
	"image"
	"image/gif"
	"image/png"
	"io/ioutil"
	"os"
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

func main() {
	err := Run("./", "./out.gif", 1)
	if err != nil {
		panic(err)
	}
}

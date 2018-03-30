package main

import (
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"io/ioutil"
	"os"
	"sync"

	"github.com/andybons/gogif"
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

	var fs = make(chan File, len(files))
	var gs = make([]Image, len(files))
	var outGif = new(gif.GIF)

	for k, v := range files {
		fs <- File{
			Index: k,
			Name:  v,
		}
	}

	var sw sync.WaitGroup
	for i := 0; i < p; i++ {
		sw.Add(1)
		go func() {
			defer sw.Done()
			for {
				select {
				case f := <-fs:
					fmt.Printf("%d\n", f.Index)
					img, err := ReadToGif(f.Name)
					gs[f.Index] = Image{
						Img: img,
						Err: err,
					}
				default:
					return
				}
			}
		}()
	}
	sw.Wait()

	for _, v := range gs {
		if v.Err != nil {
			return err
		}
		outGif.Image = append(outGif.Image, v.Img)
		outGif.Delay = append(outGif.Delay, 0)
	}

	return SveAsGif(output, outGif)
}

func main() {
	err := Run("./", "out.gif", 20)
	if err != nil {
		panic(err)
	}
}

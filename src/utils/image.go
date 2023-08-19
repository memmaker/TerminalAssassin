package utils

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"

	"github.com/memmaker/terminal-assassin/common"
)

func GetPixelsFromImage(fileSystem fs.FS, filename string) ([][]common.Color, error) {
	// You can register another format here
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	file, err := fileSystem.Open(filename)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	defer file.Close()

	pixels, err := getPixels(file)

	if err != nil {
		return nil, fmt.Errorf("error while decoding image: %w", err)
	}

	return pixels, nil
}

// Get the bi-dimensional pixel array
func getPixels(file io.Reader) ([][]common.Color, error) {
	img, _, err := image.Decode(file)

	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]common.Color
	for y := 0; y < height; y++ {
		var row []common.Color
		for x := 0; x < width; x++ {
			row = append(row, common.RGBAFrom32Bits(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}

	return pixels, nil
}

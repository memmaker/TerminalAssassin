package utils

import (
	"encoding/binary"
	"io/fs"
	"os"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	//	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type CellImage struct {
	Width  uint64
	Height uint64
	Cells  []common.Cell
}

func NewCellImage(width uint64, height uint64, cells []common.Cell) *CellImage {
	return &CellImage{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}
func (ci *CellImage) DrawOrigin(topLeftCorner geometry.Point, con console.CellInterface) {
	for y := uint64(0); y < ci.Height; y++ {
		for x := uint64(0); x < ci.Width; x++ {
			cell := ci.Cells[y*ci.Width+x]
			drawPos := topLeftCorner.Add(geometry.Point{X: int(x), Y: int(y)})
			con.SetSquare(drawPos, cell)
		}
	}
}

func (ci *CellImage) DrawCentered(con console.CellInterface) {
	screenWidth, screenHeight := con.Size().X, con.Size().Y
	topLeftCorner := geometry.Point{
		X: (screenWidth - int(ci.Width)) / 2,
		Y: (screenHeight - int(ci.Height)) / 2,
	}
	ci.DrawOrigin(topLeftCorner, con)
}
func (ci *CellImage) SaveToDisk(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	err = binary.Write(file, binary.LittleEndian, ci.Width)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.LittleEndian, ci.Height)
	if err != nil {
		return err
	}
	colorPalette := make(map[common.RGBAColor]uint8, 0)
	currentPalletteIndex := uint8(0)
	for _, cell := range ci.Cells {
		err = binary.Write(file, binary.LittleEndian, cell.Rune)
		if err != nil {
			return err
		}
		bgIndex := uint8(0)
		fgIndex := uint8(0)
		if _, ok := colorPalette[cell.Style.Background.ToRGB()]; !ok {
			colorPalette[cell.Style.Background.ToRGB()] = currentPalletteIndex
			currentPalletteIndex++
		}
		bgIndex = colorPalette[cell.Style.Background.ToRGB()]

		if _, ok := colorPalette[cell.Style.Foreground.ToRGB()]; !ok {
			colorPalette[cell.Style.Foreground.ToRGB()] = currentPalletteIndex
			currentPalletteIndex++
		}
		fgIndex = colorPalette[cell.Style.Foreground.ToRGB()]

		err = binary.Write(file, binary.LittleEndian, bgIndex)
		if err != nil {
			return err
		}
		err = binary.Write(file, binary.LittleEndian, fgIndex)
		if err != nil {
			return err
		}
	}
	paletteSize := uint32(len(colorPalette))

	orderPalette := make([]common.RGBAColor, paletteSize)
	for color, index := range colorPalette {
		orderPalette[index] = color
	}
	err = binary.Write(file, binary.LittleEndian, paletteSize)
	if err != nil {
		return err
	}
	for _, color := range orderPalette {
		err = binary.Write(file, binary.LittleEndian, color)
		if err != nil {
			return err
		}
	}
	return nil
}

type FileOpener interface {
	Open(filename string) (fs.File, error)
}
type IndexedCell struct {
	Icon        rune
	BgTileIndex uint8
	FgTileIndex uint8
}

func LoadCellImageFromDisk(fileInterface FileOpener, filename string) *CellImage {
	openFile, err := fileInterface.Open(filename)
	if err != nil {
		println(err.Error())
		return nil
	}
	defer openFile.Close()
	var width, height uint64
	binary.Read(openFile, binary.LittleEndian, &width)
	binary.Read(openFile, binary.LittleEndian, &height)
	indexedCells := make([]IndexedCell, width*height)
	for i := uint64(0); i < width*height; i++ {
		var icon rune
		binary.Read(openFile, binary.LittleEndian, &icon)
		var bgIndex, fgIndex uint8
		binary.Read(openFile, binary.LittleEndian, &bgIndex)
		binary.Read(openFile, binary.LittleEndian, &fgIndex)
		indexedCells[i] = IndexedCell{
			Icon:        icon,
			BgTileIndex: bgIndex,
			FgTileIndex: fgIndex,
		}
	}
	var paletteSize uint32
	binary.Read(openFile, binary.LittleEndian, &paletteSize)
	palette := make([]common.RGBAColor, paletteSize)
	for i := uint32(0); i < paletteSize; i++ {
		var color common.RGBAColor
		binary.Read(openFile, binary.LittleEndian, &color)
		palette[i] = color
	}
	cells := make([]common.Cell, width*height)
	for i := uint64(0); i < width*height; i++ {
		cell := indexedCells[i]
		cells[i] = common.Cell{
			Rune: cell.Icon,
			Style: common.Style{
				Background: palette[cell.BgTileIndex],
				Foreground: palette[cell.FgTileIndex],
			},
		}
	}
	return &CellImage{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}

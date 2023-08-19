package console

import (
	"embed"
	"log"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tinne26/etxt"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/geometry"
)

type CellInterface interface {
	SetSquare(p geometry.Point, cell common.Cell)
	AtSquare(p geometry.Point) common.Cell
	SetHalfWidth(p geometry.Point, cell common.Cell)
	AtHalfWidth(p geometry.Point) common.Cell
	Size() geometry.Point
	Flush()
	ClearConsole()
	SquareFill(bbox geometry.Rect, cell common.Cell)
	HalfWidthFill(bbox geometry.Rect, cell common.Cell)
	SquareBlack()
	HalfWidthTransparent()
	ClearSurface()
	HalfWidthBlack()
	Contains(pos geometry.Point) bool
}

type GridConfig struct {
	TileWidth      int
	TileHeight     int
	GridWidth      int
	GridHeight     int
	MaxVisionRange int
}

type Console struct {
	// basics
	screenDPIScale float64
	TileWidth      int
	TileHeight     int

	// fonts
	txtRenderer   *etxt.Renderer
	squareFont    *etxt.Font
	halfWidthFont *etxt.Font

	// buffers
	squareBuffer    *ebiten.Image
	halfWidthBuffer *ebiten.Image

	// delta drawing
	squareCurrentGrid      geometry.Grid    // from the user's perspective, he always draws to this console
	squarePreviousGrid     geometry.Grid    // we consolidate changes to this console
	squareAccumulatedFrame AccumulatedFrame // represents accumulated changes since the last frame draw, that's what we draw. This gets cleared after each draw
	squareDirty            bool             // indicates whether the console has been modified since the last draw

	// half width drawing
	halfWidthCurrentGrid      geometry.Grid    // we always draw to this console
	halfWidthPreviousGrid     geometry.Grid    // we consolidate changes to this console
	halfWidthAccumulatedFrame AccumulatedFrame // represents accumulated changes since the last frame draw, that's what we draw. This gets cleared after each draw
	halfWidthDirty            bool             // indicates whether the console has been modified since the last draw

	tileFontLib         *etxt.FontLibrary
	textFontLib         *etxt.FontLibrary
	clearBeforeNextDraw bool
	GridWidth           int
	GridHeight          int
}

func (c *Console) Size() geometry.Point {
	return c.squareCurrentGrid.Size()
}
func (c *Console) SetSquare(p geometry.Point, cell common.Cell) {
	c.squareCurrentGrid.Set(p, cell)
	c.squareDirty = true
}

func (c *Console) AtSquare(p geometry.Point) common.Cell {
	return c.squareCurrentGrid.At(p)
}

func (c *Console) SetHalfWidth(p geometry.Point, cell common.Cell) {
	c.halfWidthCurrentGrid.Set(p, cell)
	c.halfWidthDirty = true
}

func (c *Console) AtHalfWidth(p geometry.Point) common.Cell {
	return c.halfWidthCurrentGrid.At(p)
}

func NewConsole(config GridConfig) *Console {
	newCon := &Console{
		TileWidth:                 config.TileWidth,
		TileHeight:                config.TileHeight,
		GridWidth:                 config.GridWidth,
		GridHeight:                config.GridHeight,
		txtRenderer:               NewTextRenderer(),
		squareCurrentGrid:         geometry.NewGrid(config.GridWidth, config.GridHeight),
		squareAccumulatedFrame:    make(map[geometry.Point]common.Cell, config.GridWidth*config.GridHeight),
		halfWidthCurrentGrid:      geometry.NewGrid(config.GridWidth*2, config.GridHeight),
		halfWidthAccumulatedFrame: make(map[geometry.Point]common.Cell, config.GridWidth*config.GridHeight*2),
		squareBuffer:              ebiten.NewImage(config.GridWidth*config.TileWidth, config.GridHeight*config.TileHeight),
		halfWidthBuffer:           ebiten.NewImage(config.GridWidth*config.TileWidth, config.GridHeight*config.TileHeight),
		halfWidthPreviousGrid:     geometry.NewGrid(config.GridWidth*2, config.GridHeight),
		squarePreviousGrid:        geometry.NewGrid(config.GridWidth, config.GridHeight),
	}
	newCon.HalfWidthTransparent()
	return newCon
}

func (c *Console) ClearConsole() {
	c.SquareBlack()
	c.HalfWidthTransparent()
}
func (c *Console) ClearSurface() {
	c.clearBeforeNextDraw = true
}

func (c *Console) Draw(screen *ebiten.Image) {
	if c.clearBeforeNextDraw {
		c.clearBeforeNextDraw = false
		screen.Fill(common.Black)
	}

	c.drawCellsToScreen(c.GetSquareDeltaIterator(), screen, c.squareFont, geometry.OnePointF)
	c.drawCellsToScreen(c.GetHalfWidthDeltaIterator(), screen, c.halfWidthFont, geometry.PointF{X: 0.5, Y: 1.0})
}
func (c *Console) drawCellsToScreen(cells CellIterator, screen *ebiten.Image, font *etxt.Font, tileScale geometry.PointF) {
	count := cells.Count()
	if count == 0 || font == nil {
		return
	}
	//println(fmt.Sprintf("drawing %v cells", count))
	c.txtRenderer.SetFont(font)
	scale := c.screenDPIScale
	tilewidth := int(math.Ceil(float64(c.TileWidth) * scale * tileScale.X))
	tileheight := int(math.Ceil(float64(c.TileHeight) * scale * tileScale.Y))
	cells.Iter(func(tilePos geometry.Point, cellAt common.Cell) {
		vector.DrawFilledRect(screen, float32((tilePos.X)*tilewidth), float32((tilePos.Y)*tileheight), float32(tilewidth), float32(tileheight), cellAt.Style.Background, false)
	})

	c.txtRenderer.SetTarget(screen)
	c.txtRenderer.SetSizePx(int(math.Ceil(float64(c.TileHeight) * scale)))
	cells.Iter(func(tilePos geometry.Point, cellAt common.Cell) {
		glyph := string(cellAt.Rune)
		runes, _ := etxt.GetMissingRunes(font, glyph)
		if len(runes) > 0 {
			glyph = " "
		}
		xPos := (tilePos.X) * tilewidth
		yPos := (tilePos.Y) * tileheight
		c.txtRenderer.SetColor(cellAt.Style.Foreground)
		c.txtRenderer.Draw(glyph, xPos, yPos)
	})
	cells.Done()
}

func (c *Console) SetScale(scale float64) {
	if scale != c.screenDPIScale {
		c.screenDPIScale = scale
		c.squareBuffer = ebiten.NewImage(int(float64(c.GridWidth*c.TileWidth)*c.screenDPIScale), int(float64(c.GridHeight*c.TileHeight)*c.screenDPIScale))
		c.halfWidthBuffer = ebiten.NewImage(int(float64(c.GridWidth*c.TileWidth)*c.screenDPIScale), int(float64(c.GridHeight*c.TileHeight)*c.screenDPIScale))
		c.ClearConsole()
	}
}

// Flush will compute the delta between the last frame and the current frame.
// Call it after all your drawing is done. And before you call Draw()
func (c *Console) Flush() {
	if c.squareDirty {
		c.squareComputeFrameDelta(c.squareCurrentGrid, c.clearBeforeNextDraw)
		c.squareDirty = false
	}
	if c.halfWidthDirty {
		c.halfWidthComputeFrameDelta(c.halfWidthCurrentGrid, c.clearBeforeNextDraw)
		c.halfWidthDirty = false
	}
	c.clearBeforeNextDraw = false
}

func (c *Console) SquareBlack() {
	c.squareCurrentGrid.Fill(common.Cell{Rune: ' ', Style: common.DefaultStyle})
	c.squareDirty = true
}
func (c *Console) HalfWidthTransparent() {
	c.halfWidthCurrentGrid.Fill(common.Cell{Rune: ' ', Style: common.Style{Foreground: common.White, Background: common.Transparent}})
	c.halfWidthDirty = true
}

func (c *Console) HalfWidthBlack() {
	c.halfWidthCurrentGrid.Fill(common.Cell{Rune: ' ', Style: common.DefaultStyle})
	c.halfWidthDirty = true
}
func (c *Console) SquareFill(rect geometry.Rect, cell common.Cell) {
	c.squareCurrentGrid.Slice(rect).Fill(cell)
	c.squareDirty = true
}

func (c *Console) HalfWidthFill(rect geometry.Rect, cell common.Cell) {
	c.halfWidthCurrentGrid.Slice(rect).Fill(cell)
	c.halfWidthDirty = true
}

func (c *Console) halfWidthComputeFrameDelta(gd geometry.Grid, exposed bool) {
	if gd.Ug == nil || gd.Rg.Empty() && !exposed {
		return
	}
	if c.halfWidthPreviousGrid.Ug == nil {
		c.halfWidthPreviousGrid = geometry.NewGrid(gd.Ug.Width, gd.Ug.Height)
	} else if c.halfWidthPreviousGrid.Ug.Width != gd.Ug.Width || c.halfWidthPreviousGrid.Ug.Height != gd.Ug.Height {
		c.halfWidthPreviousGrid = c.halfWidthPreviousGrid.Resize(gd.Ug.Width, gd.Ug.Height)
	}
	if exposed {
		c.halfWidthRefresh(gd)
		return
	}

	w := gd.Ug.Width
	cells := gd.Ug.Cells
	pcells := c.halfWidthPreviousGrid.Ug.Cells // previous cells
	yimax := gd.Rg.Max.Y * w
	for y, yi := 0, gd.Rg.Min.Y*w; yi < yimax; y, yi = y+1, yi+w {
		ximax := yi + gd.Rg.Max.X
		for x, xi := 0, yi+gd.Rg.Min.X; xi < ximax; x, xi = x+1, xi+1 {
			cellAt := cells[xi]
			if cellAt == pcells[xi] {
				continue
			}
			pcells[xi] = cellAt
			p := geometry.Point{X: x, Y: y}
			c.halfWidthAccumulatedFrame[p] = cellAt
		}
	}
}

// squareComputeFrameDelta will compute the delta between the previous console state and the current console state.
// It will return the delta and also update the previous console state.
func (c *Console) squareComputeFrameDelta(gd geometry.Grid, exposed bool) {
	if gd.Ug == nil || gd.Rg.Empty() && !exposed {
		return
	}
	if c.squarePreviousGrid.Ug == nil {
		c.squarePreviousGrid = geometry.NewGrid(gd.Ug.Width, gd.Ug.Height)
	} else if c.squarePreviousGrid.Ug.Width != gd.Ug.Width || c.squarePreviousGrid.Ug.Height != gd.Ug.Height {
		c.squarePreviousGrid = c.squarePreviousGrid.Resize(gd.Ug.Width, gd.Ug.Height)
	}
	if exposed {
		c.squareRefresh(gd)
		return
	}
	w := gd.Ug.Width
	cells := gd.Ug.Cells
	pcells := c.squarePreviousGrid.Ug.Cells // previous cells
	yimax := gd.Rg.Max.Y * w
	for y, yi := 0, gd.Rg.Min.Y*w; yi < yimax; y, yi = y+1, yi+w {
		ximax := yi + gd.Rg.Max.X
		for x, xi := 0, yi+gd.Rg.Min.X; xi < ximax; x, xi = x+1, xi+1 {
			cellAt := cells[xi]
			p := geometry.Point{X: x, Y: y}
			if cellAt == pcells[xi] {
				continue
			}
			pcells[xi] = cellAt
			c.squareAccumulatedFrame[p] = cellAt
		}
	}
}

func (c *Console) squareRefresh(gd geometry.Grid) {
	gd.Rg.Min = geometry.Point{}
	gd.Rg.Max = gd.Rg.Min.Add(geometry.Point{X: gd.Ug.Width, Y: gd.Ug.Height})
	c.squarePreviousGrid.Copy(gd)
	it := gd.Iterator()
	//println("refresh (copy from current console to previous console to accumulated frame and delta frame)")
	for it.Next() {
		c.squareAccumulatedFrame[it.P()] = it.Cell()
	}
}

func (c *Console) halfWidthRefresh(gd geometry.Grid) {
	gd.Rg.Min = geometry.Point{}
	gd.Rg.Max = gd.Rg.Min.Add(geometry.Point{X: gd.Ug.Width, Y: gd.Ug.Height})
	c.halfWidthPreviousGrid.Copy(gd)
	it := gd.Iterator()
	for it.Next() {
		c.halfWidthAccumulatedFrame[it.P()] = it.Cell()
	}
}

func (c *Console) LoadEmbeddedFonts(tileFontDir, textFontDir, squareFontName, halfFontName string, fs embed.FS) {
	c.tileFontLib = etxt.NewFontLibrary()
	c.textFontLib = etxt.NewFontLibrary()

	_, _, err := c.tileFontLib.ParseEmbedDirFonts(tileFontDir, fs)
	if err != nil {
		log.Fatalf("Error while loading EmbeddedData: %s", err.Error())
	}
	_, _, err = c.textFontLib.ParseEmbedDirFonts(textFontDir, fs)
	if err != nil {
		log.Fatalf("Error while loading EmbeddedData: %s", err.Error())
	}

	if !c.tileFontLib.HasFont(squareFontName) {
		log.Fatal("missing font: " + squareFontName)
	}

	if !c.textFontLib.HasFont(halfFontName) {
		log.Fatal("missing font: " + halfFontName)
	}
	c.squareFont = c.tileFontLib.GetFont(squareFontName)
	c.halfWidthFont = c.textFontLib.GetFont(halfFontName)
}

func (c *Console) GetAvailableTileFonts() []string {
	result := make([]string, 0)
	c.tileFontLib.EachFont(func(name string, font *etxt.Font) error {
		result = append(result, name)
		return nil
	})
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func (c *Console) GetAvailableTextFonts() []string {
	result := make([]string, 0)
	c.textFontLib.EachFont(func(name string, font *etxt.Font) error {
		result = append(result, name)
		return nil
	})
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func (c *Console) SetSquareFont(fontName string) {
	if c.tileFontLib.HasFont(fontName) {
		c.squareFont = c.tileFontLib.GetFont(fontName)
	}
}

func (c *Console) SetHalfWidthFont(fontName string) {
	if c.textFontLib.HasFont(fontName) {
		c.halfWidthFont = c.textFontLib.GetFont(fontName)
	}
}
func (c *Console) Contains(p geometry.Point) bool {
	return c.squareCurrentGrid.Contains(p)
}
func (c *Console) isHWDeltaTransparentAt(squarePos geometry.Point) bool {
	halfWidthPosOne := geometry.Point{X: squarePos.X * 2, Y: squarePos.Y}
	halfWidthPosTwo := geometry.Point{X: squarePos.X*2 + 1, Y: squarePos.Y}
	deltaOneIsOpaque := false
	if deltaOne, ok := c.halfWidthAccumulatedFrame[halfWidthPosOne]; ok {
		deltaOneIsOpaque = !deltaOne.IsTransparent()
	}
	deltaTwoIsOpaque := false
	if deltaTwo, ok := c.halfWidthAccumulatedFrame[halfWidthPosTwo]; ok {
		deltaTwoIsOpaque = !deltaTwo.IsTransparent()
	}
	return !deltaOneIsOpaque && !deltaTwoIsOpaque
}
func (c *Console) isHWLayerTransparentAt(squarePos geometry.Point) OverlayState {
	halfWidthPosOne := geometry.Point{X: squarePos.X * 2, Y: squarePos.Y}
	halfWidthPosTwo := geometry.Point{X: squarePos.X*2 + 1, Y: squarePos.Y}
	previousGridCellOne := c.halfWidthPreviousGrid.At(halfWidthPosOne)
	previousGridCellTwo := c.halfWidthPreviousGrid.At(halfWidthPosTwo)
	currentOneIsOpaque := !previousGridCellOne.IsTransparent()
	currentTwoIsOpaque := !previousGridCellTwo.IsTransparent()
	deltaOneIsOpaque := false
	if deltaOne, ok := c.halfWidthAccumulatedFrame[halfWidthPosOne]; ok {
		deltaOneIsOpaque = !deltaOne.IsTransparent()
	}
	deltaTwoIsOpaque := false
	if deltaTwo, ok := c.halfWidthAccumulatedFrame[halfWidthPosTwo]; ok {
		deltaTwoIsOpaque = !deltaTwo.IsTransparent()
	}

	rightIsOpaque := currentTwoIsOpaque || deltaTwoIsOpaque
	leftIsOpaque := currentOneIsOpaque || deltaOneIsOpaque

	if leftIsOpaque && rightIsOpaque {
		return BothOpaque
	} else if leftIsOpaque {
		return RightTransparent
	} else if rightIsOpaque {
		return LeftTransparent
	} else {
		return BothTransparent
	}
}

func NewTextRenderer() *etxt.Renderer {
	txtRenderer := etxt.NewStdRenderer()
	glyphsCache := etxt.NewDefaultCache(10 * 1024 * 1024) // 10MB
	txtRenderer.SetCacheHandler(glyphsCache.NewHandler())
	txtRenderer.SetAlign(etxt.Top, etxt.Left)
	whiteColor := common.White
	txtRenderer.SetColor(whiteColor)
	return txtRenderer
}

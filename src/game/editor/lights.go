package editor

import (
	"fmt"
	"time"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

func (g *GameStateEditor) setBrushHandlerWithLightUpdate(state UIHandler, placeIcon rune, placeFunc func(point geometry.Point)) func() {
	return func() {
		g.changeUIStateTo(state)
		g.placeThingIcon = placeIcon
		g.handler.CellsSelected = func() {
			g.callOnSelection(placeFunc)
			g.engine.GetGame().GetMap().UpdateBakedLights()
			g.engine.GetGame().GetMap().UpdateDynamicLights()
		}
		//g.menuBar.SetDirty()
		g.updateStatusLine()
	}
}

func (g *GameStateEditor) placeLightAt(pos geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()

	if currentMap.IsBakedLightSource(pos) || !currentMap.Contains(pos) {
		return
	}
	lightColor := currentMap.AmbientLight
	if g.SelectedLightSource != nil {
		lightColor = g.SelectedLightSource.Color.ToRGB()
	}
	neutralLight := &gridmap.LightSource{
		Pos:          pos,
		Radius:       7,
		Color:        lightColor,
		MaxIntensity: 1.0,
	}
	currentMap.AddBakedLightSource(pos, neutralLight)
}

func (g *GameStateEditor) backInTime() {
	currentMap := g.engine.GetGame().GetMap()
	currentMap.TimeOfDay = currentMap.TimeOfDay.Add(-30 * time.Minute)
	currentMap.SetAmbientLight(common.GetAmbientLightFromDayTime(currentMap.TimeOfDay).ToRGB())

	g.PrintAsMessage(fmt.Sprintf("Time of day: %s -> %v", currentMap.TimeOfDay.String(), currentMap.AmbientLight))
	g.gridIsDirty = true
}

func (g *GameStateEditor) forwardInTime() {
	currentMap := g.engine.GetGame().GetMap()
	currentMap.TimeOfDay = currentMap.TimeOfDay.Add(30 * time.Minute)
	currentMap.SetAmbientLight(common.GetAmbientLightFromDayTime(currentMap.TimeOfDay).ToRGB())

	g.PrintAsMessage(fmt.Sprintf("Time of day: %s -> %v", currentMap.TimeOfDay.String(), currentMap.AmbientLight))
	g.gridIsDirty = true
}

func (g *GameStateEditor) updateAllLights() {
	currentMap := g.engine.GetGame().GetMap()
	currentMap.ApplyAmbientLight()
	currentMap.UpdateBakedLights()
	currentMap.UpdateDynamicLights()
	g.gridIsDirty = true
}

func (g *GameStateEditor) selectLightAt(positionInWorld geometry.Point) {
	currentMap := g.engine.GetGame().GetMap()
	if light, isBaked := currentMap.BakedLights[positionInWorld]; isBaked {
		g.SelectedLightSource = light
		g.PrintAsMessage(fmt.Sprintf("Selected light at %v", positionInWorld))
	}
}

func (g *GameStateEditor) changeSelectedLightColor() {
	if g.SelectedLightSource == nil {
		return
	}
	userInterface := g.engine.GetUI()
	currentMap := g.engine.GetGame().GetMap()

	light := g.SelectedLightSource
	changed := func(color common.Color) {
		light.Color = color.ToRGB()
		currentMap.UpdateBakedLights()
		g.gridIsDirty = true
	}
	closed := func(color common.Color) {
		changed(color)
		userInterface.ShowWidget(g.menuBar)
		userInterface.ShowWidget(g.bottomMessageLabel)
		userInterface.ShowWidget(g.topStatusLineLabel)
	}
	userInterface.HideWidget(g.menuBar)
	userInterface.HideWidget(g.bottomMessageLabel)
	userInterface.HideWidget(g.topStatusLineLabel)
	userInterface.OpenColorPicker(light.Color, changed, closed)
}

func (g *GameStateEditor) changeAmbientLight() {
	userInterface := g.engine.GetUI()
	currentMap := g.engine.GetGame().GetMap()
	changed := func(color common.Color) {
		currentMap.AmbientLight = color.ToRGB()
		currentMap.ApplyAmbientLight()
		currentMap.UpdateBakedLights()
		currentMap.UpdateDynamicLights()
		g.gridIsDirty = true
	}
	closed := func(color common.Color) {
		changed(color)
		userInterface.ShowWidget(g.menuBar)
		userInterface.ShowWidget(g.bottomMessageLabel)
		userInterface.ShowWidget(g.topStatusLineLabel)
	}
	userInterface.HideWidget(g.menuBar)
	userInterface.HideWidget(g.bottomMessageLabel)
	userInterface.HideWidget(g.topStatusLineLabel)
	userInterface.OpenColorPicker(currentMap.AmbientLight, changed, closed)
}

func (g *GameStateEditor) decreaseLightRadius() {
	if g.SelectedLightSource == nil {
		return
	}
	g.SelectedLightSource.Radius--
	currentMap := g.engine.GetGame().GetMap()
	currentMap.UpdateBakedLights()
	currentMap.UpdateDynamicLights()
	g.gridIsDirty = true
}

func (g *GameStateEditor) increaseLightRadius() {
	if g.SelectedLightSource == nil {
		return
	}
	g.SelectedLightSource.Radius++
	currentMap := g.engine.GetGame().GetMap()
	currentMap.UpdateBakedLights()
	currentMap.UpdateDynamicLights()
	g.gridIsDirty = true
}

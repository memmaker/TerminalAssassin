package editor

import (
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
)

func (g *GameStateEditor) openColorMenu() {
    menuItems := []services.MenuItem{
        {
            Label:   "Foreground Color",
            Icon:    core.GlyphPalette,
            Handler: g.changeForegroundColor,
        },
        {
            Label:   "Background Color",
            Icon:    core.GlyphPalette,
            Handler: g.changeBackgroundColor,
        },
    }
    g.OpenMenuBarDropDown("Color", (2*12)-2, menuItems)
}

// ...existing code...

func (g *GameStateEditor) changeBackgroundColor() {
    /*
       g.bottomMessageLabel.Clear()

       userInterface := g.engine.GetUI()
       userInterface.HideWidget(g.menuBar)
       userInterface.HideWidget(g.topStatusLineLabel)
       userInterface.HideWidget(g.bottomMessageLabel)
       changed := func(color common.Color) {
           g.gridIsDirty = true
       }

       closed := func(color common.Color) {
           changed(color)
           userInterface.ShowWidget(g.menuBar)
           userInterface.ShowWidget(g.topStatusLineLabel)
           userInterface.ShowWidget(g.bottomMessageLabel)
           g.updateStatusLine()
           g.clearHUD = true
       }

    */
    //userInterface.OpenColorPicker(g.currentBackgroundColor, changed, closed)
}

func (g *GameStateEditor) changeForegroundColor() {
    /*
       g.bottomMessageLabel.Clear()

       userInterface := g.engine.GetUI()
       userInterface.HideWidget(g.menuBar)
       userInterface.HideWidget(g.topStatusLineLabel)
       userInterface.HideWidget(g.bottomMessageLabel)
       changed := func(color common.Color) {
           g.currentForegroundColor = color
           g.gridIsDirty = true
       }
       closed := func(color common.Color) {
           changed(color)
           userInterface.ShowWidget(g.menuBar)
           userInterface.ShowWidget(g.topStatusLineLabel)
           userInterface.ShowWidget(g.bottomMessageLabel)
           g.updateStatusLine()
           g.clearHUD = true
       }
       userInterface.OpenColorPicker(g.currentForegroundColor, changed, closed)

    */
}

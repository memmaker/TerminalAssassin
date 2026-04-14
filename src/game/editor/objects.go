package editor

import (
    "fmt"
    "strings"

    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/objects"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
)

const gravestonePrefix = "gravestone|"

func (g *GameStateEditor) openObjectsMenu() {
    currentMap := g.engine.GetGame().GetMap()
    data := g.engine.GetData()
    defaultFloor := data.GroundTile()
    objectFactory := g.engine.GetObjectFactory()
    menuItems := []services.MenuItem{
        {
            Label: "clear object",
            Handler: g.setBrushHandlerWithLightUpdate(addObjectsUI, 'X', func(pos geometry.Point) {
                if currentMap.IsObjectAt(pos) {
                    currentMap.RemoveObjectAt(pos)
                }
            }),
        },
    }
    for _, o := range objectFactory.SimpleObjects() {
        objectCreator := o
        identifier := objectCreator.Name

        if strings.HasPrefix(identifier, gravestonePrefix) {
            // Gravestones need a unique inscription — prompt before placing.
            menuItems = append(menuItems, services.MenuItem{
                Label: "gravestone (inscription)",
                Icon:  objectCreator.Icon,
                Handler: func() {
                    g.handler = UIHandler{
                        Name: "gravestone inscription",
                        TextReceived: func(inscription string) {
                            g.setBrushHandlerWithLightUpdate(addObjectsUI, objectCreator.Icon, func(pos geometry.Point) {
                                newObject := objects.NewGravestone(inscription)
                                currentMap.AddObject(newObject, pos)
                                currentMap.SetTile(pos, defaultFloor.WithBGColor(core.CurrentTheme.MapBackground).WithFGColor(core.CurrentTheme.MapForeground))
                            })()
                        },
                    }
                    g.showTextInput("Inscription:", "")
                },
            })
            continue
        }

        menuItems = append(menuItems, services.MenuItem{
            Label: identifier,
            Icon:  objectCreator.Icon,
            Handler: g.setBrushHandlerWithLightUpdate(addObjectsUI, 'O', func(pos geometry.Point) {
                newObject := objectCreator.Create(identifier)
                currentMap.AddObject(newObject, pos)
                currentMap.SetTile(pos, defaultFloor.WithBGColor(core.CurrentTheme.MapBackground).WithFGColor(core.CurrentTheme.MapForeground))
            }),
        })
    }

    g.OpenTilePickerDropDown("Choose object", menuItems)
    return
}
func (g *GameStateEditor) selectObjectAt(pos geometry.Point) {
    currentMap := g.engine.GetGame().GetMap()
    objectAt := currentMap.ObjectAt(pos)
    if objectAt != nil {
        g.selectedObject = objectAt
        g.PrintAsMessage(fmt.Sprintf("Object: %s", objectAt.Description()))
    } else {
        g.PrintAsMessage("No object at " + pos.String())
    }
}

func (g *GameStateEditor) removeSelectedObject() {
    if g.selectedObject == nil {
        return
    }
    currentMap := g.engine.GetGame().GetMap()
    position := g.selectedObject.Pos()
    currentMap.RemoveObjectAt(position)
    description := g.selectedObject.Description()

    g.PrintAsMessage(fmt.Sprintf("Removed object %s at %s", description, position.String()))

    g.selectedObject = nil

    g.SetDirty()
}

// editContentsOfSelectedObject opens a menu that lists every item stored in the
// selected container. Clicking an item removes it; "Cancel" closes the menu.
func (g *GameStateEditor) editContentsOfSelectedObject() {
    holder, ok := g.selectedObject.(services.ContentHolder)
    if !ok {
        return
    }
    g.openContentsMenuAt(holder, g.MousePositionOnScreen)
}

// openContentsMenuAt builds (or rebuilds after a removal) the contents editing
// menu anchored at menuPos. Each item entry removes itself from the container
// and re-opens the menu; "Cancel" just closes it.
func (g *GameStateEditor) openContentsMenuAt(holder services.ContentHolder, menuPos geometry.Point) {
    contents := holder.GetContents()
    menuItems := make([]services.MenuItem, 0, len(contents)+1)

    for i, name := range contents {
        i, name := i, name // capture loop variables
        menuItems = append(menuItems, services.MenuItem{
            Label: name,
            Icon:  core.GlyphWrench,
            Handler: func() {
                current := holder.GetContents()
                if i < len(current) {
                    // remove by index using a safe three-index slice copy
                    updated := make([]string, 0, len(current)-1)
                    updated = append(updated, current[:i]...)
                    updated = append(updated, current[i+1:]...)
                    holder.SetContents(updated)
                }
                g.PrintAsMessage(fmt.Sprintf("Removed '%s' from container", name))
                g.SetDirty()
                // Reopen with the updated contents if anything remains
                if len(holder.GetContents()) > 0 {
                    g.openContentsMenuAt(holder, menuPos)
                }
            },
        })
    }

    menuItems = append(menuItems, services.MenuItem{
        Label: "Cancel",
        Icon:  'x',
    })

    g.engine.GetUI().OpenAtPosAutoCloseMenuWithCallback(menuPos, menuItems, func() {
        g.gridIsDirty = true
        g.menuBar.SetDirty()
        g.topStatusLineLabel.SetDirty()
    })
}

// setDifficultyOfSelectedObject opens a small menu to choose Easy / Medium /
// Hard for the selected lock-difficulty-bearing object (door or safe).
func (g *GameStateEditor) setDifficultyOfSelectedObject() {
    holder, ok := g.selectedObject.(services.LockDifficultyHolder)
    if !ok {
        return
    }
    difficulties := []core.LockDifficulty{
        core.LockDifficultyEasy,
        core.LockDifficultyMedium,
        core.LockDifficultyHard,
    }
    menuItems := make([]services.MenuItem, 0, len(difficulties))
    for _, diff := range difficulties {
        d := diff
        menuItems = append(menuItems, services.MenuItem{
            Label: fmt.Sprintf("%s (%d pick(s), %.0fs)", d.ToString(), d.PickCount(), d.PickTime()),
            Handler: func() {
                holder.SetLockDifficulty(d)
                g.PrintAsMessage(fmt.Sprintf("Lock difficulty set to %s", d.ToString()))
                g.SetDirty()
            },
        })
    }
    g.OpenMenuBarDropDown("Lock Difficulty", (2*4)-2, menuItems)
}

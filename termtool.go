package filterdir

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/jroimartin/gocui"
)

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// Log view
	if v, err := g.SetView("logs", 0, 0, maxX-1, maxY-9); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Autoscroll = true
		_, y := v.Size()
		fmt.Fprint(v, strings.Repeat("\n", y))
	}

	// File summary view
	if v, err := g.SetView("files", 0, maxY-9, maxX-1, maxY-7); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.FgColor = gocui.ColorGreen
		fmt.Fprintln(v, "\033[0m 0 files in the list.")
	}

	// Command options view
	if v, err := g.SetView("commands", 0, maxY-7, maxX-1, maxY-2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.FgColor = gocui.ColorMagenta
		fmt.Fprintln(v, "\033[0m c : Clear all files from IncludeList")
		fmt.Fprintln(v, " s : Save current file list to go source")
		fmt.Fprintln(v, " g : Generate go code that statically implements all files in list (using shurcooL/vfsgen)")
		fmt.Fprint(v, " q : Quit")
	}

	// Prompt view
	if v, err := g.SetView("prompt", 0, maxY-2, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		fmt.Fprintln(v, "   : ")
	}

	// Input view
	if v, err := g.SetView("input", 5, maxY-2, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Editable = true
		if err := g.SetCurrentView("input"); err != nil {
			return err
		}
	}
	return nil
}

func bindKeys(g *gocui.Gui, reqs *sortedList, f *FilterDir) error {

	// Ctr-C will shutdown the GUI
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, shutdown); err != nil {
		return err
	}

	// Process all user input from the input view
	if err := g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, processInput(reqs, f)); err != nil {
		return err
	}
	return nil
}

// Check for file changes every so often, and if changes are found push to GUI
func pushUpdates(ctx context.Context, g *gocui.Gui, reqs *sortedList) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Millisecond * 100):
			g.Execute(func(g *gocui.Gui) error {
				if reqs.Changed() {
					v, err := g.View("files")
					if err != nil {
						return err
					}
					v.Clear()
					fmt.Fprintf(v, "\033[0m %d files in the list.", len(reqs.List()))
				}
				return nil
			})
		}
	}
}

// Handle user input
func processInput(reqs *sortedList, f *FilterDir) func(*gocui.Gui, *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		input := strings.ToLower(strings.TrimSpace(v.ViewBuffer()))
		v.Clear()

		switch input {
		case "q", "quit":
			return gocui.ErrQuit
		case "c", "clear":
			reqs.Clear()
			g.Execute(func(g *gocui.Gui) error {
				v, err := g.View("logs")
				if err != nil {
					return err
				}
				fmt.Fprint(v, "\n List cleared.")
				return nil
			})
		case "s", "save":
			saveErr := f.saveList(reqs.List())
			g.Execute(func(g *gocui.Gui) error {
				v, err := g.View("logs")
				if err != nil {
					return err
				}
				if saveErr != nil {
					fmt.Fprintln(v, saveErr.Error())
				} else {
					fmt.Fprint(v, "\n IncludeList successfully saved to go source.")
				}
				return nil
			})
		case "g", "generate":
			saveErr := f.generateAssets(reqs.List())
			g.Execute(func(g *gocui.Gui) error {
				v, err := g.View("logs")
				if err != nil {
					return err
				}
				if saveErr != nil {
					fmt.Fprintln(v, saveErr.Error())
				} else {
					fmt.Fprint(v, "\n Staticly implemented assets have been generated.")
				}
				return nil
			})
		}
		return nil
	}
}

func shutdown(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

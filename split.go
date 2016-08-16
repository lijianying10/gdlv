package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"

	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/label"
	ntypes "github.com/aarzilli/nucular/types"

	"golang.org/x/mobile/event/mouse"
)

type panelKind string

const (
	fullPanelKind            panelKind = "Full"
	splitHorizontalPanelKind panelKind = "Horizontal"
	splitVerticalPanelKind   panelKind = "Vertical"
	infoPanelKind            panelKind = "Info"

	splitFlags = nucular.WindowNoScrollbar
)

const (
	infoCommand     = "Command"
	infoListing     = "Listing"
	infoDisassembly = "Disassembly"
	infoGoroutines  = "Goroutines"
	infoStacktrace  = "Stacktrace"
	infoLocals      = "Locals"
	infoGlobal      = "Globals"
	infoBps         = "Breakpoints"
	infoThreads     = "Threads"
	infoRegisters   = "Registers"
	infoSources     = "Sources"
	infoFuncs       = "Functions"
	infoTypes       = "Types"
)

var infoNameToFunc = map[string]func(mw *nucular.MasterWindow, w *nucular.Window){
	infoCommand:     updateCommandPanel,
	infoListing:     updateListingPanel,
	infoDisassembly: updateDisassemblyPanel,
	infoGoroutines:  goroutinesPanel.Update,
	infoStacktrace:  stackPanel.Update,
	infoLocals:      localsPanel.Update,
	infoGlobal:      globalsPanel.Update,
	infoBps:         breakpointsPanel.Update,
	infoThreads:     threadsPanel.Update,
	infoRegisters:   regsPanel.Update,
	infoSources:     sourcesPanel.Update,
	infoFuncs:       funcsPanel.Update,
	infoTypes:       typesPanel.Update,
}

var infoModes = []string{
	infoCommand, infoListing, infoDisassembly, infoGoroutines, infoStacktrace, infoLocals, infoGlobal, infoBps, infoThreads, infoRegisters, infoSources, infoFuncs, infoTypes,
}

var codeToInfoMode = map[byte]string{
	'C': infoCommand,
	'L': infoListing,
	'D': infoDisassembly,
	'G': infoGoroutines,
	'S': infoStacktrace,
	'l': infoLocals,
	'g': infoGlobal,
	'B': infoBps,
	'r': infoRegisters,
	's': infoSources,
	'f': infoFuncs,
	't': infoTypes,
	'T': infoThreads,
}

var infoModeToCode = map[string]byte{}

func init() {
	for k, v := range codeToInfoMode {
		infoModeToCode[v] = k
	}
}

func (kind panelKind) Internal() bool {
	switch kind {
	case fullPanelKind, splitHorizontalPanelKind, splitVerticalPanelKind:
		return true
	default:
		return false
	}
}

type panel struct {
	kind     panelKind
	size     int
	infoMode int
	child    [2]*panel
	parent   *panel

	name   string
	resize bool
}

var rootPanel *panel

const (
	headerRow         = 20
	headerCombo       = 110
	headerSplitMenu   = 70
	verticalSpacing   = 1
	horizontalSpacing = 2
	splitMinSize      = 20
)

func parsePanelDescr(in string, parent *panel) (p *panel, rest string) {
	var kind panelKind
	switch in[0] {
	case '0':
		p = &panel{kind: kind, name: randomname(), parent: parent}
		p.child[0], rest = parsePanelDescr(in[1:], p)
		return p, rest
	case '_', '|':
		kind = splitHorizontalPanelKind
		if in[0] == '|' {
			kind = splitVerticalPanelKind
		}
		var i int
		for i = 1; i < len(in); i++ {
			if in[i] < '0' || in[i] > '9' {
				break
			}
		}
		size, _ := strconv.Atoi(in[1:i])
		p = &panel{kind: kind, name: randomname(), size: size, parent: parent}
		rest = in[i:]
		p.child[0], rest = parsePanelDescr(rest, p)
		p.child[1], rest = parsePanelDescr(rest, p)
		return p, rest
	default:
		p = &panel{kind: infoPanelKind, name: randomname(), infoMode: infoModeIdx(codeToInfoMode[in[0]]), parent: parent}
		rest = in[1:]
		return p, rest
	}
}

func (p *panel) String() (s string, err error) {
	defer func() {
		if ierr := recover(); ierr != nil {
			err = ierr.(error)
		}
	}()
	var out bytes.Buffer
	p.serialize(&out)
	return out.String(), err
}

func (p *panel) serialize(out io.Writer) {
	switch p.kind {
	case fullPanelKind:
		out.Write([]byte{'0'})
		p.child[0].serialize(out)
	case splitHorizontalPanelKind:
		fmt.Fprintf(out, "_%d", p.size)
		p.child[0].serialize(out)
		p.child[1].serialize(out)
	case splitVerticalPanelKind:
		fmt.Fprintf(out, "|%d", p.size)
		p.child[0].serialize(out)
		p.child[1].serialize(out)
	case infoPanelKind:
		code, ok := infoModeToCode[infoModes[p.infoMode]]
		if !ok {
			panic(fmt.Errorf("could not convert info mode %s to a code", infoModes[p.infoMode]))
		}
		out.Write([]byte{code})
	}
}

func infoModeIdx(n string) int {
	for i := range infoModes {
		if infoModes[i] == n {
			return i
		}
	}
	return -1
}

func randomname() string {
	var alphabet = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z'}
	out := make([]byte, 8)
	for i := range out {
		out[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(out)
}

func (p *panel) update(mw *nucular.MasterWindow, w *nucular.Window) {
	style, scaling := mw.Style()

	switch p.kind {
	case fullPanelKind:
		p.child[0].update(mw, w)

	case infoPanelKind:
		w.Row(headerRow).Static(headerSplitMenu, 0, headerCombo, 2)
		w.Menu(label.TA("Split", "CC"), 120, p.splitMenu)
		w.Spacing(1)
		w.ComboSimple(infoModes, &p.infoMode, 22)
		w.Row(0).Dynamic(1)
		if p.infoMode >= 0 {
			infoNameToFunc[infoModes[p.infoMode]](mw, w)
		}

	case splitHorizontalPanelKind:
		if p.size == 0 {
			p.size = int(float64(w.LayoutAvailableHeight()-int(horizontalSpacing*scaling)) / (2 * scaling))
		}
		w.Row(p.size).Dynamic(1)
		flags := splitFlags
		if p.child[0].kind == infoPanelKind {
			flags |= nucular.WindowBorder
		}
		if sw := w.GroupBegin(p.child[0].name, flags); sw != nil {
			p.child[0].update(mw, sw)
			sw.GroupEnd()
		}

		w.Row(horizontalSpacing).Dynamic(1)
		rszbounds, _ := w.Custom(ntypes.WidgetStateInactive)
		rszbounds.Y -= style.GroupWindow.Spacing.Y
		rszbounds.H += style.GroupWindow.Spacing.Y * 2

		if w.Input().Mouse.HasClickDownInRect(mouse.ButtonLeft, rszbounds, true) {
			p.resize = true
		}
		if p.resize {
			if !w.Input().Mouse.Down(mouse.ButtonLeft) {
				p.resize = false
			} else {
				p.size += int(float64(w.Input().Mouse.Delta.Y) / scaling)
				if p.size <= splitMinSize {
					p.size = splitMinSize
				}
			}
		}

		w.Row(0).Dynamic(1)
		flags = splitFlags
		if p.child[1].kind == infoPanelKind {
			flags |= nucular.WindowBorder
		}
		if sw := w.GroupBegin(p.child[1].name, flags); sw != nil {
			p.child[1].update(mw, sw)
			sw.GroupEnd()
		}

	case splitVerticalPanelKind:
		w.Row(0).Static(p.size, verticalSpacing, 0)

		flags := splitFlags
		if p.child[0].kind == infoPanelKind {
			flags |= nucular.WindowBorder
		}
		if sw := w.GroupBegin(p.child[0].name, flags); sw != nil {
			p.child[0].update(mw, sw)
			sw.GroupEnd()
		}
		if p.size == 0 {
			p.size = int(float64(w.LastWidgetBounds.W) / scaling)
		}

		rszbounds, _ := w.Custom(ntypes.WidgetStateInactive)
		rszbounds.X -= style.NormalWindow.Spacing.X
		rszbounds.W += style.NormalWindow.Spacing.X * 2

		if w.Input().Mouse.HasClickDownInRect(mouse.ButtonLeft, rszbounds, true) {
			p.resize = true
		}
		if p.resize {
			if !w.Input().Mouse.Down(mouse.ButtonLeft) {
				p.resize = false
			} else {
				p.size += int(float64(w.Input().Mouse.Delta.X) / scaling)
				if p.size <= splitMinSize {
					p.size = splitMinSize
				}
			}
		}

		flags = splitFlags
		if p.child[1].kind == infoPanelKind {
			flags |= nucular.WindowBorder
		}
		if sw := w.GroupBegin(p.child[1].name, flags); sw != nil {
			p.child[1].update(mw, sw)
			sw.GroupEnd()
		}
	}
}

func (p *panel) splitMenu(mw *nucular.MasterWindow, w *nucular.Window) {
	w.Row(20).Dynamic(1)
	if w.MenuItem(label.TA("Horizontal", "LC")) {
		p.split(splitHorizontalPanelKind)
	}
	if w.MenuItem(label.TA("Vertical", "LC")) {
		p.split(splitVerticalPanelKind)
	}
	if w.MenuItem(label.TA("Close", "LC")) {
		p.closeMyself()
	}
}

func (p *panel) split(kind panelKind) {
	if p.parent == nil {
		return
	}

	if p.parent.kind == fullPanelKind {
		p.parent.kind = kind
		p.parent.child[1] = &panel{kind: p.kind, name: randomname(), infoMode: p.infoMode, parent: p.parent}
		return
	}

	idx := p.parent.idx(p)
	if idx < 0 {
		return
	}

	newpanel := &panel{kind: kind, name: randomname(), size: 0, parent: p.parent}

	newpanel.child[0] = p
	newpanel.child[1] = &panel{kind: p.kind, name: randomname(), infoMode: p.infoMode, parent: newpanel}

	p.parent.child[idx] = newpanel
	p.parent = newpanel
}

func (p *panel) idx(child *panel) int {
	for i := range p.child {
		if p.child[i] == child {
			return i
		}
	}
	return -1
}

func (p *panel) closeMyself() {
	if p.parent == nil || p.parent.kind == fullPanelKind {
		return
	}

	idx := p.parent.idx(p)
	if idx < 0 {
		return
	}
	p.parent.closeChild(idx)
}

func (p *panel) closeChild(idx int) {
	if p.parent == nil {
		p.kind = fullPanelKind
		if idx == 0 {
			p.child[0] = p.child[1]
		}
		return
	}

	ownidx := p.parent.idx(p)
	if ownidx < 0 {
		return
	}

	survivor := p.child[0]
	if idx == 0 {
		survivor = p.child[1]
	}

	p.parent.child[ownidx] = survivor
	survivor.parent = p.parent
}

func updateCommandPanel(mw *nucular.MasterWindow, container *nucular.Window) {
	style, _ := mw.Style()

	w := container.GroupBegin("command", nucular.WindowNoScrollbar)
	if w == nil {
		return
	}
	defer w.GroupEnd()

	w.LayoutReserveRow(commandLineHeight, 1)
	w.Row(0).Dynamic(1)
	scrollbackEditor.Edit(w)

	var p string
	if running {
		p = "running"
	} else if client == nil {
		p = "connecting"
	} else {
		if curThread < 0 {
			p = "dlv>"
		} else {
			p = prompt(curThread, curGid, curFrame) + ">"
		}
	}

	promptwidth := nucular.FontWidth(style.Font, p) + style.Text.Padding.X*2

	w.Row(commandLineHeight).StaticScaled(promptwidth, 0)
	w.Label(p, "LC")

	if client == nil || running {
		commandLineEditor.Flags |= nucular.EditReadOnly
	} else {
		commandLineEditor.Flags &= ^nucular.EditReadOnly
	}
	active := commandLineEditor.Edit(w)
	if active&nucular.EditCommitted != 0 {
		var scrollbackOut = editorWriter{&scrollbackEditor, false}

		cmd := string(commandLineEditor.Buffer)
		if cmd == "" {
			fmt.Fprintf(&scrollbackOut, "%s %s\n", p, lastCmd)
		} else {
			lastCmd = cmd
			fmt.Fprintf(&scrollbackOut, "%s %s\n", p, cmd)
		}
		go executeCommand(cmd)
		commandLineEditor.Buffer = commandLineEditor.Buffer[:0]
		commandLineEditor.Cursor = 0
		commandLineEditor.Active = true
	}
}

func updateListingPanel(mw *nucular.MasterWindow, container *nucular.Window) {
	const lineheight = 14

	listp := container.GroupBegin("listing", 0)
	if listp == nil {
		return
	}
	defer listp.GroupEnd()

	style, _ := mw.Style()

	arroww := nucular.FontWidth(style.Font, "=>") + style.Text.Padding.X*2
	starw := nucular.FontWidth(style.Font, "*") + style.Text.Padding.X*2

	idxw := style.Text.Padding.X * 2
	if len(lp.listing) > 0 {
		idxw += nucular.FontWidth(style.Font, lp.listing[len(lp.listing)-1].idx)
	}

	for _, line := range lp.listing {
		listp.Row(lineheight).StaticScaled(starw, arroww, idxw, 0)
		if line.pc {
			rowbounds := listp.WidgetBounds()
			rowbounds.W = listp.Bounds.W
			cmds := listp.Commands()
			cmds.FillRect(rowbounds, 0, style.Selectable.PressedActive.Data.Color)
		}

		if line.breakpoint {
			listp.Label("*", "CC")
		} else {
			listp.Spacing(1)
		}

		if line.pc && lp.recenterListing {
			lp.recenterListing = false
			if above, below := listp.Invisible(); above || below {
				listp.Scrollbar.Y = listp.At().Y - listp.Bounds.H/2
				if listp.Scrollbar.Y < 0 {
					listp.Scrollbar.Y = 0
				}
			}
		}

		if line.pc && curFrame == 0 {
			listp.Label("=>", "CC")
		} else {
			listp.Spacing(1)
		}
		listp.Label(line.idx, "LC")
		listp.Label(line.text, "LC")
	}
}

func updateDisassemblyPanel(mw *nucular.MasterWindow, container *nucular.Window) {
	const lineheight = 14

	listp := container.GroupBegin("disassembly", 0)
	if listp == nil {
		return
	}
	defer listp.GroupEnd()

	style, _ := mw.Style()

	arroww := nucular.FontWidth(style.Font, "=>") + style.Text.Padding.X*2
	starw := nucular.FontWidth(style.Font, "*") + style.Text.Padding.X*2

	var maxaddr uint64 = 0
	if len(lp.text) > 0 {
		maxaddr = lp.text[len(lp.text)-1].Loc.PC
	}
	addrw := nucular.FontWidth(style.Font, fmt.Sprintf("%#x", maxaddr)) + style.Text.Padding.X*2

	lastfile, lastlineno := "", 0

	if len(lp.text) > 0 && lp.text[0].Loc.Function != nil {
		listp.Row(lineheight).Dynamic(1)
		listp.Label(fmt.Sprintf("TEXT %s(SB) %s", lp.text[0].Loc.Function.Name, lp.text[0].Loc.File), "LC")
	}

	for _, instr := range lp.text {
		if instr.Loc.File != lastfile || instr.Loc.Line != lastlineno {
			listp.Row(lineheight).Dynamic(1)
			listp.Row(lineheight).Dynamic(1)
			text := ""
			if instr.Loc.File == lp.file && instr.Loc.Line-1 < len(lp.listing) {
				text = strings.TrimSpace(lp.listing[instr.Loc.Line-1].text)
			}
			listp.Label(fmt.Sprintf("%s:%d: %s", instr.Loc.File, instr.Loc.Line, text), "LC")
			lastfile, lastlineno = instr.Loc.File, instr.Loc.Line
		}
		listp.Row(lineheight).StaticScaled(starw, arroww, addrw, 0)

		if instr.AtPC {
			rowbounds := listp.WidgetBounds()
			rowbounds.W = listp.Bounds.W
			cmds := listp.Commands()
			cmds.FillRect(rowbounds, 0, style.Selectable.PressedActive.Data.Color)
		}

		if instr.Breakpoint {
			listp.Label("*", "LC")
		} else {
			listp.Label(" ", "LC")
		}

		if instr.AtPC {
			if lp.recenterDisassembly {
				lp.recenterDisassembly = false
				if above, below := listp.Invisible(); above || below {
					listp.Scrollbar.Y = listp.At().Y - listp.Bounds.H/2
					if listp.Scrollbar.Y < 0 {
						listp.Scrollbar.Y = 0
					}
				}
			}
			listp.Label("=>", "LC")
		} else {
			listp.Label(" ", "LC")
		}

		listp.Label(fmt.Sprintf("%#x", instr.Loc.PC), "LC")
		listp.Label(instr.Text, "LC")
	}
}
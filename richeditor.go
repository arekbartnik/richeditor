// Package richeditor renders a form-associated rich text editor built on
// Wordgard. The editor posts its content as HTML under its name, like any
// other form field. That HTML comes from the browser, so sanitize it on the
// server before storing or rendering it.
package richeditor

import (
	"cmp"
	"embed"
	"io/fs"
	"maps"

	"github.com/a-h/templ"
)

//go:generate npm run build
//go:generate go run github.com/a-h/templ/cmd/templ@v0.3.1070 generate

//go:embed static
var static embed.FS

// Files holds editor.js and editor.css. Serve them and load them on every page
// that renders an Editor, for example with Head.
var Files = func() fs.FS {
	files, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	return files
}()

// Layout places the formatting buttons.
type Layout int

const (
	// Toolbar shows the buttons in a fixed toolbar above the text.
	Toolbar Layout = iota
	// Popover shows them in a popover over selected text. Alt+F10 moves
	// focus into it and Escape returns to the text.
	Popover
)

// Labels names the toolbar and its buttons for assistive technology and
// tooltips. An empty field takes its English text.
type Labels struct {
	Toolbar     string
	Strong      string
	Emphasis    string
	BulletList  string
	OrderedList string
	Blockquote  string
	Clear       string
}

// English is the text of every empty Labels field.
var English = Labels{
	Toolbar:     "Text formatting",
	Strong:      "Bold",
	Emphasis:    "Italic",
	BulletList:  "Bulleted list",
	OrderedList: "Numbered list",
	Blockquote:  "Quote",
	Clear:       "Clear formatting",
}

// Icons draws the buttons. A nil field takes the built-in Tabler icon, an
// svg with data-icon set to the button's command.
type Icons struct {
	Strong      templ.Component
	Emphasis    templ.Component
	BulletList  templ.Component
	OrderedList templ.Component
	Blockquote  templ.Component
	Clear       templ.Component
}

// Props configures an Editor.
type Props struct {
	// Name is the form field the HTML is posted under.
	Name string
	// Value is the initial HTML. Form reset restores it.
	Value string
	// Label is the accessible name of the editable area.
	Label  string
	Layout Layout
	Labels Labels
	Icons  Icons
	// ButtonClass replaces the "rich-editor-button" class on each button, so
	// the buttons can take the page's own button styles.
	ButtonClass string
	// ContentClass is added to the editable text, so it can share the styles
	// of the stored HTML where it is displayed.
	ContentClass string
	// Attrs adds attributes to the rich-editor element, such as an id or
	// event handlers. The element fires input on every change.
	Attrs templ.Attributes
}

func (p Props) labels() Labels {
	return Labels{
		Toolbar:     cmp.Or(p.Labels.Toolbar, English.Toolbar),
		Strong:      cmp.Or(p.Labels.Strong, English.Strong),
		Emphasis:    cmp.Or(p.Labels.Emphasis, English.Emphasis),
		BulletList:  cmp.Or(p.Labels.BulletList, English.BulletList),
		OrderedList: cmp.Or(p.Labels.OrderedList, English.OrderedList),
		Blockquote:  cmp.Or(p.Labels.Blockquote, English.Blockquote),
		Clear:       cmp.Or(p.Labels.Clear, English.Clear),
	}
}

func (p Props) buttonClass() string {
	return cmp.Or(p.ButtonClass, "rich-editor-button")
}

func (p Props) attrs() templ.Attributes {
	attrs := templ.Attributes{"name": p.Name, "value": p.Value, "label": p.Label}
	if p.ContentClass != "" {
		attrs["content-class"] = p.ContentClass
	}
	maps.Copy(attrs, p.Attrs)
	return attrs
}

type button struct {
	command string
	label   string
	icon    templ.Component
	toggle  bool
}

func (p Props) buttons() []button {
	l, i := p.labels(), p.Icons
	return []button{
		{"strong", l.Strong, cmp.Or(i.Strong, tabler("strong", tablerBold)), true},
		{"emphasis", l.Emphasis, cmp.Or(i.Emphasis, tabler("emphasis", tablerItalic)), true},
		{"bulletList", l.BulletList, cmp.Or(i.BulletList, tabler("bulletList", tablerList)), true},
		{"orderedList", l.OrderedList, cmp.Or(i.OrderedList, tabler("orderedList", tablerListNumbers)), true},
		{"blockquote", l.Blockquote, cmp.Or(i.Blockquote, tabler("blockquote", tablerBlockquote)), true},
		{"clear", l.Clear, cmp.Or(i.Clear, tabler("clear", tablerClearFormatting)), false},
	}
}

// pressed starts toggle buttons unpressed, and the script tracks them.
func (b button) pressed() templ.Attributes {
	if !b.toggle {
		return nil
	}
	return templ.Attributes{"aria-pressed": "false"}
}

// Tabler Icons, MIT licensed. See NOTICE.
const (
	tablerBold            = `<path d="M7 5h6a3.5 3.5 0 0 1 0 7H7zm6 7h1a3.5 3.5 0 0 1 0 7H7v-7"/>`
	tablerItalic          = `<path d="M11 5h6M7 19h6m1-14l-4 14"/>`
	tablerList            = `<path d="M9 6h11M9 12h11M9 18h11M5 6v.01M5 12v.01M5 18v.01"/>`
	tablerListNumbers     = `<path d="M11 6h9m-9 6h9m-8 6h8M4 16a2 2 0 1 1 4 0c0 .591-.5 1-1 1.5L4 20h4M6 10V4L4 6"/>`
	tablerBlockquote      = `<path d="M6 15h15m0 4H6m9-8h6m0-4h-6M9 9h1a1 1 0 1 1-1 1V7.5a2 2 0 0 1 2-2M3 9h1a1 1 0 1 1-1 1V7.5a2 2 0 0 1 2-2"/>`
	tablerClearFormatting = `<path d="m17 15l4 4m0-4l-4 4M7 6V5h11v1M7 19h4m2-14L9 19"/>`
)

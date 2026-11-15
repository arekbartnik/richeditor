// Package richeditor renders a form-associated rich text editor built on
// Wordgard. The editor posts its content as HTML under its name, like any
// other form field. That HTML comes from the browser, so sanitize it on the
// server before storing or rendering it.
package richeditor

import (
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
// tooltips.
type Labels struct {
	Toolbar     string
	Strong      string
	Emphasis    string
	BulletList  string
	OrderedList string
	Blockquote  string
	Clear       string
}

// English is the Labels an Editor uses when its Labels are zero.
var English = Labels{
	Toolbar:     "Text formatting",
	Strong:      "Bold",
	Emphasis:    "Italic",
	BulletList:  "Bulleted list",
	OrderedList: "Numbered list",
	Blockquote:  "Quote",
	Clear:       "Clear formatting",
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
	// ButtonClass replaces the "rich-button" class on each button, so the
	// buttons can take the page's own button styles.
	ButtonClass string
	// ContentClass is added to the editable area, so it can share the styles
	// of the stored HTML where it is displayed.
	ContentClass string
	// Attrs adds attributes to the rich-editor element, such as an id or
	// event handlers. The element fires input on every change.
	Attrs templ.Attributes
}

func (p Props) labels() Labels {
	if p.Labels == (Labels{}) {
		return English
	}
	return p.Labels
}

func (p Props) buttonClass() string {
	if p.ButtonClass == "" {
		return "rich-button"
	}
	return p.ButtonClass
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
	icon    string
}

func (p Props) buttons() []button {
	l := p.labels()
	return []button{
		{"strong", l.Strong, iconBold},
		{"emphasis", l.Emphasis, iconItalic},
		{"bulletList", l.BulletList, iconList},
		{"orderedList", l.OrderedList, iconListNumbers},
		{"blockquote", l.Blockquote, iconBlockquote},
	}
}

// Tabler Icons, MIT licensed. See NOTICE.
const (
	iconBold            = `<path d="M7 5h6a3.5 3.5 0 0 1 0 7H7zm6 7h1a3.5 3.5 0 0 1 0 7H7v-7"/>`
	iconItalic          = `<path d="M11 5h6M7 19h6m1-14l-4 14"/>`
	iconList            = `<path d="M9 6h11M9 12h11M9 18h11M5 6v.01M5 12v.01M5 18v.01"/>`
	iconListNumbers     = `<path d="M11 6h9m-9 6h9m-8 6h8M4 16a2 2 0 1 1 4 0c0 .591-.5 1-1 1.5L4 20h4M6 10V4L4 6"/>`
	iconBlockquote      = `<path d="M6 15h15m0 4H6m9-8h6m0-4h-6M9 9h1a1 1 0 1 1-1 1V7.5a2 2 0 0 1 2-2M3 9h1a1 1 0 1 1-1 1V7.5a2 2 0 0 1 2-2"/>`
	iconClearFormatting = `<path d="m17 15l4 4m0-4l-4 4M7 6V5h11v1M7 19h4m2-14L9 19"/>`
)

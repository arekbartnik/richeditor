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
	"slices"
	"strconv"

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

// Button is one formatting button. An editor only supports the formatting
// of the buttons it shows, so leaving out Link also drops links from pasted
// text and disables Mod-k. Undo and redo stay on the keyboard either way.
type Button int

const (
	Undo Button = iota
	Redo
	Strong
	Emphasis
	Underline
	Link
	BulletList
	OrderedList
	Blockquote
	Clear
	// TextStyle is a dropdown that turns the selected lines into a paragraph
	// or a heading of one of Props.Headings, and shows the style of the line
	// with the cursor. Typing "## " at the start of a line makes an h2, and
	// Ctrl-Shift-2 does the same, for each offered level, while Ctrl-Shift-0
	// makes a paragraph. With headings offered the editor holds headings of
	// every level, so loaded and pasted h1 to h6 keep their level.
	TextStyle
)

// DefaultButtons is every button but TextStyle, in the order an Editor shows
// them when its Buttons are nil.
var DefaultButtons = []Button{Strong, Emphasis, Underline, Link, BulletList, OrderedList, Blockquote, Clear, Undo, Redo}

// trailing buttons sit at the end of the toolbar, so the toolbar layout can
// push them to the far side.
func (b Button) trailing() bool {
	return b == Clear || b == Undo || b == Redo
}

// Labels names the toolbar and its buttons for assistive technology and
// tooltips, and words the link prompt. An empty field takes its English
// text.
type Labels struct {
	Toolbar     string
	Undo        string
	Redo        string
	Strong      string
	Emphasis    string
	Underline   string
	Link        string
	BulletList  string
	OrderedList string
	Blockquote  string
	Clear       string
	// TextStyle names the text style dropdown, whose options are Paragraph
	// and Heading followed by the level, as in "Heading 2".
	TextStyle string
	Paragraph string
	Heading   string
	// LinkURL labels the address field of the link prompt.
	LinkURL string
	// LinkSubmit labels the button that adds the link.
	LinkSubmit string
	// CloseDialog labels the button that closes the link prompt.
	CloseDialog string
}

// English is the text of every empty Labels field.
var English = Labels{
	Toolbar:     "Text formatting",
	Undo:        "Undo",
	Redo:        "Redo",
	Strong:      "Bold",
	Emphasis:    "Italic",
	Underline:   "Underline",
	Link:        "Link",
	BulletList:  "Bulleted list",
	OrderedList: "Numbered list",
	Blockquote:  "Quote",
	Clear:       "Clear formatting",
	TextStyle:   "Text style",
	Paragraph:   "Paragraph",
	Heading:     "Heading",
	LinkURL:     "Link address",
	LinkSubmit:  "Add link",
	CloseDialog: "Close",
}

// Icons draws the buttons. A nil field takes the built-in Tabler icon, an
// svg with data-icon set to the button's command.
type Icons struct {
	Undo        templ.Component
	Redo        templ.Component
	Strong      templ.Component
	Emphasis    templ.Component
	Underline   templ.Component
	Link        templ.Component
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
	// Buttons lists the buttons in toolbar order. Clear, Undo and Redo keep
	// their order but always come last. Nil shows DefaultButtons, and an
	// empty slice shows none, which leaves plain paragraphs.
	Buttons []Button
	// Headings are the heading levels, 1 to 6, that the TextStyle dropdown
	// offers after Paragraph. Nil offers 1, 2 and 3.
	Headings []int
	Labels   Labels
	Icons    Icons
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
	l := p.Labels
	return Labels{
		Toolbar:     cmp.Or(l.Toolbar, English.Toolbar),
		Undo:        cmp.Or(l.Undo, English.Undo),
		Redo:        cmp.Or(l.Redo, English.Redo),
		Strong:      cmp.Or(l.Strong, English.Strong),
		Emphasis:    cmp.Or(l.Emphasis, English.Emphasis),
		Underline:   cmp.Or(l.Underline, English.Underline),
		Link:        cmp.Or(l.Link, English.Link),
		BulletList:  cmp.Or(l.BulletList, English.BulletList),
		OrderedList: cmp.Or(l.OrderedList, English.OrderedList),
		Blockquote:  cmp.Or(l.Blockquote, English.Blockquote),
		Clear:       cmp.Or(l.Clear, English.Clear),
		TextStyle:   cmp.Or(l.TextStyle, English.TextStyle),
		Paragraph:   cmp.Or(l.Paragraph, English.Paragraph),
		Heading:     cmp.Or(l.Heading, English.Heading),
		LinkURL:     cmp.Or(l.LinkURL, English.LinkURL),
		LinkSubmit:  cmp.Or(l.LinkSubmit, English.LinkSubmit),
		CloseDialog: cmp.Or(l.CloseDialog, English.CloseDialog),
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
	attrs   templ.Attributes
}

// option is one choice of a dropdown. Its value is 0 for a paragraph and the
// level for a heading.
type option struct {
	value string
	label string
}

func (p Props) textStyles() []option {
	l := p.labels()
	levels := p.Headings
	if levels == nil {
		levels = []int{1, 2, 3}
	}
	options := []option{{"0", l.Paragraph}}
	for _, level := range levels {
		if level >= 1 && level <= 6 {
			options = append(options, option{strconv.Itoa(level), l.Heading + " " + strconv.Itoa(level)})
		}
	}
	return options
}

var pressed = templ.Attributes{"aria-pressed": "false"}

func (p Props) buttons() []button {
	l, i := p.labels(), p.Icons
	list := p.Buttons
	if list == nil {
		list = DefaultButtons
	}
	leading := slices.DeleteFunc(slices.Clone(list), Button.trailing)
	trailing := slices.DeleteFunc(slices.Clone(list), func(b Button) bool { return !b.trailing() })
	var buttons []button
	for _, b := range append(leading, trailing...) {
		switch b {
		case Undo:
			buttons = append(buttons, button{"undo", l.Undo, cmp.Or(i.Undo, tabler("undo", tablerArrowBackUp)), nil})
		case Redo:
			buttons = append(buttons, button{"redo", l.Redo, cmp.Or(i.Redo, tabler("redo", tablerArrowForwardUp)), nil})
		case Strong:
			buttons = append(buttons, button{"strong", l.Strong, cmp.Or(i.Strong, tabler("strong", tablerBold)), pressed})
		case Emphasis:
			buttons = append(buttons, button{"emphasis", l.Emphasis, cmp.Or(i.Emphasis, tabler("emphasis", tablerItalic)), pressed})
		case Underline:
			buttons = append(buttons, button{"underline", l.Underline, cmp.Or(i.Underline, tabler("underline", tablerUnderline)), pressed})
		case Link:
			buttons = append(buttons, button{"link", l.Link, cmp.Or(i.Link, tabler("link", tablerLink)), templ.Attributes{
				"aria-pressed": "false", "data-url-label": l.LinkURL, "data-submit-label": l.LinkSubmit, "data-close-label": l.CloseDialog,
			}})
		case BulletList:
			buttons = append(buttons, button{"bulletList", l.BulletList, cmp.Or(i.BulletList, tabler("bulletList", tablerList)), pressed})
		case OrderedList:
			buttons = append(buttons, button{"orderedList", l.OrderedList, cmp.Or(i.OrderedList, tabler("orderedList", tablerListNumbers)), pressed})
		case Blockquote:
			buttons = append(buttons, button{"blockquote", l.Blockquote, cmp.Or(i.Blockquote, tabler("blockquote", tablerBlockquote)), pressed})
		case Clear:
			buttons = append(buttons, button{"clear", l.Clear, cmp.Or(i.Clear, tabler("clear", tablerClearFormatting)), nil})
		case TextStyle:
			buttons = append(buttons, button{"textStyle", l.TextStyle, nil, nil})
		}
	}
	return buttons
}

// Tabler Icons, MIT licensed. See NOTICE.
const (
	tablerArrowBackUp     = `<path d="m9 14l-4-4l4-4"/><path d="M5 10h11a4 4 0 1 1 0 8h-1"/>`
	tablerArrowForwardUp  = `<path d="m15 14l4-4l-4-4"/><path d="M19 10H8a4 4 0 1 0 0 8h1"/>`
	tablerBold            = `<path d="M7 5h6a3.5 3.5 0 0 1 0 7H7zm6 7h1a3.5 3.5 0 0 1 0 7H7v-7"/>`
	tablerItalic          = `<path d="M11 5h6M7 19h6m1-14l-4 14"/>`
	tablerUnderline       = `<path d="M7 5v5a5 5 0 0 0 10 0V5M5 19h14"/>`
	tablerLink            = `<path d="m9 15l6-6m-4-3l.463-.536a5 5 0 0 1 7.071 7.072L18 13m-5 5l-.397.534a5.07 5.07 0 0 1-7.127 0a4.97 4.97 0 0 1 0-7.071L6 11"/>`
	tablerList            = `<path d="M9 6h11M9 12h11M9 18h11M5 6v.01M5 12v.01M5 18v.01"/>`
	tablerListNumbers     = `<path d="M11 6h9m-9 6h9m-8 6h8M4 16a2 2 0 1 1 4 0c0 .591-.5 1-1 1.5L4 20h4M6 10V4L4 6"/>`
	tablerBlockquote      = `<path d="M6 15h15m0 4H6m9-8h6m0-4h-6M9 9h1a1 1 0 1 1-1 1V7.5a2 2 0 0 1 2-2M3 9h1a1 1 0 1 1-1 1V7.5a2 2 0 0 1 2-2"/>`
	tablerClearFormatting = `<path d="m17 15l4 4m0-4l-4 4M7 6V5h11v1M7 19h4m2-14L9 19"/>`
	tablerChevronDown     = `<path d="m6 9l6 6l6-6"/>`
)

# richeditor

A rich text editor for Go web apps that render HTML with [templ](https://templ.guide).
It is a form field. Put it inside a `<form>` and the HTML posts under its name like any input.
The editing engine is [Wordgard](https://www.npmjs.com/package/wordgard), bundled into one script, so there is no npm step in your project and no other JavaScript framework to load.

The editor supports headings, bold, italic, underline, links, bulleted and numbered lists, quotes, clearing formatting, and undo and redo.
Each editor can offer only some of these.
Typing `- `, `1. ` or `> ` at the start of a line starts a list or a quote, and `## ` an h2 heading.
Mod-b, Mod-i, Mod-u and Mod-k toggle bold, italic, underline and a link.

## Installation

Requires Go 1.26 or later and templ.

```sh
go get github.com/arekbartnik/richeditor
```

## Usage

Serve `richeditor.Files`, load it with `richeditor.Head`, and render `richeditor.Editor` inside a form.

```go
http.Handle("GET /richeditor/", http.StripPrefix("/richeditor/", http.FileServerFS(richeditor.Files)))
```

```templ
templ page(body string) {
	<head>
		@richeditor.Head("/richeditor/editor.js", "/richeditor/editor.css")
	</head>
	<form method="post">
		@richeditor.Editor(richeditor.Props{Name: "body", Label: "Body", Value: body})
		<button>Save</button>
	</form>
}
```

`examples/basic` is a runnable page without templ files. Start it with `go run ./examples/basic`.

`Files` holds `editor.js` and `editor.css`.
An app with its own asset pipeline can copy them into it to fingerprint or compress them, and pass the resulting URLs to `Head`.
The script is a module, so a page that loads it without `Head` needs `<script type="module">`.

### Sanitize on the server

The posted HTML comes from the browser, so it can hold anything.
Sanitize it before you store or display it, for example with [bluemonday](https://github.com/microcosm-cc/bluemonday).
The editor itself produces only `p`, `br`, `strong`, `em`, `u`, `a` with `href`, `ul`, `ol`, `li`, `blockquote` and, with the text style dropdown, `h1` to `h6`.

### Props

| Field | Purpose |
| --- | --- |
| `Name` | Form field the HTML posts under. |
| `Value` | Initial HTML. Form reset restores it. |
| `Label` | Accessible name of the editable area. |
| `Buttons` | The buttons, in toolbar order. See below. |
| `Headings` | The heading levels the `TextStyle` dropdown offers after Paragraph. Nil offers 1, 2 and 3. |
| `Layout` | `richeditor.Toolbar` (default) shows a fixed toolbar. `richeditor.Popover` shows the buttons over selected text. Alt+F10 moves focus into the popover and Escape returns to the text. |
| `Labels` | Button, toolbar and link prompt labels. An empty field takes its text from `richeditor.English`. |
| `Icons` | Button icons as templ components. A nil field takes the built-in Tabler icon. |
| `ButtonClass` | Replaces the `rich-editor-button` class on every button, so the buttons can use your own button styles. |
| `ContentClass` | Added to the editable text, so it can share styles with the stored HTML where you display it. |
| `Attrs` | Extra attributes on the `<rich-editor>` element, such as `id` or event handlers. |

### Element API

`<rich-editor>` is a form-associated custom element.

- `value` gets or sets the HTML.
- `textLength` counts the visible characters.
- `setCustomValidity(message)`, `checkValidity()`, `reportValidity()`, `validity` and `validationMessage` behave as on an input, and `:invalid` matches.
- `focusEnd()` focuses the text with the cursor at the end.
- An `input` event fires on every change.

### Buttons

`Buttons` picks the formatting an editor offers.
The editor only supports the formatting of its buttons, so an editor without `richeditor.Link` drops links from pasted and loaded HTML and ignores Mod-k.

```go
richeditor.Props{
	Name:    "summary",
	Buttons: []richeditor.Button{richeditor.Strong, richeditor.Emphasis, richeditor.Clear},
}
```

Nil shows `richeditor.DefaultButtons`, which is every button but `TextStyle`.
An empty slice shows none and leaves plain paragraphs.
`Clear`, `Undo` and `Redo` always come last and sit at the far end of the toolbar.
Undo and redo stay on the keyboard even without their buttons.

The link button works on selected text.
It asks for an address, adds `https://` to a bare domain such as `example.com`, and refuses schemes other than `http`, `https`, `mailto` and `tel`.
On text that already has a link, it removes the link.

`richeditor.TextStyle` is a dropdown, not a button.
It shows the style of the lines in the selection, Paragraph or a heading, and changing it gives every selected line that style.
`Headings` picks its heading levels.
Typing `## ` at the start of a line and Ctrl-Shift-2 make an h2 when the dropdown offers level 2, and so on for each offered level, while Ctrl-Shift-0 makes a paragraph.
Clear formatting turns headings back into paragraphs.
With any heading offered the editor holds headings of every level, so loaded or pasted HTML keeps an h1 even when only levels 2 and 3 are offered.
Sanitize or rewrite such headings on the server if the page allows only some levels.

```go
richeditor.Props{
	Name:     "article",
	Buttons:  append([]richeditor.Button{richeditor.TextStyle}, richeditor.DefaultButtons...),
	Headings: []int{2, 3},
}
```

### Icons

Pass your own components to use another icon set. Each one renders inside its button.

```go
richeditor.Props{
	Name: "body",
	Icons: richeditor.Icons{
		Strong: myicons.Bold(),
		Link:   myicons.Chain(),
	},
}
```

### Styling

The editor's parts carry these classes.

| Class | Element |
| --- | --- |
| `rich-editor-toolbar` | The fixed toolbar of the `Toolbar` layout. |
| `rich-editor-popover` | The popover of the `Popover` layout. |
| `rich-editor-button` | Each button, unless `ButtonClass` replaces it. |
| `rich-editor-select` | The box around the text style dropdown, holding the `select` and its arrow, an `svg` with `data-icon="textStyle"`. The editor draws the arrow itself, so the `select` has `appearance: none`. |
| `rich-editor-content` | The box around the editable text. The editable text inside it is `wg-content`. |
| `wg-link-dialog` | The link prompt. |

Wordgard adds its own styles to the content box after the page's stylesheets.
Put `rich-editor` in front of the class to outrank them, as in `rich-editor .rich-editor-content wg-content { padding: 16px; }`.

`editor.css` reads these custom properties, so a page can match its theme without overriding selectors.

| Property | Default |
| --- | --- |
| `--rich-editor-border` | `#8d8d8d` |
| `--rich-editor-radius` | `6px` |
| `--rich-editor-background` | `Canvas` |
| `--rich-editor-color` | `CanvasText` |
| `--rich-editor-font` | `inherit` |
| `--rich-editor-shadow` | `none` |
| `--rich-editor-accent` | `#0a66d6`, the focus border, focus ring and selection |
| `--rich-editor-divider` | 15% of the text color |
| `--rich-editor-hover` | 6% of the text color |
| `--rich-editor-pressed` | 10% of the text color |
| `--rich-editor-popover-shadow` | `0 4px 16px rgb(0 0 0 / 15%)` |
| `--rich-editor-button-radius` | `4px` |
| `--rich-editor-min-height` | `4lh` |
| `--rich-editor-padding` | `6px 12px` |

## Development

`src/editor.js` is the element. `npm run build` bundles it with Wordgard into `static/editor.js`, which is committed so `go get` needs no Node.
`go generate` runs that build and templ generation.

## License

MIT. `NOTICE` lists the bundled Wordgard code and the Tabler icons, both MIT licensed.

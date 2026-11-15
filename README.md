# richeditor

A rich text editor for Go web apps that render HTML with [templ](https://templ.guide).
It is a form field. Put it inside a `<form>` and the HTML posts under its name like any input.
The editing engine is [Wordgard](https://www.npmjs.com/package/wordgard), bundled into one script, so there is no npm step in your project and no other JavaScript framework to load.

The editor supports bold, italic, bulleted and numbered lists, quotes, and clearing formatting.
Typing `- `, `1. ` or `> ` at the start of a line starts a list or a quote.

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
The editor itself produces only `p`, `br`, `strong`, `em`, `ul`, `ol`, `li` and `blockquote`.

### Props

| Field | Purpose |
| --- | --- |
| `Name` | Form field the HTML posts under. |
| `Value` | Initial HTML. Form reset restores it. |
| `Label` | Accessible name of the editable area. |
| `Layout` | `richeditor.Toolbar` (default) shows a fixed toolbar. `richeditor.Popover` shows the buttons over selected text. Alt+F10 moves focus into the popover and Escape returns to the text. |
| `Labels` | Button and toolbar labels. Defaults to `richeditor.English`. |
| `ButtonClass` | Replaces the `rich-button` class on every button, so the buttons can use your own button styles. |
| `ContentClass` | Added to the editable area, so it can share styles with the stored HTML where you display it. |
| `Attrs` | Extra attributes on the `<rich-editor>` element, such as `id` or event handlers. |

### Element API

`<rich-editor>` is a form-associated custom element.

- `value` gets or sets the HTML.
- `textLength` counts the visible characters.
- `setCustomValidity(message)`, `checkValidity()`, `reportValidity()`, `validity` and `validationMessage` behave as on an input, and `:invalid` matches.
- `focusEnd()` focuses the text with the cursor at the end.
- An `input` event fires on every change.

### Styling

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

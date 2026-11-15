// Command basic serves one form with an editor and shows the HTML it posts.
// Run it with go run ./examples/basic and open http://localhost:8080.
package main

import (
	"cmp"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"

	"github.com/arekbartnik/richeditor"
)

func main() {
	http.Handle("GET /richeditor/", http.StripPrefix("/richeditor/", http.FileServerFS(richeditor.Files)))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		posted := r.PostFormValue("body")
		fmt.Fprint(w, `<!doctype html><html lang="en"><meta charset="utf-8"><title>richeditor</title>`)
		_ = richeditor.Head("/richeditor/editor.js", "/richeditor/editor.css").Render(r.Context(), w)
		fmt.Fprint(w, `<body style="max-width: 40rem; margin: 2rem auto; font-family: system-ui"><form method="post">`)
		_ = richeditor.Editor(richeditor.Props{Name: "body", Label: "Body", Value: posted}).Render(r.Context(), w)
		fmt.Fprint(w, `<p>A summary allows only bold and italic, so pasted links and lists lose their markup.</p>`)
		summary := r.PostFormValue("summary")
		buttons := []richeditor.Button{richeditor.Strong, richeditor.Emphasis}
		_ = richeditor.Editor(richeditor.Props{Name: "summary", Label: "Summary", Value: summary, Buttons: buttons}).Render(r.Context(), w)
		fmt.Fprint(w, `<p><button>Submit</button> <button type="reset">Reset</button></p></form>`)
		fmt.Fprint(w, `<h2>Posted HTML</h2><pre style="white-space: pre-wrap">`, html.EscapeString(posted), `</pre>`)
		fmt.Fprint(w, `<pre style="white-space: pre-wrap">`, html.EscapeString(summary), `</pre>`)
	})
	addr := cmp.Or(os.Getenv("ADDR"), "localhost:8080")
	log.Printf("listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

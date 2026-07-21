package site

import (
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
)

// Serve rebuilds the site on every request, so edits under the site root show up
// on refresh without a watcher.
func Serve(root, addr string) error {
	fmt.Printf("serving on %s (rebuilds per request)\n", addr)
	return http.ListenAndServe(addr, handler(root))
}

func handler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := Render(root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" || strings.HasSuffix(r.URL.Path, "/") {
			p = path.Join(p, "index.html")
		}
		b, ok := out[p]
		if !ok {
			if b, ok = out[p+"/index.html"]; !ok {
				http.NotFound(w, r)
				return
			}
			p += "/index.html"
		}
		w.Header().Set("Content-Type", contentType(p))
		w.Write(b)
	}
}

// contentType maps an output path to its Content-Type, falling back to
// plain text for extensions the platform MIME database does not know.
func contentType(p string) string {
	if ct := mime.TypeByExtension(path.Ext(p)); ct != "" {
		return ct
	}
	return "text/plain; charset=utf-8"
}

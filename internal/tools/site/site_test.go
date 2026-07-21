package site

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is the website repo root relative to this package.
const repoRoot = "../../.."

// copySources clones the site sources into a fresh root for mutation tests.
func copySources(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"templates", "content", "static"} {
		if err := os.CopyFS(filepath.Join(root, d), os.DirFS(filepath.Join(repoRoot, d))); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := os.ReadFile(filepath.Join(repoRoot, "site.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site.yaml"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSplitFrontmatter(t *testing.T) {
	fm, body, err := splitFrontmatter([]byte("---\ntitle: X\nweight: 2\n---\nbody text\n"))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Title != "X" || fm.Weight != 2 {
		t.Fatalf("frontmatter = %+v", fm)
	}
	if strings.TrimSpace(string(body)) != "body text" {
		t.Fatalf("body = %q", body)
	}
	if _, _, err := splitFrontmatter([]byte("---\ntitle: X\n")); err == nil {
		t.Fatal("unterminated frontmatter must error")
	}
}

func TestHeadingAttributeIDs(t *testing.T) {
	html, err := renderMarkdown([]byte("## Install from source {#install-source}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), `id="install-source"`) {
		t.Fatalf("heading id not rendered: %s", html)
	}
}

func TestRenderSite(t *testing.T) {
	out, err := Render(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"index.html", "docs/index.html", "styles.css", "install.sh", "install.html", "x/index.html", "x/cli/index.html", "x/router/directive/index.html"} {
		if _, ok := out[p]; !ok {
			t.Errorf("missing output %s", p)
		}
	}
	if !strings.Contains(string(out["install.html"]), "url=/docs/#install-source") {
		t.Error("install.html shim does not redirect to /docs/#install-source")
	}

	docs := string(out["docs/index.html"])
	for _, want := range []string{`id="install"`, `id="install-source"`, `id="prerequisites"`, `id="install-cli"`, "Go 1.26 or newer", `aria-current="true"`} {
		if !strings.Contains(docs, want) {
			t.Errorf("docs page missing %q", want)
		}
	}

	home := string(out["index.html"])
	if !strings.Contains(home, `href="/docs/#install-source"`) {
		t.Error("home page missing install-source link")
	}
	for _, tab := range []string{"routes", "providers", "commands", "jobs", "config"} {
		for _, want := range []string{
			`id="tab-` + tab + `" aria-controls="panel-` + tab + `"`,
			`id="panel-` + tab + `" role="tabpanel" aria-labelledby="tab-` + tab + `"`,
		} {
			if !strings.Contains(home, want) {
				t.Errorf("home page missing %q", want)
			}
		}
	}
	if !strings.Contains(home, `aria-labelledby="tab-routes" tabindex="0">`) {
		t.Error("routes panel is not the visible, focusable default")
	}
	for _, tab := range []string{"providers", "commands", "jobs", "config"} {
		if !strings.Contains(home, `aria-labelledby="tab-`+tab+`" tabindex="0" hidden>`) {
			t.Errorf("%s panel not hidden initially", tab)
		}
	}
	if strings.Count(home, `<li role="presentation">`) != 5 {
		t.Error("tab list items must not expose listitem roles inside the tablist")
	}
	for _, want := range []string{"ArrowRight", "ArrowLeft", "'Home'", "'End'", "preventDefault", "t.tabIndex = on ? 0 : -1"} {
		if !strings.Contains(home, want) {
			t.Errorf("tab switcher script missing %q", want)
		}
	}
	if !strings.Contains(string(out["styles.css"]), ".tab-panel[hidden] { display: none; }") {
		t.Error("styles missing the hidden-panel display:none override")
	}

	meta := `<meta name="go-import" content="gofabrik.dev/x git https://github.com/gofabrik/fabrik">`
	for _, p := range []string{"x/cli/index.html", "x/index.html"} {
		if !strings.Contains(string(out[p]), meta) {
			t.Errorf("%s missing go-import meta", p)
		}
	}
	if !strings.Contains(string(out["x/index.html"]), `href="/x/cli/"`) {
		t.Error("x index missing module link")
	}
}

// TestRenderNoBlog pins that an empty blog emits nothing and no nav link.
func TestRenderNoBlog(t *testing.T) {
	root := copySources(t)
	if err := os.RemoveAll(filepath.Join(root, "content", "blog")); err != nil {
		t.Fatal(err)
	}
	out, err := Render(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["blog/index.html"]; ok {
		t.Error("blog index emitted with no posts")
	}
	if strings.Contains(string(out["index.html"]), `href="/blog/"`) {
		t.Error("home page links blog with no posts")
	}
}

// TestRenderBlog adds a post, pinning that posts activate the blog output
// and the nav link.
func TestRenderBlog(t *testing.T) {
	root := copySources(t)
	if err := os.MkdirAll(filepath.Join(root, "content", "blog"), 0o755); err != nil {
		t.Fatal(err)
	}
	post := "---\ntitle: Hello\ndate: \"2026-07-21\"\ndescription: First post.\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(root, "content", "blog", "hello.md"), []byte(post), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Render(root)
	if err != nil {
		t.Fatal(err)
	}
	index, ok := out["blog/index.html"]
	if !ok {
		t.Fatal("blog index not emitted")
	}
	if !strings.Contains(string(index), `href="/blog/hello/"`) {
		t.Error("blog index missing post link")
	}
	page, ok := out["blog/hello/index.html"]
	if !ok {
		t.Fatal("post page not emitted")
	}
	if !strings.Contains(string(page), "<h1>Hello</h1>") {
		t.Error("post page missing title")
	}
	if !strings.Contains(string(page), "July 21, 2026") {
		t.Error("post page missing human-readable date")
	}
	for _, p := range []string{"index.html", "docs/index.html"} {
		if !strings.Contains(string(out[p]), `href="/blog/"`) {
			t.Errorf("%s nav missing blog link once posts exist", p)
		}
	}
}

func TestFrontmatterRequiredFields(t *testing.T) {
	root := copySources(t)
	bad := filepath.Join(root, "content", "docs", "bad.md")
	if err := os.WriteFile(bad, []byte("---\ngroup: Start here\n---\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Render(root); err == nil || !strings.Contains(err.Error(), "missing title") {
		t.Fatalf("doc without title must fail the build, got %v", err)
	}
	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "content", "blog"), 0o755); err != nil {
		t.Fatal(err)
	}
	post := filepath.Join(root, "content", "blog", "undated.md")
	for _, bad := range []string{"---\ntitle: X\n---\nBody.\n", "---\ndate: \"2026-07-21\"\n---\nBody.\n"} {
		if err := os.WriteFile(post, []byte(bad), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Render(root); err == nil || !strings.Contains(err.Error(), "title and date") {
			t.Fatalf("post %q must fail the build, got %v", bad, err)
		}
	}
}

// TestModulePathValidation pins that config module paths cannot escape
// the output tree.
func TestModulePathValidation(t *testing.T) {
	root := copySources(t)
	cfg := filepath.Join(root, "site.yaml")
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, append(b, []byte("  - ../../evil\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Render(root); err == nil || !strings.Contains(err.Error(), "invalid module path") {
		t.Fatalf("traversal module path must fail the build, got %v", err)
	}
}

// TestBuildCleansStale pins that Build replaces public/ instead of layering
// over it.
func TestBuildCleansStale(t *testing.T) {
	root := copySources(t)
	stale := filepath.Join(root, "public", "old", "index.html")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Build(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale output survived Build")
	}
	if _, err := os.Stat(filepath.Join(root, "public", "index.html")); err != nil {
		t.Errorf("built output missing: %v", err)
	}
}

func TestServeHandler(t *testing.T) {
	h := handler(repoRoot)
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("GET", path, nil))
		return w
	}

	for path, ct := range map[string]string{
		"/":           "text/html",
		"/docs/":      "text/html",
		"/docs":       "text/html", // slashless directory falls back to its index
		"/styles.css": "text/css",
	} {
		w := get(path)
		if w.Code != 200 {
			t.Errorf("GET %s = %d", path, w.Code)
			continue
		}
		if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, ct) {
			t.Errorf("GET %s Content-Type = %q, want %s", path, got, ct)
		}
	}
	if w := get("/install.sh"); w.Code != 200 || w.Body.Len() == 0 || w.Header().Get("Content-Type") == "" {
		t.Errorf("GET /install.sh = %d, %d bytes, Content-Type %q", w.Code, w.Body.Len(), w.Header().Get("Content-Type"))
	}
	if got := contentType("file.unknownext"); got != "text/plain; charset=utf-8" {
		t.Errorf("contentType fallback = %q, want text/plain; charset=utf-8", got)
	}
	if got := contentType("styles.css"); !strings.HasPrefix(got, "text/css") {
		t.Errorf("contentType(styles.css) = %q", got)
	}
	if w := get("/nope"); w.Code != 404 {
		t.Errorf("GET /nope = %d, want 404", w.Code)
	}
}

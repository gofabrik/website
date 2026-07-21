// Package site renders gofabrik.dev into public/: static pages and
// docs/blog markdown from the repo root, plus vanity-import pages for
// every module.
package site

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v3"
)

// Config is site.yaml at the site root.
type Config struct {
	Repo         string   `yaml:"repo"`
	VanityPrefix string   `yaml:"vanityPrefix"`
	Modules      []string `yaml:"modules"`
}

// DocPage is one rendered file from content/docs.
type DocPage struct {
	Title       string
	Slug        string
	Group       string
	Weight      int
	Description string
	Lede        string
	URL         string
	Body        template.HTML
}

// Post is one rendered file from content/blog.
type Post struct {
	Title       string
	Date        string
	DisplayDate string
	Description string
	URL         string
	Body        template.HTML
}

// DocGroup is a sidebar group in declared weight order.
type DocGroup struct {
	Name  string
	Pages []*DocPage
}

// Site is everything the templates see.
type Site struct {
	Config    Config
	DocGroups []DocGroup
	Posts     []*Post
}

type frontmatter struct {
	Title       string `yaml:"title"`
	Slug        string `yaml:"slug"`
	Group       string `yaml:"group"`
	Weight      int    `yaml:"weight"`
	Description string `yaml:"description"`
	Lede        string `yaml:"lede"`
	Date        string `yaml:"date"`
}

var markdown = goldmark.New(goldmark.WithParserOptions(parser.WithAttribute()))

// splitFrontmatter separates the leading --- YAML block from the body.
func splitFrontmatter(src []byte) (frontmatter, []byte, error) {
	var fm frontmatter
	rest, ok := bytes.CutPrefix(src, []byte("---\n"))
	if !ok {
		return fm, src, nil
	}
	head, body, ok := bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		return fm, nil, fmt.Errorf("unterminated frontmatter")
	}
	if err := yaml.Unmarshal(head, &fm); err != nil {
		return fm, nil, err
	}
	return fm, body, nil
}

func renderMarkdown(body []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := markdown.Convert(body, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func loadConfig(root string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(filepath.Join(root, "site.yaml"))
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	for _, m := range cfg.Modules {
		if m == "" || !filepath.IsLocal(m) {
			return cfg, fmt.Errorf("site.yaml: invalid module path %q", m)
		}
	}
	return cfg, nil
}

// loadDocs reads content/docs/*.md; index.md maps to /docs/,
// any other file to /docs/<name>/.
func loadDocs(root string) ([]*DocPage, error) {
	files, err := filepath.Glob(filepath.Join(root, "content", "docs", "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var pages []*DocPage
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		fm, body, err := splitFrontmatter(src)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		html, err := renderMarkdown(body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if fm.Title == "" {
			return nil, fmt.Errorf("%s: missing title in frontmatter", f)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		url := "/docs/"
		if name != "index" {
			url = "/docs/" + name + "/"
		}
		slug := fm.Slug
		if slug == "" {
			slug = name
		}
		pages = append(pages, &DocPage{
			Title:       fm.Title,
			Slug:        slug,
			Group:       fm.Group,
			Weight:      fm.Weight,
			Description: fm.Description,
			Lede:        fm.Lede,
			URL:         url,
			Body:        html,
		})
	}
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].Weight < pages[j].Weight })
	return pages, nil
}

// loadPosts reads content/blog/*.md into date-descending posts.
func loadPosts(root string) ([]*Post, error) {
	files, err := filepath.Glob(filepath.Join(root, "content", "blog", "*.md"))
	if err != nil {
		return nil, err
	}
	var posts []*Post
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		fm, body, err := splitFrontmatter(src)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		html, err := renderMarkdown(body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if fm.Title == "" || fm.Date == "" {
			return nil, fmt.Errorf("%s: posts need title and date in frontmatter", f)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		display := fm.Date
		if t, err := time.Parse("2006-01-02", fm.Date); err == nil {
			display = t.Format("January 2, 2006")
		}
		posts = append(posts, &Post{
			Title:       fm.Title,
			Date:        fm.Date,
			DisplayDate: display,
			Description: fm.Description,
			URL:         "/blog/" + name + "/",
			Body:        html,
		})
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].Date > posts[j].Date })
	return posts, nil
}

func groupDocs(pages []*DocPage) []DocGroup {
	var groups []DocGroup
	index := map[string]int{}
	for _, p := range pages {
		i, ok := index[p.Group]
		if !ok {
			i = len(groups)
			index[p.Group] = i
			groups = append(groups, DocGroup{Name: p.Group})
		}
		groups[i].Pages = append(groups[i].Pages, p)
	}
	return groups
}

// Render produces the whole site as output-path -> content for public/
// (index.html, docs/, blog/, x/, and copies of static/).
func Render(root string) (map[string][]byte, error) {
	cfg, err := loadConfig(root)
	if err != nil {
		return nil, err
	}
	docs, err := loadDocs(root)
	if err != nil {
		return nil, err
	}
	posts, err := loadPosts(root)
	if err != nil {
		return nil, err
	}
	s := &Site{Config: cfg, DocGroups: groupDocs(docs), Posts: posts}

	tmplDir := filepath.Join(root, "templates")
	base, err := template.ParseFiles(filepath.Join(tmplDir, "layout.html"))
	if err != nil {
		return nil, err
	}
	page := func(name string, data any) ([]byte, error) {
		t, err := template.Must(base.Clone()).ParseFiles(filepath.Join(tmplDir, name))
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "page", data); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	out := map[string][]byte{}

	home, err := page("home.html", map[string]any{"Site": s})
	if err != nil {
		return nil, err
	}
	out["index.html"] = home

	for _, d := range docs {
		b, err := page("docs.html", map[string]any{"Site": s, "Page": d})
		if err != nil {
			return nil, err
		}
		out[strings.TrimPrefix(d.URL, "/")+"index.html"] = b
	}

	if len(posts) > 0 {
		b, err := page("blog.html", map[string]any{"Site": s})
		if err != nil {
			return nil, err
		}
		out["blog/index.html"] = b
		for _, p := range posts {
			b, err := page("post.html", map[string]any{"Site": s, "Page": p})
			if err != nil {
				return nil, err
			}
			out[strings.TrimPrefix(p.URL, "/")+"index.html"] = b
		}
	}

	vanity, err := template.ParseFiles(filepath.Join(tmplDir, "vanity.html"))
	if err != nil {
		return nil, err
	}
	// The go command verifies the non-exact gofabrik.dev/x prefix by
	// fetching /x?go-get=1, so the index page carries the same meta tag.
	var xindex bytes.Buffer
	err = vanity.Execute(&xindex, map[string]any{
		"Prefix":  cfg.VanityPrefix,
		"Repo":    cfg.Repo,
		"Modules": cfg.Modules,
	})
	if err != nil {
		return nil, err
	}
	out["x/index.html"] = xindex.Bytes()
	for _, m := range cfg.Modules {
		var buf bytes.Buffer
		err := vanity.Execute(&buf, map[string]any{
			"Prefix": cfg.VanityPrefix,
			"Repo":   cfg.Repo,
			"Module": m,
		})
		if err != nil {
			return nil, err
		}
		out[path.Join("x", m, "index.html")] = buf.Bytes()
	}

	staticDir := filepath.Join(root, "static")
	err = filepath.WalkDir(staticDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(staticDir, p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Build renders the site and replaces root/public with it, so removed
// pages and assets do not linger in the output.
func Build(root string) error {
	out, err := Render(root)
	if err != nil {
		return err
	}
	pub := filepath.Join(root, "public")
	if err := os.RemoveAll(pub); err != nil {
		return err
	}
	for p, b := range out {
		dst := filepath.Join(pub, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

package renderly3

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"text/template/parse"

	"github.com/bokwoon95/erro"
	"github.com/oxtoacart/bpool"
)

// envfuncs occur in n different places:
// 1) They can be packaged with an asset (only for js, envfuncs in css assets are ignored)
// 2) They can be included as part of the GlobalEnvfuncs
// 3) They can be passed in with render.PageWithEnv()
//	- page.Render(w, r, data, filenames, )
type Page struct {
	bufpool  *bpool.BufferPool
	html     *template.Template
	css      []*Asset
	js       []*Asset
	prehooks []Prehook
	envfuncs []func(*http.Request) (name string, value interface{}, err error)
	config   map[string]string
}

func (ry *Renderly) Lookup(filenames ...string) (Page, error) {
	if len(filenames) == 0 {
		return Page{}, fmt.Errorf("no files were passed in")
	}
	fullname := strings.Join(filenames, "\n")
	// If page is already cached for the given fullname, return that page and exit
	if ry.cacheenabled {
		ry.mu.RLock()
		page, ok := ry.cachepage[fullname]
		ry.mu.RUnlock()
		if ok {
			return page, nil
		}
	}
	var err error
	// Else construct the page from scratch
	page := Page{
		bufpool:  ry.bufpool,
		css:      ry.css[""],      // global css assets
		js:       ry.js[""],       // global js assets
		prehooks: ry.prehooks[""], // global prehooks
	}
	// Clone the page template from the base template
	page.html, err = ry.html.Clone()
	if err != nil {
		return page, erro.Wrap(err)
	}
	page.html = page.html.Funcs(ry.funcs).Option(ry.opts...)
	HTML, CSS, JS := categorize(filenames)
	if len(HTML) == 0 {
		return Page{}, fmt.Errorf("no html files were passed in")
	}
	// Add user-specified HTML templates to the page template
	for _, filename := range HTML {
		var t *template.Template
		// If the template is already cached for the given filename, use that template
		if ry.cacheenabled {
			ry.mu.RLock()
			t = ry.cachehtml[filename]
			ry.mu.RUnlock()
		}
		// Else construct the template from scratch
		if t == nil {
			b, err := ry.ReadFile(filename)
			if err != nil {
				return page, erro.Wrap(err)
			}
			t, err = template.New(filename).Funcs(ry.funcs).Option(ry.opts...).Parse(string(b))
			if err != nil {
				return page, erro.Wrap(err)
			}
			// Cache the template if the user enabled it
			if ry.cacheenabled {
				ry.mu.Lock()
				ry.cachehtml[filename] = t
				ry.mu.Unlock()
			}
		}
		// Add to page template
		err := addParseTree(page.html, t, t.Name())
		if err != nil {
			return page, erro.Wrap(err)
		}
	}
	page.html = page.html.Lookup(HTML[0])
	if page.html == nil {
		return page, fmt.Errorf(`no template found for name "%s"`, HTML[0])
	}
	// Find the list of dependency templates invoked by the main HTML template
	depedencies, err := listAllDeps(page.html, HTML[0])
	if err != nil {
		return page, erro.Wrap(err)
	}
	// For each depedency template, figure out the corresponding set of
	// CSS/JS/Prehooks/Posthooks to include in the page. A map is used keep
	// track of every included CSS/JS asset (identified by their hash) so that
	// we do not include the same asset twice.
	cssset := make(map[[32]byte]struct{})
	jsset := make(map[[32]byte]struct{})
	for _, asset := range page.css {
		cssset[asset.Hash] = struct{}{}
	}
	for _, asset := range page.js {
		jsset[asset.Hash] = struct{}{}
	}
	for _, templateName := range depedencies {
		// css
		for _, asset := range ry.css[templateName] {
			if _, ok := cssset[asset.Hash]; ok {
				continue
			}
			cssset[asset.Hash] = struct{}{}
			page.css = append(page.css, asset)
		}
		// js
		for _, asset := range ry.js[templateName] {
			if _, ok := jsset[asset.Hash]; ok {
				continue
			}
			jsset[asset.Hash] = struct{}{}
			page.js = append(page.js, asset)
		}
		// prehooks
		page.prehooks = append(page.prehooks, ry.prehooks[templateName]...)
	}
	// Add the user-specified CSS files to the page
	for _, filename := range CSS {
		var asset *Asset
		// If CSS asset is already cached for the given file name, use that asset
		if ry.cacheenabled {
			ry.mu.RLock()
			asset = ry.cachecss[filename]
			ry.mu.RUnlock()
		}
		// Else construct the CSS asset from scratch
		if asset == nil {
			b, err := ry.ReadFile(filename)
			if err != nil {
				return page, erro.Wrap(err)
			}
			asset = &Asset{
				Data: string(b),
				Hash: sha256.Sum256(b),
			}
			// Cache the CSS asset if the user enabled it
			if ry.cacheenabled {
				ry.mu.Lock()
				ry.cachecss[filename] = asset
				ry.mu.Unlock()
			}
		}
		// Add CSS asset to page if it hasn't already been added
		if _, ok := cssset[asset.Hash]; !ok {
			cssset[asset.Hash] = struct{}{}
			page.css = append(page.css, asset)
		}
	}
	// Add the user-specified JS files to the page
	for _, filename := range JS {
		var asset *Asset
		// If JS asset is already cached for the given file name, use that asset
		if ry.cacheenabled {
			ry.mu.RLock()
			asset = ry.cachejs[filename]
			ry.mu.RUnlock()
		}
		// Else construct the JS asset from scratch
		if asset == nil {
			b, err := ry.ReadFile(filename)
			if err != nil {
				return page, erro.Wrap(err)
			}
			asset = &Asset{
				Data: string(b),
				Hash: sha256.Sum256(b),
			}
			// Cache the JS asset if the user enabled it
			if ry.cacheenabled {
				ry.mu.Lock()
				ry.cachejs[filename] = asset
				ry.mu.Unlock()
			}
		}
		// Add JS asset to page if it hasn't already been added
		if _, ok := jsset[asset.Hash]; !ok {
			jsset[asset.Hash] = struct{}{}
			page.js = append(page.js, asset)
		}
	}
	// Cache the page if the user enabled it
	if ry.cacheenabled {
		ry.mu.Lock()
		ry.cachepage[fullname] = page
		ry.mu.Unlock()
	}
	return page, nil
}

var r1 = regexp.MustCompile(`(?:;|^)\s*(?:frame-ancestors|report-uri|sandbox)[^;]*\s*`)

func (page Page) Render(w io.Writer, r *http.Request, data interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	if page.bufpool == nil || page.html == nil {
		return fmt.Errorf("tried to render an empty page")
	}
	var err error
	for _, fn := range page.prehooks {
		data, err = fn(w, r, data)
		if err != nil {
			return erro.Wrap(err)
		}
	}
	if data == nil {
		data = make(map[string]interface{})
	}
	if mapdata, ok := data.(map[string]interface{}); ok {
		if len(page.css) > 0 {
			mapdata["__css__"] = page.CSS(w)
		}
		if len(page.js) > 0 {
			mapdata["__js__"] = page.JS(w, r)
		}
		if w, ok := w.(http.ResponseWriter); ok {
			w.Header().Set("Content-Type", "text/html")
			// this must be computed -AFTER- making the necessary changes to the
			// CSP header! So that it will reflect the latest version of CSP.
			if CSP := w.Header().Get("Content-Security-Policy"); CSP != "" {
				CSP = r1.ReplaceAllString(CSP, "") // not sure if this is worth doing but ok
				mapdata["__Content_Security_Policy__"] = template.HTML(fmt.Sprintf(`<meta http-equiv="Content-Security-Policy" content="%s">`, CSP))
			}
		} else {
			mapdata["__Content_Security_Policy__"] = template.HTML(`<meta http-equiv="Content-Security-Policy" content="">`)
		}
		data = mapdata
	}
	err = executeTemplate(page.html, page.bufpool, w, page.html.Name(), data)
	if err != nil {
		return erro.Wrap(err)
	}
	return nil
}

func (page Page) Nonce(w http.ResponseWriter) (template.HTMLAttr, error) {
	arr := make([]byte, 32)
	_, err := rand.Read(arr)
	if err != nil {
		return "", erro.Wrap(err)
	}
	nonce := base64.StdEncoding.EncodeToString(arr)
	_ = appendCSP(w, "style-src", `'nonce-`+nonce+`'`)
	_ = appendCSP(w, "script-src", `'nonce-`+nonce+`'`)
	return template.HTMLAttr(`nonce="` + nonce + `"`), nil
}

func (page Page) CSS(w io.Writer) template.HTML {
	// Generate Content-Security-Policy script-src
	styles := &strings.Builder{}
	styleHashes := &strings.Builder{}
	for i, asset := range page.css {
		if i > 0 {
			styles.WriteString("\n")
			styleHashes.WriteString(" ")
		}
		styles.WriteString("<style>")
		styles.WriteString(asset.Data)
		styles.WriteString("</style>")
		styleHashes.WriteString("'sha256-")
		styleHashes.WriteString(base64.StdEncoding.EncodeToString(asset.Hash[0:]))
		styleHashes.WriteString("'")
	}
	if styleHashes.Len() > 0 {
		if w, ok := w.(http.ResponseWriter); ok {
			_ = appendCSP(w, "style-src", "'self'")
			_ = appendCSP(w, "style-src", styleHashes.String())
		}
	}
	return template.HTML(styles.String())
}

func (page Page) JS(w io.Writer, r *http.Request) template.HTML {
	// Generate Content-Security-Policy script-src
	scripts := &strings.Builder{}
	scriptHashes := &strings.Builder{}
	writepayload := func(name, payload string) {
		data := "const " + name + " = function(){ return " + payload + " }"
		hash := sha256.Sum256([]byte(data))
		scripts.WriteString("<script>")
		scripts.WriteString(data)
		scripts.WriteString("</script>")
		scriptHashes.WriteString(" 'sha256-")
		scriptHashes.WriteString(base64.StdEncoding.EncodeToString(hash[0:]))
		scriptHashes.WriteString("' ")
	}
	for _, fn := range page.envfuncs {
		var payload string
		name, value, err := fn(r)
		if err != nil {
			payload = "new Error(" + err.Error() + ")"
			writepayload(name, payload)
			continue
		}
		b, err := json.Marshal(value)
		if err != nil {
			payload = "new Error(" + err.Error() + ")"
			writepayload(name, payload)
			continue
		}
		writepayload(name, string(b))
		scripts.WriteString("\n")
	}
	for i, asset := range page.js {
		if i > 0 {
			scripts.WriteString("\n")
			scriptHashes.WriteString(" ")
		}
		scripts.WriteString("<script>")
		scripts.WriteString(asset.Data)
		scripts.WriteString("</script>")
		scriptHashes.WriteString(" 'sha256-")
		scriptHashes.WriteString(base64.StdEncoding.EncodeToString(asset.Hash[0:]))
		scriptHashes.WriteString("' ")
	}
	if scriptHashes.Len() > 0 {
		if w, ok := w.(http.ResponseWriter); ok {
			_ = appendCSP(w, "script-src", "'self'")
			_ = appendCSP(w, "script-src", scriptHashes.String())
		}
	}
	return template.HTML(scripts.String())
}

func listAllDeps(t *template.Template, name string) ([]string, error) {
	t = t.Lookup(name) // set the main template to `name`
	if t == nil {
		return nil, fmt.Errorf(`no such template "%s"`, name)
	}
	var allnames = []string{t.Name()}
	var set = make(map[string]struct{})
	var root parse.Node = t.Tree.Root
	var roots []parse.Node
	for {
		names := listDeps(root)
		for _, name := range names {
			if _, ok := set[name]; ok {
				continue
			}
			set[name] = struct{}{}
			allnames = append(allnames, name)
			t = t.Lookup(name)
			if t == nil {
				return allnames, fmt.Errorf(`{{ template "%s" }} was referenced, but was not found`, name)
			}
			roots = append(roots, t.Tree.Root)
		}
		if len(roots) == 0 {
			break
		}
		root, roots = roots[0], roots[1:]
	}
	return allnames, nil
}

func listDeps(node parse.Node) []string {
	var names []string
	switch node := node.(type) {
	case *parse.TemplateNode:
		names = append(names, node.Name)
	case *parse.ListNode:
		for _, n := range node.Nodes {
			names = append(names, listDeps(n)...)
		}
	}
	return names
}

func appendCSP(w http.ResponseWriter, policy, value string) error {
	const key = "Content-Security-Policy"
	CSP := w.Header().Get(key)
	if CSP == "" {
		w.Header().Set(key, policy+" "+value)
		return nil
	}
	CSP = strings.ReplaceAll(CSP, "\n", " ") // newlines screw up the regex matching, remove them
	re, err := regexp.Compile(`(.*` + policy + `[^;]*)(;|$)(.*)`)
	if err != nil {
		return erro.Wrap(err)
	}
	matches := re.FindStringSubmatch(CSP)
	if len(matches) == 0 {
		w.Header().Set(key, CSP+"; "+policy+" "+value)
		return nil
	}
	newCSP := matches[1] + " " + value + matches[2] + matches[3]
	w.Header().Set("Content-Security-Policy", newCSP)
	return nil
}

func AppendCSP(rw http.ResponseWriter, policy, value string) error {
	const key = "Content-Security-Policy"
	CSP := rw.Header().Get(key)
	if CSP == "" {
		rw.Header().Set(key, policy+" "+value)
		return nil
	}
	CSP = strings.ReplaceAll(CSP, "\n", " ") // newlines screw up the regex matching, remove them
	re, err := regexp.Compile(`(.*` + policy + `[^;]*)(;|$)(.*)`)
	if err != nil {
		return erro.Wrap(err)
	}
	matches := re.FindStringSubmatch(CSP)
	if len(matches) == 0 {
		rw.Header().Set(key, CSP+"; "+policy+" "+value)
		return nil
	}
	newCSP := matches[1] + " " + value + matches[2] + matches[3]
	rw.Header().Set("Content-Security-Policy", newCSP)
	return nil
}

func executeTemplate(t *template.Template, bufpool *bpool.BufferPool, w io.Writer, name string, data interface{}) error {
	tempbuf := bufpool.Get()
	defer bufpool.Put(tempbuf)
	err := t.ExecuteTemplate(tempbuf, name, data)
	if err != nil {
		return erro.Wrap(err)
	}
	_, err = tempbuf.WriteTo(w)
	if err != nil {
		return erro.Wrap(err)
	}
	return nil
}

func categorize(names []string) (html, css, js []string) {
	for _, name := range names {
		truncatedName := name
		// if i := strings.IndexRune(name, '?'); i > 0 {
		// 	truncatedName = name[:i]
		// }
		ext := strings.ToLower(filepath.Ext(truncatedName))
		switch ext {
		case ".css":
			css = append(css, name)
		case ".js":
			js = append(js, name)
		default:
			html = append(html, name)
		}
	}
	return html, css, js
}

func addParseTree(parent, child *template.Template, childName string) error {
	var err error
	if childName == "" {
		childName = child.Name()
	}
	for _, t := range child.Templates() {
		if t == child {
			_, err = parent.AddParseTree(childName, t.Tree)
			if err != nil {
				return erro.Wrap(err)
			}
			continue
		}
		_, err = parent.AddParseTree(t.Name(), t.Tree)
		if err != nil {
			return erro.Wrap(err)
		}
	}
	return nil
}

func (ry *Renderly) ReadFile(filename string) ([]byte, error) {
	// ReadFile copied from function of the same name in
	// $GOROOT/src/io/fs/readfile.go, with minor adjustments.
	//
	// Copyright 2020 The Go Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.
	file, err := ry.Open(filename)
	if err != nil {
		return nil, erro.Wrap(err)
	}
	defer file.Close()
	var size int
	if info, err := file.Stat(); err == nil {
		size64 := info.Size()
		if int64(int(size64)) == size64 {
			size = int(size64)
		}
	}
	data := make([]byte, 0, size+1)
	for {
		if len(data) >= cap(data) {
			d := append(data[:cap(data)], 0)
			data = d[:len(data)]
		}
		n, err := file.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return data, erro.Wrap(err)
		}
	}
}

func (ry *Renderly) Resolve(name string) (fs.FS, string) {
	var fsys = ry.fs
	if name != "" && name[0] == '~' {
		i := strings.IndexRune(name, '/')
		if i > 0 {
			fsName := name[1:i]
			altfs := ry.altfs[fsName]
			if altfs != nil {
				fsys = altfs
			}
		}
		name = name[i+1:]
	}
	return fsys, name
}

// Open implements fs.FS, which can be converted to a http.Filesystem using http.FS
func (ry *Renderly) Open(name string) (fs.File, error) {
	var fsys = ry.fs
	if name != "" && name[0] == '~' {
		i := strings.IndexRune(name, '/')
		if i > 0 {
			fsName := name[1:i]
			altfs := ry.altfs[fsName]
			if altfs != nil {
				fsys = altfs
			}
		}
		name = name[i+1:]
	}
	return fsys.Open(name)
}

func (ry *Renderly) FileServer() http.Handler {
	// render := renderly.New(os.DirFS("."))
	//
	// // net/http
	// http.Handle("/static/", http.StripPrefix("/static/", render.FileServer()))
	//
	// // go-chi/chi
	// r.Handle("/static/*", http.StripPrefix("/static/", render.FileServer()))
	//
	// // gorilla/mux
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", render.FileServer()))
	//
	// // julienschmidt/httprouter
	// r.ServeFiles("/static/*filepath", http.FS(render))
	return http.FileServer(http.FS(ry))
}

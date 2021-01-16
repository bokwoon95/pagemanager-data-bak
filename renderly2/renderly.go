package renderly

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"text/template/parse"

	"github.com/bokwoon95/erro"
	"github.com/google/uuid"
	"github.com/oxtoacart/bpool"
)

type Asset struct {
	Data string
	hash [32]byte
}

type EnvFunc func(w io.Writer, r *http.Request, env map[string]interface{}) error

type Plugin struct {
	pluginID          int
	InitErr           error
	FuncMap           map[string]interface{}
	Assets            map[string]Asset
	EnvFuncs          map[string]EnvFunc
	Components        []Component
	GlobalCSS         []string
	GlobalJS          []string
	GlobalHTMLEnvFunc []string
	GlobalJSEnvFunc   []string
}

type Component struct {
	pluginID     int
	HTML         *template.Template
	CSS          []string
	JS           []string
	HTMLEnvFuncs []string
	JSEnvFuncs   []string
}

type mapkey struct {
	pluginID int
	name     string
}

type Page struct {
	bufpool *bpool.BufferPool
	html    *template.Template
	// plugin components
	css          []mapkey
	js           []mapkey
	htmlenvfuncs []mapkey
	jsenvfuncs   []mapkey
	assets       map[mapkey]Asset
	envfuncs     map[mapkey]EnvFunc
	fsys         MuxFS
}

type MuxFS struct {
	DefaultFS fs.FS
	AltFS     map[string]fs.FS
	// TODO: if the fs path starts with a space followed by the tilde prefix, the prefix is taken literally to be a part of the path (?)
}

type Renderly struct {
	mu      *sync.RWMutex
	bufpool *bpool.BufferPool
	fsys    MuxFS
	funcmap map[string]interface{}
	opts    []string
	// plugins
	pluginCount        int
	basetemplate       *template.Template
	components         map[string]Component
	assets             map[mapkey]Asset
	envfuncs           map[mapkey]EnvFunc
	globalcss          []mapkey
	globaljs           []mapkey
	globalhtmlenvfuncs []mapkey
	globaljsenvfuncs   []mapkey
	// fs cache (user's templates only, not plugins')
	cacheenabled bool
	cachepage    map[string]Page
	cachehtml    map[string]*template.Template
	cachecss     map[string]Asset // not strictly necessary to cache fsys assets, just a premature optimization to make serving files closer to 'static' websites
	cachejs      map[string]Asset
	//
	errorhandler func(http.ResponseWriter, *http.Request, error)
}

type Option func(*Renderly) error

func New(fsys fs.FS, opts ...Option) (*Renderly, error) {
	ry := &Renderly{
		mu:      &sync.RWMutex{},
		fsys:    MuxFS{AltFS: make(map[string]fs.FS)},
		bufpool: bpool.NewBufferPool(64),
		funcmap: make(map[string]interface{}),
		// plugin
		basetemplate: template.New(""),
		assets:       make(map[mapkey]Asset),
		envfuncs:     make(map[mapkey]EnvFunc),
		// fs cache
		cachepage: make(map[string]Page),
		cachehtml: make(map[string]*template.Template),
		cachecss:  make(map[string]Asset),
		cachejs:   make(map[string]Asset),
	}
	var err error
	for _, opt := range opts {
		err = opt(ry)
		if err != nil {
			return ry, erro.Wrap(err)
		}
	}
	// ry.cacheenabled = true
	return ry, nil
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

// Open implements fs.FS, which can be converted to a http.Filesystem using http.FS
func (muxfs MuxFS) Open(name string) (fs.File, error) {
	var fsys = muxfs.DefaultFS
	if name != "" && name[0] == '~' {
		i := strings.IndexRune(name, '/')
		if i > 0 {
			fsName := name[1:i]
			altfs := muxfs.AltFS[fsName]
			if altfs != nil {
				fsys = altfs
			}
		}
		name = name[i+1:]
	}
	return fsys.Open(name)
}

func (muxfs MuxFS) ReadFile(filename string) ([]byte, error) {
	// ReadFile copied from function of the same name in
	// $GOROOT/src/io/fs/readfile.go, with minor adjustments.
	//
	// Copyright 2020 The Go Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.
	file, err := muxfs.Open(filename)
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

// TODO: rewrite the recursive part into a loop
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

func TemplateFuncs(funcmaps ...map[string]interface{}) Option {
	return func(ry *Renderly) error {
		if ry.funcmap == nil {
			ry.funcmap = make(map[string]interface{})
		}
		for _, funcmap := range funcmaps {
			for name, fn := range funcmap {
				ry.funcmap[name] = fn
			}
		}
		return nil
	}
}

func TemplateOpts(option ...string) Option {
	return func(ry *Renderly) error {
		ry.opts = option
		return nil
	}
}

func GlobalCSS(fsys fs.FS, filenames ...string) Option {
	return func(ry *Renderly) error {
		for _, filename := range filenames {
			k := mapkey{name: filename}
			b, err := fs.ReadFile(fsys, filename)
			if err != nil {
				return erro.Wrap(err)
			}
			ry.globalcss = append(ry.globalcss, k)
			ry.assets[k] = Asset{
				Data: string(b),
				hash: sha256.Sum256(b),
			}
		}
		return nil
	}
}

func GlobalJS(fsys fs.FS, filenames ...string) Option {
	return func(ry *Renderly) error {
		for _, filename := range filenames {
			k := mapkey{name: filename}
			b, err := fs.ReadFile(fsys, filename)
			if err != nil {
				return erro.Wrap(err)
			}
			ry.globaljs = append(ry.globaljs, k)
			ry.assets[k] = Asset{
				Data: string(b),
				hash: sha256.Sum256(b),
			}
		}
		return nil
	}
}

func GlobalTemplates(fsys fs.FS, filenames ...string) Option {
	return func(ry *Renderly) error {
		if ry.basetemplate == nil {
			ry.basetemplate = template.New("")
		}
		for _, name := range filenames {
			b, err := fs.ReadFile(fsys, name)
			if err != nil {
				return erro.Wrap(err)
			}
			t, err := template.New(name).Funcs(ry.funcmap).Option(ry.opts...).Parse(string(b))
			if err != nil {
				return erro.Wrap(err)
			}
			err = addParseTree(ry.basetemplate, t, t.Name())
			if err != nil {
				return erro.Wrap(err)
			}
		}
		return nil
	}
}

func GlobalHTMLEnvFuncs(fns ...EnvFunc) Option {
	return func(ry *Renderly) error {
		for _, fn := range fns {
			name := uuid.New().String()
			k := mapkey{name: name}
			ry.globalhtmlenvfuncs = append(ry.globalhtmlenvfuncs, k)
			ry.envfuncs[k] = fn
		}
		return nil
	}
}

func GlobalJSEnvFuncs(fns ...EnvFunc) Option {
	return func(ry *Renderly) error {
		for _, fn := range fns {
			name := uuid.New().String()
			k := mapkey{name: name}
			ry.globaljsenvfuncs = append(ry.globaljsenvfuncs, k)
			ry.envfuncs[k] = fn
		}
		return nil
	}
}

func AltFS(name string, fsys fs.FS) Option {
	return func(ry *Renderly) error {
		ry.fsys.AltFS[name] = fsys
		return nil
	}
}

func AbsDir(relativePath string) string {
	_, absolutePath, _, _ := runtime.Caller(1)
	return filepath.Join(absolutePath, "..", relativePath) + string(os.PathSeparator)
}

func Plugins(plugins ...Plugin) Option {
	return func(ry *Renderly) error {
		if ry.basetemplate == nil {
			ry.basetemplate = template.New("")
		}
		for _, plugin := range plugins {
			ry.mu.Lock()
			ry.pluginCount++
			pluginID := ry.pluginCount
			ry.mu.Unlock()
			if plugin.InitErr != nil {
				return erro.Wrap(plugin.InitErr)
			}
			for _, component := range plugin.Components {
				if component.HTML == nil {
					return erro.Wrap(fmt.Errorf("component with nil template"))
				}
				name := component.HTML.Name()
				ry.components[name] = component
				err := addParseTree(ry.basetemplate, component.HTML, name) // TODO: actually I don't want to put all component templates into the basetemplate
				if err != nil {
					return erro.Wrap(err)
				}
			}
			for name, asset := range plugin.Assets {
				k := mapkey{pluginID: pluginID, name: name}
				asset.hash = sha256.Sum256([]byte(asset.Data))
				ry.assets[k] = asset
			}
			for name, fn := range plugin.EnvFuncs {
				k := mapkey{pluginID: pluginID, name: name}
				ry.envfuncs[k] = fn
			}
			for name, fn := range plugin.FuncMap {
				ry.funcmap[name] = fn
			}
			for _, name := range plugin.GlobalCSS {
				ry.globalcss = append(ry.globalcss, mapkey{pluginID: pluginID, name: name})
			}
			for _, name := range plugin.GlobalJS {
				ry.globaljs = append(ry.globaljs, mapkey{pluginID: pluginID, name: name})
			}
			for _, name := range plugin.GlobalHTMLEnvFunc {
				ry.globalhtmlenvfuncs = append(ry.globalhtmlenvfuncs, mapkey{pluginID: pluginID, name: name})
			}
			for _, name := range plugin.GlobalJSEnvFunc {
				ry.globaljsenvfuncs = append(ry.globaljsenvfuncs, mapkey{pluginID: pluginID, name: name})
			}
		}
		return nil
	}
}

func (ry *Renderly) Lookup(mainfile string, includefiles ...string) (Page, error) {
	fullname := strings.Join(append([]string{mainfile}, includefiles...), "\n")
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
		assets:   make(map[mapkey]Asset),
		envfuncs: make(map[mapkey]EnvFunc),
	}
	// Clone the page template from the base template
	page.html, err = ry.basetemplate.Clone()
	if err != nil {
		return page, erro.Wrap(err)
	}
	page.html = page.html.Funcs(ry.funcmap).Option(ry.opts...)
	HTMLFiles, CSSFiles, JSFiles := categorize(includefiles)
	// Add user-specified HTML templates to the page template
	for _, filename := range append([]string{mainfile}, HTMLFiles...) {
		var t *template.Template
		// If the template is already cached for the given filename, use that template
		if ry.cacheenabled {
			ry.mu.RLock()
			t = ry.cachehtml[filename]
			ry.mu.RUnlock()
		}
		// Else construct the template from scratch
		if t == nil {
			b, err := ry.fsys.ReadFile(filename)
			if err != nil {
				return page, erro.Wrap(err)
			}
			t, err = template.New(filename).Funcs(ry.funcmap).Option(ry.opts...).Parse(string(b))
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
	page.html = page.html.Lookup(mainfile)
	if page.html == nil {
		return page, fmt.Errorf(`no template found for name "%s"`, mainfile)
	}
	// Add global assets/envfuncs
	page.css = append(page.css, ry.globalcss...)
	page.js = append(page.js, ry.globaljs...)
	page.htmlenvfuncs = append(page.htmlenvfuncs, ry.globalhtmlenvfuncs...)
	page.jsenvfuncs = append(page.jsenvfuncs, ry.globaljsenvfuncs...)
	// Find the list of dependency templates invoked by the main HTML template
	depedencies, err := listAllDeps(page.html, mainfile)
	if err != nil {
		return page, erro.Wrap(err)
	}
	// For each depedency template, figure out the corresponding set of
	// CSS/JS/HTMLEnvFuncs/JSEnvFuncs to include in the page.
	for _, componentName := range depedencies {
		component, ok := ry.components[componentName]
		if !ok {
			continue
		}
		err = addParseTree(page.html, component.HTML, componentName)
		if err != nil {
			return page, erro.Wrap(err)
		}
		for _, name := range component.CSS {
			page.css = append(page.css, mapkey{
				pluginID: component.pluginID,
				name:     name,
			})
		}
		for _, name := range component.JS {
			page.js = append(page.js, mapkey{
				pluginID: component.pluginID,
				name:     name,
			})
		}
		for _, name := range component.HTMLEnvFuncs {
			page.htmlenvfuncs = append(page.htmlenvfuncs, mapkey{
				pluginID: component.pluginID,
				name:     name,
			})
		}
		for _, name := range component.JSEnvFuncs {
			page.jsenvfuncs = append(page.jsenvfuncs, mapkey{
				pluginID: component.pluginID,
				name:     name,
			})
		}
	}
	// Add the user-specified CSS files to the page
	for _, filename := range CSSFiles {
		page.css = append(page.css, mapkey{name: filename})
	}
	// Add the user-specified JS files to the page
	for _, filename := range JSFiles {
		page.js = append(page.js, mapkey{name: filename})
	}
	// TODO: dedup page.css, page.js, page.htmlenvfuncs, page.jsenvfuncs
	// Cache the page if the user enabled it
	if ry.cacheenabled {
		ry.mu.Lock()
		ry.cachepage[fullname] = page
		ry.mu.Unlock()
	}
	return page, nil
}

func (page Page) Render(w io.Writer, r *http.Request, data interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	if page.bufpool == nil || page.html == nil {
		return fmt.Errorf("tried to render an empty page")
	}
	htmlenv := make(map[string]interface{})
	jsenv := make(map[string]interface{})
	var err error
	for _, key := range page.htmlenvfuncs {
		fn := page.envfuncs[key]
		if fn == nil {
			return erro.Wrap(fmt.Errorf("function %+v is nil or doesn't exist", key))
		}
		err = fn(w, r, htmlenv)
		if err != nil {
			return erro.Wrap(err)
		}
	}
	for _, key := range page.jsenvfuncs {
		fn := page.envfuncs[key]
		if fn == nil {
			return erro.Wrap(fmt.Errorf("function %+v is nil or doesn't exist", key))
		}
		err = fn(w, r, jsenv)
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

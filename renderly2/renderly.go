package renderly

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"text/template/parse"

	"github.com/bokwoon95/erro"
	"github.com/oxtoacart/bpool"
)

type Asset struct {
	Data string
	hash [32]byte
}

type EnvFunc func(w http.ResponseWriter, r *http.Request, env map[string]interface{}) error

type Plugin struct {
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

type mapkey struct {
	index int
	name  string
}

type Component struct {
	index        int
	html         *template.Template
	css          []mapkey
	js           []mapkey
	htmlenvfuncs []mapkey
	jsenvfuncs   []mapkey
	// shared with *Renderly
	bufpool  *bpool.BufferPool
	assets   map[mapkey]Asset
	envfuncs map[mapkey]EnvFunc
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
	fs      fs.FS
	altfs   map[string]fs.FS
	funcmap map[string]interface{}
	opts    []string
	// plugins
	basetemplate   *template.Template
	components     map[string]Component
	assets         map[mapkey]Asset
	envfuncs       map[mapkey]EnvFunc
	globalassets   []mapkey
	globalenvfuncs []mapkey
	// fs cache (user's templates only, not plugins')
	cacheenabled bool
	cachepage    map[string]Component
	cachehtml    map[string]*template.Template
	cachecss     map[string]Asset
	cachejs      map[string]Asset
	//
	errorhandler func(http.ResponseWriter, *http.Request, error)
}

type Option func(*Renderly) error

func New(fsys fs.FS, opts ...Option) (*Renderly, error) {
	ry := &Renderly{
		mu:      &sync.RWMutex{},
		fs:      fsys,
		altfs:   make(map[string]fs.FS),
		bufpool: bpool.NewBufferPool(64),
		funcmap: make(map[string]interface{}),
		// plugin
		basetemplate: template.New(""),
		assets:       make(map[mapkey]Asset),
		envfuncs:     make(map[mapkey]EnvFunc),
		// fs cache
		cachepage: make(map[string]Component),
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

func (ry *Renderly) Lookup(mainfile string, includefiles ...string) (Component, error) {
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
	page := Component{
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
	HTMLFiles = append([]string{mainfile}, HTMLFiles...)
	// Add user-specified HTML templates to the page template
	for _, filename := range HTMLFiles {
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
	page.html = page.html.Lookup(HTMLFiles[0])
	if page.html == nil {
		return page, fmt.Errorf(`no template found for name "%s"`, HTMLFiles[0])
	}
	// Find the list of dependency templates invoked by the main HTML template
	depedencies, err := listAllDeps(page.html, mainfile)
	if err != nil {
		return page, erro.Wrap(err)
	}
	// For each depedency template, figure out the corresponding set of
	// CSS/JS/Prehooks/Posthooks to include in the page. A map is used keep
	// track of every included CSS/JS asset (identified by their hash) so that
	// we do not include the same asset twice.
	for _, componentName := range depedencies {
		component := ry.components[componentName] // TODO: check _, ok
		page.css = append(page.css, component.css...)
		page.js = append(page.js, component.js...)
		page.htmlenvfuncs = append(page.htmlenvfuncs, component.htmlenvfuncs...)
		page.jsenvfuncs = append(page.jsenvfuncs, component.jsenvfuncs...)
	}
	// Add the user-specified CSS files to the page
	for _, filename := range CSSFiles {
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
	for _, filename := range JSFiles {
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

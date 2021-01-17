package renderly2

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
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

// TODO: I don't fucking know
type Page struct {
	bufpool      *bpool.BufferPool
	html         *template.Template
	htmlinclude  []string
	css          []mapkey
	js           []mapkey
	htmlenvfuncs []mapkey
	jsenvfuncs   []mapkey
	assets       map[mapkey]Asset
	envfuncs     map[mapkey]EnvFunc
	fsys         MuxFS
	jsonifydata  bool
	inlineassets bool
}

type MuxFS struct {
	DefaultFS fs.FS
	AltFS     map[string]fs.FS
	// TODO: if the fs path starts with a space followed by the tilde prefix, the prefix is taken literally to be a part of the path (?)
}

type Renderly struct {
	mu         *sync.RWMutex
	bufpool    *bpool.BufferPool
	fsys       MuxFS
	funcmap    map[string]interface{}
	opts       []string
	fsysprefix string
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

func dedupkeys(keys []mapkey) []mapkey {
	set := make(map[mapkey]struct{})
	n := 0
	for _, key := range keys {
		if _, ok := set[key]; ok {
			continue
		}
		set[key] = struct{}{}
		keys[n] = key
		n++
	}
	keys = keys[:n]
	return keys
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

func appendCSP(w http.ResponseWriter, policy, value string) error {
	const key = "Content-Security-Policy"
	CSP := w.Header().Get(key)
	if CSP == "" {
		// NOTE: if no CSP exists, I may not want to create it in case the user wants CSP off
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

func (page Page) CSS(w io.Writer) template.HTML {
	// Generate Content-Security-Policy script-src
	styles := &strings.Builder{}
	styleHashes := &strings.Builder{}
	for i, key := range page.css {
		if i > 0 {
			styles.WriteString("\n")
			styleHashes.WriteString(" ")
		}
		asset := page.assets[key]
		styles.WriteString("<style>")
		styles.WriteString(asset.Data)
		styles.WriteString("</style>")
		styleHashes.WriteString("'sha256-")
		styleHashes.WriteString(base64.StdEncoding.EncodeToString(asset.hash[0:]))
		styleHashes.WriteString("'")
	}
	if styleHashes.Len() > 0 {
		if w, ok := w.(http.ResponseWriter); ok {
			_ = appendCSP(w, "style-src", "'self'") // NOTE: this may need to be removed
			_ = appendCSP(w, "style-src", styleHashes.String())
		}
	}
	return template.HTML(styles.String())
}

func (page Page) JS(w io.Writer, jsenv, htmlenv map[string]interface{}) (template.HTML, error) {
	// Generate Content-Security-Policy script-src
	scripts := &strings.Builder{}
	scriptHashes := &strings.Builder{}
	b, err := json.Marshal(jsenv)
	if err != nil {
		return "", erro.Wrap(err)
	}
	b = bytes.ReplaceAll(b, []byte(`"`), []byte(`\"`))
	envscript := `const Env = (function () {
  const env = JSON.parse("` + string(b) + `");
  return function (key) {
    return env[key];
  };
})();`
	envhash := sha256.Sum256([]byte(envscript))
	scripts.WriteString("<script>")
	scripts.WriteString(envscript)
	scripts.WriteString("</script>")
	scriptHashes.WriteString(" 'sha256-")
	scriptHashes.WriteString(base64.StdEncoding.EncodeToString(envhash[0:]))
	scriptHashes.WriteString("' ")
	for i, key := range page.js {
		if i > 0 {
			scripts.WriteString("\n")
			scriptHashes.WriteString(" ")
		}
		asset := page.assets[key]
		scripts.WriteString("<script>")
		scripts.WriteString(asset.Data)
		scripts.WriteString("</script>")
		scriptHashes.WriteString(" 'sha256-")
		scriptHashes.WriteString(base64.StdEncoding.EncodeToString(asset.hash[0:]))
		scriptHashes.WriteString("' ")
	}
	if scriptHashes.Len() > 0 {
		if w, ok := w.(http.ResponseWriter); ok {
			_ = appendCSP(w, "script-src", "'self'") // NOTE: this may need to be removed
			_ = appendCSP(w, "script-src", scriptHashes.String())
		}
	}
	htmlenv["ContentSecurityPolicy"] = template.HTML(`<meta http-equiv="Content-Security-Policy" content="` + scriptHashes.String() + `">`)
	return template.HTML(scripts.String()), nil
}

type RenderOption func(*renderConfig)

type renderConfig struct {
	includefiles []string
}

func (ry *Renderly) Page(w http.ResponseWriter, r *http.Request, data interface{}, mainfile string) {
}

func Files() RenderOption {
	return func(config *renderConfig) {
	}
}

func JSEnv(env map[string]interface{}) RenderOption {
	return func(config *renderConfig) {
	}
}

func HTMLEnv(env map[string]interface{}) RenderOption {
	return func(config *renderConfig) {
	}
}

func JSONifyData(jsonify bool) RenderOption {
	return func(config *renderConfig) {
	}
}

func InlineAssets(inline bool) RenderOption {
	return func(config *renderConfig) {
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
		if t := page.html.Lookup(componentName); t != nil {
			continue
		}
		component, ok := ry.components[componentName]
		if !ok {
			return page, erro.Wrap(fmt.Errorf("{{ template %s }} was referenced but does not exist", componentName))
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
	page.css = dedupkeys(page.css)
	page.js = dedupkeys(page.js)
	page.htmlenvfuncs = dedupkeys(page.htmlenvfuncs)
	page.jsenvfuncs = dedupkeys(page.jsenvfuncs)
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
	switch data := data.(type) {
	case map[string]interface{}:
		env, _ := data["Env"].(map[string]interface{})
		if env != nil {
			htmlenv = env
		} else {
			data["Env"] = htmlenv
		}
	default:
		vptr := reflect.ValueOf(data)
		v := reflect.Indirect(vptr)
		if v.Kind() != reflect.Struct {
			break
		}
		vtype := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if vtype.Field(i).Tag.Get("renderly") != "Env" {
				continue
			}
			field := v.Field(i)
			switch env := field.Interface().(type) {
			case map[string]interface{}:
				if !field.CanSet() {
					if vptr.Kind() == v.Kind() {
						return erro.Wrap(fmt.Errorf("unable to set tagged field, please pass in a pointer instead"))
					}
					return erro.Wrap(fmt.Errorf("unable to set field tagged `Env`"))
				}
				if env != nil {
					htmlenv = env
				} else {
					field.Set(reflect.ValueOf(htmlenv)) // NOTE: may panic if type don't match
				}
			default:
				return erro.Wrap(fmt.Errorf("tagged field is not a map[string]interface{}"))
			}
			break
		}
	}
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
	htmlenv["CSS"] = page.CSS(w)
	htmlenv["JS"], err = page.JS(w, jsenv, htmlenv)
	if err != nil {
		return erro.Wrap(err)
	}
	err = executeTemplate(page.html, page.bufpool, w, page.html.Name(), data)
	if err != nil {
		return erro.Wrap(err)
	}
	return nil
}

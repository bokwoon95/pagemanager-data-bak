package renderly3

import (
	"crypto/sha256"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/bokwoon95/erro"
	"github.com/oxtoacart/bpool"
)

type Renderly struct {
	mu      *sync.RWMutex
	bufpool *bpool.BufferPool
	fs      fs.FS
	altfs   map[string]fs.FS
	funcs   map[string]interface{}
	opts    []string
	// plugin
	html     *template.Template
	css      map[string][]*Asset
	js       map[string][]*Asset
	prehooks map[string][]Prehook
	// fs cache
	cacheenabled bool
	cachepage    map[string]Page
	cachehtml    map[string]*template.Template
	cachecss     map[string]*Asset
	cachejs      map[string]*Asset
	//
	errorhandler func(http.ResponseWriter, *http.Request, error)
}

type Asset struct {
	Data        string
	Hash        [32]byte
	External    bool
	PayloadFunc func(*http.Request) (name string, value interface{}, err error)
}

type Prehook func(w io.Writer, r *http.Request, input interface{}) (output interface{}, err error)

func New(fsys fs.FS, opts ...Option) (*Renderly, error) {
	ry := &Renderly{
		mu:      &sync.RWMutex{},
		fs:      fsys,
		altfs:   make(map[string]fs.FS),
		bufpool: bpool.NewBufferPool(64),
		funcs:   make(map[string]interface{}),
		// plugin
		html:     template.New(""),
		css:      make(map[string][]*Asset),
		js:       make(map[string][]*Asset),
		prehooks: make(map[string][]Prehook),
		// fs cache
		cachepage: make(map[string]Page),
		cachehtml: make(map[string]*template.Template),
		cachecss:  make(map[string]*Asset),
		cachejs:   make(map[string]*Asset),
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

func (ry *Renderly) Page(w http.ResponseWriter, r *http.Request, data interface{}, filenames ...string) error {
	page, err := ry.Lookup(filenames...)
	if err != nil {
		return erro.Wrap(err)
	}
	err = page.Render(w, r, data)
	if err != nil {
		return erro.Wrap(err)
	}
	return nil
}

func (ry *Renderly) InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	if ry.errorhandler != nil {
		ry.errorhandler(w, r, err)
		return
	}
	io.WriteString(w, err.Error())
}

type Option func(*Renderly) error

func TemplateFuncs(funcmaps ...map[string]interface{}) Option {
	return func(ry *Renderly) error {
		if ry.funcs == nil {
			ry.funcs = make(map[string]interface{})
		}
		for _, funcmap := range funcmaps {
			for name, fn := range funcmap {
				ry.funcs[name] = fn
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
		for _, name := range filenames {
			b, err := fs.ReadFile(fsys, name)
			if err != nil {
				return erro.Wrap(err)
			}
			ry.css[""] = append(ry.css[""], &Asset{
				Data: string(b),
				Hash: sha256.Sum256(b),
			})
		}
		return nil
	}
}

func GlobalJS(fsys fs.FS, filenames ...string) Option {
	return func(ry *Renderly) error {
		for _, name := range filenames {
			b, err := fs.ReadFile(fsys, name)
			if err != nil {
				return erro.Wrap(err)
			}
			ry.js[""] = append(ry.js[""], &Asset{
				Data: string(b),
				Hash: sha256.Sum256(b),
			})
		}
		return nil
	}
}

func GlobalTemplates(fsys fs.FS, filenames ...string) Option {
	return func(ry *Renderly) error {
		if ry.html == nil {
			ry.html = template.New("")
		}
		for _, name := range filenames {
			b, err := fs.ReadFile(fsys, name)
			if err != nil {
				return erro.Wrap(err)
			}
			t, err := template.New(name).Funcs(ry.funcs).Option(ry.opts...).Parse(string(b))
			if err != nil {
				return erro.Wrap(err)
			}
			err = addParseTree(ry.html, t, t.Name())
			if err != nil {
				return erro.Wrap(err)
			}
		}
		return nil
	}
}

func GlobalPrehooks(fns ...Prehook) Option {
	return func(ry *Renderly) error {
		ry.prehooks[""] = append(ry.prehooks[""], fns...)
		return nil
	}
}

func AltFS(name string, fsys fs.FS) Option {
	return func(ry *Renderly) error {
		ry.altfs[name] = fsys
		return nil
	}
}

func AbsDir(relativePath string) string {
	_, absolutePath, _, _ := runtime.Caller(1)
	return filepath.Join(absolutePath, "..", relativePath) + string(os.PathSeparator)
}

type Plugin struct {
	Err      error
	Names    []string
	HTML     *template.Template
	CSS      []*Asset
	JS       []*Asset
	Prehooks []Prehook
	// global assets
	FuncMap            map[string]interface{}
	GlobalCSS          []*Asset
	GlobalJS           []*Asset
	GlobalPrehooks     []Prehook
	GlobalPayloadFuncs []func(*http.Request) (name string, value interface{}, err error)
}

func Plugins(plugins ...Plugin) Option {
	return func(ry *Renderly) error {
		if ry.html == nil {
			ry.html = template.New("")
		}
		for _, plugin := range plugins {
			if plugin.Err != nil {
				return plugin.Err
			}
			if plugin.HTML != nil {
				err := addParseTree(ry.html, plugin.HTML, plugin.HTML.Name())
				if err != nil {
					return erro.Wrap(err)
				}
			}
			// Compute the hash for each asset
			for _, assets := range [][]*Asset{plugin.CSS, plugin.JS, plugin.GlobalCSS, plugin.GlobalJS} {
				for _, asset := range assets {
					asset.Hash = sha256.Sum256([]byte(asset.Data))
				}
			}
			for _, name := range plugin.Names {
				ry.css[name] = append(ry.css[name], plugin.CSS...)
				ry.js[name] = append(ry.js[name], plugin.JS...)
				ry.prehooks[name] = append(ry.prehooks[name], plugin.Prehooks...)
			}
			for name, fn := range plugin.FuncMap {
				ry.funcs[name] = fn
			}
			ry.css[""] = append(ry.css[""], plugin.GlobalCSS...)
			ry.js[""] = append(ry.js[""], plugin.GlobalJS...)
			ry.prehooks[""] = append(ry.prehooks[""], plugin.GlobalPrehooks...)
		}
		return nil
	}
}

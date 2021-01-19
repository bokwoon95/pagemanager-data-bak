package pagemanager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/bokwoon95/erro"
	"github.com/bokwoon95/pagemanager-data/renderly"
	"github.com/dgraph-io/ristretto"
	"github.com/dop251/goja"
	_ "github.com/mattn/go-sqlite3"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml"
)

var builtin fs.FS

func init() {
	flag.Parse()
	if builtin == nil {
		builtin = os.DirFS(renderly.AbsDir("."))
	}
}

type PageManager struct {
	dbdriver   string
	db         *sql.DB
	routemap   map[string]Route
	routecache *ristretto.Cache // TODO: make this a Cache interface instead
	restart    chan struct{}
	fsys       fs.FS
	fsHandler  http.Handler
	notfound   http.Handler
	render     *renderly.Renderly
	htmlPolicy *bluemonday.Policy

	firsttime     bool
	trailingslash bool
}

func New() (*PageManager, error) {
	pm := &PageManager{}
	pm.firsttime = true
	err := pm.Setup()
	if err != nil {
		return pm, erro.Wrap(err)
	}
	return pm, nil
}

func (pm *PageManager) Setup() error {
	pm.routemap = make(map[string]Route)
	pm.restart = make(chan struct{}, 1)
	pm.notfound = http.NotFoundHandler()
	datafolder, err := LocateDataFolder()
	if err != nil {
		return erro.Wrap(err)
	}
	if datafolder == "" {
		return fmt.Errorf("couldn't locate PageManager datafolder")
	}
	// fsys
	pm.fsys = os.DirFS(datafolder)
	pm.fsHandler = http.FileServer(http.FS(pm.fsys))
	// db
	pm.dbdriver = "sqlite3"
	pm.db, err = sql.Open(pm.dbdriver, datafolder+string(os.PathSeparator)+"database.sqlite3")
	if err != nil {
		return erro.Wrap(err)
	}
	err = pm.db.Ping()
	if err != nil {
		return erro.Wrap(err)
	}
	_, err = pm.db.Exec("PRAGMA journal_mode = WAL")
	if err != nil {
		return erro.Wrap(err)
	}
	_, err = pm.db.Exec("PRAGMA synchronous = normal")
	if err != nil {
		return erro.Wrap(err)
	}
	_, err = pm.db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return erro.Wrap(err)
	}
	err = ensuretables(pm.dbdriver, pm.db)
	if err != nil {
		return erro.Wrap(err)
	}
	// cache
	pm.routecache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
		Metrics:     true,
	})
	if err != nil {
		return erro.Wrap(err)
	}
	// render
	pm.render, err = renderly.New(
		pm.fsys,
		renderly.AltFS("builtin", builtin),
		renderly.TemplateFuncs(pm.FuncMap()),
		renderly.GlobalCSS(nil, "builtin::tachyons.min.css"),
		renderly.GlobalHTMLEnvFuncs(EnvFunc),
		renderly.GlobalJSEnvFuncs(EnvFunc),
	)
	if err != nil {
		return erro.Wrap(err)
	}
	// bluemonday
	pm.htmlPolicy = bluemonday.UGCPolicy()
	pm.htmlPolicy.AllowStyling()
	return nil
}

func (pm *PageManager) newmux(defaultHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", defaultHandler)
	mux.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		select {
		case pm.restart <- struct{}{}:
		default:
		}
	})
	return mux
}

type Route struct {
	URL         sql.NullString
	Disabled    sql.NullBool
	RedirectURL sql.NullString
	HandlerURL  sql.NullString
	Content     sql.NullString
	Template    sql.NullString
}

func (pm *PageManager) getroute(path string) (Route, error) {
	negapath := path
	if strings.HasSuffix(negapath, "/") {
		negapath = strings.TrimRight(negapath, "/")
	} else {
		negapath = negapath + "/"
	}
	value, found := pm.routecache.Get(path)
	route, ok := value.(Route)
	if found && ok {
		return route, nil
	}
	route, ok = pm.routemap[path]
	if ok {
		return route, nil
	}
	route, ok = pm.routemap[negapath]
	if ok {
		return route, nil
	}
	query := `SELECT url, disabled, redirect_url, handler_url, content, template
		FROM pm_routes WHERE url IN (?, ?)
		ORDER BY CASE url WHEN ? THEN 1 ELSE 2 END
		LIMIT 1`
	err := pm.db.
		QueryRow(query, path, negapath, path).
		Scan(&route.URL, &route.Disabled, &route.RedirectURL, &route.HandlerURL, &route.Content, &route.Template)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return route, erro.Wrap(err)
	}
	// _ = pm.routecache.Set(r.URL.Path, route, 0)
	return route, nil
}

func (pm *PageManager) Middleware(next http.Handler) http.Handler {
	mux := pm.render.FileServerMiddleware()(pm.newmux(next))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, err := pm.getroute(r.URL.Path)
		if err != nil {
			http.Error(w, erro.Sdump(err), http.StatusInternalServerError)
			return
		}
		if route.Disabled.Valid && route.Disabled.Bool {
			pm.notfound.ServeHTTP(w, r)
			return
		}
		if route.RedirectURL.Valid {
			http.Redirect(w, r, route.RedirectURL.String, http.StatusMovedPermanently)
			return
		}
		if route.HandlerURL.Valid {
			r2 := &http.Request{}
			*r2 = *r
			r2.URL = &url.URL{}
			r2.URL.Path = route.HandlerURL.String
			mux.ServeHTTP(w, r2)
			return
		}
		if route.Content.Valid {
			io.WriteString(w, route.Content.String)
			return
		}
		var editTemplate bool
		if !route.Template.Valid {
			path := r.URL.Path
			if strings.HasSuffix(path, "edit/") {
				path = strings.TrimSuffix(path, "edit/")
			} else if strings.HasSuffix(path, "edit") {
				path = strings.TrimSuffix(path, "edit")
			}
			route, err = pm.getroute(path)
			if err != nil {
				http.Error(w, erro.Sdump(err), http.StatusInternalServerError)
				return
			}
			editTemplate = route.Template.Valid
		}
		if route.Template.Valid {
			metadata, err := GetTemplateMetadata(pm.fsys, route.Template.String) // TODO: cache the metadata
			if err != nil {
				http.Error(w, erro.Sdump(err), http.StatusInternalServerError)
				return
			}
			for policy, values := range metadata.CSP {
				allvalues := strings.Join(values, " ")
				_ = renderly.AppendCSP(w, policy, allvalues)
			}
			var mainfile = metadata.Name
			var includefiles []string
			if metadata.MainTemplate != "" {
				mainfile = metadata.MainTemplate
				includefiles = append(includefiles, metadata.Name)
			}
			includefiles = append(includefiles, metadata.Include...)
			// files = append(files, metadata.Name)
			// files = append(files, metadata.Include...)
			err = r.ParseForm()
			if err != nil {
				http.Error(w, erro.Sdump(err), http.StatusInternalServerError)
				return
			}
			if editTemplate {
				includefiles = append(includefiles, "builtin::editor.js", "builtin::editor.css")
			}
			err = pm.render.Page(w, r, mainfile, includefiles, nil)
			if err != nil {
				http.Error(w, erro.Sdump(err), http.StatusInternalServerError)
				return
			}
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (pm *PageManager) ListenAndServe(addr string, handler http.Handler) error {
	for {
		if !pm.firsttime {
			fmt.Println("restarted")
			err := pm.Setup()
			if err != nil {
				return erro.Wrap(err)
			}
		} else {
			pm.firsttime = false
		}
		srv := http.Server{
			Addr:    addr,
			Handler: handler,
		}
		go func() {
			<-pm.restart
			if err := srv.Shutdown(context.Background()); err != nil {
				log.Printf("srv.Shutdown error: %v\n", err)
			}
		}()
		fmt.Println("Listening on " + addr)
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			continue
		}
		return erro.Wrap(err)
	}
}

type TemplateMetadata struct {
	Name         string
	MainTemplate string                 `json,toml,mapstructure:"main_template"`
	Include      []string               `json,toml,mapstructure:"include"`
	CSP          map[string][]string    `json,toml,mapstructure:"content_security_policy"`
	Args         map[string]interface{} `json,toml,mapstructure:"args"`
}

func GetTemplateMetadata(fsys fs.FS, filename string) (TemplateMetadata, error) {
	const tomlfile = "templates-config.toml"
	const jsfile = "templates-config.js"
	// var configPath string
	metadata := TemplateMetadata{
		Name: filename,
	}
	currentPath := filename
	var b []byte
	var err error
	var parentPath string
	var path string
	for {
		parentPath = filepath.Dir(currentPath)
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
		// try js
		path = currentPath + string(os.PathSeparator) + jsfile
		if currentPath == "." {
			path = jsfile
		}
		b, err = fs.ReadFile(fsys, path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return metadata, erro.Wrap(err)
		}
		if err == nil {
			vm := goja.New()
			vm.Set("log", func(f goja.FunctionCall) goja.Value {
				a := make([]interface{}, len(f.Arguments))
				for i := range f.Arguments {
					a[i] = f.Argument(i).Export()
				}
				fmt.Println(a...)
				return goja.Undefined()
			})
			res, err := vm.RunString("(function(){" + string(b) + "})()")
			if err != nil {
				return metadata, erro.Wrap(err)
			}
			if res == nil {
				return metadata, nil
			}
			m, ok := res.Export().(map[string]interface{})
			if !ok {
				return metadata, nil
			}
			err = mapstructure.Decode(m[filename], &metadata)
			if err != nil {
				return metadata, erro.Wrap(err)
			}
			return metadata, nil
		}
		// try toml
		path = currentPath + string(os.PathSeparator) + tomlfile
		if currentPath == "." {
			path = tomlfile
		}
		b, err = fs.ReadFile(fsys, path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return metadata, erro.Wrap(err)
		}
		if err == nil {
			mainTree, err := toml.LoadBytes(b)
			if err != nil {
				return metadata, erro.Wrap(err)
			}
			subTree, _ := mainTree.GetPath([]string{filename}).(*toml.Tree)
			if subTree != nil {
				err = subTree.Unmarshal(&metadata)
				if err != nil {
					return metadata, erro.Wrap(err)
				}
				return metadata, nil
			}
		}
	}
	return metadata, nil
}

// aliasing to dynamic URLs is not supported. If a plugin wishes to make a URL aliasable, it has to make the route static i.e. no :colon prefix, or {curly braces}/<angle brackets> delimiters.
type Plugin interface {
	HTTPHandler() (defaultPrefix string, handler http.Handler)
	URLs() []string
}

var datafolder = flag.String("pm-datafolder", "", "")

func LocateDataFolder() (string, error) {
	const datafoldername = "pagemanager-data"
	var dirnames []string
	cwd, err := os.Getwd()
	if err != nil {
		return "", erro.Wrap(err)
	}
	if *datafolder != "" {
		if strings.HasPrefix(*datafolder, ".") {
			return cwd + (*datafolder)[1:], nil
		}
		return *datafolder, nil
	}
	dirnames = append(dirnames, cwd, filepath.Dir(cwd))
	userhome, err := os.UserHomeDir()
	if err != nil {
		return "", erro.Wrap(err)
	}
	dirnames = append(dirnames, userhome)
	exePath, err := os.Executable()
	if err != nil {
		return "", erro.Wrap(err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", erro.Wrap(err)
	}
	exeDir := filepath.Dir(exePath)
	dirnames = append(dirnames, filepath.Dir(exeDir), exeDir)
	var dir *os.File
	var names []string
	var found string
	for _, dirname := range dirnames {
		if filepath.Base(dirname) == datafoldername {
			return dirname, nil
		}
		found, err = func() (string, error) {
			dir, err = os.Open(dirname)
			if err != nil {
				return "", erro.Wrap(err)
			}
			defer dir.Close()
			for {
				names, err = dir.Readdirnames(1)
				if err != nil {
					if errors.Is(err, io.EOF) {
						return "", nil
					}
					return "", erro.Wrap(err)
				}
				if names[0] == datafoldername {
					return dirname + string(os.PathSeparator) + datafoldername, nil
				}
			}
		}()
		if err != nil {
			return "", erro.Wrap(err)
		}
		if found != "" {
			return found, nil
		}
	}
	// TODO: else create datafolder in user's homedir
	return "", nil
}

type table struct {
	name        string
	columns     []column
	constraints []string
}

func (t table) ddl() string {
	buf := &strings.Builder{}
	buf.WriteString("CREATE TABLE ")
	buf.WriteString(t.name)
	buf.WriteString(" (")
	for i, c := range t.columns {
		buf.WriteString("\n    ")
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(c.name)
		buf.WriteString(" ")
		buf.WriteString(c.typ)
		if len(c.constraints) > 0 {
			buf.WriteString(" ")
			buf.WriteString(strings.Join(c.constraints, " "))
		}
	}
	if len(t.constraints) > 0 {
		buf.WriteString("\n    ,")
		buf.WriteString(strings.Join(t.constraints, "\n    ,"))
	}
	buf.WriteString("\n)")
	return buf.String()
}

type column struct {
	name        string
	typ         string
	constraints []string
}

var tables = []table{
	{
		name: "pm_routes",
		columns: []column{
			{name: "url", typ: "TEXT", constraints: []string{"NOT NULL", "PRIMARY KEY"}},
			{name: "disabled", typ: "BOOLEAN"},
			{name: "redirect_url", typ: "TEXT"},
			{name: "handler_url", typ: "TEXT"},
			{name: "content", typ: "TEXT"},
			{name: "template", typ: "TEXT"},
		},
	},
	{
		name: "pm_templatedata",
		columns: []column{
			{name: "id", typ: "TEXT", constraints: []string{"NOT NULL", "PRIMARY KEY"}},
			{name: "data", typ: "JSON"},
		},
	},
}

func ensuretables(driver string, db *sql.DB) error {
	var err error
	for _, table := range tables {
		// does table exist?
		var exists sql.NullBool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE name = ?)", table.name).Scan(&exists)
		// if not exists, create table from scratch and continue
		if !exists.Valid || !exists.Bool {
			_, err = db.Exec(table.ddl())
			if err != nil {
				return erro.Wrap(err)
			}
			continue
		}
		// do columns exist?
		columnset := make(map[string]struct{})
		rows, err := db.Query("SELECT name FROM pragma_table_info(?)", table.name)
		if err != nil {
			return erro.Wrap(err)
		}
		defer rows.Close()
		var name sql.NullString
		for rows.Next() {
			err = rows.Scan(&name)
			if err != nil {
				return erro.Wrap(err)
			}
			if name.Valid {
				columnset[name.String] = struct{}{}
			}
		}
		for _, column := range table.columns {
			if _, ok := columnset[column.name]; ok {
				continue
			}
			query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table.name, column.name, column.typ)
			if len(column.constraints) > 0 {
				query = query + strings.Join(column.constraints, " ")
			}
			_, err = db.Exec(query)
			if err != nil {
				return erro.Wrap(err)
			}
		}
	}
	return nil
}

func EnvFunc(w io.Writer, r *http.Request, env map[string]interface{}) error {
	env["PageID"] = r.URL.Path
	env["EditMode"] = false
	return nil
}

func (pm *PageManager) FuncMap() map[string]interface{} {
	funcmap := map[string]interface{}{
		"safeHTML":       func(s string) template.HTML { return template.HTML(s) },
		"safeJS":         func(s string) template.JS { return template.JS(s) },
		"getValue":       pm.getValue,
		"getValueWithID": pm.getValueWithID,
		"getRows":        pm.getRows,
		"getRowsWithID":  pm.getRowsWithID,
	}
	return funcmap
}

func (pm *PageManager) getValue(env map[string]interface{}, key string) (interface{}, error) {
	id, ok := env["PageID"].(string)
	if !ok {
		return nil, nil
	}
	return pm.getValueWithID(env, key, id)
}

func (pm *PageManager) getValueWithID(env map[string]interface{}, key, id string) (interface{}, error) {
	var value sql.NullString
	query := "SELECT json_extract(data, ?) FROM pm_templatedata WHERE id = ?"
	err := pm.db.QueryRow(query, "$."+key, id).Scan(&value)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if value.Valid {
		return template.HTML(value.String), nil
	}
	return nil, nil
}

func (pm *PageManager) getRows(env map[string]interface{}, key string) ([]interface{}, error) {
	id, ok := env["PageID"].(string)
	if !ok {
		return nil, nil
	}
	return pm.getRowsWithID(env, key, id)
}

func (pm *PageManager) getRowsWithID(env map[string]interface{}, key, id string) ([]interface{}, error) {
	var s sql.NullString
	query := "SELECT json_extract(data, ?) FROM pm_templatedata WHERE id = ?"
	err := pm.db.QueryRow(query, "$."+key, id).Scan(&s)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var array []interface{}
	if s.Valid {
		err = json.Unmarshal([]byte(s.String), &array)
		if err != nil {
			return array, err
		}
		return array, nil
	}
	return nil, nil
}

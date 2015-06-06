package main

import (
	"fmt"
	"net/http"

	"github.com/naoina/denco"
	"github.com/pocke/gfm-viewer/env"
	"github.com/pocke/hlog.go"
	"github.com/yosssi/ace"
)

type Server struct {
	storage *Storage
}

func NewServer() *Server {
	s := &Server{
		storage: NewStorage(),
	}

	go func() {
		wsm := NewWSManager(s.storage.OnUpdate())

		mux := denco.NewMux()
		f, err := mux.Build([]denco.Handler{
			mux.GET("/", s.indexHandler),
			mux.POST("/auth", s.authHandler),
			mux.GET("/files/*path", s.ServeFile),
			mux.GET("/css/github-markdown.css", s.serveCSS),
			mux.GET("/js/main.js", s.serveJS),
			mux.GET("/ws/*path", wsm.ServeWS),
		})
		if err != nil {
			panic(err)
		}
		handler := f.ServeHTTP
		if env.DEBUG {
			handler = hlog.Wrap(f.ServeHTTP)
		}
		http.HandleFunc("/", handler)
		// TODO: port
		http.ListenAndServe(":1124", nil)
	}()

	return s
}

func (s *Server) ServeFile(w http.ResponseWriter, r *http.Request, p denco.Params) {
	path := p.Get("path")
	html, ok := s.storage.Get(path)
	if !ok {
		http.Error(w, fmt.Sprintf("%s page not found", path), http.StatusNotFound)
		return
	}
	w.Write([]byte(html))
}

func (s *Server) authHandler(w http.ResponseWriter, r *http.Request, _ denco.Params) {
	r.ParseForm()
	v := r.PostForm
	user := v.Get("username")
	pass := v.Get("password")

	err := s.storage.token.Init(user, pass)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.storage.AddAll()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request, _ denco.Params) {
	if s.storage.token.hasToken() {
		loadAce(w, "index", s.storage.Index())
	} else {
		loadAce(w, "before_auth", nil)
	}
}

func loadAce(w http.ResponseWriter, action string, data interface{}) {
	tpl, err := ace.Load("assets/"+action, "", &ace.Options{
		DynamicReload: env.DEBUG,
		Asset:         Asset,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) serveCSS(w http.ResponseWriter, r *http.Request, _ denco.Params) {
	file, err := Asset("assets/github-markdown.css")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/css")
	w.Write(file)
}

func (s *Server) serveJS(w http.ResponseWriter, r *http.Request, _ denco.Params) {
	file, err := Asset("assets/main.js")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(file)
}

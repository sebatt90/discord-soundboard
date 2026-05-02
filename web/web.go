package web

import (
	"net/http"
	"html/template"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebatt90/discord-soundboard/db"
)

func Start(sc chan os.Signal) {
	exe, _ := os.Executable()
	viewsDir := filepath.Join(filepath.Dir(exe), "web/views")
	staticDir := filepath.Join(filepath.Dir(exe), "web/static")
	
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		tmpl, err := template.ParseFiles(
			filepath.Join(viewsDir, "index.html"),
			filepath.Join(viewsDir, "partial/navbar.html"),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		guilds, err := db.GetGuilds()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, guilds) 
	})

	mux.HandleFunc("GET /guild/{id}", func(w http.ResponseWriter, req *http.Request) {
		tmpl, err := template.ParseFiles(
			filepath.Join(viewsDir, "guild.html"),
			filepath.Join(viewsDir, "partial/navbar.html"),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tracks, err := db.GetTracksPerGuild(req.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, tracks)
	})

	err := http.ListenAndServe(":8080",mux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] web server error (%s)\n",err)
		sc <- os.Interrupt
	}
}

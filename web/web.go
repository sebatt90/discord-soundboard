package web

import (
	"net/http"
	"html/template"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"io"
	
	"github.com/h2non/filetype"
	"github.com/sebatt90/discord-soundboard/db"
	"github.com/sebatt90/discord-soundboard/models"	
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
			filepath.Join(viewsDir, "partial/delete_modal.html"),
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

		guild, err := db.GetGuildByID(req.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, struct {
			Guild models.Guild
			Tracks []models.Track
		}{
			Guild: guild,
			Tracks: tracks,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return			
		}
	})

	mux.HandleFunc("GET /guild/{id}/new", func(w http.ResponseWriter, req *http.Request) {
		tmpl, err := template.ParseFiles(
			filepath.Join(viewsDir, "new.html"),
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

		guild, err := db.GetGuildByID(req.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, struct {
			Guild models.Guild
			Tracks []models.Track
		}{
			Guild: guild,
			Tracks: tracks,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return			
		}
	})

	// Delete track
	mux.HandleFunc("DELETE /guild/{guildid}/track/{trackid}", func (w http.ResponseWriter, req *http.Request) {
		guildID := req.PathValue("guildid")
		trackID, err := strconv.Atoi(req.PathValue("trackid"))
		if err != nil {
			http.Error(w, "invalid track id", http.StatusBadRequest)
			return
		}

		err = db.DeleteTrack(guildID,trackID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		fmt.Printf("[INFO] delete track %d from guild %s\n", trackID, guildID)
		w.WriteHeader(http.StatusOK)
	})

	// Insert track
	mux.HandleFunc("POST /guild/{id}/track", func(w http.ResponseWriter, req *http.Request) {
		req.ParseMultipartForm(32 << 20)

		name := req.FormValue("name")
		if name == "" {
			http.Error(w, "missing name", http.StatusBadRequest)
			return
		}

		file, _, err := req.FormFile("data")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// to read mime type
		data, err := io.ReadAll(file)
		if err != nil || !filetype.IsAudio(data) {
			http.Error(w, "file must be an audio file", http.StatusBadRequest)
			return
		}
		
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := db.InsertTrack(req.PathValue("id"), name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	err := http.ListenAndServe(":8080",mux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] web server error (%s)\n",err)
		sc <- os.Interrupt
	}
}

package api

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/lerko/helm/internal/api/handlers"
	"github.com/lerko/helm/internal/api/middleware"
	"github.com/lerko/helm/internal/broker"
	"github.com/lerko/helm/internal/config"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NewRouter(cfg *config.Config, db *sql.DB, uiFS fs.FS, b *broker.Broker) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Public: auth + shared memo links + layout config
	r.With(middleware.LoginRateLimit).Post("/api/auth/login", handlers.Login(cfg))
	r.Get("/s/{token}", handlers.GetSharedMemo(db))
	r.Get("/api/config/pages", configPagesHandler(cfg))

	// SSE — auth handled inline via ?token= (EventSource can't set headers)
	r.Get("/api/events", handlers.SSEEvents(b, cfg.Auth.Secret))

	// Protected API
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.Auth.Secret))

		// Tags
		r.Get("/api/tags", handlers.ListTags(db))
		r.Post("/api/tags", handlers.CreateTag(db))
		r.Delete("/api/tags/{id}", handlers.DeleteTag(db))

		// Tag <-> entity attachment/detachment
		r.Post("/api/tags/{id}/attach", handlers.AttachTagHandler(db))
		r.Delete("/api/tags/{id}/detach", handlers.DetachTagHandler(db))

		// Reminders
		r.Get("/api/reminders", handlers.ListReminders(db))
		r.Post("/api/reminders", handlers.CreateReminder(db))
		r.Delete("/api/reminders/{id}", handlers.DeleteReminder(db))

		// Notes
		r.Get("/api/note-folders", handlers.ListNoteFolders(db))
		r.Post("/api/note-folders", handlers.CreateNoteFolder(db))
		r.Delete("/api/note-folders/{id}", handlers.DeleteNoteFolder(db))

		r.Get("/api/notes", handlers.ListNotes(db))
		r.Post("/api/notes", handlers.CreateNote(db, b))
		r.Get("/api/notes/{id}", handlers.GetNote(db))
		r.Put("/api/notes/{id}", handlers.UpdateNote(db, b))
		r.Delete("/api/notes/{id}", handlers.DeleteNote(db, b))

		// Todos
		r.Get("/api/todo-lists", handlers.ListTodoLists(db))
		r.Post("/api/todo-lists", handlers.CreateTodoList(db))
		r.Delete("/api/todo-lists/{id}", handlers.DeleteTodoList(db))

		r.Get("/api/todos", handlers.ListTodos(db))
		r.Post("/api/todos", handlers.CreateTodo(db, b))
		r.Get("/api/todos/{id}", handlers.GetTodo(db))
		r.Put("/api/todos/{id}", handlers.UpdateTodo(db, b))
		r.Delete("/api/todos/{id}", handlers.DeleteTodo(db, b))
		r.Post("/api/todos/{id}/recurrences", handlers.CreateTodoRecurrence(db))
		r.Delete("/api/todos/{id}/recurrences", handlers.DeleteTodoRecurrence(db))

		// Calendar
		r.Get("/api/calendar/sources", handlers.ListCalendarSources(db))
		r.Post("/api/calendar/sources", handlers.CreateCalendarSource(db, cfg.Auth.Secret))
		r.Delete("/api/calendar/sources/{id}", handlers.DeleteCalendarSource(db))
		r.Post("/api/calendar/sources/{id}/sync", handlers.SyncCalendarSource(db, cfg.Auth.Secret, b))

		r.Get("/api/calendar/events", handlers.ListCalendarEvents(db))
		r.Post("/api/calendar/events", handlers.CreateCalendarEvent(db))
		r.Put("/api/calendar/events/{id}", handlers.UpdateCalendarEvent(db))
		r.Delete("/api/calendar/events/{id}", handlers.DeleteCalendarEvent(db))

		// Clipboard
		r.Get("/api/clipboard", handlers.ListClipboardItems(db))
		r.Post("/api/clipboard", handlers.CreateClipboardItem(db, b))
		r.Get("/api/clipboard/{id}", handlers.GetClipboardItem(db))
		r.Put("/api/clipboard/{id}", handlers.UpdateClipboardItem(db, b))
		r.Delete("/api/clipboard/{id}", handlers.DeleteClipboardItem(db, b))

		// Bookmarks
		r.Get("/api/bookmark-collections", handlers.ListBookmarkCollections(db))
		r.Post("/api/bookmark-collections", handlers.CreateBookmarkCollection(db))
		r.Delete("/api/bookmark-collections/{id}", handlers.DeleteBookmarkCollection(db))

		r.Get("/api/bookmarks", handlers.ListBookmarks(db))
		r.Post("/api/bookmarks", handlers.CreateBookmark(db, b))
		r.Put("/api/bookmarks/{id}", handlers.UpdateBookmark(db, b))
		r.Delete("/api/bookmarks/{id}", handlers.DeleteBookmark(db, b))

		// Memos
		r.Get("/api/memos", handlers.ListMemos(db))
		r.Post("/api/memos", handlers.CreateMemo(db, b))
		r.Put("/api/memos/{id}", handlers.UpdateMemo(db, b))
		r.Delete("/api/memos/{id}", handlers.DeleteMemo(db, b))

		// Attachments
		r.Post("/api/attachments", handlers.UploadAttachment(db, cfg))
		r.Get("/api/attachments", handlers.ListAttachments(db))
		r.Get("/api/attachments/{id}/download", handlers.DownloadAttachment(db))
		r.Delete("/api/attachments/{id}", handlers.DeleteAttachment(db))
	})

	// SPA catch-all: serve static files, fall back to index.html
	if uiFS != nil {
		fileServer := http.FileServer(http.FS(uiFS))
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			// If the file exists, serve it directly; otherwise serve index.html.
			if _, err := fs.Stat(uiFS, req.URL.Path[1:]); err == nil {
				fileServer.ServeHTTP(w, req)
				return
			}
			// Rewrite to index.html for SPA routing.
			req.URL.Path = "/"
			fileServer.ServeHTTP(w, req)
		})
	}

	return r
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	return strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

type apiWidget struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Title  string         `json:"title"`
	Config map[string]any `json:"config,omitempty"`
}

type apiColumn struct {
	ID      string      `json:"id"`
	Size    string      `json:"size"`
	Widgets []apiWidget `json:"widgets"`
}

type apiPage struct {
	ID      string      `json:"id"`
	Label   string      `json:"label"`
	Slug    string      `json:"slug"`
	Columns []apiColumn `json:"columns"`
}

func configPagesHandler(cfg *config.Config) http.HandlerFunc {
	pages := make([]apiPage, len(cfg.Pages))
	for i, p := range cfg.Pages {
		pageSlug := "/" + slugify(p.Name)
		if i == 0 {
			pageSlug = "/"
		}
		pageID := slugify(p.Name)
		cols := make([]apiColumn, len(p.Columns))
		for j, c := range p.Columns {
			widgets := make([]apiWidget, len(c.Widgets))
			for k, w := range c.Widgets {
				widgets[k] = apiWidget{
					ID:     pageID + "-col-" + strconv.Itoa(j) + "-" + w.Type + "-" + strconv.Itoa(k),
					Type:   w.Type,
					Title:  cases.Title(language.Und).String(strings.ReplaceAll(w.Type, "-", " ")),
					Config: w.Config,
				}
			}
			cols[j] = apiColumn{
				ID:      pageID + "-col-" + strconv.Itoa(j),
				Size:    c.Size,
				Widgets: widgets,
			}
		}
		pages[i] = apiPage{
			ID:      pageID,
			Label:   p.Name,
			Slug:    pageSlug,
			Columns: cols,
		}
	}

	data, err := json.Marshal(pages)
	if err != nil {
		log.Fatalf("config: failed to marshal pages: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

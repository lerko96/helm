package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/lerko/helm/internal/api/handlers"
	"github.com/lerko/helm/internal/api/middleware"
	"github.com/lerko/helm/internal/config"
)

func NewRouter(cfg *config.Config, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Public: auth + shared memo links
	r.Post("/api/auth/login", handlers.Login(cfg))
	r.Get("/s/{token}", handlers.GetSharedMemo(db))

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
		r.Post("/api/notes", handlers.CreateNote(db))
		r.Get("/api/notes/{id}", handlers.GetNote(db))
		r.Put("/api/notes/{id}", handlers.UpdateNote(db))
		r.Delete("/api/notes/{id}", handlers.DeleteNote(db))

		// Todos
		r.Get("/api/todo-lists", handlers.ListTodoLists(db))
		r.Post("/api/todo-lists", handlers.CreateTodoList(db))
		r.Delete("/api/todo-lists/{id}", handlers.DeleteTodoList(db))

		r.Get("/api/todos", handlers.ListTodos(db))
		r.Post("/api/todos", handlers.CreateTodo(db))
		r.Get("/api/todos/{id}", handlers.GetTodo(db))
		r.Put("/api/todos/{id}", handlers.UpdateTodo(db))
		r.Delete("/api/todos/{id}", handlers.DeleteTodo(db))

		// Calendar
		r.Get("/api/calendar/sources", handlers.ListCalendarSources(db))
		r.Post("/api/calendar/sources", handlers.CreateCalendarSource(db))
		r.Delete("/api/calendar/sources/{id}", handlers.DeleteCalendarSource(db))
		r.Post("/api/calendar/sources/{id}/sync", handlers.SyncCalendarSource(db))

		r.Get("/api/calendar/events", handlers.ListCalendarEvents(db))
		r.Post("/api/calendar/events", handlers.CreateCalendarEvent(db))
		r.Put("/api/calendar/events/{id}", handlers.UpdateCalendarEvent(db))
		r.Delete("/api/calendar/events/{id}", handlers.DeleteCalendarEvent(db))

		// Clipboard
		r.Get("/api/clipboard", handlers.ListClipboardItems(db))
		r.Post("/api/clipboard", handlers.CreateClipboardItem(db))
		r.Get("/api/clipboard/{id}", handlers.GetClipboardItem(db))
		r.Put("/api/clipboard/{id}", handlers.UpdateClipboardItem(db))
		r.Delete("/api/clipboard/{id}", handlers.DeleteClipboardItem(db))

		// Bookmarks
		r.Get("/api/bookmark-collections", handlers.ListBookmarkCollections(db))
		r.Post("/api/bookmark-collections", handlers.CreateBookmarkCollection(db))
		r.Delete("/api/bookmark-collections/{id}", handlers.DeleteBookmarkCollection(db))

		r.Get("/api/bookmarks", handlers.ListBookmarks(db))
		r.Post("/api/bookmarks", handlers.CreateBookmark(db))
		r.Put("/api/bookmarks/{id}", handlers.UpdateBookmark(db))
		r.Delete("/api/bookmarks/{id}", handlers.DeleteBookmark(db))

		// Memos
		r.Get("/api/memos", handlers.ListMemos(db))
		r.Post("/api/memos", handlers.CreateMemo(db))
		r.Put("/api/memos/{id}", handlers.UpdateMemo(db))
		r.Delete("/api/memos/{id}", handlers.DeleteMemo(db))
	})

	return r
}

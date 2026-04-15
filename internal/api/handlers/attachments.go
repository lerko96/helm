package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lerko/helm/internal/config"
)

type Attachment struct {
	ID           int64     `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
}

const maxUploadSize = 20 << 20 // 20 MB

func UploadAttachment(db *sql.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			respondError(w, http.StatusBadRequest, "file too large or invalid form")
			return
		}

		entityType := r.FormValue("entity_type")
		entityID := r.FormValue("entity_id")
		if entityType == "" || entityID == "" {
			respondError(w, http.StatusBadRequest, "entity_type and entity_id required")
			return
		}
		if !validEntityType(entityType) {
			respondError(w, http.StatusBadRequest, "invalid entity_type")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			respondError(w, http.StatusBadRequest, "file field required")
			return
		}
		defer file.Close()

		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to generate filename")
			return
		}
		ext := filepath.Ext(header.Filename)
		filename := hex.EncodeToString(b[:]) + ext

		if err := os.MkdirAll(cfg.Storage.AttachmentsPath, 0755); err != nil {
			respondError(w, http.StatusInternalServerError, "storage directory unavailable")
			return
		}

		diskPath := filepath.Join(cfg.Storage.AttachmentsPath, filename)
		dst, err := os.Create(diskPath)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to create file")
			return
		}
		defer dst.Close()

		size, err := io.Copy(dst, file)
		if err != nil {
			os.Remove(diskPath) //nolint:errcheck
			respondError(w, http.StatusInternalServerError, "failed to write file")
			return
		}

		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		tx, err := db.BeginTx(r.Context(), nil)
		if err != nil {
			os.Remove(diskPath) //nolint:errcheck
			respondError(w, http.StatusInternalServerError, "db transaction failed")
			return
		}
		defer tx.Rollback() //nolint:errcheck

		res, err := tx.ExecContext(r.Context(),
			`INSERT INTO attachments (user_id, filename, original_name, mime_type, size, disk_path) VALUES (?, ?, ?, ?, ?, ?)`,
			defaultUserID, filename, header.Filename, mimeType, size, diskPath,
		)
		if err != nil {
			os.Remove(diskPath) //nolint:errcheck
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()

		if _, err := tx.ExecContext(r.Context(),
			`INSERT INTO entity_attachments (attachment_id, entity_type, entity_id) VALUES (?, ?, ?)`,
			id, entityType, entityID,
		); err != nil {
			os.Remove(diskPath) //nolint:errcheck
			respondError(w, http.StatusInternalServerError, "link failed")
			return
		}

		if err := tx.Commit(); err != nil {
			os.Remove(diskPath) //nolint:errcheck
			respondError(w, http.StatusInternalServerError, "commit failed")
			return
		}

		respond(w, http.StatusCreated, Attachment{
			ID:           id,
			Filename:     filename,
			OriginalName: header.Filename,
			MimeType:     mimeType,
			Size:         size,
		})
	}
}

func ListAttachments(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityType := r.URL.Query().Get("entity_type")
		entityID := r.URL.Query().Get("entity_id")
		if entityType == "" || entityID == "" {
			respondError(w, http.StatusBadRequest, "entity_type and entity_id required")
			return
		}
		if !validEntityType(entityType) {
			respondError(w, http.StatusBadRequest, "invalid entity_type")
			return
		}

		rows, err := db.QueryContext(r.Context(), `
			SELECT a.id, a.filename, a.original_name, a.mime_type, a.size, a.created_at
			FROM attachments a
			JOIN entity_attachments ea ON ea.attachment_id = a.id
			WHERE ea.entity_type = ? AND ea.entity_id = ? AND a.user_id = ?
			ORDER BY a.created_at ASC`,
			entityType, entityID, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		attachments := []Attachment{}
		for rows.Next() {
			var a Attachment
			if err := rows.Scan(&a.ID, &a.Filename, &a.OriginalName, &a.MimeType, &a.Size, &a.CreatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			attachments = append(attachments, a)
		}
		respond(w, http.StatusOK, attachments)
	}
}

func DownloadAttachment(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var a Attachment
		var diskPath string
		err = db.QueryRowContext(r.Context(),
			`SELECT id, filename, original_name, mime_type, size, disk_path FROM attachments WHERE id = ? AND user_id = ?`,
			id, defaultUserID,
		).Scan(&a.ID, &a.Filename, &a.OriginalName, &a.MimeType, &a.Size, &diskPath)
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "attachment not found")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}

		f, err := os.Open(diskPath)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "file not found on disk")
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", a.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.OriginalName))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", a.Size))
		io.Copy(w, f) //nolint:errcheck
	}
}

func DeleteAttachment(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var diskPath string
		err = db.QueryRowContext(r.Context(),
			`SELECT disk_path FROM attachments WHERE id = ? AND user_id = ?`,
			id, defaultUserID,
		).Scan(&diskPath)
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "attachment not found")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}

		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM attachments WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}

		os.Remove(diskPath) //nolint:errcheck
		respond(w, http.StatusNoContent, nil)
	}
}

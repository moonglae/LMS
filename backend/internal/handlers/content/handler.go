package content

import (
	"database/sql"
)

type ContentHandler struct {
	DB *sql.DB
}

func NewContentHandler(db *sql.DB) *ContentHandler {
	return &ContentHandler{DB: db}
}

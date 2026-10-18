package auth

import "database/sql"

const formDataSize = 5 << 20 // 5 MB

type Handler struct {
	DB           *sql.DB
	CloseSession func(string)
}

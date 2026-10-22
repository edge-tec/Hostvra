module hostvra/api

go 1.26.0

require (
	github.com/go-chi/chi/v5 v5.3.2
	github.com/go-chi/cors v1.2.2
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.12.3
	golang.org/x/crypto v0.57.0
	hostvra/agent v0.0.0
)

replace hostvra/agent => ../agent

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/creack/pty v1.1.24 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e // indirect
	golang.org/x/sys v0.48.0 // indirect
)

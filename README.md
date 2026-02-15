# MiniTwit (Go/Gin Port)

A microblogging application ported from Python/Flask to Go/Gin.

## Database
Stored at `/tmp/minitwit.db`.
- **Initialize:** `go run main.go init`

## Running
- **Foreground:** `go run main.go`
- **Background:** `./control.sh start`

## Stopping
- **Foreground:** Press `Ctrl + C`
- **Background:** `./control.sh stop` or `pkill -f minitwit`

## Notes
- **Environment Variables:** Set the port by prefixing the run command: `PORT=8080 go run main.go`. Defaults to `5001`.
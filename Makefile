init:
	./minitwit init

build:
	go build -o minitwit main.go

build-flag:
	gcc flag_tool.c -l sqlite3 -o flag_tool

clean:
	rm -f minitwit flag_tool

run:
	go run main.go

test:
	go test -v ./...

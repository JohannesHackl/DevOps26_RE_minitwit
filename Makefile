.PHONY: run build clean init init-fresh-db

run:
	go run .

build:
	go build -o minitwit .

init-fresh-db:
	@echo "WARNING: This will DELETE all data in the database!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
	   cd tmp/legacy && python -c"from minitwit import init_db; init_db()"; \
	else \
	   echo "Aborted."; \
	fi

init:
	@if [ -f ./tmp/minitwit.db ]; then \
	   echo "ERROR: Database exists. Use 'make init-fresh-db'."; \
	   exit 1; \
	fi
	@echo "Initializing database..."
	cd tmp/legacy && python -c"from minitwit import init_db; init_db()"

clean:
	rm -f minitwit
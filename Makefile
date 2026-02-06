init-fresh-db:
	@echo "WARNING: This will DELETE all data in the database!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		python -c"from minitwit import init_db; init_db()"; \
	else \
		echo "Aborted."; \
	fi

init:
	@if [ -f ./tmp/minitwit.db ]; then \
		echo "ERROR: Database exists. Use 'make init-fresh-db' to delete it and initialize a new database."; \
		exit 1; \
	fi
	@echo "Initializing new database..."
	python -c"from minitwit import init_db; init_db()"

build:
	gcc flag_tool.c -l sqlite3 -o flag_tool

clean:
	rm flag_tool

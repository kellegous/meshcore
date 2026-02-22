.PHONY: docs

docs:
	go run etc/update-readme/main.go README.md

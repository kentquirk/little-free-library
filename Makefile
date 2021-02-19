APP?=library-server
ZIPFILE?=library.zip
COMMIT_SHA=$(shell git rev-parse --short HEAD)
PORT?=3333
STATIC_ROOT?=`pwd`/static
NO_CACHE_TEMPLATES?=true

.PHONY: zip
## zip: constructs the zip file with all necessary artifacts for deploy
zip: ${ZIPFILE}

${ZIPFILE}: build
	rm -f ${ZIPFILE}
	zip -R ${ZIPFILE} bin/${APP} "./static/*" "./templates/*/*.tmpl"

.PHONY: build
## build: build the little free library server
build: clean
	@echo "Building..."
	go build ./...
	go build -o bin/${APP} cmd/${APP}/*.go

.PHONY: race
## race: runs the app with -race and some environment defaults
race: build
	go run -race cmd/${APP}/*.go

.PHONY: run
## run: runs the app
run: build
	go run cmd/${APP}/*.go

.PHONY: install
## install: install the little free library
install: clean
	@echo "Building and installing..."
	go install ./...

.PHONY: clean
## clean: cleans the binary
clean:
	@echo "Cleaning"
	@go clean
	@rm -f bin/*

.PHONY: tidy
## tidy: clean up the go mod file
tidy:
	@go mod tidy

.PHONY: help
## help: prints a help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

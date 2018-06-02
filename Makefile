all: build

.PHONY: build

ifndef ($(GOPATH))
	GOPATH = $(HOME)/go
endif

PATH := $(GOPATH)/bin:$(PATH)
APP_NAME = main

deps:
	go get -u github.com/golang/dep/...
	go get -u github.com/golang/lint/golint
	dep ensure -vendor-only -v

clean:
	rm -rf build/
	rm -f *.zip
	rm main

.pre-build:
	mkdir -p build/

build: .pre-build
	GOOS=linux go build -o ${APP_NAME} ./main.go ./sync.go
	zip build/main.zip ${APP_NAME} libgit2.so libgit2.so.0.26.3 libgit2.so.26 libhttp_parser.so libhttp_parser.so.2 libhttp_parser.so.2.0

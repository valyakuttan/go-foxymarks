SHELL := /bin/bash

HOME_BIN := "$$HOME/.local/bin"

TARGET := foxymarks

MAIN := cmd/main.go

# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

$(TARGET): build

build: $(SRC)
	@go build -o $(TARGET) $(MAIN)

test:
	@echo "running go tests "

clean:
	@rm -f $(TARGET)

install: $(TARGET)
	@cp -f $(TARGET) $(HOME_BIN)

.PHONY: build clean install test
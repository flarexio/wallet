.PHONY: all build_angular build_golang

all: build_angular build_golang

# 編譯 Angular 專案
build_angular:
	cd app && ng build --base-href /app/

# 編譯 Golang 專案
build_golang:
	go build -o wallet cmd/wallet/main.go

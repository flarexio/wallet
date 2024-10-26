.PHONY: all build_angular build_golang serve_angular

all: build_angular build_golang

build_angular:
	cd app && ng build --base-href /app/

build_golang:
	go build -o wallet cmd/wallet/main.go

serve_angular:
	cd app && ng serve --host 0.0.0.0 --public-host wallet.flarex.io:443

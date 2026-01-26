.PHONY: build run clean

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-a \
		-trimpath \
		-ldflags="-s -w -extldflags '-static'" \
		-tags netgo,osusergo \
		-o caffeine \
		./cmd/caffeine

run: build
	./caffeine

clean:
	rm -f ./caffeine

install: build
	cp ./caffeine ~/bin

uninstall:
	rm ~/bin/caffeine

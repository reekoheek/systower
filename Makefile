.PHONY: build run clean

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-a \
		-trimpath \
		-ldflags="-s -w -extldflags '-static'" \
		-tags netgo,osusergo \
		-o systower \
		./cmd/systower

run: build
	./systower

clean:
	rm -f ./systower

install: build
	cp ./systower ~/bin

uninstall:
	rm ~/bin/systower

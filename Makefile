.PHONY: build local clean

build:
	npm run build
	go build -o server .

local:
	./server

clean:
	rm -f server
	rm -rf public/build .routify

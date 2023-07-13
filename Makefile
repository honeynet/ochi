build:
	npm run build
	go build server.go

local:
	./server localhost:3000 token secret

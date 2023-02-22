build:
	npm run build
	go build -o server main.go

local:
	./server localhost:3000 token secret

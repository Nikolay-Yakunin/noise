






.PONY=run-cli run-http

run-cli: 
	go run cmd/cli/main.go

run-http:
	go run cmd/server/main.go

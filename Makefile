all: main

main: dnsserver.go lookupdb.go utils.go
	go build

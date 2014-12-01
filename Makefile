default: linux-amd64

cleanup:
	rm -Rf bin/
bin: cleanup
	mkdir -p bin/

linux-amd64: bin
	@echo ... Building client
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/smg smg/smg.go
	@echo ... Building server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/smgd smgd/smgd.go

osx: bin
	@echo ... Building client
	go build -o bin/smg smg/smg.go
	@echo ... Building server
	go build -o bin/smgd smgd/smgd.go


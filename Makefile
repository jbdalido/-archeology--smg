GOPATH := $(CURDIR)/vendor:$(GOPATH)
default: linux-amd64

cleanup:
	rm -Rf bin/
bin: cleanup
	mkdir -p bin/

linux-amd64: bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/smg smg/smg.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/smgd smgd/smgd.go

osx: bin
	go build -o bin/smg smg/smg.go
#	go build -o bin/smgd smgd/smgd.go


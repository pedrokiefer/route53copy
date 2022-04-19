sources = cmd/route53copy/main.go
name = route53copy
dist/$(name).exe: $(sources)
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -a -installsuffix cgo -ldflags '-s' -o dist/$(name).exe ./cmd/$(name)

dist/$(name)-osx: $(sources)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -installsuffix cgo -ldflags '-s' -o dist/$(name)-osx ./cmd/$(name)

dist/$(name)-linux: $(sources)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -installsuffix cgo -ldflags '-s' -o dist/$(name)-linux ./cmd/$(name)

.PHONY: build release clean
build: dist/$(name).exe dist/$(name)-osx dist/$(name)-linux

release: build
	./release.sh $(name) $(VERSION) dist/*

clean :
	-rm -r dist

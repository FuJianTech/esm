SHELL=/bin/bash
#CWD=$(shell pwd)
#OLDGOPATH=${GOPATH}
#NEWGOPATH:=${CWD}:${OLDGOPATH}
#export GOPATH=$(NEWGOPATH)
#PATH := $(PATH):$(GOPATH)/bin
#GO111MODULE=off

build: clean config
	@go version
	go build  -o bin/esm

tar: build
	tar cfz bin/esm.tar.gz bin/esm

cross-build-all-platform: clean config
	go test
	GOOS=windows GOARCH=amd64     go build -o bin/windows64/esm.exe
	GOOS=windows GOARCH=386       go build -o bin/windows32/esm.exe
	GOOS=darwin  GOARCH=amd64     go build -o bin/darwin64/esm
	#GOOS=darwin  GOARCH=386       go build -o bin/darwin32/esm
	GOOS=linux  GOARCH=amd64      go build -o bin/linux64/esm
	GOOS=linux  GOARCH=386        go build -o bin/linux32/esm
	GOOS=linux  GOARCH=arm        go build -o bin/linux_arm/esm
    GOOS=linux  GOARCH=arm64      go build -o bin/linux_arm64/esm
	GOOS=freebsd  GOARCH=amd64    go build -o bin/freebsd64/esm
	GOOS=freebsd  GOARCH=386      go build -o bin/freebsd32/esm
	GOOS=netbsd  GOARCH=amd64     go build -o bin/netbsd64/esm
	GOOS=netbsd  GOARCH=386       go build -o bin/netbsd32/esm
	GOOS=openbsd  GOARCH=amd64    go build -o bin/openbsd64/esm
	GOOS=openbsd  GOARCH=386      go build -o bin/openbsd32/esm


gox-cross-build-all-platform: clean config
	go get github.com/mitchellh/gox
	go test
	gox -output="bin/esm_{{.OS}}_{{.Arch}}"

cross-gox-build-all-platform: clean config
	go get github.com/mitchellh/gox
	go test
	gox -os=windows -arch=amd64  -output="bin/windows64/esm"
	gox -os=windows -arch=386       -output=bin/windows32/esm
	gox -os=darwin  -arch=amd64     -output=bin/darwin64/esm
	gox -os=darwin  -arch=386       -output=bin/darwin32/esm
	gox -os=linux  -arch=amd64      -output=bin/linux64/esm
	gox -os=linux  -arch=386        -output=bin/linux32/esm
	gox -os=linux  -arch=arm64      -output=bin/linux_arm64/esm
	gox -os=linux  -arch=arm        -output=bin/linux_arm/esm
	gox -os=freebsd  -arch=amd64    -output=bin/freebsd64/esm
	gox -os=freebsd  -arch=386      -output=bin/freebsd32/esm
	gox -os=netbsd  -arch=amd64     -output=bin/netbsd64/esm
	gox -os=netbsd  -arch=386       -output=bin/netbsd32/esm
	gox -os=openbsd  -arch=amd64    -output=bin/openbsd64/esm
	gox -os=openbsd  -arch=386      -output=bin/openbsd32/esm

cross-build: clean config
	go test
	GOOS=windows GOARCH=amd64     go build -ldflags '-w -s' -o bin/windows64/esm.exe
	GOOS=darwin  GOARCH=amd64     go build -ldflags '-w -s' -o bin/darwin64/esm
	GOOS=linux  GOARCH=amd64      go build -ldflags '-w -s' -o bin/linux64/esm
	GOOS=linux  GOARCH=arm64      go build -ldflags '-w -s' -o bin/linux_arm64/esm

all: clean config cross-build

all-platform: clean config cross-build-all-platform

format:
	gofmt -s -w -tabs=false -tabwidth=4 main.go

clean:
	rm -rif bin
	mkdir bin

config:
	@echo "get Dependencies"
	go env
	go get gopkg.in/cheggaaa/pb.v1
	go get github.com/jessevdk/go-flags
	go get github.com/olekukonko/ts
	go get github.com/cihub/seelog
	go get github.com/parnurzeal/gorequest
	go get github.com/mattn/go-isatty

dist: cross-build package

dist-all: all package

dist-all-platform: all-platform package-all-platform

package:
	@echo "Packaging"
	tar cfz 	 bin/windows64.tar.gz    bin/windows64/esm.exe
	tar cfz 	 bin/darwin64.tar.gz      bin/darwin64/esm
	tar cfz 	 bin/linux64.tar.gz      bin/linux64/esm

package-all-platform:
	@echo "Packaging"
	tar cfz 	 bin/windows64.tar.gz    bin/windows64/esm.exe
	tar cfz 	 bin/windows32.tar.gz    bin/windows32/esm.exe
	tar cfz 	 bin/darwin64.tar.gz      bin/darwin64/esm
	tar cfz 	 bin/darwin32.tar.gz      bin/darwin32/esm
	tar cfz 	 bin/linux64.tar.gz      bin/linux64/esm
	tar cfz 	 bin/linux32.tar.gz      bin/linux32/esm
	tar cfz 	 bin/linux_arm.tar.gz     bin/linux_arm/esm
	tar cfz 	 bin/freebsd64.tar.gz    bin/freebsd64/esm
	tar cfz 	 bin/freebsd32.tar.gz    bin/freebsd32/esm
	tar cfz 	 bin/netbsd64.tar.gz     bin/netbsd64/esm
	tar cfz 	 bin/netbsd32.tar.gz     bin/netbsd32/esm
	tar cfz 	 bin/openbsd64.tar.gz     bin/openbsd64/esm
	tar cfz 	 bin/openbsd32.tar.gz     bin/openbsd32/esm


cross-compile:
	@echo "Prepare Cross Compiling"
	cd $(GOROOT)/src && GOOS=windows GOARCH=amd64 ./make.bash --no-clean
	cd $(GOROOT)/src && GOOS=darwin  GOARCH=amd64 ./make.bash --no-clean 2> /dev/null 1> /dev/null
	cd $(GOROOT)/src && GOOS=linux  GOARCH=amd64 ./make.bash --no-clean 2> /dev/null 1> /dev/null

	cd $(CWD)



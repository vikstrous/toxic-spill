.PHONY: all

all: main/app.go
	docker run -v $(PWD)/Godeps/_workspace:/Godeps -v $(PWD):/usr/src/github.com/vikstrous/toxic-spill -e "GOPATH=/Godeps:/usr/src" -w /usr/src/github.com/vikstrous/toxic-spill golang:1.4.2 go build -o toxic-spill ./main

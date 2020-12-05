PKG = github.com/rmanzoku/mackerel-plugin-aws-billing-per-account
NAME = mackerel-plugin-aws-billing-per-account

# Build options
GITHUB_REF = $(shell git describe --all HEAD)
GITHUB_SHA = $(shell git rev-parse HEAD)
BUILD_LDFLAGS = "-w -s -X 'main.version=$(GITHUB_REF)' -X 'main.revision=$(GITHUB_SHA)'"

.PHONY: build test upload crossbuild

test:
	go test -v ./...

build: $(NAME)

$(NAME):
	go build -ldflags "-w -s -X 'main.version=$(GITHUB_REF)' -X 'main.revision=$(GITHUB_SHA)'" -o $(NAME)

crossbuild:
	godzil crossbuild -pv=v$(GITHUB_REF) -build-ldflags=$(BUILD_LDFLAGS) \
	-os=linux,darwin,windows -arch=amd64,arm64 -d=./dist/v$(GITHUB_REF) ./

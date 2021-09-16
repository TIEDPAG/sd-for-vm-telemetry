.PHONY: format docker

BINARY=vm-discovery
DOCKER_REPO=tiedpag
IMAGE=$(DOCKER_REPO)/vm-discovery
IMAGE_TAG=v0.3.0
FINDFILES=find . \( -path ./common-protos -o -path ./.git -o -path ./out -o -path ./.github -o -path ./licenses -o -path ./vendor \) -prune -o -type f
XARGS = xargs -0 -r

clean:
	rm $(BINARY)

build:
	GOOS=linux GOARCH=amd64 go build -o out/$(BINARY)

test:
	go test `go list ./...`

docker: BUILD_PRE=&& chmod 755 vm-discovery
docker: out/vm-discovery
docker: docker/Dockerfile
	mkdir -p out/$@ && cp -r $^ out/$@ && cd out/$@ $(BUILD_PRE) && docker build -t $(IMAGE):$(IMAGE_TAG) -f Dockerfile .

docker.push: docker
	docker push $(IMAGE):$(IMAGE_TAG)

format: fmt ## Auto formats all code. This should be run before sending a PR.
fmt: format-go tidy-go

tidy-go:
	@go mod tidy

format-go: tidy-go
	@${FINDFILES} -name '*.go' \( ! \( -name '*.gen.go' -o -name '*.pb.go' \) \) -print0 | ${XARGS} goimports -w -local "istio.io"
.PHONY : docker-build docker-push release PIPCOMPILE

NAME 					:= eduseal
LDFLAGS                 := -ldflags "-w -s --extldflags '-static'"
LDFLAGS_DYNAMIC			:= -ldflags "-w -s"
PYTHON					:= $(shell which python)
PIPCOMPILE				:= pip-compile -v --upgrade --generate-hashes --allow-unsafe --index-url https://pypi.sunet.se/simple
PIPSYNC					:= pip-sync --index-url https://pypi.sunet.se/simple --python-executable $(PYTHON)

test: test-verifier test-datastore

test-verifier:
	$(info Testing verifier)
	go test -v ./cmd/verifier

test-datastore:
	$(info Testing datastore)
	go test -v ./cmd/datastore

gosec:
	$(info Run gosec)
	gosec -color -nosec -tests ./...

staticcheck:
	$(info Run staticcheck)
	staticcheck ./...

start:
	$(info Run!)
	docker-compose -f docker-compose.yaml up -d --remove-orphans

stop:
	$(info stopping eduSeal)
	docker-compose -f docker-compose.yaml rm -s -f

sync_py_deps:
	$(PIPSYNC) requirements.txt

update_py_deps:
	$(PIPCOMPILE) requirements.in

restart: stop start

get_release-tag:
	@date +'%Y%m%d%H%M%S%9N'

ifndef VERSION
VERSION := latest
endif


DOCKER_TAG_APIGW 		:= docker.sunet.se/eduseal/apigw:$(VERSION)
DOCKER_TAG_CACHE 		:= docker.sunet.se/eduseal/cache:$(VERSION)
DOCKER_TAG_PERSISTENT 	:= docker.sunet.se/eduseal/persistent:$(VERSION)
DOCKER_TAG_GOBUILD 		:= docker.sunet.se/eduseal/gobuild:$(VERSION)


build: proto build-cache build-persistent build-apigw


build-cache:
	$(info Building cache)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/$(NAME)_cache ${LDFLAGS} ./cmd/cache/main.go

build-persistent:
	$(info Building persistent)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/$(NAME)_persistent ${LDFLAGS} ./cmd/persistent/main.go

build-apigw:
	$(info Building apigw)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/$(NAME)_apigw ${LDFLAGS} ./cmd/apigw/main.go


docker-build: docker-build-cache docker-build-persistent docker-build-apigw

docker-build-gobuild:
	$(info Docker Building gobuild with tag: $(VERSION))
	docker build --tag $(DOCKER_TAG_GOBUILD) --file dockerfiles/gobuild .

docker-build-cache:
	$(info Docker Building cache with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=cache --tag $(DOCKER_TAG_CACHE) --file dockerfiles/worker .

docker-build-persistent:
	$(info Docker Building persistent with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=persistent --tag $(DOCKER_TAG_PERSISTENT) --file dockerfiles/worker .


docker-build-apigw:
	$(info Docker building apigw with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=apigw --build-arg VERSION=$(VERSION) --tag $(DOCKER_TAG_APIGW) --file dockerfiles/worker .

docker-push-gobuild:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_GOBUILD)

docker-push-cache:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_CACHE)

docker-push-persistent:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_PERSISTENT)

docker-push-apigw:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_APIGW)

docker-push: docker-push-cache docker-push-persistent docker-push-apigw
	$(info Pushing docker images)

docker-tag-apigw:
	$(info Tagging docker images)
	docker tag $(DOCKER_TAG_APIGW) docker.sunet.se/dc4eu/apigw:$(NEWTAG)

docker-tag-verifier:
	$(info Tagging docker images)
	docker tag $(DOCKER_TAG_VERIFIER) docker.sunet.se/dc4eu/verifier:$(NEWTAG)

docker-tag-cache:
	$(info Tagging docker images)
	docker tag $(DOCKER_TAG_CACHE) docker.sunet.se/dc4eu/cache:$(NEWTAG)

docker-tag-persistent:
	$(info Tagging docker images)
	docker tag $(DOCKER_TAG_PERSISTENT) docker.sunet.se/dc4eu/persistent:$(NEWTAG)

docker-tag: docker-tag-apigw docker-tag-cache docker-tag-persistent
	$(info Tagging docker images)

release:
	$(info Release version: $(VERSION))
	git tag $(VERSION)
	git push origin ${VERSION}
	make docker-build
	make docker-push
	$(info Release version $(VERSION) done)
	$(info tag $(NEWTAG) from $(VERSION))
	make docker-tag
	make VERSION=$(NEWTAG) docker-push

docker-pull:
	$(info Pulling docker images)
	docker pull $(DOCKER_TAG_APIGW)
	docker pull $(DOCKER_TAG_GOBUILD)
	docker pull $(DOCKER_TAG_PERSISTENT)

docker-archive:
	docker save --output docker_archives/eduseal_$(VERSION).tar $(DOCKER_TAG_VERIFIER) $(DOCKER_TAG_DATASTORE) $(DOCKER_TAG_REGISTRY)


clean_redis:
	$(info Cleaning redis volume)
	docker volume rm eduseal_redis_data 

ci_build: docker-build docker-push
	$(info CI Build)

proto: proto-status


proto-status:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-status-model.proto 

swagger: swagger-apigw swagger-fmt

swagger-fmt:
	swag fmt

swagger-apigw:
	swag init -d internal/apigw/apiv1/ -g client.go --output docs/apigw --parseDependency --packageName docs

install-tools:
	$(info Install from apt)
	apt-get update && apt-get install -y \
		protobuf-compiler \
		netcat-openbsd

	make clean-apt-cache

	$(info Install from go)
	go install github.com/swaggo/swag/cmd/swag@latest && \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest


clean-apt-cache:
	$(info Cleaning apt cache)
	rm -rf /var/lib/apt/lists/*

vscode:
	$(info Install APT packages)
	sudo apt-get update && sudo apt-get install -y \
		protobuf-compiler \
		netcat-openbsd \
		python3-pip \
		python3.11-venv
	$(info Install go packages)
	go install github.com/swaggo/swag/cmd/swag@latest && \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
	go install golang.org/x/tools/cmd/deadcode@latest && \
	go install github.com/securego/gosec/v2/cmd/gosec@latest && \
	go install honnef.co/go/tools/cmd/staticcheck@latest

	$(info Create python environment)
	python3.11 -m venv .venv
	. .venv/bin/activate && pip install -r requirements.txt && pip3 install pip-tools
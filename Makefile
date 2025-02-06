.PHONY : docker-build docker-push release PIPCOMPILE

NAME 					:= eduseal
LDFLAGS                 := -ldflags "-w -s --extldflags '-static'"
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

vulncheck:
	$(info Run vulncheck)
	govulncheck -show verbose ./...

start:
	$(info Run!)
	docker compose -f docker-compose.yaml up -d --remove-orphans

stop:
	$(info stopping eduSeal)
	docker compose -f docker-compose.yaml rm -s -f

sync_py_deps:
	$(PIPSYNC) requirements.txt

update_py_deps:
	$(PIPCOMPILE) requirements.in

restart: stop start

clean_nats_volumes:
	$(info deleting nats volumes)
	docker volume rm nats1 nats2 nats3

create_nats_volumes:
	$(info Creating nats volumes)
	docker volume create nats1
	docker volume create nats2
	docker volume create nats3

get_release-tag:
	@date +'%Y%m%d%H%M%S%9N'

ifndef VERSION
VERSION := latest
endif


DOCKER_TAG_APIGW 				:= docker.sunet.se/eduseal/apigw:$(VERSION)
DOCKER_TAG_GOBUILD 				:= docker.sunet.se/eduseal/gobuild:$(VERSION)
DOCKER_TAG_SEALER_SECTIGO		:= docker.sunet.se/eduseal/sealer_sectigo:$(VERSION)
DOCKER_TAG_SEALER_SOFTHSM		:= docker.sunet.se/eduseal/sealer_softhsm:$(VERSION)
DOCKER_TAG_VALIDATOR			:= docker.sunet.se/eduseal/validator:$(VERSION)


#### Docker build
docker-build-non-pkcs11-containers: docker-build-apigw docker-build-validator
docker-build-sectigo: docker-build-non-pkcs11-containers docker-build-sealer-sectigo
docker-build-softhsm: docker-build-non-pkcs11-containers docker-build-sealer-softhsm

docker-build-apigw:
	$(info Docker building apigw with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=apigw --build-arg VERSION=$(VERSION) --tag $(DOCKER_TAG_APIGW) --file docker/worker .

docker-build-sealer-sectigo:
	$(info building docker image $(DOCKER_TAG_SEALER_SECTIGO) )
	docker build --tag $(DOCKER_TAG_SEALER_SECTIGO) --file docker/sealer/sectigo/Dockerfile .

docker-build-sealer-softhsm:
	$(info building docker image $(DOCKER_TAG_SEALER_SOFTHSM) )
	docker build --tag $(DOCKER_TAG_SEALER_SOFTHSM) --file docker/sealer/softhsm/Dockerfile .

docker-build-validator:
	$(info building docker image $(DOCKER_TAG_VALIDATOR) )
	docker build --tag $(DOCKER_TAG_VALIDATOR) --file docker/validator/Dockerfile .

docker-build-gobuild:
	$(info Docker Building gobuild with tag: $(VERSION))
	docker build --tag $(DOCKER_TAG_GOBUILD) --file docker/gobuild .

#### Docker push
docker-push: docker-push-apigw docker-push-sealer-sectigo docker-push-sealer-softhsm docker-push-validator
	$(info Pushing docker images)

docker-push-apigw:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_APIGW)

docker-push-sealer-softhsm:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_SEALER_SOFTHSM)

docker-push-sealer-sectigo:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_SEALER_SECTIGO)

docker-push-validator:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_VALIDATOR)

docker-push-gobuild:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_GOBUILD)

docker-tag-apigw:
	$(info Tagging docker images)
	docker tag $(DOCKER_TAG_APIGW) docker.sunet.se/eduseal/apigw:$(NEWTAG)

docker-tag-verifier:
	$(info Tagging docker images)
	docker tag $(DOCKER_TAG_VERIFIER) docker.sunet.se/eduseal/verifier:$(NEWTAG)

docker-tag: docker-tag-apigw
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

proto-golang: proto-status-golang proto-sealer-golang proto-validator-golang

proto-status-golang:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-status-model.proto 

proto-sealer-golang:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-sealer.proto 

proto-validator-golang:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-validator.proto 

proto-python: proto-sealer-python proto-validator-python

proto-sealer-python:
	python -m grpc_tools.protoc --proto_path=./proto/ --python_out=./src/eduseal/sealer --grpc_python_out=./src/eduseal/sealer ./proto/v1-sealer.proto

proto-validator-python:
	python -m grpc_tools.protoc --proto_path=./proto/ --python_out=./src/eduseal/validator --grpc_python_out=./src/eduseal/validator ./proto/v1-validator.proto

proto: proto-golang proto-python


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
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/nats-io/nats-top@latest

	$(info Create python environment)
	python3.11 -m venv .venv
	. .venv/bin/activate && pip install -r requirements.txt && pip3 install pip-tools
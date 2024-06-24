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


DOCKER_TAG_APIGW 				:= docker.sunet.se/eduseal/apigw:$(VERSION)
DOCKER_TAG_CACHE 				:= docker.sunet.se/eduseal/cache:$(VERSION)
DOCKER_TAG_PERSISTENT 			:= docker.sunet.se/eduseal/persistent:$(VERSION)
DOCKER_TAG_GOBUILD 				:= docker.sunet.se/eduseal/gobuild:$(VERSION)
DOCKER_TAG_SEALER_USB			:= docker.sunet.se/eduseal/sealer_usb:$(VERSION)
DOCKER_TAG_SEALER_SOFTHSM		:= docker.sunet.se/eduseal/sealer_softhsm:$(VERSION)
DOCKER_TAG_SEALER_SOFTHSM_GRPC	:= docker.sunet.se/eduseal/sealer_softhsm_grpc:$(VERSION)
DOCKER_TAG_VALIDATOR_GRPC		:= docker.sunet.se/eduseal/validator_grpc:$(VERSION)


#### Docker build
docker-build-non-pkcs11-containers: docker-build-cache docker-build-persistent docker-build-apigw docker-build-validator
docker-build-usb: docker-build-non-pkcs11-containers docker-build-sealer-usb
docker-build-softhsm: docker-build-non-pkcs11-containers docker-build-sealer-softhsm-grpc

docker-build-apigw:
	$(info Docker building apigw with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=apigw --build-arg VERSION=$(VERSION) --tag $(DOCKER_TAG_APIGW) --file docker/worker .

docker-build-cache:
	$(info Docker Building cache with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=cache --tag $(DOCKER_TAG_CACHE) --file docker/worker .

docker-build-persistent:
	$(info Docker Building persistent with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=persistent --tag $(DOCKER_TAG_PERSISTENT) --file docker/worker .

docker-build-sealer-usb:
	$(info building docker image $(DOCKER_TAG_SEALER_USB) )
	docker build --tag $(DOCKER_TAG_SEALER_USB) --file docker/sealer_usb .

docker-build-sealer-softhsm:
	$(info building docker image $(DOCKER_TAG_SEALER_SOFTHSM) )
	docker build --tag $(DOCKER_TAG_SEALER_SOFTHSM) --file docker/sealer_softhsm .

docker-build-sealer-softhsm-grpc:
	$(info building docker image $(DOCKER_TAG_SEALER_SOFTHSM_GRPC) )
	docker build --tag $(DOCKER_TAG_SEALER_SOFTHSM_GRPC) --file docker/sealer/softhsm/grpc/Dockerfile .

docker-build-validator:
	$(info building docker image $(DOCKER_TAG_VALIDATOR_GRPC) )
	docker build --tag $(DOCKER_TAG_VALIDATOR_GRPC) --file docker/validator/grpc/Dockerfile .

docker-build-gobuild:
	$(info Docker Building gobuild with tag: $(VERSION))
	docker build --tag $(DOCKER_TAG_GOBUILD) --file docker/gobuild .

#### Docker push
docker-push: docker-push-cache docker-push-persistent docker-push-apigw docker-push-sealer-usb docker-push-validator
	$(info Pushing docker images)

docker-push-apigw:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_APIGW)

docker-push-cache:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_CACHE)

docker-push-persistent:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_PERSISTENT)

docker-push-sealer-usb:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_SEALER_USB)

docker-push-validator:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_VALIDATOR)

docker-push-gobuild:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_GOBUILD)


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

proto: proto-status proto-sealer proto-validator

proto-status:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-status-model.proto 

proto-sealer:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-sealer.proto 

proto-validator:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-validator.proto 

proto-python: proto-sealer-python proto-validator-python

proto-sealer-python:
	python -m grpc_tools.protoc --proto_path=./proto/ --python_out=./src/eduseal/sealer --grpc_python_out=./src/eduseal/sealer ./proto/v1-sealer.proto

proto-validator-python:
	python -m grpc_tools.protoc --proto_path=./proto/ --python_out=./src/eduseal/validator --grpc_python_out=./src/eduseal/validator ./proto/v1-validator.proto


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
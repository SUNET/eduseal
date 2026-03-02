.PHONY : docker-build docker-push docker-tag-testing docker-push-testing docker-pull-release docker-tag-prod docker-push-prod docker-tag-prod-version docker-push-prod-version local-publish release release-local release-prod release-noctool PIPCOMPILE

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

GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
NOCTOOL_LDFLAGS := -ldflags "-w -s --extldflags '-static' -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

build-noctool:
	$(info Building noctool)
	go build $(NOCTOOL_LDFLAGS) -o bin/noctool ./cmd/noctool

release-noctool:
	@echo "$(BUMP)" | grep -qE '^(major|minor|patch)$$' || \
		{ echo "Error: BUMP must be major, minor, or patch (got: $(BUMP))"; exit 1; }
	@if ! git diff --quiet HEAD 2>/dev/null; then \
		echo "Error: working tree is dirty — commit or stash changes first"; exit 1; \
	fi
	@LATEST=$$(git tag -l "noctool-v*" --sort=-v:refname | grep -E '^noctool-v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n1); \
	if [ -z "$$LATEST" ]; then \
		echo "No existing noctool tags found, starting at noctool-v0.0.0"; \
		LATEST="noctool-v0.0.0"; \
	fi; \
	CURRENT=$$(echo "$$LATEST" | sed 's/^noctool-v//'); \
	MAJOR=$$(echo "$$CURRENT" | cut -d. -f1); \
	MINOR=$$(echo "$$CURRENT" | cut -d. -f2); \
	PATCH=$$(echo "$$CURRENT" | cut -d. -f3); \
	case "$(BUMP)" in \
		major) MAJOR=$$((MAJOR + 1)); MINOR=0; PATCH=0 ;; \
		minor) MINOR=$$((MINOR + 1)); PATCH=0 ;; \
		patch) PATCH=$$((PATCH + 1)) ;; \
	esac; \
	NEW_TAG="noctool-v$$MAJOR.$$MINOR.$$PATCH"; \
	echo ""; \
	echo "$$LATEST -> $$NEW_TAG"; \
	echo ""; \
	if git rev-parse "$$NEW_TAG" >/dev/null 2>&1; then \
		echo "Error: tag $$NEW_TAG already exists"; exit 1; \
	fi; \
	git tag -a "$$NEW_TAG" -m "Release $$NEW_TAG"; \
	git push origin "$$NEW_TAG"; \
	echo ""; \
	echo "==> $$NEW_TAG pushed. GitHub Actions will build noctool binaries."; \
	echo ""


DOCKER_TAG_APIGW 				:= docker.sunet.se/eduseal/apigw:$(VERSION)
DOCKER_TAG_GOBUILD 				:= docker.sunet.se/eduseal/gobuild:$(VERSION)
DOCKER_TAG_SEALER_SECTIGO		:= docker.sunet.se/eduseal/sealer_sectigo:$(VERSION)
DOCKER_TAG_SEALER_SOFTHSM		:= docker.sunet.se/eduseal/sealer_softhsm:$(VERSION)
DOCKER_TAG_SEALER_LUNAHSM		:= docker.sunet.se/eduseal/sealer_lunahsm:$(VERSION)
DOCKER_TAG_VALIDATOR			:= docker.sunet.se/eduseal/validator:$(VERSION)


#### Docker build
docker-build-non-pkcs11-containers: docker-build-apigw docker-build-validator
docker-build: docker-build-non-pkcs11-containers docker-build-sealer-lunahsm
docker-build-softhsm: docker-build-non-pkcs11-containers docker-build-sealer-softhsm

docker-build-apigw:
	$(info Docker building apigw with tag: $(VERSION))
	docker build --build-arg SERVICE_NAME=apigw --build-arg VERSION=$(VERSION) --tag $(DOCKER_TAG_APIGW) --file docker/apigw/Dockerfile .

docker-build-sealer-sectigo:
	$(info building docker image $(DOCKER_TAG_SEALER_SECTIGO) )
	docker build --tag $(DOCKER_TAG_SEALER_SECTIGO) --file docker/sealer/sectigo/Dockerfile .

docker-build-sealer-softhsm:
	$(info building docker image $(DOCKER_TAG_SEALER_SOFTHSM) )
	docker build --tag $(DOCKER_TAG_SEALER_SOFTHSM) --file docker/sealer/softhsm/Dockerfile .

docker-build-sealer-lunahsm:
	$(info building docker image $(DOCKER_TAG_SEALER_LUNAHSM) )
	docker build --tag $(DOCKER_TAG_SEALER_LUNAHSM) --file docker/sealer/lunahsm/Dockerfile .

docker-build-validator:
	$(info building docker image $(DOCKER_TAG_VALIDATOR) )
	docker build --tag $(DOCKER_TAG_VALIDATOR) --file docker/validator/Dockerfile .

docker-build-gobuild:
	$(info Docker Building gobuild with tag: $(VERSION))
	docker build --tag $(DOCKER_TAG_GOBUILD) --file docker/gobuild .

#### Docker push
docker-push: docker-push-apigw docker-push-sealer-lunahsm docker-push-validator
	$(info Pushing docker images)

docker-tag-testing:
	$(info Tagging release images as :testing)
	docker tag $(DOCKER_TAG_APIGW) $(patsubst %:$(VERSION),%:testing,$(DOCKER_TAG_APIGW))
	docker tag $(DOCKER_TAG_SEALER_LUNAHSM) $(patsubst %:$(VERSION),%:testing,$(DOCKER_TAG_SEALER_LUNAHSM))
	docker tag $(DOCKER_TAG_VALIDATOR) $(patsubst %:$(VERSION),%:testing,$(DOCKER_TAG_VALIDATOR))

docker-push-testing:
	$(info Pushing :testing image tags)
	docker push $(patsubst %:$(VERSION),%:testing,$(DOCKER_TAG_APIGW))
	docker push $(patsubst %:$(VERSION),%:testing,$(DOCKER_TAG_SEALER_LUNAHSM))
	docker push $(patsubst %:$(VERSION),%:testing,$(DOCKER_TAG_VALIDATOR))

docker-pull-release:
	$(info Pulling release-tagged images)
	docker pull $(DOCKER_TAG_APIGW)
	docker pull $(DOCKER_TAG_SEALER_LUNAHSM)
	docker pull $(DOCKER_TAG_VALIDATOR)

docker-tag-prod:
	$(info Tagging release images as :prod)
	docker tag $(DOCKER_TAG_APIGW) $(patsubst %:$(VERSION),%:prod,$(DOCKER_TAG_APIGW))
	docker tag $(DOCKER_TAG_SEALER_LUNAHSM) $(patsubst %:$(VERSION),%:prod,$(DOCKER_TAG_SEALER_LUNAHSM))
	docker tag $(DOCKER_TAG_VALIDATOR) $(patsubst %:$(VERSION),%:prod,$(DOCKER_TAG_VALIDATOR))

docker-push-prod:
	$(info Pushing :prod image tags)
	docker push $(patsubst %:$(VERSION),%:prod,$(DOCKER_TAG_APIGW))
	docker push $(patsubst %:$(VERSION),%:prod,$(DOCKER_TAG_SEALER_LUNAHSM))
	docker push $(patsubst %:$(VERSION),%:prod,$(DOCKER_TAG_VALIDATOR))

docker-tag-prod-version:
	$(info Tagging release images as :prod-$(VERSION))
	docker tag $(DOCKER_TAG_APIGW) $(patsubst %:$(VERSION),%:prod-$(VERSION),$(DOCKER_TAG_APIGW))
	docker tag $(DOCKER_TAG_SEALER_LUNAHSM) $(patsubst %:$(VERSION),%:prod-$(VERSION),$(DOCKER_TAG_SEALER_LUNAHSM))
	docker tag $(DOCKER_TAG_VALIDATOR) $(patsubst %:$(VERSION),%:prod-$(VERSION),$(DOCKER_TAG_VALIDATOR))

docker-push-prod-version:
	$(info Pushing :prod-$(VERSION) image tags)
	docker push $(patsubst %:$(VERSION),%:prod-$(VERSION),$(DOCKER_TAG_APIGW))
	docker push $(patsubst %:$(VERSION),%:prod-$(VERSION),$(DOCKER_TAG_SEALER_LUNAHSM))
	docker push $(patsubst %:$(VERSION),%:prod-$(VERSION),$(DOCKER_TAG_VALIDATOR))

docker-push-apigw:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_APIGW)

docker-push-sealer-sectigo:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_SEALER_SECTIGO)

docker-push-sealer-softhsm:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_SEALER_SOFTHSM)

docker-push-sealer-lunahsm:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_SEALER_LUNAHSM)

docker-push-validator:
	$(info Pushing docker image)
	docker push $(DOCKER_TAG_VALIDATOR)

docker-push-gobuild:
	$(info Pushing docker images)
	docker push $(DOCKER_TAG_GOBUILD)

#### Release targets
# Creates a single vX.Y.Z tag for ALL services (apigw, sealer, validator).
# Usage:
#   make release               # defaults to patch bump
#   make release BUMP=patch    # v1.0.0 -> v1.0.1
#   make release BUMP=minor    # v1.0.0 -> v1.1.0
#   make release BUMP=major    # v1.0.0 -> v2.0.0

BUMP ?= patch

release:
	@set -e; \
	echo "$(BUMP)" | grep -qE '^(major|minor|patch)$$' || \
		{ echo "Error: BUMP must be major, minor, or patch (got: $(BUMP))"; exit 1; }
	@if ! git diff --quiet HEAD 2>/dev/null; then \
		echo "Error: working tree is dirty — commit or stash changes first"; exit 1; \
	fi
	@LATEST=$$(git tag -l "v*" --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n1); \
	if [ -z "$$LATEST" ]; then \
		echo "No existing version tags found, starting at v0.0.0"; \
		LATEST="v0.0.0"; \
	fi; \
	CURRENT=$$(echo "$$LATEST" | sed 's/^v//'); \
	MAJOR=$$(echo "$$CURRENT" | cut -d. -f1); \
	MINOR=$$(echo "$$CURRENT" | cut -d. -f2); \
	PATCH=$$(echo "$$CURRENT" | cut -d. -f3); \
	case "$(BUMP)" in \
		major) MAJOR=$$((MAJOR + 1)); MINOR=0; PATCH=0 ;; \
		minor) MINOR=$$((MINOR + 1)); PATCH=0 ;; \
		patch) PATCH=$$((PATCH + 1)) ;; \
	esac; \
	NEW_TAG="v$$MAJOR.$$MINOR.$$PATCH"; \
	echo ""; \
	echo "$$LATEST -> $$NEW_TAG"; \
	echo ""; \
	if git rev-parse "$$NEW_TAG" >/dev/null 2>&1; then \
		echo "Error: tag $$NEW_TAG already exists"; exit 1; \
	fi; \
	git tag -a "$$NEW_TAG" -m "Release $$NEW_TAG (apigw, sealer, validator)"; \
	git push origin "$$NEW_TAG"; \
	git ls-remote --exit-code --tags origin "refs/tags/$$NEW_TAG" >/dev/null; \
	$(MAKE) local-publish VERSION=$$NEW_TAG; \
	echo ""; \
	echo "==> $$NEW_TAG pushed and published locally."; \
	echo ""

#### Local release publish
# Builds and pushes apigw, sealer_lunahsm, validator images from a specific tag.
# Also updates :testing tags to point at that release.
# Usage:
#   make release-local TAG=v1.2.3

local-publish:
	@if [ "$(VERSION)" = "latest" ] || [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required (example: make local-publish VERSION=v1.2.3)"; exit 1; \
	fi
	@echo ""; \
	echo "Publishing images from local environment with VERSION=$(VERSION)"; \
	echo ""; \
	$(MAKE) docker-build VERSION=$(VERSION) && \
	$(MAKE) docker-push VERSION=$(VERSION) && \
	$(MAKE) docker-tag-testing VERSION=$(VERSION) && \
	$(MAKE) docker-push-testing VERSION=$(VERSION); \
	echo ""; \
	echo "==> Local publish complete for VERSION=$(VERSION)"; \
	echo ""

release-local:
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required (example: make release-local TAG=v1.2.3)"; exit 1; \
	fi
	@TAG_CLEAN=$$(echo "$(TAG)" | sed 's#^refs/tags/##'); \
	echo "$$TAG_CLEAN" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$' || \
		{ echo "Error: TAG must match vX.Y.Z (got: $$TAG_CLEAN)"; exit 1; }; \
	$(MAKE) local-publish VERSION=$$TAG_CLEAN

#### Prod promotion
# Promotes a version to prod by locally pulling :vX.Y.Z images
# and re-tagging/pushing as :prod. No rebuild.
# Usage:
#   make release-prod              # promotes latest vX.Y.Z tag to prod
#   make release-prod TAG=v1.2.3   # promotes v1.2.3 to prod

release-prod:
	@set -e; \
	if [ -n "$(TAG)" ]; then \
		SRC_TAG=$$(echo "$(TAG)" | sed 's#^refs/tags/##'); \
	else \
		SRC_TAG=$$(git tag -l "v*" --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n1); \
		if [ -z "$$SRC_TAG" ]; then \
			echo "Error: no version tags found. Run 'make release' first."; exit 1; \
		fi; \
	fi; \
	echo "$$SRC_TAG" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$' || \
		{ echo "Error: TAG must match vX.Y.Z (got: $$SRC_TAG)"; exit 1; }; \
	if docker manifest inspect "docker.sunet.se/eduseal/apigw:prod-$$SRC_TAG" >/dev/null 2>&1; then \
		echo "Error: prod-$$SRC_TAG already exists in registry. Refusing to re-push."; exit 1; \
	fi; \
	echo ""; \
	echo "Promoting $$SRC_TAG -> prod"; \
	echo ""; \
	$(MAKE) docker-pull-release VERSION=$$SRC_TAG && \
	$(MAKE) docker-tag-prod VERSION=$$SRC_TAG && \
	$(MAKE) docker-tag-prod-version VERSION=$$SRC_TAG && \
	$(MAKE) docker-push-prod VERSION=$$SRC_TAG && \
	$(MAKE) docker-push-prod-version VERSION=$$SRC_TAG; \
	echo ""; \
	echo "==> Local prod re-tag/push complete for $$SRC_TAG (:prod and :prod-$$SRC_TAG)."; \
	echo ""

docker-pull:
	$(info Pulling docker images)
	docker pull $(DOCKER_TAG_APIGW)
	docker pull $(DOCKER_TAG_GOBUILD)

docker-archive:
	docker save --output docker_archives/eduseal_$(VERSION).tar $(DOCKER_TAG_VERIFIER) $(DOCKER_TAG_DATASTORE) $(DOCKER_TAG_REGISTRY)


clean_redis:
	$(info Cleaning redis volume)
	docker volume rm eduseal_redis_data 

ci_build: docker-build docker-push
	$(info CI Build)

proto-golang: proto-status-golang proto-sealer-golang proto-validator-golang

proto-status-golang:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-status.proto

proto-sealer-golang:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-sealer.proto 

proto-validator-golang:
	protoc --proto_path=./proto/ --go-grpc_opt=module=eduseal --go_opt=module=eduseal --go_out=. --go-grpc_out=. ./proto/v1-validator.proto 

proto-python: proto-sealer-python proto-validator-python proto-python-fix-imports

proto-sealer-python:
	python -m grpc_tools.protoc --proto_path=./proto/ --python_out=./src/eduseal/sealer --grpc_python_out=./src/eduseal/sealer ./proto/v1-sealer.proto

proto-validator-python:
	python -m grpc_tools.protoc --proto_path=./proto/ --python_out=./src/eduseal/validator --grpc_python_out=./src/eduseal/validator ./proto/v1-validator.proto

proto-python-fix-imports:
	@sed -i "s/^import v1_sealer_pb2 as /import eduseal.sealer.v1_sealer_pb2 as /" ./src/eduseal/sealer/v1_sealer_pb2_grpc.py
	@sed -i "s/^import v1_validator_pb2 as /import eduseal.validator.v1_validator_pb2 as /" ./src/eduseal/validator/v1_validator_pb2_grpc.py

proto-health-python:
	python -m grpc_tools.protoc --proto_path=./proto/ --python_out=./src/eduseal/sealer --grpc_python_out=./src/eduseal/sealer ./proto/v1-status.proto

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


diagram:
	plantuml docs/diagrams/*.puml

vscode:
	$(info Install APT packages)
	sudo apt-get update && sudo apt-get install -y \
		protobuf-compiler \
		netcat-openbsd \
		python3-pip \
		python3.13-venv \
		plantuml
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
	python3 -m venv .venv
	. .venv/bin/activate && pip install -r requirements.txt && pip
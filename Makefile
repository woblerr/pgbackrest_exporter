SHELL := /bin/bash
APP_NAME := pgbackrest_exporter
BRANCH_FULL := $(shell git rev-parse --abbrev-ref HEAD)
BRANCH := $(subst /,-,$(BRANCH_FULL))
GIT_REV := $(shell git describe --abbrev=7 --always)
BUILD_DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)
BUILD_USER ?= pgbackrest_exporter
SERVICE_CONF_DIR := /etc/systemd/system
HTTP_PORT := 9854
BACKREST_VERSION := 2.56.0
DOCKER_BACKREST_VERSION := v0.34
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
DOCKER_CONTAINER_E2E := $(shell docker ps -a -q -f name=$(APP_NAME)_e2e)
HTTP_PORT_E2E := $(shell echo $$((10000 + ($$RANDOM % 10000))))
BACKREST_STANZA_EXCLUDE ?= demo
BACKREST_STANZA_INCLUDE ?= demo
LDFLAGS = -X github.com/prometheus/common/version.Version=$(BRANCH)-$(GIT_REV) \
		  -X github.com/prometheus/common/version.Branch=$(BRANCH) \
		  -X github.com/prometheus/common/version.Revision=$(GIT_REV) \
		  -X github.com/prometheus/common/version.BuildDate=$(BUILD_DATE) \
		  -X github.com/prometheus/common/version.BuildUser=$(BUILD_USER)

.PHONY: test
test:
	@echo "Run tests for $(APP_NAME)"
	TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1  ./...

.PHONY: test-e2e
test-e2e:
	@echo "Run end-to-end tests for $(APP_NAME)"
	@if [ -n "$(DOCKER_CONTAINER_E2E)" ]; then docker rm -f "$(DOCKER_CONTAINER_E2E)"; fi;
	DOCKER_BUILDKIT=1 docker build --pull -f e2e_tests/Dockerfile --build-arg REPO_BUILD_TAG=$(BRANCH)-$(GIT_REV) --build-arg BACKREST_VERSION=$(BACKREST_VERSION) --build-arg DOCKER_BACKREST_VERSION=$(DOCKER_BACKREST_VERSION) -t $(APP_NAME)_e2e .
	$(call e2e_basic)
	$(call e2e_exclude)
	$(call e2e_include)
	$(call e2e_disable_collector)
	$(call e2e_tls_auth,/e2e_tests/web_config_empty.yml,false,false)
	$(call e2e_tls_auth,/e2e_tests/web_config_TLS_noAuth.yml,true,false)
	$(call e2e_tls_auth,/e2e_tests/web_config_TLSInLine_noAuth.yml,true,false)
	$(call e2e_tls_auth,/e2e_tests/web_config_TLS_Auth.yml,true,basic)
	$(call e2e_tls_auth,/e2e_tests/web_config_noTLS_Auth.yml,false,basic)
	$(call e2e_tls_auth,/e2e_tests/web_config_TLS_RequireAnyClientCert.yml,true,cert,$(ROOT_DIR)/e2e_tests)
	$(call e2e_tls_auth,/e2e_tests/web_config_TLS_RequireAndVerifyClientCert.yml,true,cert,$(ROOT_DIR)/e2e_tests)


.PHONY: build
build:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-mod=vendor -trimpath \
		-ldflags "$(LDFLAGS)" \
		-o $(APP_NAME) $(APP_NAME).go

.PHONY: build-arm
build-arm:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
		-mod=vendor -trimpath \
		-ldflags "$(LDFLAGS)" \
		-o $(APP_NAME) $(APP_NAME).go

.PHONY: build-darwin
build-darwin:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
		-mod=vendor -trimpath \
		-ldflags "$(LDFLAGS)" \
		-o $(APP_NAME) $(APP_NAME).go

.PHONY: dist
dist:
	- @mkdir -p dist
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.artifacts --progress=plain -t pgbackrest_exporter_dist .
	- @docker rm -f pgbackrest_exporter_dist 2>/dev/null || exit 0
	docker run -d --name=pgbackrest_exporter_dist pgbackrest_exporter_dist
	docker cp pgbackrest_exporter_dist:/artifacts dist/
	docker rm -f pgbackrest_exporter_dist

.PHONY: docker
docker:
	@echo "Build $(APP_NAME) docker container"
	@echo "Version $(BRANCH)-$(GIT_REV)"
	DOCKER_BUILDKIT=1 docker build --pull -f Dockerfile --build-arg REPO_BUILD_TAG=$(BRANCH)-$(GIT_REV) --build-arg BACKREST_VERSION=$(BACKREST_VERSION) --build-arg DOCKER_BACKREST_VERSION=$(DOCKER_BACKREST_VERSION) -t $(APP_NAME) .

.PHONY: docker-alpine
docker-alpine:
	@echo "Build $(APP_NAME) alpine docker container"
	@echo "Version $(BRANCH)-$(GIT_REV)"
	DOCKER_BUILDKIT=1 docker build --pull -f Dockerfile --build-arg REPO_BUILD_TAG=$(BRANCH)-$(GIT_REV) --build-arg BACKREST_VERSION=$(BACKREST_VERSION)-alpine --build-arg DOCKER_BACKREST_VERSION=$(DOCKER_BACKREST_VERSION) -t $(APP_NAME)-alpine .

.PHONY: prepare-service
prepare-service:
	@echo "Prepare config file $(APP_NAME).service for systemd"
	cp $(ROOT_DIR)/$(APP_NAME).service.template $(ROOT_DIR)/$(APP_NAME).service
	sed -i.bak "s|/usr/bin|$(ROOT_DIR)|g" $(APP_NAME).service
	rm $(APP_NAME).service.bak

.PHONY: install-service
install-service:
	@echo "Install $(APP_NAME) as systemd service"
	$(call service-install)

.PHONY: remove-service
remove-service:
	@echo "Delete $(APP_NAME) systemd service"
	$(call service-remove)

define service-install
	cp $(ROOT_DIR)/$(APP_NAME).service $(SERVICE_CONF_DIR)/$(APP_NAME).service
	systemctl daemon-reload
	systemctl enable $(APP_NAME)
	systemctl restart $(APP_NAME)
	systemctl status $(APP_NAME)
endef

define service-remove
	systemctl stop $(APP_NAME)
	systemctl disable $(APP_NAME)
	rm $(SERVICE_CONF_DIR)/$(APP_NAME).service
	systemctl daemon-reload
	systemctl reset-failed
endef

define e2e_basic
	docker run -d -p $(HTTP_PORT_E2E):$(HTTP_PORT) --name=$(APP_NAME)_e2e $(APP_NAME)_e2e
	@sleep 30
	$(ROOT_DIR)/e2e_tests/run_e2e.sh $(HTTP_PORT_E2E)
	docker rm -f $(APP_NAME)_e2e
endef

define e2e_exclude
	docker run -d -p $(HTTP_PORT_E2E):$(HTTP_PORT) --env STANZA_EXCLUDE="$(BACKREST_STANZA_EXCLUDE)" --name=$(APP_NAME)_e2e $(APP_NAME)_e2e
	@sleep 30
	$(ROOT_DIR)/e2e_tests/run_e2e.sh $(HTTP_PORT_E2E) false false "" exclude
	docker rm -f $(APP_NAME)_e2e
endef

define e2e_disable_collector
	docker run -d -p $(HTTP_PORT_E2E):$(HTTP_PORT) --env COLLECTOR_PGBACKREST="false" --name=$(APP_NAME)_e2e $(APP_NAME)_e2e
	@sleep 30
	$(ROOT_DIR)/e2e_tests/run_e2e.sh $(HTTP_PORT_E2E) false false "" disable-collector
	docker rm -f $(APP_NAME)_e2e
endef

define e2e_include
	docker run -d -p $(HTTP_PORT_E2E):$(HTTP_PORT) --env STANZA_INCLUDE="$(BACKREST_STANZA_INCLUDE)" --name=$(APP_NAME)_e2e $(APP_NAME)_e2e
	@sleep 30
	$(ROOT_DIR)/e2e_tests/run_e2e.sh $(HTTP_PORT_E2E) false false "" include
	docker rm -f $(APP_NAME)_e2e
endef

define e2e_tls_auth
	docker run -d -p $(HTTP_PORT_E2E):$(HTTP_PORT) --env EXPORTER_CONFIG="${1}" --name=$(APP_NAME)_e2e $(APP_NAME)_e2e
	@sleep 30
	$(ROOT_DIR)/e2e_tests/run_e2e.sh $(HTTP_PORT_E2E) ${2} ${3} "${4}"
	docker rm -f $(APP_NAME)_e2e
endef

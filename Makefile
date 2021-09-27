APP_NAME = pgbackrest_exporter
BRANCH_FULL=$(shell git rev-parse --abbrev-ref HEAD)
BRANCH=$(subst /,-,$(BRANCH_FULL))
GIT_REV=$(shell git describe --abbrev=7 --always)
SERVICE_CONF_DIR = /etc/systemd/system
HTTP_PORT = 9854
BACKREST_VERSION = 2.35
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

test:
	@echo "Run tests for $(APP_NAME)"
	go test -mod=vendor -timeout=60s -count 1  ./...

.PHONY: build
build:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags "-X main.version=$(BRANCH)-$(GIT_REV)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: build-arm
build-arm:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -mod=vendor -trimpath -ldflags "-X main.version=$(BRANCH)-$(GIT_REV)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: build-darwin
build-darwin:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -mod=vendor -trimpath -ldflags "-X main.version=$(BRANCH)-$(GIT_REV)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: dist
dist:
	- @mkdir -p dist
	docker build -f Dockerfile.artifacts --progress=plain -t pgbackrest_exporter_dist .
	- @docker rm -f pgbackrest_exporter_dist 2>/dev/null || exit 0
	docker run -d --name=pgbackrest_exporter_dist pgbackrest_exporter_dist
	docker cp pgbackrest_exporter_dist:/artifacts dist/
	docker rm -f pgbackrest_exporter_dist

.PHONY: docker
docker:
	@echo "Build $(APP_NAME) docker container"
	@echo "Version $(BRANCH)-$(GIT_REV)"
	docker build --pull -f Dockerfile --build-arg REPO_BUILD_TAG=$(BRANCH)-$(GIT_REV) --build-arg BACKREST_VERSION=$(BACKREST_VERSION) -t $(APP_NAME) .

.PHONY: prepare-service
prepare-service:
	@echo "Prepare config file $(APP_NAME).service for systemd"
	cp $(ROOT_DIR)/$(APP_NAME).service.template $(ROOT_DIR)/$(APP_NAME).service
	sed -i.bak "s|{PATH_TO_FILE}|$(ROOT_DIR)|g" $(APP_NAME).service
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


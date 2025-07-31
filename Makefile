# ==============================================================================
# Joyride - Docker-based DNS service with Serf clustering
# ==============================================================================

# Docker BuildKit optimizations
export DOCKER_BUILDKIT := 1
export DOCKER_SCAN_SUGGEST := false
export COMPOSE_DOCKER_CLI_BUILD := 1

# ==============================================================================
# Host IP Detection
# ==============================================================================
ifndef HOSTIP
	ifeq ($(OS),Windows_NT)
		HOSTIP := $(shell powershell -noprofile -command '(Get-NetIPConfiguration | Where-Object {$$_.IPv4DefaultGateway -ne $$null -and $$_.NetAdapter.Status -ne "Disconnected"}).IPv4Address.IPAddress' )
	else
		UNAME_S := $(shell uname)
		ifeq ($(UNAME_S),Linux)
			HOSTIP := $(shell ip route get 1 | head -1 | awk '{print $$7}' )
		endif
		ifeq ($(UNAME_S),Darwin)
			HOSTIP := $(shell ifconfig | grep "inet " | grep -Fv 127.0.0.1 | awk '{print $$2}' )
		endif
	endif
endif

export HOSTIP

# ==============================================================================
# Container Runtime Detection (Docker vs Podman)
# ==============================================================================
ifeq (, $(shell which podman))
	DOCKER_COMMAND := docker
	DOCKER_SOCKET := /var/run/docker.sock
else
	DOCKER_COMMAND := podman
	DOCKER_SOCKET := $(XDG_RUNTIME_DIR)/podman/podman.sock
endif

export DOCKER_COMPOSE
export DOCKER_COMMAND
export DOCKER_SOCKET

# ==============================================================================
# Semantic Versioning
# ==============================================================================
SEMVER_TAG := $(shell git tag --list 'v*.*.*' --sort=-v:refname | head -n 1)
VERSION := $(shell echo $(SEMVER_TAG) | sed 's/^v//')

define bump_version
	@echo "Latest version: $(SEMVER_TAG)"
	@NEW_VERSION=`echo $(VERSION) | awk -F. 'BEGIN {OFS="."} { \
		if ("$(1)" == "patch") {$$3+=1} \
		else if ("$(1)" == "minor") {$$2+=1; $$3=0} \
		else if ("$(1)" == "major") {$$1+=1; $$2=0; $$3=0} \
		print $$1, $$2, $$3}'` && \
	echo "New version: $$NEW_VERSION" && \
	git tag -a "v$$NEW_VERSION" -m "Release v$$NEW_VERSION" && \
	git push --tags && \
	echo "Tagged and pushed as v$$NEW_VERSION"
endef

# ==============================================================================
# Development Commands
# ==============================================================================

.PHONY: build up start restart down logs bash attach clean echo

build:				## Build the Docker image
	$(DOCKER_COMMAND) compose build

up: build			## Build and run containers in foreground (with logs)
	$(DOCKER_COMMAND) compose up --force-recreate --abort-on-container-exit --remove-orphans

start: build		## Build and start containers in background (daemon mode)
	$(DOCKER_COMMAND) compose up --force-recreate --remove-orphans -d

restart: build down start	## Rebuild, stop, and restart containers

down:				## Stop and remove containers
	$(DOCKER_COMMAND) compose down

logs:				## Follow container logs
	$(DOCKER_COMMAND) compose logs -f

bash: build			## Run interactive bash shell in new container
	$(DOCKER_COMMAND) compose run --rm joyride bash

attach:				## Attach to running container's bash shell
	$(DOCKER_COMMAND) compose exec joyride bash

clean:				## Remove containers, volumes, networks, and images
	$(DOCKER_COMMAND) compose down --volumes --remove-orphans --rmi local
	$(DOCKER_COMMAND) compose rm -f
	-$(DOCKER_COMMAND) image rm -f $(shell docker image ls -q --filter label=ilude-project=joyride)

echo:				## Display environment variables
	@echo "OS: $(OS)"
	@echo "HOSTIP: $(HOSTIP)"
	@echo "UPSTREAM_DNS: $(UPSTREAM_DNS)"

# ==============================================================================
# Serf Clustering
# ==============================================================================

.PHONY: serf

serf: build			## Run Serf agent for clustering
	$(DOCKER_COMMAND) compose run --rm joyride serf agent -advertise=$(HOSTIP):7946 -log-level=debug

# ==============================================================================
# Testing & Production
# ==============================================================================

.PHONY: test whoami production

test: build			## Run tests with whoami service
	$(DOCKER_COMMAND) compose -f docker-compose.yml -f docker-compose.whoami.yml up --force-recreate --abort-on-container-exit --remove-orphans

whoami:				## Run standalone whoami test service
	$(DOCKER_COMMAND) compose -f docker-compose.whoami.yml up --force-recreate --abort-on-container-exit --remove-orphans

production:			## Run in production mode
	$(DOCKER_COMMAND) compose -f docker-compose.production.yml up --force-recreate --abort-on-container-exit --remove-orphans

# ==============================================================================
# Version Management & Publishing
# ==============================================================================

.PHONY: bump-patch bump-minor bump-major publish

bump-patch:			## Increment patch version (x.y.Z+1) for bug fixes
	$(call bump_version,patch)

bump-minor:			## Increment minor version (x.Y+1.0) for new features
	$(call bump_version,minor)

bump-major:			## Increment major version (X+1.0.0) for breaking changes
	$(call bump_version,major)

publish: bump-patch	## Bump patch version and push all branches to remote
	@git push --all

# ==============================================================================
# Help
# ==============================================================================

.PHONY: help

help:				## Show this help message
	@echo "Joyride - Docker-based DNS service with Serf clustering"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Environment:"
	@echo "  HOSTIP: $(HOSTIP)"
	@echo "  DOCKER_COMMAND: $(DOCKER_COMMAND)"

.DEFAULT_GOAL := help

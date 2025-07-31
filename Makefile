# https://docs.docker.com/develop/develop-images/build_enhancements/
# https://www.docker.com/blog/faster-builds-in-compose-thanks-to-buildkit-support/
export DOCKER_BUILDKIT := 1
export DOCKER_SCAN_SUGGEST := false
export COMPOSE_DOCKER_CLI_BUILD := 1

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

# check if we should use podman compose or docker compose
# no one should be using docker-compose anymore
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

# -------------------------------
# Semantic version bumping logic
# -------------------------------
SEMVER_TAG := $(shell git tag --list 'v*.*.*' --sort=-v:refname | head -n 1)
VERSION := $(shell echo $(SEMVER_TAG) | sed 's/^v//')

define bump_version
  @echo "Latest version: $(SEMVER_TAG)"
  @NEW_VERSION=`echo $(VERSION) | awk -F. 'BEGIN {OFS="."} { \
		if ("$(1)" == "patch") {$3+=1} \
		else if ("$(1)" == "minor") {$2+=1; $3=0} \
		else if ("$(1)" == "major") {$1+=1; $2=0; $3=0} \
		print $1, $2, $3}'` && \
	echo "New version: $$NEW_VERSION" && \
	git tag -a "v$$NEW_VERSION" -m "Release v$$NEW_VERSION" && \
	git push --tags && \
	echo "Tagged and pushed as v$$NEW_VERSION"
endef

bump-patch:
	$(call bump_version,patch)

bump-minor:
	$(call bump_version,minor)

bump-major:
	$(call bump_version,major)

publish: bump-patch
	@git push --all

# -------------------------------
# Container commands
# -------------------------------

start: build
	$(DOCKER_COMMAND) compose up --force-recreate --remove-orphans -d

up: build 
	$(DOCKER_COMMAND) compose up --force-recreate --abort-on-container-exit --remove-orphans

production:  
	$(DOCKER_COMMAND) compose -f docker-compose.production.yml up --force-recreate --abort-on-container-exit --remove-orphans	

restart: build down start

down: 
	$(DOCKER_COMMAND) compose down

echo:
	@echo "OS: $(OS)"
	@echo "HOSTIP: $(HOSTIP)"
	@echo "UPSTREAM_DNS: $(UPSTREAM_DNS)"
	
logs:
	$(DOCKER_COMMAND) compose logs -f

bash: build 
	$(DOCKER_COMMAND) compose run --rm joyride bash

serf: build
	$(DOCKER_COMMAND) compose run --rm joyride serf agent -advertise=$(HOSTIP):7946 -log-level=debug

attach:
	$(DOCKER_COMMAND) compose exec joyride bash
	
build:
	$(DOCKER_COMMAND) compose build 

clean: 
	$(DOCKER_COMMAND) compose down --volumes --remove-orphans --rmi local
	$(DOCKER_COMMAND) compose rm -f
	-$(DOCKER_COMMAND) image rm -f $(shell docker image ls -q --filter label=ilude-project=joyride)

test: build
	$(DOCKER_COMMAND) compose -f docker-compose.yml -f docker-compose.whoami.yml up --force-recreate --abort-on-container-exit --remove-orphans

whoami: 
	$(DOCKER_COMMAND) compose -f docker-compose.whoami.yml up --force-recreate --abort-on-container-exit --remove-orphans

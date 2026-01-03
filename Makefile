# https://docs.docker.com/develop/develop-images/build_enhancements/
# https://www.docker.com/blog/faster-builds-in-compose-thanks-to-buildkit-support/
export DOCKER_BUILDKIT := 1
export DOCKER_SCAN_SUGGEST := false
export COMPOSE_DOCKER_CLI_BUILD := 1

# HOSTIP is now auto-detected inside the container (host network mode)
# Set it here only if you need to override the detected value
ifdef HOSTIP
export HOSTIP
endif
export UPSTREAM_DNS

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
	docker compose up --force-recreate --remove-orphans -d

up: build 
	docker compose up --force-recreate --abort-on-container-exit --remove-orphans

restart: build down start

down: 
	docker compose down

echo:
	@echo "HOSTIP: $(HOSTIP)"
	@echo "UPSTREAM_DNS: $(UPSTREAM_DNS)"
	
logs:
	docker compose logs -f

bash: build 
	docker compose run --rm joyride bash

attach:
	docker compose exec joyride bash
	
build:
	docker compose build 

clean: 
	docker compose down --volumes --remove-orphans --rmi local
	docker compose rm -f
	-docker image rm -f $(shell docker image ls -q --filter label=ilude-project=joyride)
	-rm Dockerfile

test: build
	docker compose -f docker compose.yml -f docker compose.whoami.yml up --force-recreate --abort-on-container-exit --remove-orphans

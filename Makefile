# https://docs.docker.com/develop/develop-images/build_enhancements/
# https://www.docker.com/blog/faster-builds-in-compose-thanks-to-buildkit-support/
export DOCKER_BUILDKIT := 1
export DOCKER_SCAN_SUGGEST := false
export COMPOSE_DOCKER_CLI_BUILD := 1

ifndef HOSTIP
	ifeq ($(OS),Windows_NT)
		HOSTIP := $(shell powershell -command '(Get-NetIPConfiguration | Where-Object {$$_.IPv4DefaultGateway -ne $$null -and $$_.NetAdapter.Status -ne "Disconnected"}).IPv4Address.IPAddress' )
#   UPSTREAM_DNS :=  $(shell powershell -command '(Get-NetRoute | where {$$_.DestinationPrefix -eq '0.0.0.0/0'} | select { $$_.NextHop }' )
	else
#   UPSTREAM_DNS = $(shell /sbin/ip route | awk '/default/ { print $$3 }')
		ifeq ($(UNAME_S),Linux)
				HOSTIP := $(shell ip route get 1 | head -1 | awk '{print $$7}' )
		endif
		ifeq ($(UNAME_S),Darwin)
				HOSTIP := $(shell ifconfig | grep "inet " | grep -Fv 127.0.0.1 | awk '{print $$2}' )
		endif
	endif
endif

export HOSTIP
export UPSTREAM_DNS

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

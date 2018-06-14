PACKAGES = $(shell go list ./... | grep -v /vendor/)
DONE = echo ">> $@ done"

DOCKER_TEAM_NAME 	?= operations-reliability
DOCKER_IMAGE_NAME   ?= prometheus-nagios-exporter
DOCKER_TAG 			?= latest

all: format build test

test: ## Run the tests 🚀.
	@echo ">> running tests"
	go test -short $(PACKAGES)
	@$(DONE)

style: ## Check the formatting of the Go source code.
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'
	@$(DONE)

format: ## Format the Go source code.
	@echo ">> formatting code"
	go fmt $(PACKAGES)
	@$(DONE)

vet: ## Examine the Go source code.
	@echo ">> vetting code"
	go vet $(PACKAGES)
	@$(DONE)

build: ## Build the Docker image.
	@echo ">> building the docker image"
	docker build -t "financial-times/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)" .
	@$(DONE)

run: ## Run the Docker image.
	@echo ">> building the docker image"
	docker run -p 9942:9942 "financial-times/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"
	@$(DONE)

publish: ## Push the docker image to the FT private repository.
	@echo ">> pushing the docker image"
	docker tag "financial-times/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)" "nexus.in.ft.com:5000/$(DOCKER_TEAM_NAME)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"
	docker push "nexus.in.ft.com:5000/$(DOCKER_TEAM_NAME)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)"
	@$(DONE)

help: ## Show this help message.
	@echo "usage: make [target] ..."
	@echo ""
	@echo "targets:"
	@grep -Eh '^.+:\ ##\ .+' ${MAKEFILE_LIST} | column -t -s ':#'

.PHONY: all style format build test vet docker

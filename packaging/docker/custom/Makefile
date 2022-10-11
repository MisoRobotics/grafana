.DEFAULT_GOAL := help

VERSION ?= latest

container := grafana
image := gcr.io/miso-cook-691c/$(container)

.PHONY: help
help: ## Show usage information for this Makefile.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the container image locally.
	docker build . \
		--build-arg "GRAFANA_VERSION=$(VERSION)" \
		--build-arg "GF_INSTALL_PLUGINS=grafana-clock-panel,grafana-simple-json-datasource,grafana-timestream-datasource" \
		-t $(image) -f Dockerfile


.PHONY: run
run: ## Run the container image locally.
	docker run --rm -it --publish 8080:8080 --name $(container) $(image)

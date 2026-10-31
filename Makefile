.DEFAULT_GOAL := help
.PHONY: help build test test-design lint swiftlint format format-check

help: ## Show this list
	@grep -hE '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-13s %s\n", $$1, $$2}'

build: ## Compile every target, app included, with warnings as errors
	swift build -Xswiftc -warnings-as-errors

test: ## Run the Core suite without building the app
	./Tools/test-core.sh

test-design: ## Run the OrbitDesign suite
	swift test --filter OrbitDesignTests

lint: ## Check the house rules, and the rules themselves
	./Tools/house-rules.sh --self-test
	./Tools/house-rules.sh

swiftlint: ## Run SwiftLint strictly against .swiftlint.yml
	./Tools/swiftlint.sh

format: ## Rewrite sources with swift-format
	./Tools/format.sh

format-check: ## Fail when a source is not formatted
	./Tools/format.sh --check

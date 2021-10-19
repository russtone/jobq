#
# Build
#

.PHONY: build
build:
	@go build

#
# Test
#

.PHONY: test
test:
	@go test -v -race -coverprofile coverage.out

#
# Coverage
#

.PHONY: coverage
coverage:
	@go tool cover -html coverage.out

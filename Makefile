#
# Build
#

.PHONY: build
build:
	@go build -o jobq .

#
# Test
#

.PHONY: test
test:
	@cd test && \
	 go generate && \
	 go test -v -coverprofile ../coverage.out

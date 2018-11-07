.PHONY: test.nocache test fmt vet lint setup vendor

PACKAGES := $(shell go list ./...)
DIRS := $(shell go list -f '{{.Dir}}' ./...)

setup:
	which dep > /dev/null 2>&1 || go get -u github.com/golang/dep/cmd/dep
	which goimports > /dev/null 2>&1 || go get -u golang.org/x/tools/cmd/goimports
	which golint > /dev/null 2>&1 || go get -u golang.org/x/lint/golint

vendor: vendor/.timestamp

vendor/.timestamp: $(shell find $(DIRS) -name '*.go')
	dep ensure -v
	touch vendor/.timestamp

vet:
	go vet $(PACKAGES)

lint:
	! find $(DIRS) -name '*.go' | xargs goimports -d | grep '^'
	echo $(PACKAGES) | xargs -n 1 golint -set_exit_status

fmt:
	find $(DIRS) -name '*.go' | xargs goimports -w

test:
	go test -v -race $(PACKAGES)

test.nocache:
	go test -count=1 -v -race $(PACKAGES)


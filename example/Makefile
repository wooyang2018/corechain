# init project PATH
HOMEDIR := $(shell pwd)
OUTDIR  := $(HOMEDIR)/output
COMPILECACHEDIR := $(HOMEDIR)/.compile_cache
TESTNETDIR := $(HOMEDIR)/testnet

# init command params
export GO111MODULE=on
X_ROOT_PATH := $(HOMEDIR)
export X_ROOT_PATH
export PATH := $(OUTDIR)/bin:$(PATH)

# make, make all
all: clean compile
# make compile, go build
compile: chain
chain:
	bash $(HOMEDIR)/auto/build.sh
# make test, test your code
test: unit
unit:
	go test -coverprofile=coverage.txt -covermode=atomic ./...
# make clean
cleanall: clean cleantest cleancache
clean:
	rm -rf $(OUTDIR)
cleantest:
	rm -rf $(TESTNETDIR)
cleancache:
	rm -rf $(COMPILECACHEDIR)
# deploy test network
testnet:
	bash $(HOMEDIR)/auto/testnet.sh
# avoid filename conflict and speed up build
.PHONY: all compile test clean

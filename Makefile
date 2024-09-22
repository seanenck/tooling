OS   := $(shell uname | tr '[:upper:]' '[:lower:]')
CMD  := OS=$(OS) go run build.go
OPTS := clean install

all:
	$(CMD)

$(OPTS):
	$(CMD) $@

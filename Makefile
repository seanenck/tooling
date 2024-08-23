CMD  := go run build.go
OPTS := clean install

all:
	$(CMD)

$(OPTS):
	$(CMD) $@

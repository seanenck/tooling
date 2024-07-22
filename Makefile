GOFLAGS := -trimpath -buildmode=pie -mod=readonly -modcacherw -buildvcs=false
BUILD   := build/
TARGETS := $(shell ls cmd/)

all: $(TARGETS)

clean:
	rm -rf $(BUILD)

$(TARGETS): go.mod $(shell find cmd/ -type f)
	go build $(GOFLAGS) -o $(BUILD)$@ cmd/$@/main.go

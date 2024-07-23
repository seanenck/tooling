GOFLAGS := -trimpath -buildmode=pie -mod=readonly -modcacherw -buildvcs=false
BUILD   := build/
TARGETS := $(BUILD)git-uncommitted $(BUILD)git-current-state
DESTDIR :=

all: $(TARGETS)

clean:
	rm -rf $(BUILD)

$(TARGETS): go.mod $(shell find cmd/ -type f)
	go build $(GOFLAGS) -o $@ cmd/$(shell basename $@)/main.go

install:
	@for file in $(shell ls $(BUILD)) ; do \
		echo $$file; \
		install -m755 $(BUILD)$$file $(DESTDIR)/$$file; \
	done

GOFLAGS := -trimpath -buildmode=pie -mod=readonly -modcacherw -buildvcs=false
BUILD   := build/
TARGETS := $(addprefix $(BUILD),$(shell ls cmd/))
DESTDIR := $(HOME)/.local/bin/

all: $(TARGETS)

clean:
	rm -rf $(BUILD)

$(TARGETS): go.mod generated.template $(shell find cmd/ -type f)
	cp generated.template cmd/$(shell basename $@)/generated.go
	go build $(GOFLAGS) -o $@ cmd/$(shell basename $@)/*.go

install: $(TARGETS)
	@for file in $(shell ls $(BUILD)) ; do \
		echo $$file; \
		install -m755 $(BUILD)$$file $(DESTDIR)/$$file; \
	done

_OS     := $(shell uname | tr '[:upper:]' '[:lower:]')
OS      := $(_OS)
TARGET  := target
BUILD   := $(TARGET)/$(OS)
INSTALL := $(BUILD)/Makefile

all:
	BUILDDIR=$(BUILD) OS=$(OS) go run build.go

clean:
	rm -rf $(TARGET)

install:
ifneq ($(_OS), $(OS))
	$(error "can not install, $(_OS) != $(OS)")
endif
	test -e $(INSTALL) && make -C $(dir $(INSTALL))

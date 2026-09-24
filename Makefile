PREFIX ?= $(HOME)/.local
BIN ?= $(PREFIX)/bin
SHARE ?= $(PREFIX)/share
ifeq ($(PREFIX),$(HOME)/.local)
FISH ?= $(HOME)/.config/fish/completions
else
FISH ?= $(SHARE)/fish/vendor_completions.d
endif
BASH ?= $(SHARE)/bash-completion/completions
ZSH ?= $(SHARE)/zsh/site-functions

sio: go.mod $(shell find . -name "*.go" -o -name "*.jsonc")
	go build -o sio ./cmd/sio

install: sio
	mkdir -p $(BIN) $(FISH) $(BASH) $(ZSH)
	cp sio $(BIN)/sio
	cp completions/sio.fish $(FISH)/sio.fish
	cp completions/sio.bash $(BASH)/sio
	cp completions/_sio $(ZSH)/_sio

uninstall:
	rm -f $(BIN)/sio $(FISH)/sio.fish $(BASH)/sio $(ZSH)/_sio

clean:
	rm -f sio

.PHONY: install uninstall clean

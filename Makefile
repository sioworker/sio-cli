BIN := $(HOME)/.local/bin
FISH := $(HOME)/.config/fish/completions
BASH := $(HOME)/.local/share/bash-completion/completions
ZSH := $(HOME)/.local/share/zsh/site-functions

install:
	go build -o $(BIN)/sio ./cmd/sio
	mkdir -p $(FISH) $(BASH) $(ZSH)
	cp completions/sio.fish $(FISH)/sio.fish
	cp completions/sio.bash $(BASH)/sio
	cp completions/_sio $(ZSH)/_sio

.PHONY: install

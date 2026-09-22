BIN := $(HOME)/.local/bin
FISH := $(HOME)/.config/fish/completions

install:
	go build -o $(BIN)/sio ./cmd/sio
	mkdir -p $(FISH)
	cp completions/sio.fish $(FISH)/sio.fish

.PHONY: install

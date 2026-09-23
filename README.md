# Sio CLI

CLI for submitting to [OIOIOI](https://github.com/sio2project/oioioi)

# Install

<details>
<summary><b>Autobuild</b></summary>

<br>

Grabs the newest build for your OS/arch (Linux, macOS) and installs it to `~/.local/bin/sio` with fish/bash/zsh completions:

```sh
curl -fsSL https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.sh | sh
```

You can also specify where to put the binary:

```sh
curl -fsSL https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.sh | SIO_BIN=/usr/local/bin/ sh
```

It asks which completions to install (the shells you have are ticked already, bash only if its your shell). To skip the question:

```sh
curl -fsSL https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.sh | SIO_COMP="fish zsh" sh # or SIO_COMP=none
```

</details>

<details>
<summary><b>Source</b></summary>

<br>

Needs Go 1.27.1+, git and make:

```sh
git clone https://github.com/sioworker/sio-cli && cd sio-cli && make install
```

</details>

<details>
<summary><b>Nix Flake</b></summary>

<br>

Needs flakes enabled (`experimental-features = nix-command flakes`).

Run without installing:

```sh
nix run github:sioworker/sio-cli -- info
```

Install:

```sh
nix profile install github:sioworker/sio-cli
```

Dev shell with Go + gopls:

```sh
nix develop github:sioworker/sio-cli
```

</details>

<details>
<summary><b>AUR</b></summary>

<br>

TODO (currently unavailable)

</details>

<details>
<summary><b>Windows</b></summary>

<br>

Currently unsupported

</details>

# Usage

[TO-DO]

Run `sio` for all cmds.

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

It asks which completions to install, to skip the question:

```sh
curl -fsSL https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.sh | SIO_COMP=none sh
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

In PowerShell (installs to `%LOCALAPPDATA%\Programs\sio`, adds it to your PATH):

```powershell
irm https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.ps1 | iex
```

`$env:SIO_BIN` and `$env:SIO_COMP` (`pwsh` / `none`) work like on linux. Still experimental, tell us if something breaks.

</details>

# Usage

## Setup

Add a host with your API token, which you get from `<host>/api/token` while logged in on the site:

```sh
sio config hosts add szkopul szkopul.edu.pl   # asks for the token
sio config hosts add --main oboz oboz.talent.edu.pl
sio ping                                      # ✓ oboz: logged in as ...
```

The first host you add becomes the main one; `--main` or `sio config hosts main <name>` switch it. Any command that takes a contest also takes `host/contest` to pick a host just for that call, e.g. `sio probs szkopul/kurs-oi`.

Older OIOIOI instances (like the camp one) accept submissions over the API but can't report results. To still see your verdicts in the terminal, log in once:

```sh
sio config hosts login oboz   # username + password, only the session gets saved
```

## Browsing

```sh
sio info          # who you're logged in as + the contests you can see
sio tree          # browse contests -> problems, see keys below
sio probs c1      # problems in c1, with your score and submissions left
sio subs c1       # your submissions in c1, newest first
sio subs c1 abc   # only for problem abc
```

`sio tree` keys:

| key | does |
|---|---|
| ↑↓ / j k | move |
| space / enter | open or close a contest, open a problem in the browser |
| ← → / h l | close / open, ← on a problem jumps to its contest |
| u | submit a file to the selected problem |
| g G, PgUp PgDn | jump |
| q | quit |

## Submitting

```sh
sio submit c1 abc          # finds abc.cpp (or the only abc.*) in the current dir
sio submit c1 abc sol.py   # or name the file yourself
sio sub c1 abc.cpp         # sub = submit, prob is taken from the file name
sio submit -n c1 abc       # dont wait for the results
```

It waits until the submission is judged and shows the verdict, colored by score: red under 50, yellow under 80, green from 80 up.

## Config

```sh
sio config                 # overview: config file, hosts, language, quotes
sio config hosts           # list hosts, ● = main
sio config hosts rm <name>
sio config hosts token <name>
sio config lang pl         # en, pl, lolcat(best)
sio config quotes off      # the random programming quote after each command
```

Everything lives in `~/.config/sio/cfg.json`. Your own translations go in `~/.config/sio/lang/<code>.jsonc`.

Run `sio` for all cmds.

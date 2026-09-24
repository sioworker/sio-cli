# Sio CLI

CLI for submitting to [OIOIOI](https://github.com/sio2project/oioioi)

# Install

<details>
<summary><b>Autobuild</b></summary>

<br>

Grabs the newest autobuild:

```sh
curl -fsSL https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.sh | sh
```

Specify where to put the binary:

```sh
curl -fsSL https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.sh | SIO_BIN=/usr/local/bin/ sh
```

To skip the shell completions question:

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

Registration on the AUR is paused while they deal with a wave of automated account creation. This is a temporary measure and it is not specific to you or your network. There's no manual registration queue, and we will not be able to respond to requests for new accounts during this time.

</details>

<details>
<summary><b>Windows</b></summary>

<br>

In PowerShell (installs to `%LOCALAPPDATA%\Programs\sio`):

```powershell
irm https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.ps1 | iex
```

`$env:SIO_BIN` and `$env:SIO_COMP` (`pwsh` / `none`) work like on linux. Still experimental, tell us if something breaks.

</details>

# Usage

## Setup

```sh
sio config hosts add szkopul szkopul.edu.pl   # asks for the token
sio config hosts add --main oboz oboz.talent.edu.pl
sio ping                                      # ✓ oboz: logged in as ...
```

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

## Config

```sh
sio config                 # overview: config file, hosts, language, quotes
sio config hosts           # list hosts, ● = main
sio config hosts rm <name>
sio config hosts token <name>
sio config lang pl         # en, pl, lolcat(best)
sio config quotes off      # the random programming quote after each command
```

Run `sio` for all cmds.

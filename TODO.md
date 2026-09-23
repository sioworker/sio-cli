# TODO

## output
- [x] colors (ok=green, err=red, host/contest/prob=cyan), off when not a tty or `NO_COLOR` set
- [x] nicer msgs, icons + boxes, basically make it look like ai slop (✓ submitted, ✗ 401: bad token, etc)
- [ ] spinner while uploading
- [ ] `--json` flag for scripts
- [x] langs (`lang/*.jsonc`, `sio config lang`, `SIO_LANG`/`LANG`, overrides in `~/.config/sio/lang/`)

## statements / tests
- [ ] `sio pdf <contest> <prob>` - get the pdf
- [ ] parse the pdf (pdftotext or a go lib) and pull the sample in/out into `tests/<prob>0.in`/`.out`
- [ ] (or if you can js get 0 tests from api do that)
- [ ] `sio test <prob> <bin>` - run the bin on the samples locally, diff vs `.out`, before uploading - stonks (make it look like ai slop 2)

## results
- [ ] after upload, poll the submission status and print the score instead of just the id
- [x] `sio subs` - list own submissions for a contest
- [x] `sio probs <contest>` - list problems (short name + title)

## misc
- [ ] remember last contest per dir so `sio up <prob> <file>` works without it
- [ ] guess prob from filename (`abc.cpp` -> `abc`)
- [ ] update completions for any new cmds
- [ ] AUR pkg

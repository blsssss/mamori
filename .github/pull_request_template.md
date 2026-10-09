## What changes

## Why

## How it was checked

- [ ] `make check`
- [ ] `make build` and a run of the changed check in the app
- [ ] internal/fwprobe or internal/avprobe changed: run in the test VM with Defender, Windows Firewall and a third-party antivirus, each on and off
- [ ] no test string (EICAR, AMSI) in plain text in the source or the binary
- [ ] docs/ru and docs/en updated together

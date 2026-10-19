# Gates: preserve donor Steam identity for the target

OWNS: GATES.md, internal/launch/command.go, internal/launch/command_test.go, internal/session/helper_test.go, docs/architecture.md, GLOSSARY.md, docs/compatibility.md, .scratch/better-rpt/issues/03-proton-launch-session.md, .scratch/better-rpt/issues/15-proton-command-builder.md, .scratch/better-rpt/notes/03-proton-launch-session-proof.md, .scratch/better-rpt/notes/steam-mechanics.md

Scope: make the helper launch the target with the donor's Steam identity so Steam registers only SpaceWar and Remote Play Together guests are not told the host is busy.

- [x] G1: the gate ledger is lint-clean
  CHECK: node /home/srnoob/.agents/skills/unlazy/scripts/gate-lint.mjs GATES.md
  EXPECT: LINT OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=7ac42e5bc687311a4301536eea2e5c645e79aaf039958f397bc5e5b76984d29a; exit=0; EXPECT=matched; output-sha256=1166862ed4bd530892551c8fb62b876dd866137927e0d25bb987cc29dd89acf4; output-bytes=268; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=c2a2a5586ce5/20 entries

- [x] G2: BuildCommand preserves SteamAppId/SteamGameId from the donor environment for native and Proton targets and never writes the target AppID into Steam identity variables
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./internal/launch -count=1 && printf 'LAUNCH_ENV_TESTS_OK\n'
  EXPECT: LAUNCH_ENV_TESTS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=ce1ed7a6cdebbd5619d654d8fccbaeb91c6e18016555c6ae0085deffd19f9c5a; exit=0; EXPECT=matched; output-sha256=f4d519dbc0ea8a63c031a2ae1024c603380f6c15cf64a81facac7269b7094053; output-bytes=81; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=c2a2a5586ce5/20 entries

- [x] G3: the full Go test suite passes
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./... && printf 'GO_TESTS_OK\n'
  EXPECT: GO_TESTS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=965391a7a51fa310cec9c40ab90067c6543b36c6f19e7ee485f5f7980d9f80e3; exit=0; EXPECT=matched; output-sha256=e98dbc152412e70ea21f7f1b509fe37844071ced9c6cce723dd89d8555a2f09a; output-bytes=1049; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=c2a2a5586ce5/20 entries

- [x] G4: go vet reports no issues
  CHECK: GOTOOLCHAIN=go1.27.2 go vet ./... && printf 'GO_VET_OK\n'
  EXPECT: GO_VET_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=d52ca2cb8fd062edfb95edfd5b7adefe34e3687bb7a5ed16e2d64e553ced1ac4; exit=0; EXPECT=matched; output-sha256=a267a9eb927f181e192d863ee6d20d3444909ccf2c46121a5664fbaa543e65e5; output-bytes=10; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=c2a2a5586ce5/20 entries

- [x] G5: the GUI and helper packages build
  CHECK: GOTOOLCHAIN=go1.27.2 go build ./... && printf 'GO_BUILD_OK\n'
  EXPECT: GO_BUILD_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=66c4c627af021d834abb0ece32bbffeb1f29689e84104d35872e229ae4c4b5de; exit=0; EXPECT=matched; output-sha256=250b9b2b0f161708a5cce8d225769628bbf63e1165683eaf77779d85c78eaea5; output-bytes=12; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=c2a2a5586ce5/20 entries

- [x] G6: tracked documentation states that the target inherits the donor Steam identity and uses its AppID only for launch paths, not for Steam presence
  EVIDENCE: reviewed docs/architecture.md, GLOSSARY.md and docs/compatibility.md on 2026-10-19; all three state that the target inherits SteamAppId/SteamGameId from the donor and that its AppID is used only for executable/prefix paths, never for Steam presence.

- [ ] G7: a real Steam session tracks the donor as the only running game and a Remote Play Together guest connects without the host-busy message
  EVIDENCE: pending

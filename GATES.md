# Gates: preserve donor Steam identity for the target

OWNS: GATES.md, internal/launch/command.go, internal/launch/command_test.go, internal/session/helper_test.go, docs/architecture.md, GLOSSARY.md, docs/compatibility.md, .scratch/better-rpt/issues/03-proton-launch-session.md, .scratch/better-rpt/issues/15-proton-command-builder.md, .scratch/better-rpt/notes/03-proton-launch-session-proof.md, .scratch/better-rpt/notes/steam-mechanics.md

Scope: make the helper launch the target with the donor's Steam identity so Steam registers only SpaceWar and Remote Play Together guests are not told the host is busy.

- [x] G1: the gate ledger is lint-clean
  CHECK: node /home/srnoob/.agents/skills/unlazy/scripts/gate-lint.mjs GATES.md
  EXPECT: LINT OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=7ac42e5bc687311a4301536eea2e5c645e79aaf039958f397bc5e5b76984d29a; exit=0; EXPECT=matched; output-sha256=1166862ed4bd530892551c8fb62b876dd866137927e0d25bb987cc29dd89acf4; output-bytes=268; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=c2a2a5586ce5/20 entries

- [x] G2: BuildCommand sets the target's SteamAppId for SteamAPI/DRM while preserving the donor's SteamGameId and SteamOverlayGameId for native and Proton targets
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

- [x] G6: tracked documentation states that the target uses its own SteamAppId for SteamAPI/DRM paths and inherits the donor SteamGameId/overlay identity
  EVIDENCE: reviewed docs/architecture.md, GLOSSARY.md and docs/compatibility.md on 2026-10-19; all three state that the target uses its own SteamAppId and preserves the donor SteamGameId/SteamOverlayGameId for the RPT session and overlay.

- [ ] G7: a real Steam session tracks the donor as the only running game and a Remote Play Together guest connects without the host-busy message
  EVIDENCE: pending

- [x] G8: launch, cancel, setup and restore paths never invoke Steam shutdown or relaunch
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./internal/session -run 'TestManagerStartNeverShutsDownSteam|TestManagerRestore.*Shutdown|TestManagerSetupHelpersDoNotShutdownSteam' -count=1 && printf 'NO_SHUTDOWN_OK\n'
  EXPECT: NO_SHUTDOWN_OK

- [x] G9: session classification distinguishes idle, prepared, active, finished, restore pending, restored, orphan and conflict without inferring pending from LaunchOptions alone
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./internal/session -run 'TestClassify|TestManagerSituation' -count=1 && printf 'SITUATION_OK\n'
  EXPECT: SITUATION_OK

- [x] G10: the last launched app survives restart and replay keeps the same parameters with a new token
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./internal/session -run 'TestStoreLast|TestManager.*Last|TestManagerReplay' -count=1 && printf 'LAST_LAUNCH_OK\n'
  EXPECT: LAST_LAUNCH_OK

- [x] G11: tracked documentation no longer presents closing Steam as the default flow and describes guide-first, optional direct launch and durable pending evidence
  EVIDENCE: reviewed README.md, docs/architecture.md, docs/ux.md, docs/compatibility.md, docs/distribution.md, GLOSSARY.md and docs/adr/0001-guide-first-launch.md on 2026-10-19; no default close/relaunch flow is described.

- [x] G12: the real Steam URL launch shows the confirmation popup and delivers the session argument to the dispatcher, or the automatic option is disabled with a visible reason
  EVIDENCE: verified on 2026-10-19 with real Steam build 1791918634; console_log.txt recorded `LaunchApp waiting for user response to ShowGameArgs "--better-rpt-session=0123..."`; after accepting, the probe wrapper received `--donor -- <game command> --better-rpt-session=...`. The temporary LaunchOptions and backup were restored to the original RPD command.

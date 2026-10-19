# Gates: Relay Link brand kit and app rollout

OWNS: packaging/brand/**, packaging/better-rpt.svg, packaging/better-rpt.png, packaging/icons/**, packaging/nfpm.yaml, internal/ui/assets/better-rpt.png, internal/ui/assets/better-rpt-small.png, internal/ui/theme.go, internal/ui/dashboard.go, scripts/verify-brand.py, GATES.md

Scope: produce the approved Relay Link identity kit and install it in the Fyne app, Linux desktop metadata and package icon ladder without changing runtime behavior.

- [x] G0: this completion ledger is lint-clean
  CHECK: node /home/srnoob/.agents/skills/unlazy/scripts/gate-lint.mjs GATES.md
  EXPECT: LINT OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=7ac42e5bc687311a4301536eea2e5c645e79aaf039958f397bc5e5b76984d29a; exit=0; EXPECT=matched; output-sha256=958052e1be78d9acdd6d8ab18bb3126f4a183151119a69de38c6f4b36cac193d; output-bytes=150; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G1: the brand kit contains the complete colour, mono, reversed, lockup, small-size, favicon, palette and guide family
  CHECK: python3 scripts/verify-brand.py kit
  EXPECT: BRAND_KIT_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=832b4394fff3aeeff4527eb67890ac51143d070d88f26b3660c2f834379c0dc9; exit=0; EXPECT=matched; output-sha256=e733df260cd3576cd67ba44a5c7473e2c3029e9bc4c5d3f64fa12006e09db30a; output-bytes=13; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G2: the app icon, embedded header icon and hicolor size ladder are the approved Relay Link
  CHECK: python3 scripts/verify-brand.py app-icons
  EXPECT: APP_ICONS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=6f77dd0265ffc15de2c051a9e4542abced89c9e03820e9f02dcf6eeba2d97e9d; exit=0; EXPECT=matched; output-sha256=7bd7239c816898bf7f009af37a0cea1620d88571e517a8d20dbe75d645a5f005; output-bytes=13; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G3: the Fyne app and Linux package consume the new icon assets
  CHECK: python3 scripts/verify-brand.py integration
  EXPECT: APP_INTEGRATION_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=4c6ba3ab00c25de8ff21590fab04da4f35acac7a9258b7a5babc2ead52f743f4; exit=0; EXPECT=matched; output-sha256=26eddebae3d31ad00c923b9963af14b39b90eb1e2859875508054e841e6fd3e6; output-bytes=19; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G4: the brand verifier rejects known-bad SVGs instead of passing absence cases blindly
  CHECK: python3 scripts/verify-brand.py selftest
  EXPECT: BRAND_SELFTEST_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=e110307a0bf3cd8190844ff2915dafdf4b8e0490563e3c7ce61ec512728a86c9; exit=0; EXPECT=matched; output-sha256=221ff53b87793c9dddd319cdd7e07fd76178305f524957a4a654ccdedafc622f; output-bytes=18; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G5: the Go test suite passes
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./... && printf 'GO_TESTS_OK\n'
  EXPECT: GO_TESTS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=965391a7a51fa310cec9c40ab90067c6543b36c6f19e7ee485f5f7980d9f80e3; exit=0; EXPECT=matched; output-sha256=6660f50f1c8c6954f1575901dec88852e909cffe49fb07d39552a724a416369a; output-bytes=1053; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G6: the GUI and helper packages build
  CHECK: GOTOOLCHAIN=go1.27.2 go build ./... && printf 'GO_BUILD_OK\n'
  EXPECT: GO_BUILD_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=66c4c627af021d834abb0ece32bbffeb1f29689e84104d35872e229ae4c4b5de; exit=0; EXPECT=matched; output-sha256=250b9b2b0f161708a5cce8d225769628bbf63e1165683eaf77779d85c78eaea5; output-bytes=12; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G7: the board and the small-size icon cuts were visually reviewed against the approved Relay Link
  EVIDENCE: reviewed packaging/brand/brand-board.png, icon-16.png, icon-32.png and the 28 px header preview on 2026-10-19; the board, outlined wordmark, one-colour/reversed marks and small-size ladder match the approved Relay Link.

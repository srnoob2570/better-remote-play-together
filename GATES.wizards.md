# Gates: setup and repair wizards

OWNS: GATES.wizards.md, cmd/better-rpt/main.go, internal/app/app.go, internal/app/session.go, internal/app/session_test.go, internal/bridge/bridge.go, internal/bridge/bridge_test.go, internal/diagnostics/checks_app.go, internal/diagnostics/testdata/fedora-appimage.json, internal/i18n/catalogs/es.json, internal/i18n/catalogs/en.json, frontend/src/App.tsx, frontend/src/App.test.tsx, frontend/src/bridge/client.ts, frontend/src/bridge/wails.ts, frontend/bindings/github.com/srnoob2570/better-rpt/internal/bridge/bridge.ts, frontend/src/clipboard.ts, frontend/src/docs.ts, frontend/src/diagnostics/useDiagnostics.ts, frontend/src/guide.ts, frontend/src/components/DispatcherSetup.tsx, frontend/src/components/SetupIndicator.tsx, frontend/src/components/DiagnosticsPane.tsx, frontend/src/components/SessionPanel.tsx, frontend/src/session/useSession.ts, frontend/src/styles.css, frontend/src/wizards/**, frontend/src/e2e/mockTransport.ts, frontend/e2e/wizards.spec.ts, frontend/e2e/a11y.spec.ts, docs/appimage.md, docs/distribution.md, README.md, docs/ux.md, docs/architecture.md, frontend/e2e/visual.spec.ts-snapshots/visual-guide-1024x600-linux.png

Scope: add a durable setup wizard and an on-demand repair wizard, make the packaged helper install at startup, and reuse the existing setup, guide, diagnostics and restore flows without deleting user state.

- [x] G0: the wizard gate ledger is lint-clean
  CHECK: node /home/srnoob/.agents/skills/unlazy/scripts/gate-lint.mjs GATES.wizards.md
  EXPECT: LINT OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=db215cf1afed6f2e650e788d743d759413576e9f36b341c8a83a594784a3900a; exit=0; EXPECT=matched; output-sha256=9aba413f4bf824c5a19204d3808a534b7458eb109e68644e11012a3b4f99586e; output-bytes=158; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G1: the Go suite passes with the helper startup path and bridge changes
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./... && printf 'GO_TESTS_OK\n'
  EXPECT: GO_TESTS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=965391a7a51fa310cec9c40ab90067c6543b36c6f19e7ee485f5f7980d9f80e3; exit=0; EXPECT=matched; output-sha256=3a3fab4215333b556493e9b2cadbf91c270ffe2b80358a9fe04e25100d167857; output-bytes=1381; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G2: the Go tree builds and vets cleanly
  CHECK: GOTOOLCHAIN=go1.27.2 go vet ./... && GOTOOLCHAIN=go1.27.2 go build ./... && printf 'GO_BUILD_VET_OK\n'
  EXPECT: GO_BUILD_VET_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=ef6fa7f7218a82dc8471c487d4be98171d725042b8c04912d417285b048a7d87; exit=0; EXPECT=matched; output-sha256=269a60dfff0fc332ed03926d788db775a653523b8942a8dfe1ccb8f1da396bda; output-bytes=16; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G3: the frontend unit suite passes with the wizard state and dialog interactions
  CHECK: NO_COLOR=1 npm test
  EXPECT: /Tests +[0-9]+ passed/
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=0869df0fcceb7f21c01fa2a853a176d086c313f798c0251f86b2db4d66143ae3; exit=0; EXPECT=matched; output-sha256=f7277f59b9489c4ac8ee49906cb612e2258411440f7e416860f708b105af7435; output-bytes=322; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G4: the production frontend build succeeds
  CHECK: NO_COLOR=1 npm run build
  EXPECT: built in
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=098f4b5c9671af99f330f1fee1b5812841c2ef596388c9473f1bdd861cb23404; exit=0; EXPECT=matched; output-sha256=1c4b5fb42ad63c280160e00b059ef18c7811674dd6edf19f62c554a60449c23f; output-bytes=828; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G5: the wizard parity and accessibility browser checks pass
  CHECK: PLAYWRIGHT_PORT=9254 NO_COLOR=1 npx playwright test --output=/tmp/pw-wizards e2e/wizards.spec.ts e2e/a11y.spec.ts
  EXPECT: /^ *[0-9]+ passed \(/m
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=2721c833d5f1a2b6af3b9a8815f11e4e1e8768a53b25e6ce9aadba80d2316956; exit=0; EXPECT=matched; output-sha256=4f06ba00b5f3bc9fa965d81b16ff2f28861431f149992621335534d3fb0fe149; output-bytes=1724; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G6: the existing unrelated gate ledger is unchanged
  CHECK: git diff --quiet -- GATES.md && echo "GATES.md unchanged"
  EXPECT: GATES.md unchanged
  EVIDENCE: automatic-evidence=v1; definition-sha256=2145d5c64a69774ca94dc628fcdaa055340f481ab79ac5a497028ee55d35cc38; exit=0; EXPECT=matched; output-sha256=aa3195128058d39782d864c505a7a98e7e8db14dd2e733482147e98be49da149; output-bytes=19; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G7: visual and keyboard review confirms the setup wizard is one focused sequential flow and the repair wizard is a read-only issue workbench on both desktop and narrow widths
  EVIDENCE: Reviewed /tmp/wizard-setup-1280.png, /tmp/wizard-setup-1024.png, /tmp/wizard-dependencies-1280.png, /tmp/wizard-repair-1280.png and /tmp/wizard-repair-1024.png on 2026-10-20. The setup wizard keeps one active step, statuses are text plus color, the footer stays visible at 1024x600, and the dependency command fits without horizontal overflow. The repair wizard keeps the issue list and detail side by side at 1280 and stacks them at 1024, with the read-only banner visible and the command reachable by scrolling. Keyboard coverage is exercised by the a11y Escape/Tab tests and the wizard resume test.

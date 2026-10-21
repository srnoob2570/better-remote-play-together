# Gates: launch feedback and Steam focus

OWNS: GATES.launch-feedback.md, internal/steam/lifecycle.go, internal/steam/lifecycle_test.go, frontend/src/session/useSession.ts, frontend/src/components/SessionPanel.tsx, frontend/src/App.test.tsx, frontend/e2e/parity.spec.ts, frontend/e2e/visual.spec.ts-snapshots/**, internal/i18n/catalogs/es.json, internal/i18n/catalogs/en.json, docs/ux.md, docs/architecture.md, README.md, CHANGELOG.md

Scope: show an immediate launching state with a Steam confirmation prompt, and ask Steam to focus its window after the launch request, without failing the launch if activation is denied.

- [x] G0: this ledger is structurally valid
  CHECK: node /home/srnoob/.agents/skills/unlazy/scripts/gate-lint.mjs GATES.launch-feedback.md
  EXPECT: LINT OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=5461e8ce5a6ff0a106c62a3e1f4239551ba1eaab25f348abe654545eb8a716c1; exit=0; EXPECT=matched; output-sha256=48630b7361dd44ee870917b12c3d19b9d7bdea738aaca16bb04d4cab83b772d2; output-bytes=8; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G1: the Steam launch requests the focus window after the run URL and a focus failure does not fail the launch
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./internal/steam -run 'TestAppLaunchSession' -count=1 && printf 'STEAM_FOCUS_OK\n'
  EXPECT: STEAM_FOCUS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=73fe5e65bd809295e51189cf6f46e19a504081b0fb3acbe08eb20ab7f0e8c53c; exit=0; EXPECT=matched; output-sha256=440d2347f3d1a4cb9b19214aa2942abbe9d6fcc10d19ab5c421da79d96fb3233; output-bytes=75; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G2: the full Go test suite still passes
  CHECK: GOTOOLCHAIN=go1.27.2 go test ./... && printf 'GO_TESTS_OK\n'
  EXPECT: GO_TESTS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=965391a7a51fa310cec9c40ab90067c6543b36c6f19e7ee485f5f7980d9f80e3; exit=0; EXPECT=matched; output-sha256=3a3fab4215333b556493e9b2cadbf91c270ffe2b80358a9fe04e25100d167857; output-bytes=1381; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G3: go vet reports no issues
  CHECK: GOTOOLCHAIN=go1.27.2 go vet ./... && printf 'GO_VET_OK\n'
  EXPECT: GO_VET_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=d52ca2cb8fd062edfb95edfd5b7adefe34e3687bb7a5ed16e2d64e553ced1ac4; exit=0; EXPECT=matched; output-sha256=a267a9eb927f181e192d863ee6d20d3444909ccf2c46121a5664fbaa543e65e5; output-bytes=10; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G4: the frontend unit suite proves the launching label and Steam prompt
  CHECK: NO_COLOR=1 npm test
  EXPECT: /Tests +[0-9]+ passed/
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=0869df0fcceb7f21c01fa2a853a176d086c313f798c0251f86b2db4d66143ae3; exit=0; EXPECT=matched; output-sha256=ab3d183b9d2c2fd2c9d008f3b9b93e6126709f0151b7e04a8a09b395c43a6198; output-bytes=322; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G5: the production frontend build succeeds
  CHECK: NO_COLOR=1 npm run build
  EXPECT: built in
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=098f4b5c9671af99f330f1fee1b5812841c2ef596388c9473f1bdd861cb23404; exit=0; EXPECT=matched; output-sha256=662525169a3ad9fdf7c5faa40c72822069c0b3bd73acd32e9c15627a299f288b; output-bytes=828; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G6: the preparing flow and its updated visual snapshots pass in the browser
  CHECK: PLAYWRIGHT_PORT=9254 NO_COLOR=1 npx playwright test --output=/tmp/pw-gates-launch --grep "preparing" e2e/parity.spec.ts e2e/visual.spec.ts
  EXPECT: /^ *[0-9]+ passed \(/m
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=893816fdfcd0585e379eb36f2ffb6925e1be2563eac1c030683b9dcdcc43db46; exit=0; EXPECT=matched; output-sha256=80960a9e62c5ee083838ee8633e0daf8a02f79fa59a6567d8a28d3d35f0775db; output-bytes=1049; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G7: the UI documentation describes the launching label, the Steam prompt and the best-effort focus
  CHECK: node -e "const fs=require('fs'); const checks=[['docs/ux.md','## Lanzamiento'],['docs/ux.md','Lanzando…'],['docs/architecture.md','steam://open/main'],['README.md','trae Steam al frente'],['CHANGELOG.md','Lanzando…']]; for (const [file,needle] of checks) { if (!fs.readFileSync(file,'utf8').includes(needle)) throw new Error('missing '+needle+' in '+file); } console.log('DOCS_OK')"
  EXPECT: DOCS_OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=c76830e81a1a832fdd917dc701ef2d7bbb4d05e68f3ef8f131f67c50a7b25a0c; exit=0; EXPECT=matched; output-sha256=6d36de704b81554dfb84505a4da24630d2ffb6dc40ba29dd836d0c858704f7bc; output-bytes=8; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

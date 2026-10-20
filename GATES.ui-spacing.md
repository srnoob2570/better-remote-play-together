# Gates: UI spacing and toast alignment

OWNS: frontend/src/components/SessionPanel.tsx, frontend/src/styles.css, frontend/src/components/ui/toast.tsx, frontend/e2e/spacing.spec.ts, frontend/e2e/visual.spec.ts, frontend/e2e/visual.spec.ts-snapshots/**, docs/ux.md, GATES.ui-spacing.md

Scope: align the guide refresh action with the command heading, give inputs and buttons consistent breathing room, center toast content vertically, and lock the result with geometry and visual checks.

- [x] G0: the spacing ledger is structurally valid
  CHECK: node /home/srnoob/.agents/skills/unlazy/scripts/gate-lint.mjs GATES.ui-spacing.md
  EXPECT: LINT OK
  EVIDENCE: automatic-evidence=v1; definition-sha256=e220b77352e2cba978b9ae6f03be154feca2cc0736723f3fb76cc81f4733c63f; exit=0; EXPECT=matched; output-sha256=0ac834be81e929f921f13dc6cd912357fb927f20a06157026e60ae860120af96; output-bytes=161; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

- [x] G1: the frontend unit suite passes with the spacing and toast changes
  CHECK: NO_COLOR=1 npm test
  EXPECT: /Tests +[0-9]+ passed/
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=0869df0fcceb7f21c01fa2a853a176d086c313f798c0251f86b2db4d66143ae3; exit=0; EXPECT=matched; output-sha256=c0f65fcf1d5a6cf8d1954ac52c5b471514ece6246630f6930b8487928aece2ee; output-bytes=322; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G2: the production build succeeds with the new layout
  CHECK: NO_COLOR=1 npm run build
  EXPECT: built in
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=098f4b5c9671af99f330f1fee1b5812841c2ef596388c9473f1bdd861cb23404; exit=0; EXPECT=matched; output-sha256=d690b7b6e8b96a64f94451143f8dd1d57a4b6ec237218ffe58b3079a10ae76bb; output-bytes=496; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G3: guide command, library toolbar and toast geometry pass their spacing assertions
  CHECK: PLAYWRIGHT_PORT=9253 NO_COLOR=1 npx playwright test --output=/tmp/pw-gates-spacing e2e/spacing.spec.ts
  EXPECT: /^ *[0-9]+ passed \(/m
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=6a0d2b4ce05061d9db4cc37961c055e00773c2a0dfc216cff430160af057e43f; exit=0; EXPECT=matched; output-sha256=f1036568b57573263709e1d47c9a8f5501b080b030bf1a5c459328ebed69149b; output-bytes=1142; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G4: the complete Playwright suite passes, including visual snapshots
  CHECK: PLAYWRIGHT_PORT=9253 NO_COLOR=1 npx playwright test --output=/tmp/pw-gates-e2e
  EXPECT: /^ *[0-9]+ passed \(/m
  CWD: frontend
  EVIDENCE: automatic-evidence=v1; definition-sha256=4486d3954e8be2402c99aa7d9b363d4eee1956777b929ada33f2b103e91aa85b; exit=0; EXPECT=matched; output-sha256=7f68b4217caa4caf1ca5de7b335aad79fa4656f6c47e603e1d404eff4c2dd705; output-bytes=5593; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together/frontend; path=b4c0271c31ce/22 entries

- [x] G5: visual review confirms guide alignment, command spacing, library gap and toast centering
  EVIDENCE: Reviewed visual-guide-1280x800-linux.png, visual-guide-1024x600-linux.png, /tmp/guide-after-1770.png and /tmp/toast-after-1280.png; measured 10 px command gaps, <=0.1 px heading/refresh center delta, 6 px search/toolbar gap and 0 px toast center deltas.

- [x] G6: the unrelated GATES.md ledger is unchanged
  CHECK: git diff --quiet -- GATES.md && echo "GATES.md unchanged"
  EXPECT: GATES.md unchanged
  EVIDENCE: automatic-evidence=v1; definition-sha256=2145d5c64a69774ca94dc628fcdaa055340f481ab79ac5a497028ee55d35cc38; exit=0; EXPECT=matched; output-sha256=aa3195128058d39782d864c505a7a98e7e8db14dd2e733482147e98be49da149; output-bytes=19; shell=/bin/sh; cwd=/home/srnoob/Documentos/srnoob2570/better-remote-play-together; path=b4c0271c31ce/22 entries

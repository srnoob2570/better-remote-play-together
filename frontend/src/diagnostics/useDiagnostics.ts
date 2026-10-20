import { useCallback, useEffect, useMemo, useState } from "react";

import { toBridgeError } from "../bridge/client";
import { bridge } from "../bridge/wails";
import type { DiagnosticsResult, Problem } from "../bridge/types";
import type { Language } from "../i18n";

export type DiagnosticsMode = "loading" | "ready" | "error";

export interface DiagnosticsController {
  mode: DiagnosticsMode;
  result: DiagnosticsResult | null;
  problem: Problem | null;
  refresh: (options?: { refreshCatalog?: boolean }) => Promise<void>;
}

export function useDiagnostics(
  lang: Language,
  enabled = true,
): DiagnosticsController {
  const [mode, setMode] = useState<DiagnosticsMode>("loading");
  const [result, setResult] = useState<DiagnosticsResult | null>(null);
  const [problem, setProblem] = useState<Problem | null>(null);

  const refresh = useCallback(
    async (options?: { refreshCatalog?: boolean }) => {
      setMode("loading");
      setProblem(null);
      try {
        if (options?.refreshCatalog) {
          await bridge.RefreshCatalog();
        }
        const next = await bridge.RunDiagnostics();
        setResult(next);
        setMode("ready");
      } catch (reason: unknown) {
        setProblem(toBridgeError(reason).problem);
        setMode("error");
      }
    },
    [],
  );

  useEffect(() => {
    if (!enabled) return;
    void refresh();
  }, [enabled, lang, refresh]);

  return useMemo(
    () => ({ mode, result, problem, refresh }),
    [mode, problem, refresh, result],
  );
}

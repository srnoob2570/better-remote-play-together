import { useEffect, useState } from "react";

import { ExternalLinkIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/toast";
import { bridge } from "../bridge/wails";
import { copyText } from "../clipboard";
import { docsUrl, openDoc as openDocs } from "../docs";
import { diagnosticStatusLabel } from "../diagnostics/labels";
import type { DiagnosticCheck, DiagnosticStatus } from "../bridge/types";
import { useDiagnostics } from "../diagnostics/useDiagnostics";
import { translate, type Language, type MessageKey } from "../i18n";

const STATUS_RANK: Record<DiagnosticStatus, number> = {
  fail: 0,
  warn: 1,
  ok: 2,
  skip: 3,
};

type Translator = (key: MessageKey, ...args: Array<string | number>) => string;

export function DiagnosticsPane({
  lang,
  onBack,
  onRepair,
}: {
  lang: Language;
  onBack: () => void;
  onRepair: (checkId?: string) => void;
}) {
  const { mode, result, problem, refresh } = useDiagnostics(lang);
  const [busyAction, setBusyAction] = useState<"copy" | "save" | null>(null);

  const t: Translator = (key, ...args) => translate(lang, key, ...args);

  useEffect(() => {
    if (!problem) return;
    toast.add({
      type: "error",
      title: translate(lang, "diagnostics.error"),
      description: problem.message,
    });
  }, [lang, problem]);

  async function copy() {
    if (!result || busyAction) return;
    setBusyAction("copy");
    try {
      await copyText(result.markdown);
      toast.add({ type: "success", title: t("diagnostics.copied") });
    } catch {
      toast.add({ type: "error", title: t("diagnostics.copyError") });
    } finally {
      setBusyAction(null);
    }
  }

  async function save() {
    if (!result || busyAction) return;
    setBusyAction("save");
    try {
      const path = await bridge.SaveDiagnostics(result.markdown);
      toast.add({ type: "success", title: t("diagnostics.saved", path) });
    } catch {
      toast.add({ type: "error", title: t("diagnostics.saveError") });
    } finally {
      setBusyAction(null);
    }
  }

  async function openDoc(doc: string) {
    try {
      await openDocs(doc);
    } catch {
      toast.add({ type: "error", title: t("diagnostics.docsError") });
    }
  }

  const checks = result?.report.checks ?? [];
  const problems = checks
    .filter((check) => check.status === "fail" || check.status === "warn")
    .sort((left, right) => STATUS_RANK[left.status] - STATUS_RANK[right.status]);
  const others = checks.filter((check) => check.status === "ok" || check.status === "skip");
  const attentionTone = problems.some((check) => check.status === "fail") ? "fail" : "warn";

  return (
    <section className="diagnostics-pane" aria-labelledby="diagnostics-title">
      <div className="diagnostics-heading">
        <div className="diagnostics-heading-main">
          <Button variant="ghost" className="diagnostics-back" onClick={onBack}>
            <span aria-hidden="true">←</span>
            {t("diagnostics.back")}
          </Button>
          <div className="diagnostics-intro">
            <h1 id="diagnostics-title" className="diagnostics-title" tabIndex={-1}>
              {t("diagnostics.title")}
            </h1>
            <p className="hint">{t("diagnostics.intro")}</p>
          </div>
        </div>
        <div className="diagnostics-actions">
          <Button
            variant="ghost"
            className="refresh-button"
            onClick={() => onRepair()}
            disabled={mode === "loading"}
          >
            {t("wizard.repair.open")}
          </Button>
          <LoadingButton
            variant="ghost"
            className="refresh-button"
            onClick={() => void refresh()}
            loading={mode === "loading"}
          >
            {t("diagnostics.refresh")}
          </LoadingButton>
          <LoadingButton
            variant="ghost"
            className="refresh-button"
            onClick={() => void copy()}
            loading={busyAction === "copy"}
            disabled={!result || busyAction !== null}
          >
            {t("diagnostics.copy")}
          </LoadingButton>
          <LoadingButton
            variant="ghost"
            className="refresh-button"
            onClick={() => void save()}
            loading={busyAction === "save"}
            disabled={!result || busyAction !== null}
          >
            {t("diagnostics.save")}
          </LoadingButton>
        </div>
      </div>

      {mode === "loading" && (
        <div
          className="diagnostic-list diagnostic-skeletons"
          role="status"
          aria-busy="true"
          aria-label={t("diagnostics.loading")}
        >
          {[0, 1, 2, 3].map((index) => (
            <Skeleton key={index} className="diagnostic-card-skeleton" />
          ))}
        </div>
      )}

      {mode === "error" && (
        <div className="diagnostics-error">
          <p>{t("diagnostics.error")}</p>
          {problem && <p className="mono">{problem.message}</p>}
          <Button variant="ghost" className="refresh-button" onClick={() => void refresh()}>
            {t("action.retry")}
          </Button>
        </div>
      )}

      {mode === "ready" && result && (
        <>
          <div
            className="diagnostics-summary"
            data-verdict={result.report.summary.verdict}
            aria-label={t("diagnostics.summary")}
          >
            <span className="diagnostics-verdict">
              <span className="diagnostics-verdict-label">{t("diagnostics.verdict")}</span>
              <strong className="diagnostics-verdict-value">
                {diagnosticStatusLabel(t, result.report.summary.verdict)}
              </strong>
            </span>
            <span className="diagnostics-counts">
              {t(
                "diagnostics.counts",
                result.report.summary.counts.ok,
                result.report.summary.counts.warn,
                result.report.summary.counts.fail,
                result.report.summary.counts.skip,
              )}
            </span>
            <span className="diagnostics-meta mono">
              {result.report.version} · {result.report.format}
            </span>
          </div>

          {checks.length === 0 ? (
            <p className="hint">{t("diagnostics.empty")}</p>
          ) : (
            <>
              {problems.length > 0 ? (
                <section className="diagnostic-group" aria-labelledby="diagnostics-attention">
                  <h2
                    id="diagnostics-attention"
                    className="diagnostic-group-title"
                    data-verdict={attentionTone}
                  >
                    {t("diagnostics.attentionCount", problems.length)}
                  </h2>
                  <div className="diagnostic-list">
                    {problems.map((check) => (
                      <DiagnosticCard
                        key={check.id}
                        check={check}
                        t={t}
                        onOpenDoc={(doc) => void openDoc(doc)}
                        onRepair={() => onRepair(check.id)}
                      />
                    ))}
                  </div>
                </section>
              ) : (
                <p className="diagnostics-clear">{t("diagnostics.noProblems")}</p>
              )}

              {others.length > 0 && (
                <details className="diagnostic-others">
                  <summary>
                    <span>{t("diagnostics.othersCount", others.length)}</span>
                  </summary>
                  <div className="diagnostic-rows">
                    {others.map((check) => (
                      <DiagnosticRow
                        key={check.id}
                        check={check}
                        t={t}
                        onOpenDoc={(doc) => void openDoc(doc)}
                      />
                    ))}
                  </div>
                </details>
              )}
            </>
          )}
        </>
      )}
    </section>
  );
}

function DiagnosticCard({
  check,
  t,
  onOpenDoc,
  onRepair,
}: {
  check: DiagnosticCheck;
  t: Translator;
  onOpenDoc: (doc: string) => void;
  onRepair: () => void;
}) {
  return (
    <article className="diagnostic-card" data-status={check.status}>
      <div className="diagnostic-card-head">
        <span className="diagnostic-status">{diagnosticStatusLabel(t, check.status)}</span>
        <h3 className="diagnostic-card-title">{check.title}</h3>
        <span className="diagnostic-id mono">{check.id}</span>
      </div>
      <p className="diagnostic-message">{check.message}</p>
      {check.evidence.length > 0 && (
        <div className="diagnostic-evidence">
          <span className="diagnostic-label">{t("diagnostics.evidence")}</span>
          <ul>
            {check.evidence.map((line, index) => (
              <li key={`${check.id}-${index}`} className="mono">
                {line}
              </li>
            ))}
          </ul>
        </div>
      )}
      {check.nextStep && (
        <p className="diagnostic-next">
          <strong>{t("diagnostics.next")}:</strong> {check.nextStep}
        </p>
      )}
      <DocsLine docs={check.docs} t={t} onOpenDoc={onOpenDoc} />
      <Button
        variant="ghost"
        className="diagnostic-repair"
        onClick={onRepair}
      >
        {t("wizard.repair.review")}
      </Button>
    </article>
  );
}

function DiagnosticRow({
  check,
  t,
  onOpenDoc,
}: {
  check: DiagnosticCheck;
  t: Translator;
  onOpenDoc: (doc: string) => void;
}) {
  return (
    <article className="diagnostic-row" data-status={check.status}>
      <span className="diagnostic-row-status">{diagnosticStatusLabel(t, check.status)}</span>
      <div className="diagnostic-row-main">
        <div className="diagnostic-row-head">
          <h3 className="diagnostic-row-title">{check.title}</h3>
          <span className="diagnostic-id mono">{check.id}</span>
        </div>
        <p className="diagnostic-row-message">{check.message}</p>
        {check.evidence.length > 0 && (
          <p className="diagnostic-row-evidence mono">{check.evidence.join(" · ")}</p>
        )}
        {check.nextStep && (
          <p className="diagnostic-row-next">
            <strong>{t("diagnostics.next")}:</strong> {check.nextStep}
          </p>
        )}
        <DocsLine docs={check.docs} t={t} onOpenDoc={onOpenDoc} />
      </div>
    </article>
  );
}

function DocsLine({
  docs,
  t,
  onOpenDoc,
}: {
  docs: string[] | undefined;
  t: Translator;
  onOpenDoc: (doc: string) => void;
}) {
  if (!docs || docs.length === 0) return null;
  return (
    <p className="diagnostic-docs">
      <strong>{t("diagnostics.docs")}:</strong>{" "}
      {docs.map((doc, index) => (
        <span key={doc}>
          {index > 0 && " · "}
          <a
            href={docsUrl(doc)}
            target="_blank"
            rel="noreferrer"
            onClick={(event) => {
              event.preventDefault();
              onOpenDoc(doc);
            }}
          >
            {doc}
            <ExternalLinkIcon aria-hidden="true" className="diagnostic-docs-icon" />
            <span className="sr-only"> {t("common.opensExternal")}</span>
          </a>
        </span>
      ))}
    </p>
  );
}

import { useEffect, useMemo, useRef, useState } from "react";

import { XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DialogClose } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/toast";

import { toBridgeError } from "../bridge/client";
import type {
  DiagnosticCheck,
  DiagnosticReport,
  SetupStatus,
} from "../bridge/types";
import { copyText } from "../clipboard";
import { DispatcherSetup } from "../components/DispatcherSetup";
import { openDoc } from "../docs";
import { translate, type Language, type MessageKey } from "../i18n";
import type { SessionController } from "../session/useSession";
import { useDiagnostics } from "../diagnostics/useDiagnostics";
import {
  appImageFallback,
  assessSetup,
  checkById,
  dependencyChecksFor,
  dependencyCommand,
  firstPendingStep,
  SETUP_STEP_COUNT,
  SETUP_STEP_LABEL,
  type SetupAssessment,
  type SetupStep,
  type SetupStepId,
  type SetupStepStatus,
} from "./readiness";

export function SetupWizard({
  lang,
  session,
  onClose,
  onOpenGuide,
  onOpenRepair,
}: {
  lang: Language;
  session: SessionController;
  onClose: () => void;
  onOpenGuide: () => void;
  onOpenRepair: () => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const diagnostics = useDiagnostics(lang);
  const [step, setStep] = useState(0);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const initialized = useRef(false);
  const titleRef = useRef<HTMLHeadingElement>(null);

  const assessment = useMemo(
    () =>
      assessSetup(
        diagnostics.result?.report ?? null,
        session.setup,
        session.situation,
      ),
    [diagnostics.result, session.setup, session.situation],
  );

  useEffect(() => {
    if (diagnostics.mode === "ready" && !initialized.current) {
      initialized.current = true;
      setStep(firstPendingStep(assessment.steps));
    }
  }, [assessment, diagnostics.mode]);

  useEffect(() => {
    titleRef.current?.focus();
  }, []);

  const current = assessment.steps[step] ?? assessment.steps[0];
  const currentId = current?.id ?? "welcome";

  async function recheck() {
    setChecking(true);
    setError(null);
    try {
      await session.refreshContext();
    } catch (reason: unknown) {
      setError(toBridgeError(reason).problem.message);
    }
    await diagnostics.refresh({ refreshCatalog: true });
    setChecking(false);
  }

  async function ensureHelper() {
    await session.ensureHelper();
    await recheck();
  }

  function next() {
    setStep((value) => Math.min(value + 1, SETUP_STEP_COUNT - 1));
  }

  function back() {
    setStep((value) => Math.max(value - 1, 0));
  }

  return (
    <section className="wizard-view" aria-labelledby="setup-wizard-title">
      <header className="wizard-header">
        <div className="wizard-heading">
          <h1
            ref={titleRef}
            id="setup-wizard-title"
            className="wizard-title"
            tabIndex={-1}
          >
            {t("wizard.setup.title")}
          </h1>
          <p className="wizard-intro">{t("wizard.setup.intro")}</p>
        </div>
        <DialogClose asChild>
          <Button
            variant="ghost"
            className="wizard-close"
            aria-label={t("common.close")}
          >
            <XIcon aria-hidden="true" />
          </Button>
        </DialogClose>
      </header>

      <div className="wizard-body">
        <nav className="wizard-rail" aria-label={t("wizard.setup.steps")}>
          <ol className="wizard-step-list">
            {assessment.steps.map((item, index) => (
              <li key={item.id}>
                <button
                  type="button"
                  className="wizard-step-button"
                  data-active={index === step}
                  data-status={item.status}
                  aria-current={index === step ? "step" : undefined}
                  onClick={() => setStep(index)}
                >
                  <span className="wizard-step-number">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <span className="wizard-step-copy">
                    <span className="wizard-step-label">
                      {t(SETUP_STEP_LABEL[item.id])}
                    </span>
                    <span className="wizard-step-state">
                      {statusLabel(t, item.status)}
                    </span>
                  </span>
                </button>
              </li>
            ))}
          </ol>
        </nav>

        <div className="wizard-content" aria-live="polite">
          {diagnostics.mode === "loading" && (
            <div
              className="wizard-skeleton"
              role="status"
              aria-busy="true"
              aria-label={t("wizard.checking")}
            >
              <Skeleton className="h-5 w-56" />
              <Skeleton className="h-4 w-full max-w-xl" />
              <Skeleton className="h-4 w-full max-w-lg" />
              <Skeleton className="h-4 w-2/3" />
            </div>
          )}
          {diagnostics.mode === "error" && (
            <div className="wizard-error" role="alert">
              <p>{t("diagnostics.error")}</p>
              {diagnostics.problem && (
                <p className="mono">{diagnostics.problem.message}</p>
              )}
              <Button
                variant="outline"
                className="secondary-button"
                onClick={() => void diagnostics.refresh()}
              >
                {t("action.retry")}
              </Button>
            </div>
          )}
          {error && (
            <p className="wizard-error" role="alert">
              {error}
            </p>
          )}
          {diagnostics.mode === "ready" && current && (
            <StepBody
              id={currentId}
              assessment={assessment}
              report={diagnostics.result?.report ?? null}
              current={current}
              lang={lang}
              session={session}
              checking={checking}
              onRecheck={recheck}
              onEnsureHelper={ensureHelper}
              onOpenGuide={onOpenGuide}
              onOpenRepair={onOpenRepair}
            />
          )}
        </div>
      </div>

      <footer className="wizard-footer">
        <span className="wizard-progress">
          {t(
            "wizard.setup.stepOf",
            step + 1,
            SETUP_STEP_COUNT,
          )}
        </span>
        <div className="wizard-footer-actions">
          <Button
            variant="ghost"
            className="wizard-secondary"
            onClick={back}
            disabled={step === 0}
          >
            {t("wizard.back")}
          </Button>
          {step < SETUP_STEP_COUNT - 1 ? (
            <Button className="wizard-primary" onClick={next}>
              {t("wizard.next")}
            </Button>
          ) : (
            <Button
              className="wizard-primary"
              onClick={onClose}
              disabled={checking}
            >
              {assessment.complete
                ? t("wizard.verify.finish")
                : t("wizard.verify.finishGuide")}
            </Button>
          )}
        </div>
      </footer>
    </section>
  );
}

function StepBody({
  id,
  assessment,
  report,
  current,
  lang,
  session,
  checking,
  onRecheck,
  onEnsureHelper,
  onOpenGuide,
  onOpenRepair,
}: {
  id: SetupStepId;
  assessment: SetupAssessment;
  report: DiagnosticReport | null;
  current: SetupStep;
  lang: Language;
  session: SessionController;
  checking: boolean;
  onRecheck: () => Promise<void>;
  onEnsureHelper: () => Promise<void>;
  onOpenGuide: () => void;
  onOpenRepair: () => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const steam = checkById(report, "steam.install");
  const spacewar = checkById(report, "steam.spacewar");
  const helper = checkById(report, "app.helper");
  const dispatcher = checkById(report, "steam.dispatcher");

  switch (id) {
    case "welcome":
      return (
        <section className="wizard-section">
          <h2 className="wizard-section-title">{t("wizard.welcome.title")}</h2>
          <p className="hint">{t("wizard.welcome.body")}</p>
          <div className="wizard-facts">
            <Fact
              label={t("wizard.format")}
              value={formatLabel(t, assessment.format)}
            />
            <Fact
              label={t("wizard.setup.state")}
              value={setupStatusLabel(t, session.setup?.status ?? "missing")}
            />
            <Fact
              label={t("wizard.steam.steamLabel")}
              value={checkSummary(t, steam)}
            />
            <Fact
              label={t("wizard.steam.spacewarLabel")}
              value={checkSummary(t, spacewar)}
            />
            <Fact
              label={t("wizard.helper.title")}
              value={checkSummary(t, helper)}
            />
            <Fact
              label={t("wizard.step.dispatcher")}
              value={checkSummary(t, dispatcher)}
            />
          </div>
          <p className="hint">{t("wizard.setup.exclusions")}</p>
        </section>
      );
    case "steam":
      return (
        <section className="wizard-section">
          <h2 className="wizard-section-title">{t("wizard.steam.title")}</h2>
          <p className="hint">{t("wizard.steam.body")}</p>
          <CheckRow check={steam} label={t("wizard.steam.steamLabel")} t={t} />
          <CheckRow check={spacewar} label={t("wizard.steam.spacewarLabel")} t={t} />
          <p className={session.situation?.steamRunning ? "hint-amber" : "hint"}>
            {session.situation?.steamRunning
              ? t("wizard.steam.running")
              : t("wizard.steam.closed")}
          </p>
          <div className="wizard-actions">
            <Button
              className="wizard-primary"
              onClick={() => void session.openSpaceWarInstall()}
              disabled={session.busy}
            >
              {t("action.addSpacewar")}
            </Button>
            {steam?.status === "fail" && (
              <Button
                variant="outline"
                className="secondary-button"
                onClick={() =>
                  void openDoc("docs/compatibility.md").catch(() =>
                    toast.add({
                      type: "error",
                      title: t("diagnostics.docsError"),
                    }),
                  )
                }
              >
                {t("wizard.steam.installSteam")}
              </Button>
            )}
          </div>
          <p className="hint">{t("wizard.steam.recheckHint")}</p>
          <Button
            variant="outline"
            className="secondary-button"
            onClick={() => void onRecheck()}
            disabled={checking}
          >
            {checking ? t("wizard.checking") : t("wizard.recheck")}
          </Button>
          {current.status !== "done" && (
            <p className="wizard-note">{t("wizard.steam.nextHint")}</p>
          )}
        </section>
      );
    case "helper":
      return (
        <section className="wizard-section">
          <h2 className="wizard-section-title">{t("wizard.helper.title")}</h2>
          <p className="hint">{t("wizard.helper.body")}</p>
          <CheckRow check={helper} label={t("wizard.helper.title")} t={t} />
          <HelperFacts check={helper} t={t} />
          {current.reasonKey && (
            <p className="hint-amber">{t(current.reasonKey)}</p>
          )}
          <div className="wizard-actions">
            <Button
              className="wizard-primary"
              onClick={() => void onEnsureHelper()}
              disabled={session.busy || current.status === "blocked"}
            >
              {t("wizard.helper.install")}
            </Button>
            <Button
              variant="outline"
              className="secondary-button"
              onClick={() => void onRecheck()}
              disabled={checking}
            >
              {checking ? t("wizard.checking") : t("wizard.recheck")}
            </Button>
          </div>
          <p className="hint">{t("wizard.helper.pathHint")}</p>
        </section>
      );
    case "dispatcher":
      return (
        <section className="wizard-section">
          <h2 className="wizard-section-title">{t("wizard.dispatcher.title")}</h2>
          <p className="hint">{t("wizard.dispatcher.body")}</p>
          <DispatcherSetup
            lang={lang}
            session={session}
            requireBackup
            onVerify={onRecheck}
            onOpenGuide={onOpenGuide}
          />
          {current.status === "blocked" && (
            <Button
              variant="outline"
              className="secondary-button"
              onClick={onOpenRepair}
            >
              {t("wizard.repair.open")}
            </Button>
          )}
        </section>
      );
    case "dependencies":
      return (
        <DependenciesStep
          report={report}
          assessment={assessment}
          lang={lang}
          checking={checking}
          onRecheck={onRecheck}
        />
      );
    case "verify":
      return (
        <section className="wizard-section">
          <h2 className="wizard-section-title">{t("wizard.verify.title")}</h2>
          <p className="hint">{t("wizard.verify.body")}</p>
          <div className="wizard-verify-list">
            {assessment.steps.map((item) => (
              <div className="wizard-verify-row" key={item.id} data-status={item.status}>
                <span>{t(SETUP_STEP_LABEL[item.id])}</span>
                <span>{statusLabel(t, item.status)}</span>
              </div>
            ))}
          </div>
          <p
            className={
              assessment.complete
                ? "wizard-ok"
                : assessment.blocked
                  ? "hint-amber"
                  : "hint"
            }
          >
            {assessment.complete
              ? t("wizard.verify.done")
              : assessment.blocked
                ? t("wizard.verify.blocked")
                : t("wizard.verify.pending")}
          </p>
          <div className="wizard-actions">
            <Button
              className="wizard-primary"
              onClick={() => void onRecheck()}
              disabled={checking}
            >
              {checking ? t("wizard.checking") : t("wizard.verify.refresh")}
            </Button>
            {!assessment.complete && (
              <Button
                variant="outline"
                className="secondary-button"
                onClick={onOpenRepair}
              >
                {t("wizard.repair.open")}
              </Button>
            )}
          </div>
          <p className="hint">{t("wizard.verify.preserve")}</p>
        </section>
      );
  }
}

function DependenciesStep({
  report,
  assessment,
  lang,
  checking,
  onRecheck,
}: {
  report: DiagnosticReport | null;
  assessment: SetupAssessment;
  lang: Language;
  checking: boolean;
  onRecheck: () => Promise<void>;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const checks = dependencyChecksFor(report);
  const fallback = appImageFallback(report?.version ?? "");
  return (
    <section className="wizard-section">
      <h2 className="wizard-section-title">{t("wizard.dependencies.title")}</h2>
      <p className="hint">{t("wizard.dependencies.body")}</p>
      {assessment.format === "deb" ? (
        <p className="wizard-ok">{t("wizard.dependencies.notApplicable")}</p>
      ) : (
        <>
          {checks.map((check) => {
            const command = dependencyCommand(check.id);
            return (
              <div className="wizard-check-row" key={check.id} data-status={check.status}>
                <span className="wizard-check-mark">{statusLabel(t, check.status)}</span>
                <div>
                  <h3 className="wizard-check-title">{check.title}</h3>
                  <p className="hint">{check.message}</p>
                  {command && (
                    <div className="wizard-command-line">
                      <code>{command}</code>
                      <Button
                        variant="ghost"
                        className="wizard-copy"
                        onClick={() =>
                          void copyText(command)
                            .then(() => toast.add({ type: "success", title: t("action.copied") }))
                            .catch(() => toast.add({ type: "error", title: t("guide.copyError") }))
                        }
                      >
                        {t("wizard.dependencies.copy")}
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
          <div className="wizard-command-line">
            <code>{fallback}</code>
            <Button
              variant="ghost"
              className="wizard-copy"
              onClick={() =>
                void copyText(fallback)
                  .then(() => toast.add({ type: "success", title: t("action.copied") }))
                  .catch(() => toast.add({ type: "error", title: t("guide.copyError") }))
              }
            >
              {t("wizard.dependencies.copy")}
            </Button>
          </div>
          <p className="hint">{t("wizard.dependencies.fallback")}</p>
          <p className="hint-amber">{t("wizard.dependencies.preGui")}</p>
        </>
      )}
      <Button
        variant="outline"
        className="secondary-button"
        onClick={() => void onRecheck()}
        disabled={checking}
      >
        {checking ? t("wizard.checking") : t("wizard.recheck")}
      </Button>
    </section>
  );
}

function CheckRow({
  check,
  label,
  t,
}: {
  check: DiagnosticCheck | undefined;
  label: string;
  t: (key: MessageKey) => string;
}) {
  return (
    <div className="wizard-check-row" data-status={check?.status ?? "skip"}>
      <span className="wizard-check-mark">
        {check ? statusLabel(t, check.status) : t("wizard.state.pending")}
      </span>
      <div>
        <h3 className="wizard-check-title">{label}</h3>
        {check?.message && <p className="hint">{check.message}</p>}
        {check?.nextStep && <p className="wizard-note">{check.nextStep}</p>}
      </div>
    </div>
  );
}

function HelperFacts({
  check,
  t,
}: {
  check: DiagnosticCheck | undefined;
  t: (key: MessageKey) => string;
}) {
  if (!check) return null;
  const value = (prefix: string) =>
    check.evidence.find((line) => line.startsWith(prefix))?.slice(prefix.length);
  return (
    <div className="wizard-facts">
      <Fact label={t("wizard.helper.pathLabel")} value={value("path=") ?? "…"} />
      <Fact label={t("wizard.helper.versionLabel")} value={value("version=") ?? "…"} />
      <Fact label={t("wizard.helper.modeLabel")} value={value("exists=true mode=") ?? "…"} />
      <Fact label={t("wizard.helper.sourceLabel")} value={value("source=") ?? "…"} />
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="wizard-fact">
      <span className="wizard-fact-label">{label}</span>
      <span className="wizard-fact-value">{value}</span>
    </div>
  );
}

function checkSummary(
  t: (key: MessageKey) => string,
  check: DiagnosticCheck | undefined,
): string {
  return check ? check.message || statusLabel(t, check.status) : t("wizard.state.pending");
}

function formatLabel(
  t: (key: MessageKey) => string,
  format: SetupAssessment["format"],
): string {
  if (format === "deb") return t("wizard.format.deb");
  if (format === "appimage") return t("wizard.format.appimage");
  return t("wizard.format.unknown");
}

function setupStatusLabel(
  t: (key: MessageKey) => string,
  status: SetupStatus,
): string {
  return t(`setup.${status}` as MessageKey);
}

function statusLabel(
  t: (key: MessageKey) => string,
  status: SetupStepStatus | DiagnosticCheck["status"],
): string {
  switch (status) {
    case "done":
    case "ok":
      return t("wizard.state.done");
    case "blocked":
    case "fail":
      return t("wizard.state.blocked");
    case "skip":
      return t("wizard.state.pending");
    default:
      return t("wizard.state.pending");
  }
}

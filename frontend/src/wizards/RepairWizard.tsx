import { useEffect, useMemo, useRef, useState } from "react";

import { RefreshCwIcon, XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DialogClose } from "@/components/ui/dialog";
import { LoadingButton } from "@/components/ui/loading-button";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/toast";

import { toBridgeError } from "../bridge/client";
import { copyText } from "../clipboard";
import { DispatcherSetup } from "../components/DispatcherSetup";
import { diagnosticStatusLabel } from "../diagnostics/labels";
import { openDoc } from "../docs";
import { translate, type Language, type MessageKey } from "../i18n";
import { restoreReasonKey, type SessionController } from "../session/useSession";
import { useDiagnostics } from "../diagnostics/useDiagnostics";
import { buildRepairIssues, type RepairIssue } from "./repair";

export function RepairWizard({
  lang,
  session,
  initialCheckId,
  onClose,
  onOpenGuide,
  onOpenSetup,
}: {
  lang: Language;
  session: SessionController;
  initialCheckId?: string;
  onClose: () => void;
  onOpenGuide: () => void;
  onOpenSetup: () => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const diagnostics = useDiagnostics(lang);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [checking, setChecking] = useState(false);
  const [copying, setCopying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<MessageKey | null>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);

  const issues = useMemo(
    () =>
      buildRepairIssues(
        diagnostics.result?.report ?? null,
        session.setup,
        session.situation,
      ),
    [diagnostics.result, session.setup, session.situation],
  );

  useEffect(() => {
    void session.refreshContext().catch(() => {});
  }, [session.refreshContext]);

  useEffect(() => {
    const preferred = initialCheckId
      ? issues.find((issue) => issue.checkId === initialCheckId)
      : undefined;
    setSelectedKey((current) => {
      if (current && issues.some((issue) => issueKey(issue) === current)) {
        return current;
      }
      const next = preferred ?? issues[0];
      return next ? issueKey(next) : null;
    });
  }, [initialCheckId, issues]);

  useEffect(() => {
    titleRef.current?.focus();
  }, []);

  const selected =
    issues.find((issue) => issueKey(issue) === selectedKey) ?? issues[0];

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

  async function copyCommand(command: string) {
    if (copying) return;
    setCopying(true);
    try {
      await copyText(command);
      toast.add({ type: "success", title: t("action.copied") });
    } catch {
      toast.add({ type: "error", title: t("guide.copyError") });
    } finally {
      setCopying(false);
    }
  }

  async function runAction(issue: RepairIssue) {
    if (issue.disabledReasonKey) return;
    setError(null);
    setNotice(null);
    try {
      switch (issue.action) {
        case "helper":
          await session.ensureHelper();
          await recheck();
          break;
        case "spacewar":
          await session.openSpaceWarInstall();
          setNotice("wizard.repair.spacewarHint");
          break;
        case "steam":
          await openDoc("docs/compatibility.md");
          break;
        case "deps":
          if (issue.command) {
            await copyCommand(issue.command);
          } else if (issue.docs?.[0]) {
            await openDoc(issue.docs[0]);
          }
          break;
        case "restore":
          onClose();
          session.restore();
          break;
        case "recover":
          await session.recover();
          await recheck();
          break;
        case "cancel":
          await session.cancel();
          await recheck();
          break;
        case "none":
          if (issue.docs?.[0]) await openDoc(issue.docs[0]);
          break;
      }
    } catch (reason: unknown) {
      setError(toBridgeError(reason).problem.message);
    }
  }

  return (
    <section
      className="wizard-view repair-view"
      aria-labelledby="repair-wizard-title"
    >
      <header className="wizard-header">
        <div className="wizard-heading">
          <h1
            ref={titleRef}
            id="repair-wizard-title"
            className="wizard-title"
            tabIndex={-1}
          >
            {t("wizard.repair.title")}
          </h1>
          <p className="wizard-intro">{t("wizard.repair.intro")}</p>
        </div>
        <div className="wizard-header-actions">
          <LoadingButton
            variant="ghost"
            className="wizard-secondary"
            onClick={() => void recheck()}
            loading={checking}
            disabled={checking || diagnostics.mode === "loading"}
          >
            {!checking && <RefreshCwIcon data-icon="inline-start" aria-hidden="true" />}
            {t("wizard.repair.refresh")}
          </LoadingButton>
          <DialogClose asChild>
            <Button
              variant="ghost"
              className="wizard-close"
              aria-label={t("common.close")}
            >
              <XIcon aria-hidden="true" />
            </Button>
          </DialogClose>
        </div>
      </header>

      <div className="repair-readonly">
        <span>{t("wizard.repair.readOnly")}</span>
        <span className="hint">{t("wizard.repair.readOnlyHint")}</span>
      </div>

      <div className="wizard-body repair-body">
        {diagnostics.mode === "loading" && (
          <div
            className="repair-skeleton"
            role="status"
            aria-busy="true"
            aria-label={t("wizard.checking")}
          >
            <Skeleton className="h-5 w-48" />
            <Skeleton className="h-4 w-full max-w-lg" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-24 w-full max-w-2xl" />
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

        {diagnostics.mode === "ready" && issues.length === 0 && (
          <div className="wizard-section">
            <h2 className="wizard-section-title">{t("wizard.repair.empty")}</h2>
            <p className="hint">{t("wizard.repair.emptyHint")}</p>
            {session.setup?.status !== "installed" && (
              <Button
                variant="outline"
                className="secondary-button"
                onClick={onOpenSetup}
              >
                {t("wizard.repair.openSetup")}
              </Button>
            )}
          </div>
        )}

        {diagnostics.mode === "ready" && issues.length > 0 && (
          <>
            <nav className="repair-issues" aria-label={t("wizard.repair.issues", issues.length)}>
              <p className="repair-issues-title">
                {t("wizard.repair.issues", issues.length)}
              </p>
              <ul>
                {issues.map((issue) => (
                  <li key={issueKey(issue)}>
                    <button
                      type="button"
                      className="repair-issue"
                      data-active={issueKey(issue) === issueKey(selected)}
                      data-status={issue.status}
                      onClick={() => setSelectedKey(issueKey(issue))}
                    >
                      <span className="repair-issue-status">
                        {diagnosticStatusLabel(t, issue.status)}
                      </span>
                      <span className="repair-issue-title">
                        {issue.titleKey ? t(issue.titleKey) : issue.title}
                      </span>
                      <span className="repair-issue-id mono">{issue.checkId}</span>
                    </button>
                  </li>
                ))}
              </ul>
            </nav>

            {selected && (
              <section className="repair-detail" aria-live="polite">
                <div className="repair-detail-head">
                  <div>
                    <h2 className="wizard-section-title">
                      {selected.titleKey
                        ? t(selected.titleKey)
                        : selected.title}
                    </h2>
                    <p className="mono repair-check-id">{selected.checkId}</p>
                  </div>
                  <span
                    className="repair-detail-status"
                    data-status={selected.status}
                  >
                    {diagnosticStatusLabel(t, selected.status)}
                  </span>
                </div>

                <p className="wizard-detail-message">
                  {selected.messageKey
                    ? t(selected.messageKey)
                    : selected.message}
                </p>
                {selected.nextStep && <p className="wizard-note">{selected.nextStep}</p>}

                {selected.evidence.length > 0 && (
                  <div className="wizard-evidence">
                    <h3 className="wizard-label">{t("wizard.repair.evidence")}</h3>
                    <ul>
                      {selected.evidence.map((line, index) => (
                        <li key={`${selected.checkId}-${index}`} className="mono">
                          {line}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {selected.restoreReason &&
                  selected.restoreReason !== "available" &&
                  selected.restoreReason !== "not_needed" && (
                    <p className="hint-amber">
                      {t(restoreReasonKey(selected.restoreReason))}
                    </p>
                  )}

                {selected.hintKey && (
                  <p className="hint">{t(selected.hintKey)}</p>
                )}
                {selected.disabledReasonKey && (
                  <p className="hint-amber">{t(selected.disabledReasonKey)}</p>
                )}

                <div className="repair-action">
                  <h3 className="wizard-label">{t("wizard.repair.proposed")}</h3>
                  {selected.action === "dispatcher" ? (
                    <div className="wizard-section">
                      <DispatcherSetup
                        lang={lang}
                        session={session}
                        requireBackup
                        onVerify={recheck}
                        onOpenGuide={onOpenGuide}
                      />
                    </div>
                  ) : selected.action === "deps" ? (
                    <div className="wizard-section">
                      {selected.command ? (
                        <div className="wizard-command-line">
                          <code>{selected.command}</code>
                          <LoadingButton
                            variant="ghost"
                            className="wizard-copy"
                            onClick={() => void runAction(selected)}
                            loading={copying}
                            disabled={copying}
                          >
                            {t("wizard.dependencies.copy")}
                          </LoadingButton>
                        </div>
                      ) : (
                        <p className="hint">{t("wizard.repair.noAction")}</p>
                      )}
                    </div>
                  ) : selected.action !== "none" ? (
                    <div className="wizard-actions">
                      {!selected.disabledReasonKey && (
                        <LoadingButton
                          className="wizard-primary"
                          onClick={() => void runAction(selected)}
                          loading={
                            (selected.action === "helper" &&
                              session.busyAction === "ensureHelper") ||
                            (selected.action === "spacewar" &&
                              session.busyAction === "openSpaceWar") ||
                            (selected.action === "recover" &&
                              session.busyAction === "recover") ||
                            (selected.action === "cancel" &&
                              session.busyAction === "cancel")
                          }
                          disabled={session.busy}
                        >
                          {t(selected.actionLabelKey)}
                        </LoadingButton>
                      )}
                    </div>
                  ) : (
                    <p className="hint">{t("wizard.repair.noAction")}</p>
                  )}
                  {notice && <p className="wizard-note">{t(notice)}</p>}
                  {error && (
                    <p className="wizard-error" role="alert">
                      {error}
                    </p>
                  )}
                </div>

                {selected.docs && selected.docs.length > 0 && (
                  <div className="wizard-docs">
                    <h3 className="wizard-label">{t("wizard.repair.docs")}</h3>
                    <div className="wizard-actions">
                      {selected.docs.map((doc) => (
                        <Button
                          key={doc}
                          variant="ghost"
                          className="wizard-secondary"
                          onClick={() =>
                            void openDoc(doc).catch(() =>
                              toast.add({
                                type: "error",
                                title: t("diagnostics.docsError"),
                              }),
                            )
                          }
                        >
                          {doc}
                        </Button>
                      ))}
                    </div>
                  </div>
                )}
              </section>
            )}
          </>
        )}
      </div>

      <footer className="wizard-footer">
        <span className="wizard-progress">{t("wizard.repair.footer")}</span>
        <div className="wizard-footer-actions">
          {session.setup?.status !== "installed" && (
            <Button
              variant="ghost"
              className="wizard-secondary"
              onClick={onOpenSetup}
            >
              {t("wizard.repair.openSetup")}
            </Button>
          )}
        </div>
      </footer>
    </section>
  );
}

function issueKey(issue: RepairIssue): string {
  return `${issue.key}:${issue.checkId}`;
}

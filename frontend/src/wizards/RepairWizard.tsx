import { useEffect, useMemo, useState } from "react";

import {
  Dialog as UIDialog,
  DialogContent as UIDialogContent,
  DialogDescription as UIDialogDescription,
  DialogTitle as UIDialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/toast";

import { toBridgeError } from "../bridge/client";
import type { DiagnosticStatus } from "../bridge/types";
import { copyText } from "../clipboard";
import { DispatcherSetup } from "../components/DispatcherSetup";
import { openDoc } from "../docs";
import { translate, type Language, type MessageKey } from "../i18n";
import { restoreReasonKey, type SessionController } from "../session/useSession";
import { useDiagnostics } from "../diagnostics/useDiagnostics";
import { buildRepairIssues, type RepairIssue } from "./repair";

export function RepairWizard({
  open,
  lang,
  session,
  initialCheckId,
  onClose,
  onOpenGuide,
  onOpenSetup,
}: {
  open: boolean;
  lang: Language;
  session: SessionController;
  initialCheckId?: string;
  onClose: () => void;
  onOpenGuide: () => void;
  onOpenSetup: () => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const diagnostics = useDiagnostics(lang, open);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<MessageKey | null>(null);

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
    if (!open) {
      setSelectedKey(null);
      setNotice(null);
      setError(null);
      return;
    }
    void session.refreshContext().catch(() => {});
  }, [open, session.refreshContext]);

  useEffect(() => {
    if (!open) return;
    const preferred = initialCheckId
      ? issues.find((issue) => issue.checkId === initialCheckId)
      : undefined;
    const next = preferred ?? issues[0];
    setSelectedKey(next ? issueKey(next) : null);
  }, [initialCheckId, issues, open]);

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
            await copyText(issue.command);
            toast.add({ type: "success", title: t("action.copied") });
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
    <UIDialog open={open} onOpenChange={(value) => !value && onClose()}>
      <UIDialogContent
        className="dialog-content wizard-dialog repair-dialog"
        showCloseButton={false}
      >
        <header className="wizard-header">
          <div className="wizard-heading">
            <UIDialogTitle className="wizard-title">
              {t("wizard.repair.title")}
            </UIDialogTitle>
            <UIDialogDescription className="wizard-intro">
              {t("wizard.repair.intro")}
            </UIDialogDescription>
          </div>
          <div className="wizard-header-actions">
            <Button
              variant="ghost"
              className="wizard-secondary"
              onClick={() => void recheck()}
              disabled={checking || diagnostics.mode === "loading"}
            >
              {checking ? t("wizard.checking") : t("wizard.repair.refresh")}
            </Button>
            <Button
              variant="ghost"
              className="wizard-close"
              onClick={onClose}
              aria-label={t("common.close")}
            >
              {t("common.close")}
            </Button>
          </div>
        </header>

        <div className="repair-readonly">
          <span>{t("wizard.repair.readOnly")}</span>
          <span className="hint">{t("wizard.repair.readOnlyHint")}</span>
        </div>

        <div className="wizard-body repair-body">
          {diagnostics.mode === "loading" && (
            <p className="hint wizard-loading" aria-busy="true">
              {t("wizard.checking")}
            </p>
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
                            <Button
                              variant="ghost"
                              className="wizard-copy"
                              onClick={() => void runAction(selected)}
                            >
                              {t("wizard.dependencies.copy")}
                            </Button>
                          </div>
                        ) : (
                          <p className="hint">{t("wizard.repair.noAction")}</p>
                        )}
                      </div>
                    ) : selected.action !== "none" ? (
                      <div className="wizard-actions">
                        {!selected.disabledReasonKey && (
                          <Button
                            className="wizard-primary"
                            onClick={() => void runAction(selected)}
                            disabled={session.busy}
                          >
                            {t(selected.actionLabelKey)}
                          </Button>
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
                            onClick={() => void openDoc(doc)}
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
            <Button
              className="wizard-primary"
              onClick={onClose}
              disabled={checking}
            >
              {t("common.close")}
            </Button>
          </div>
        </footer>
      </UIDialogContent>
    </UIDialog>
  );
}

function issueKey(issue: RepairIssue): string {
  return `${issue.key}:${issue.checkId}`;
}

function diagnosticStatusLabel(
  t: (key: MessageKey) => string,
  status: DiagnosticStatus,
): string {
  switch (status) {
    case "ok":
      return t("diag.status.ok");
    case "warn":
      return t("diag.status.warn");
    case "fail":
      return t("diag.status.fail");
    case "skip":
      return t("diag.status.skip");
  }
}

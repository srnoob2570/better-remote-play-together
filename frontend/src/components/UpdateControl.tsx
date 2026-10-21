import { useEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";
import { Browser, Events } from "@wailsio/runtime";
import {
  CircleCheckIcon,
  DownloadIcon,
  FlaskConicalIcon,
  FolderCheckIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  TriangleAlertIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import { toBridgeError } from "../bridge/client";
import { bridge } from "../bridge/wails";
import { focusInvoker } from "../lib/focus";
import type {
  Problem,
  UpdateCheck,
  UpdateInstallResult,
  UpdateProgress,
} from "../bridge/types";
import { RELEASES_URL } from "../docs";
import { translate, type Language, type MessageKey } from "../i18n";

type Phase =
  | "idle"
  | "checking"
  | "upToDate"
  | "available"
  | "downloading"
  | "installing"
  | "restartPending"
  | "saved"
  | "error"
  | "development";

const ERROR_KEYS: Record<string, MessageKey> = {
  update_network: "update.error.network",
  update_timeout: "update.error.timeout",
  timeout: "update.error.timeout",
  update_rate_limited: "update.error.rateLimit",
  update_invalid: "update.error.invalid",
  update_no_asset: "update.error.noAsset",
  update_checksum: "update.error.checksum",
  update_download: "update.error.download",
  update_too_large: "update.error.tooLarge",
  update_install: "update.error.install",
  update_no_pkexec: "update.error.noPkexec",
  update_canceled: "update.error.canceled",
  update_busy: "update.error.busy",
  update_not_checked: "update.error.notChecked",
  update_no_update: "update.error.noUpdate",
  update_unsupported: "update.error.unsupported",
  update_no_restart: "update.error.noRestart",
  update_restart: "update.error.restart",
};

export function UpdateControl({ lang, version }: { lang: Language; version: string }) {
  const [phase, setPhase] = useState<Phase>("idle");
  const [check, setCheck] = useState<UpdateCheck | null>(null);
  const [result, setResult] = useState<UpdateInstallResult | null>(null);
  const [progress, setProgress] = useState<UpdateProgress | null>(null);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [restartFailed, setRestartFailed] = useState(false);
  const startupChecked = useRef(false);
  const manualCheck = useRef(false);
  const dialogInvoker = useRef<HTMLElement | null>(null);

  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const busy = phase === "checking" || phase === "downloading" || phase === "installing";
  const percent = Math.max(0, Math.min(100, progress?.percent ?? 0));
  const percentLabel = `${percent}%`;
  const latestVersion = check?.latestVersion ?? "";
  const releaseUrl = check?.releaseUrl || RELEASES_URL;
  const showBadge = Boolean(check?.available && latestVersion);
  const versionText = formatVersion(version);

  useEffect(() => {
    const off = Events.On("update.progress", (event) => {
      const value = event.data as UpdateProgress | null;
      if (!value || typeof value.percent !== "number") return;
      setProgress(value);
      if (value.stage === "installing") {
        setPhase("installing");
      } else {
        setPhase("downloading");
      }
    });
    return off;
  }, []);

  useEffect(() => {
    if (startupChecked.current) return;
    startupChecked.current = true;
    bridge
      .CheckForUpdateOnStartup()
      .then((value) => {
        if (!value || manualCheck.current) return;
        setCheck(value);
        if (value.available) {
          setPhase("available");
        } else if (!value.development) {
          setPhase("upToDate");
        }
      })
      .catch(() => {
        // Startup failures are deliberately silent; the button remains usable.
      });
  }, []);

  async function checkNow() {
    if (busy) return;
    manualCheck.current = true;
    setDialogOpen(true);
    setPhase("checking");
    setProblem(null);
    setResult(null);
    setProgress(null);
    setRestartFailed(false);
    try {
      const value = await bridge.CheckForUpdate();
      setCheck(value);
      if (value.development) {
        setPhase("development");
      } else if (value.available) {
        setPhase("available");
      } else {
        setPhase("upToDate");
      }
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
      setPhase("error");
    }
  }

  async function installNow() {
    if (!check?.canInstall || busy) return;
    setPhase("downloading");
    setProgress({
      stage: "downloading",
      downloaded: 0,
      total: 0,
      percent: 0,
    });
    setProblem(null);
    setResult(null);
    setRestartFailed(false);
    try {
      const value = await bridge.InstallUpdate();
      setResult(value);
      setProgress(null);
      setPhase(value.status === "saved" ? "saved" : "restartPending");
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
      setPhase("error");
      setDialogOpen(true);
    }
  }

  async function restartNow() {
    setRestartFailed(false);
    setProblem(null);
    try {
      await bridge.RestartToUpdate();
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
      setRestartFailed(true);
    }
  }

  function onButtonClick() {
    const active = document.activeElement;
    dialogInvoker.current = active instanceof HTMLElement ? active : null;
    if (phase === "restartPending" || phase === "saved") {
      setDialogOpen(true);
      return;
    }
    void checkNow();
  }

  function openRelease(event: MouseEvent<HTMLAnchorElement>) {
    event.preventDefault();
    void Browser.OpenURL(releaseUrl);
  }

  const buttonLabel = (() => {
    switch (phase) {
      case "checking":
        return t("update.checking");
      case "upToDate":
        return t("update.upToDate");
      case "downloading":
        return progress?.stage === "verifying"
          ? t("update.verifying")
          : t("update.downloading", percentLabel);
      case "installing":
        return t("update.installing");
      case "restartPending":
        return t("update.restartPending");
      case "saved":
        return t("update.saved");
      case "error":
        return t("update.retry");
      default:
        return t("update.check");
    }
  })();

  const ariaLabel =
    showBadge && phase === "available" ? t("update.badge", latestVersion) : buttonLabel;

  const statusText = (() => {
    switch (phase) {
      case "checking":
        return t("update.checking");
      case "available":
        return t("update.statusAvailable", latestVersion);
      case "upToDate":
        return t("update.statusUpToDate", check?.currentVersion ?? version);
      case "downloading":
        return progress?.stage === "verifying"
          ? t("update.verifying")
          : t("update.statusDownloading", percentLabel);
      case "installing":
        return t("update.installing");
      case "restartPending":
        return restartFailed ? t("update.error.restart") : t("update.statusRestart");
      case "saved":
        return t("update.statusSaved", result?.path ?? "");
      case "error":
        return t("update.statusError");
      case "development":
        return t("update.statusDevelopment");
      default:
        return "";
    }
  })();

  return (
    <div className="update-control" data-state={phase}>
      {versionText && <span className="app-version">{versionText}</span>}
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            className="update-button"
            data-state={phase}
            disabled={busy}
            aria-busy={busy}
            aria-label={ariaLabel}
            onClick={onButtonClick}
          >
            <PhaseIcon phase={phase} />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="top" align="start" sideOffset={8} className="app-tooltip">
          {ariaLabel}
        </TooltipContent>
      </Tooltip>
      <span className="sr-only" role="status" aria-live="polite">
        {statusText}
      </span>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent
          className="dialog-content update-dialog"
          showCloseButton={false}
          aria-describedby="update-dialog-description"
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            focusInvoker(dialogInvoker.current);
          }}
        >
          <DialogHeader className="dialog-header">
            <DialogTitle className="dialog-title">{t("update.dialogTitle")}</DialogTitle>
            <DialogDescription
              className="dialog-body"
              id="update-dialog-description"
              aria-live="polite"
              aria-atomic="true"
            >
              {bodyText({
                lang,
                phase,
                check,
                result,
                problem,
                progress,
                percentLabel,
                version,
                restartFailed,
              })}
            </DialogDescription>
          </DialogHeader>

          {(phase === "downloading" || phase === "installing") && (
            <div
              className="update-progress"
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={phase === "installing" || progress?.stage === "verifying" ? 100 : percent}
              aria-label={
                phase === "installing"
                  ? t("update.installing")
                  : progress?.stage === "verifying"
                    ? t("update.verifying")
                    : t("update.downloading", percentLabel)
              }
              data-indeterminate={phase === "installing" || progress?.stage === "verifying"}
            >
              <span
                className="update-progress-fill"
                style={{
                  width:
                    phase === "installing" || progress?.stage === "verifying"
                      ? "100%"
                      : `${percent}%`,
                }}
              />
            </div>
          )}

          {problem?.detail && (phase === "error" || restartFailed) && (
            <p className="update-detail mono">{problem.detail}</p>
          )}

          {phase !== "downloading" && phase !== "installing" && (
            <DialogFooter className="dialog-actions">
              {phase === "available" && check?.canInstall && (
                <Button
                  variant="ghost"
                  className="dialog-button"
                  data-primary="true"
                  onClick={() => void installNow()}
                >
                  {t("update.updateNow")}
                </Button>
              )}
              {phase === "restartPending" && (
                <Button
                  variant="ghost"
                  className="dialog-button"
                  data-primary="true"
                  onClick={() => void restartNow()}
                >
                  {t("update.restartNow")}
                </Button>
              )}
              {phase === "error" && (
                <Button
                  variant="ghost"
                  className="dialog-button"
                  data-primary="true"
                  onClick={() => void checkNow()}
                >
                  {t("update.retry")}
                </Button>
              )}
              <Button
                variant="ghost"
                className="dialog-button"
                onClick={() => setDialogOpen(false)}
              >
                {t("common.close")}
              </Button>
            </DialogFooter>
          )}

          {showReleaseLink(phase) && (
            <a className="update-link" href={releaseUrl} onClick={openRelease}>
              {phase === "error" ? t("update.openRelease") : t("update.releaseNotes")}
            </a>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function showReleaseLink(phase: Phase): boolean {
  return phase === "available" || phase === "development" || phase === "error";
}

function bodyText({
  lang,
  phase,
  check,
  result,
  problem,
  progress,
  percentLabel,
  version,
  restartFailed,
}: {
  lang: Language;
  phase: Phase;
  check: UpdateCheck | null;
  result: UpdateInstallResult | null;
  problem: Problem | null;
  progress: UpdateProgress | null;
  percentLabel: string;
  version: string;
  restartFailed: boolean;
}): ReactNode {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const current = check?.currentVersion ?? version;
  const latest = check?.latestVersion ?? "";
  const rows = (showLatest: boolean) => (
    <span className="update-versions">
      <span>{t("update.currentVersion", current)}</span>
      {showLatest && latest && <span>{t("update.newVersion", latest)}</span>}
    </span>
  );

  switch (phase) {
    case "checking":
      return t("update.checkingBody");
    case "upToDate":
      return (
        <>
          {rows(false)}
          <span>{t("update.upToDateBody", current)}</span>
        </>
      );
    case "available":
      return (
        <>
          {rows(true)}
          <span>
            {check?.development
              ? t("update.availableDevBody")
              : t("update.availableBody", latest)}
          </span>
        </>
      );
    case "development":
      return (
        <>
          {rows(Boolean(latest))}
          <strong className="update-state-label">{t("update.development")}</strong>
          <span>{t("update.developmentBody")}</span>
        </>
      );
    case "downloading":
      return (
        <>
          {rows(true)}
          <span>
            {progress?.stage === "verifying"
              ? t("update.verifyingBody")
              : t("update.downloadingBody", percentLabel)}
          </span>
        </>
      );
    case "installing":
      return (
        <>
          {rows(true)}
          <span>{t("update.installingBody")}</span>
        </>
      );
    case "restartPending":
      return (
        <>
          {rows(true)}
          <span>
            {restartFailed
              ? t("update.error.restart")
              : t("update.restartBody", latest || current)}
          </span>
        </>
      );
    case "saved":
      return (
        <>
          {rows(true)}
          <span>{t("update.savedBody", result?.path ?? "", latest || current)}</span>
        </>
      );
    case "error":
      return (
        <>
          {rows(Boolean(check?.available && latest))}
          <span>{translate(lang, errorKey(problem?.code))}</span>
        </>
      );
    default:
      return null;
  }
}

function errorKey(code?: string): MessageKey {
  if (!code) return "update.error.generic";
  return ERROR_KEYS[code] ?? "update.error.generic";
}

function PhaseIcon({ phase }: { phase: Phase }) {
  switch (phase) {
    case "checking":
    case "downloading":
    case "installing":
      return <Spinner aria-hidden="true" className="update-spinner" />;
    case "upToDate":
      return <CircleCheckIcon aria-hidden="true" />;
    case "available":
      return <DownloadIcon aria-hidden="true" />;
    case "restartPending":
      return <RotateCcwIcon aria-hidden="true" />;
    case "saved":
      return <FolderCheckIcon aria-hidden="true" />;
    case "error":
      return <TriangleAlertIcon aria-hidden="true" />;
    case "development":
      return <FlaskConicalIcon aria-hidden="true" />;
    default:
      return <RefreshCwIcon aria-hidden="true" />;
  }
}

function formatVersion(version: string): string {
  if (!version || version === "dev" || version.startsWith("v")) return version;
  return `v${version}`;
}

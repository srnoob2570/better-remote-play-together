import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/toast";

import { bridge } from "../bridge/wails";
import type { SetupStatus } from "../bridge/types";
import { copyText } from "../clipboard";
import { setupMessageKey } from "../guide";
import { translate, type Language, type MessageKey } from "../i18n";
import type { SessionController } from "../session/useSession";

export function DispatcherSetup({
  lang,
  session,
  requireBackup = false,
  onVerify,
  onOpenGuide,
}: {
  lang: Language;
  session: SessionController;
  requireBackup?: boolean;
  onVerify?: () => void | Promise<void>;
  onOpenGuide?: () => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const [command, setCommand] = useState("");
  const setup = session.setup;
  const status: SetupStatus = setup?.status ?? "missing";
  const steamRunning = session.situation?.steamRunning ?? false;
  const needsRecord = status === "missing" || status === "restored";
  const canCopy =
    Boolean(command) &&
    status !== "conflict" &&
    (!requireBackup || !needsRecord);

  useEffect(() => {
    let alive = true;
    bridge
      .DispatcherCommand()
      .then((value) => {
        if (alive) setCommand(value);
      })
      .catch(() => {
        if (alive) setCommand("");
      });
    return () => {
      alive = false;
    };
  }, []);

  async function copyCommand() {
    try {
      await copyText(command);
      toast.add({ type: "success", title: t("action.copied") });
    } catch {
      toast.add({ type: "error", title: t("guide.copyError") });
    }
  }

  async function verify() {
    await session.refreshSetup();
    await onVerify?.();
  }

  return (
    <div className="dispatcher-setup">
      <p className="hint">{t(setupMessageKey(status))}</p>
      {status === "installed" && setup?.hasBackup === false && (
        <p className="hint-amber">{t("wizard.warning.noBackup")}</p>
      )}

      <div className="guide-command-block">
        <div className="guide-command-head">
          <h3 className="section-title">{t("guide.command")}</h3>
          <Button
            variant="ghost"
            className="refresh-button"
            onClick={() => void verify()}
            disabled={session.busy}
          >
            {t("action.refreshSetup")}
          </Button>
        </div>
        <pre className="guide-command">{command || "…"}</pre>
        <p className="hint">{t("guide.commandHint")}</p>
        <div className="wizard-actions">
          <Button
            variant="outline"
            className="secondary-button"
            onClick={() => void copyCommand()}
            disabled={!canCopy}
          >
            {t("action.copyCommand")}
          </Button>
          {onOpenGuide && (
            <Button
              variant="outline"
              className="secondary-button"
              onClick={onOpenGuide}
            >
              {t("wizard.dispatcher.openGuide")}
            </Button>
          )}
        </div>
        {requireBackup && needsRecord && (
          <p className="hint-amber">{t("wizard.dispatcher.saveFirst")}</p>
        )}
        {status === "conflict" && (
          <p className="hint-amber">{t("wizard.dispatcher.conflict")}</p>
        )}
      </div>

      {status !== "installed" && status !== "conflict" && (
        <p className={steamRunning ? "hint-amber" : "hint"}>
          {steamRunning ? t("guide.steamOpen") : t("guide.steamClosed")}
        </p>
      )}
      {(needsRecord || status === "guide") && (
        <div className="session-actions guide-actions">
          {needsRecord && (
            <Button
              className="secondary-button"
              variant="outline"
              onClick={() => void session.recordGuide()}
              disabled={session.busy}
            >
              {t(
                status === "restored"
                  ? "action.reconfigure"
                  : "action.prepareGuide",
              )}
            </Button>
          )}
          <Button
            className="primary-button"
            onClick={() => void session.installSetup()}
            disabled={steamRunning || session.busy}
          >
            {t("action.installAuto")}
          </Button>
        </div>
      )}
      {needsRecord && <p className="hint">{t("guide.prepareHint")}</p>}
    </div>
  );
}

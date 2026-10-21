import { Button } from "@/components/ui/button";
import {
  Dialog as UIDialog,
  DialogContent as UIDialogContent,
  DialogDescription as UIDialogDescription,
  DialogFooter as UIDialogFooter,
  DialogHeader as UIDialogHeader,
  DialogTitle as UIDialogTitle,
} from "@/components/ui/dialog";

import { translate, type Language, type MessageKey } from "../i18n";
import type { SessionController } from "../session/useSession";

export function SessionDialog({
  lang,
  session,
}: {
  lang: Language;
  session: SessionController;
}) {
  const t = (key: MessageKey) => translate(lang, key);
  const dialog = session.dialog;
  const unknown = dialog?.kind === "unknown";

  return (
    <UIDialog
      open={dialog !== null}
      onOpenChange={(open) => {
        if (!open) session.closeDialog();
      }}
    >
      <UIDialogContent
        className="dialog-content"
        showCloseButton={false}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          session.restoreDialogFocus();
        }}
      >
        <UIDialogHeader className="dialog-header">
          <UIDialogTitle className="dialog-title">
            {t(unknown ? "dialog.unknown.title" : "dialog.restore.title")}
          </UIDialogTitle>
          <UIDialogDescription className="dialog-body">
            {t(unknown ? "dialog.unknown.body" : "dialog.restore.body")}
          </UIDialogDescription>
        </UIDialogHeader>
        <UIDialogFooter className="dialog-actions">
          <Button
            variant="outline"
            className="dialog-button"
            onClick={() => session.closeDialog()}
          >
            {t("common.cancel")}
          </Button>
          <Button
            className="dialog-button"
            data-primary="true"
            onClick={() => void session.confirmDialog()}
            disabled={session.busy}
          >
            {t(unknown ? "common.continue" : "action.restore")}
          </Button>
        </UIDialogFooter>
      </UIDialogContent>
    </UIDialog>
  );
}

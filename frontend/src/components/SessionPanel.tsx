import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Tabs as UITabs,
  TabsContent as UITabsContent,
  TabsList as UITabsList,
  TabsTrigger as UITabsTrigger,
} from "@/components/ui/tabs";
import { ChevronRightIcon } from "lucide-react";

import { artUrl } from "../art";
import { GAME_STATUS_KEY, gameKindLabel } from "../session/gameLabels";
import type {
  Game,
  LastOutcome,
  LastSession,
  LastSource,
  SessionState,
  SetupStatus,
} from "../bridge/types";
import { GUIDE_STEPS } from "../guide";
import { translate, type Language, type MessageKey } from "../i18n";
import {
  isPreparing,
  restoreReasonKey,
  type SessionController,
} from "../session/useSession";
import { problemText } from "../session/problem";
import { DispatcherSetup } from "./DispatcherSetup";

type Tab = "session" | "details" | "guide";

const OUTCOME_KEY: Record<LastOutcome, MessageKey> = {
  pending: "last.outcome.pending",
  ok: "last.outcome.ok",
  error: "last.outcome.error",
  timeout: "last.outcome.timeout",
  canceled: "last.outcome.canceled",
  unknown: "last.outcome.unknown",
};

const SOURCE_KEY: Record<LastSource, MessageKey> = {
  "better-rpt": "last.source.betterRpt",
  steam: "last.source.steam",
};

const PREP_STEPS: MessageKey[] = [
  "prep.verify",
  "prep.installHelper",
  "prep.armIntent",
  "prep.startDonor",
  "prep.waitHelper",
];

export function SessionPanel({
  lang,
  games,
  selected,
  donorInstalled,
  session,
  onSelectLast,
  guideRequest = 0,
  onGuideRequestHandled,
  onOpenSetup,
  onOpenRepair,
}: {
  lang: Language;
  games: Game[];
  selected?: Game;
  donorInstalled: boolean;
  session: SessionController;
  onSelectLast: (appId: number) => void;
  guideRequest?: number;
  onGuideRequestHandled?: () => void;
  onOpenSetup: () => void;
  onOpenRepair: (checkId?: string) => void;
}) {
  const [tab, setTab] = useState<Tab>("session");
  useEffect(() => {
    if (guideRequest > 0) {
      setTab("guide");
      onGuideRequestHandled?.();
    }
  }, [guideRequest, onGuideRequestHandled]);
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const state = session.state;
  const lastGame = session.last
    ? games.find((game) => game.appId === session.last?.appId)
    : undefined;
  const showSession = state && isPreparing(state.phase);
  const showActive = state?.phase === "active";
  const showEnded =
    (state?.phase === "ended" || state?.phase === "error") &&
    (!selected || selected.appId === state?.targetAppId);

  return (
    <aside className="session-pane" aria-label={t("session.section")}>
      <UITabs
        className="session-tabs-root"
        value={tab}
        onValueChange={(value) => setTab(value as Tab)}
      >
        <UITabsList
          variant="line"
          className="session-tabs"
          aria-label={t("session.section")}
        >
          <UITabsTrigger className="session-tab" value="session">
            {t("session.section")}
          </UITabsTrigger>
          <UITabsTrigger className="session-tab" value="details">
            {t("details.title")}
          </UITabsTrigger>
          <UITabsTrigger className="session-tab" value="guide">
            {t("session.tabGuide")}
          </UITabsTrigger>
        </UITabsList>

        <UITabsContent value="session">
          <div className="session-body">
            <SessionContext lang={lang} session={session} />
            {showSession && state && (
              <PreparingPanel
                lang={lang}
                game={games.find((game) => game.appId === state.targetAppId) ?? selected}
                state={state}
                session={session}
              />
            )}

            {showActive && state && (
              <ActivePanel
                lang={lang}
                game={games.find((game) => game.appId === state.targetAppId) ?? selected}
                state={state}
              />
            )}

            {showEnded && state && (
              <EndedPanel
                lang={lang}
                game={games.find((game) => game.appId === state.targetAppId) ?? selected}
                state={state}
                session={session}
                onOpenGuide={() => setTab("guide")}
                onOpenRepair={onOpenRepair}
              />
            )}

            {!showSession && !showActive && !showEnded && selected && (
              <SelectionPanel
                lang={lang}
                game={selected}
                donorInstalled={donorInstalled}
                setupStatus={session.setup?.status ?? "missing"}
                session={session}
                last={session.last}
                lastGame={lastGame}
                onOpenSetup={onOpenSetup}
                onSelectLast={onSelectLast}
              />
            )}

            {!showSession && !showActive && !showEnded && !selected && (
              <LastOrEmptyPanel
                lang={lang}
                last={session.last}
                lastGame={lastGame}
                session={session}
                onSelectLast={onSelectLast}
              />
            )}
          </div>
        </UITabsContent>

        <UITabsContent value="details">
          <DetailsPanel lang={lang} selected={selected} />
        </UITabsContent>

        <UITabsContent value="guide">
          <GuidePanel lang={lang} session={session} />
        </UITabsContent>
      </UITabs>
    </aside>
  );
}

function SessionContext({
  lang,
  session,
}: {
  lang: Language;
  session: SessionController;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const situation = session.situation;

  return (
    <>
      {situation && (situation.kind === "restore_pending" || situation.kind === "orphan") && (
        <div className="callout" data-tone="amber">
          <h2 className="callout-title">
            {t(situation.kind === "orphan" ? "pending.orphanTitle" : "pending.title")}
          </h2>
          <p className="callout-body">
            {situation.kind === "orphan"
              ? t("pending.orphanBody")
              : t(
                  "pending.body",
                  session.last?.name ??
                    (situation.targetAppId
                      ? `#${situation.targetAppId}`
                      : t("common.unknown")),
                )}
          </p>
          <RestoreAction lang={lang} session={session} />
        </div>
      )}

      {situation?.kind === "conflict" && (
        <div className="callout" data-tone="coral">
          <h2 className="callout-title">{t("pending.title")}</h2>
          <p className="callout-body">{t("restore.reason.conflict")}</p>
        </div>
      )}
    </>
  );
}

function RestoreAction({
  lang,
  session,
}: {
  lang: Language;
  session: SessionController;
}) {
  const t = (key: MessageKey) => translate(lang, key);
  const available = session.situation?.restore === "available";
  const reason = session.situation?.restore;
  return (
    <div className="inline-actions">
      <Button
        variant="outline"
        className="secondary-button"
        onClick={session.restore}
        disabled={!available || session.busy}
      >
        {t("action.restoreSpaceWar")}
      </Button>
      {!available && reason && reason !== "not_needed" && (
        <span className="hint-amber">{t(restoreReasonKey(reason))}</span>
      )}
    </div>
  );
}

function Hero({ game, title }: { game?: Game; title: string }) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [game?.appId]);

  return (
    <div className="hero-frame">
      {game && !failed ? (
        <img
          className="hero-image"
          src={artUrl(game.appId, "cover")}
          alt=""
          loading="lazy"
          decoding="async"
          onError={() => setFailed(true)}
        />
      ) : (
        <div className="hero-fallback">{game?.name ?? title}</div>
      )}
    </div>
  );
}

function SelectionPanel({
  lang,
  game,
  donorInstalled,
  setupStatus,
  session,
  last,
  lastGame,
  onOpenSetup,
  onSelectLast,
}: {
  lang: Language;
  game: Game;
  donorInstalled: boolean;
  setupStatus: SetupStatus;
  session: SessionController;
  last?: LastSession;
  lastGame?: Game;
  onOpenSetup: () => void;
  onSelectLast: (appId: number) => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const installed = setupStatus === "installed";
  const launchable = game.status === "ready" || game.status === "unknown";
  const blocked = !launchable || !donorInstalled;
  const showCompactLast = last && last.appId !== game.appId;

  return (
    <>
      <Hero game={game} title={game.name} />
      <div>
        <h2 className="game-title">{game.name}</h2>
        <div className="game-meta">
          <Meta label={t("details.kind")} value={gameKindLabel(lang, game.kind)} />
          <Meta
            label={t("details.problem")}
            value={translate(lang, GAME_STATUS_KEY[game.status])}
            tone={game.status === "ready" ? "mint" : "amber"}
          />
          {game.protonName && (
            <Meta label={t("details.proton")} value={game.protonName} />
          )}
          <Meta label={t("details.appid")} value={String(game.appId)} />
        </div>
      </div>

      <div className="session-actions selection-actions">
        {!donorInstalled ? (
          <Button
            className="primary-button"
            onClick={() => void session.openSpaceWarInstall()}
            disabled={session.busy}
          >
            {t("action.addSpacewar")}
          </Button>
        ) : blocked ? (
          <Button className="primary-button" disabled>
            {translate(lang, GAME_STATUS_KEY[game.status])}
          </Button>
        ) : installed ? (
          <Button
            className="primary-button"
            onClick={() => session.requestStart(game)}
            disabled={session.busy}
          >
            {session.launching === game.appId ? t("action.launching") : t("action.play")}
          </Button>
        ) : (
          <Button className="primary-button" onClick={onOpenSetup}>
            {t("action.configure")}
          </Button>
        )}
      </div>

      {game.problem && <p className="hint">{problemText(lang, game.problem)}</p>}
      {!donorInstalled && <p className="hint-amber">{t("selection.donorHint")}</p>}
      {donorInstalled && !installed && (
        <p className="hint-amber">{t(`setup.${setupStatus}` as MessageKey)}</p>
      )}
      <p className="hint">{t("selection.privacy")}</p>

      {session.launching === game.appId && <LaunchPrompt lang={lang} />}

      {showCompactLast && last && (
        <LastCompact
          lang={lang}
          last={last}
          game={lastGame}
          onSelectLast={onSelectLast}
        />
      )}
    </>
  );
}

function LastOrEmptyPanel({
  lang,
  last,
  lastGame,
  session,
  onSelectLast,
}: {
  lang: Language;
  last?: LastSession;
  lastGame?: Game;
  session: SessionController;
  onSelectLast: (appId: number) => void;
}) {
  const t = (key: MessageKey) => translate(lang, key);
  if (!last) {
    return (
      <div className="selection-empty">
        <h2 className="selection-empty-title">{t("primary.select")}</h2>
        <p className="hint">{t("selection.hint")}</p>
      </div>
    );
  }
  return (
    <LastFull
      lang={lang}
      last={last}
      game={lastGame}
      session={session}
      onSelectLast={onSelectLast}
    />
  );
}

function LastFull({
  lang,
  last,
  game,
  session,
  onSelectLast,
}: {
  lang: Language;
  last: LastSession;
  game?: Game;
  session: SessionController;
  onSelectLast: (appId: number) => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  return (
    <>
      <Hero game={game} title={last.name ?? `#${last.appId}`} />
      <div>
        <h2 className="section-title">{t("last.title")}</h2>
        <h3 className="game-title">{last.name ?? `#${last.appId}`}</h3>
        <div className="game-meta">
          <Meta label={t("last.outcome")} value={t(OUTCOME_KEY[last.outcome])} />
          <Meta label={t("last.duration")} value={duration(last, t)} />
          <Meta label={t("last.started")} value={formatDate(last.startedAt, lang)} />
          <Meta label={t("last.ended")} value={formatDate(last.endedAt, lang)} />
        </div>
      </div>
      <div className="session-actions">
        <Button
          className="primary-button"
          onClick={() => void session.replay()}
          disabled={session.busy}
        >
          {session.launching === last.appId ? t("action.launching") : t("action.playAgain")}
        </Button>
        <Button
          variant="outline"
          className="secondary-button"
          onClick={() => onSelectLast(last.appId)}
        >
          {t("action.openLast")}
        </Button>
        <RestoreAction lang={lang} session={session} />
      </div>
      <p className="hint">{t(SOURCE_KEY[last.source])}</p>
      {session.launching === last.appId && <LaunchPrompt lang={lang} />}
    </>
  );
}

function LastCompact({
  lang,
  last,
  game,
  onSelectLast,
}: {
  lang: Language;
  last: LastSession;
  game?: Game;
  onSelectLast: (appId: number) => void;
}) {
  const t = (key: MessageKey) => translate(lang, key);
  return (
    <div className="last-compact">
      <span className="last-compact-label">{t("last.title")}</span>
      <span className="last-compact-name">{last.name ?? game?.name ?? `#${last.appId}`}</span>
      <span className="last-compact-outcome">{t(OUTCOME_KEY[last.outcome])}</span>
      <Button
        variant="ghost"
        className="refresh-button"
        onClick={() => onSelectLast(last.appId)}
      >
        {t("action.openLast")}
      </Button>
    </div>
  );
}

function PreparingPanel({
  lang,
  game,
  state,
  session,
}: {
  lang: Language;
  game?: Game;
  state: SessionState;
  session: SessionController;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const index =
    state.phase === "donor_starting" ? 3 : state.phase === "target_starting" ? 4 : 0;
  const cancelable = state.phase === "preparing";

  return (
    <>
      <Hero game={game} title={game?.name ?? `#${state.targetAppId}`} />
      <div>
        <h2 className="section-title">{t("prep.title")}</h2>
        <p className="hint">{t("prep.step", index + 1, PREP_STEPS.length)}</p>
      </div>
      <div className="session-actions">
        <Button
          className="primary-button"
          onClick={() => void session.cancel()}
          disabled={!cancelable || session.busy}
        >
          {cancelable ? t("action.cancel") : t("session.starting")}
        </Button>
      </div>
      <ol className="steps">
        {PREP_STEPS.map((key, step) => (
          <li
            className="step"
            key={key}
            data-state={step < index ? "done" : step === index ? "active" : "pending"}
          >
            <span className="step-index">{String(step + 1).padStart(2, "0")}</span>
            <span>{t(key)}</span>
          </li>
        ))}
      </ol>
      {state.phase === "preparing" ? (
        <LaunchPrompt lang={lang} />
      ) : (
        <p className="hint">{t("prep.cancelHint")}</p>
      )}
    </>
  );
}

function ActivePanel({
  lang,
  game,
  state,
}: {
  lang: Language;
  game?: Game;
  state: SessionState;
}) {
  const t = (key: MessageKey) => translate(lang, key);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const tick = () => {
      if (document.visibilityState === "visible") setNow(Date.now());
    };
    const timer = window.setInterval(tick, 1000);
    const onVisibility = () => {
      if (document.visibilityState === "visible") tick();
    };
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, []);
  const started = state.startedAt ? Date.parse(state.startedAt) : now;
  return (
    <>
      <Hero game={game} title={game?.name ?? `#${state.targetAppId}`} />
      <div>
        <h2 className="section-title">{t("session.title")}</h2>
        <h3 className="game-title">{game?.name ?? `#${state.targetAppId}`}</h3>
        <div className="game-meta">
          <Meta label={t("session.targetRole")} value={game?.name ?? `#${state.targetAppId}`} />
          <Meta label={t("session.elapsed")} value={clock(now - started)} />
        </div>
      </div>
      <div className="session-actions">
        <Button className="primary-button" disabled>
          {t("status.active")}
        </Button>
      </div>
      <section>
        <h3 className="section-title">{t("session.guide")}</h3>
        <ol className="steps">
          {(["session.guide1", "session.guide2", "session.guide3"] as MessageKey[]).map(
            (key, index) => (
              <li className="step" key={key} data-state="active">
                <span className="step-index">{String(index + 1).padStart(2, "0")}</span>
                <span>{t(key)}</span>
              </li>
            ),
          )}
        </ol>
      </section>
      <p className="hint">{t("session.locked")}</p>
      <p className="hint-amber">{t("session.actionHint")}</p>
    </>
  );
}

function EndedPanel({
  lang,
  game,
  state,
  session,
  onOpenGuide,
  onOpenRepair,
}: {
  lang: Language;
  game?: Game;
  state: SessionState;
  session: SessionController;
  onOpenGuide: () => void;
  onOpenRepair: (checkId?: string) => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const error = state.phase === "error" ? state.error : undefined;
  const durationMs =
    state.startedAt && state.endedAt
      ? Date.parse(state.endedAt) - Date.parse(state.startedAt)
      : 0;
  const restoreAvailable = session.situation?.restore === "available";

  return (
    <>
      <Hero game={game} title={game?.name ?? `#${state.targetAppId}`} />
      <div>
        <h2 className="section-title">
          {error ? t("error.title") : t("last.title")}
        </h2>
        <h3 className="game-title">{game?.name ?? `#${state.targetAppId}`}</h3>
        {!error && (
          <div className="game-meta">
            <Meta
              label={t("last.outcome")}
              value={t(OUTCOME_KEY[state.outcome ?? "unknown"])}
            />
            <Meta label={t("last.duration")} value={clock(Math.max(0, durationMs))} />
          </div>
        )}
      </div>
      <div className="session-actions">
        {restoreAvailable && (
          <Button
            className="primary-button"
            onClick={session.restore}
            disabled={session.busy}
          >
            {t("action.restoreSpaceWar")}
          </Button>
        )}
        {!error && (
          <Button
            className={restoreAvailable ? "secondary-button" : "primary-button"}
            variant={restoreAvailable ? "outline" : "default"}
            onClick={() => void session.replay()}
            disabled={session.busy}
          >
            {session.launching === state.targetAppId
              ? t("action.launching")
              : t("action.playAgain")}
          </Button>
        )}
        {error && (
          <Button
            className="primary-button"
            onClick={() => game && session.requestStart(game)}
            disabled={!game || session.busy}
          >
            {session.launching === state.targetAppId ? t("action.launching") : t("action.retry")}
          </Button>
        )}
        {error && (
          <Button
            variant="outline"
            className="secondary-button"
            onClick={() => onOpenRepair("session.state")}
          >
            {t("wizard.repair.open")}
          </Button>
        )}
        <Button variant="outline" className="secondary-button" onClick={onOpenGuide}>
          {t("action.viewGuide")}
        </Button>
      </div>
      {error && (
        <div className="callout" data-tone="coral">
          <p className="callout-body">{problemText(lang, error)}</p>
          {error.detail && (
            <details className="error-detail">
              <summary>{t("error.detail")}</summary>
              <pre>{error.detail}</pre>
            </details>
          )}
        </div>
      )}
      {!error && session.situation?.restore !== "available" && (
        <p className="hint-amber">
          {t(restoreReasonKey(session.situation?.restore ?? "not_needed"))}
        </p>
      )}
      {session.launching === state.targetAppId && <LaunchPrompt lang={lang} />}
    </>
  );
}

function DetailsPanel({ lang, selected }: { lang: Language; selected?: Game }) {
  const t = (key: MessageKey) => translate(lang, key);
  if (!selected) {
    return (
      <div className="selection-empty">
        <h2 className="selection-empty-title">{t("details.emptyTitle")}</h2>
        <p className="hint">{t("details.emptyHint")}</p>
      </div>
    );
  }
  return (
    <div className="session-body">
      <h2 className="section-title">{t("details.title")}</h2>
      <h3 className="game-title">{selected.name}</h3>
      <div className="details-list">
        <DetailRow label={t("details.appid")} value={String(selected.appId)} />
        <DetailRow label={t("details.kind")} value={gameKindLabel(lang, selected.kind)} />
        {selected.protonName && (
          <DetailRow label={t("details.proton")} value={selected.protonName} />
        )}
        <DetailRow label={t("details.library")} value={selected.libraryPath} />
        <DetailRow label={t("details.path")} value={selected.installDir} />
        <DetailRow
          label={t("details.problem")}
          value={
            selected.problem
              ? problemText(lang, selected.problem)
              : translate(lang, GAME_STATUS_KEY[selected.status])
          }
        />
      </div>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="details-row">
      <span className="details-label">{label}</span>
      <span className="details-value">{value || "—"}</span>
    </div>
  );
}

function GuidePanel({ lang, session }: { lang: Language; session: SessionController }) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const setup = session.setup;

  return (
    <div className="session-body">
      <div className="guide-intro">
        <h2 className="section-title">{t("guide.title")}</h2>
        <p className="hint">{t("guide.intro")}</p>
      </div>

      <DispatcherSetup lang={lang} session={session} requireBackup />

      <ol className="guide-steps">
        {GUIDE_STEPS.map((step, index) => (
          <li className="guide-step" key={step.title}>
            <span className="step-index">{String(index + 1).padStart(2, "0")}</span>
            <div>
              <h3 className="section-title">{t(step.title)}</h3>
              <p className="hint">{t(step.body)}</p>
              {step.image && step.alt && (
                <img
                  className="guide-image"
                  src={step.image}
                  alt={t(step.alt)}
                  loading="lazy"
                  decoding="async"
                />
              )}
            </div>
          </li>
        ))}
      </ol>

      {setup?.hasBackup && (
        <Collapsible className="guide-restore">
          <CollapsibleTrigger asChild>
            <Button variant="ghost" className="guide-restore-trigger">
              <ChevronRightIcon aria-hidden="true" className="guide-restore-chevron" />
              {t("guide.manualRestore")}
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="guide-restore-body">
            <h3 className="section-title">{t("guide.originalValue")}</h3>
            <pre className="guide-command">
              {setup.originalValue || t("guide.originalEmpty")}
            </pre>
            <p className="hint">{t("guide.restoreHint")}</p>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  );
}

function LaunchPrompt({ lang }: { lang: Language }) {
  return (
    <p className="hint-amber" role="status">
      {translate(lang, "launch.steamPrompt")}
    </p>
  );
}

function Meta({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "amber" | "mint";
}) {
  return (
    <span className="meta-item">
      <span className="meta-label">{label}</span>
      <span className="meta-value" data-tone={tone}>
        {value}
      </span>
    </span>
  );
}

function clock(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  if (hours > 0) {
    return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
  }
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function duration(last: LastSession, t: (key: MessageKey) => string): string {
  if (!last.startedAt) return t("common.unknown");
  const end = last.endedAt ? Date.parse(last.endedAt) : Date.now();
  const start = Date.parse(last.startedAt);
  if (Number.isNaN(start) || Number.isNaN(end)) return t("common.unknown");
  return clock(end - start);
}

function formatDate(value: string | undefined, lang: Language): string {
  if (!value) return translate(lang, "common.unknown");
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(lang === "es" ? "es-ES" : "en-US", {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

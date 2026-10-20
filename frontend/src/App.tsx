import { useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";

import { bridge } from "./bridge/wails";
import { toBridgeError } from "./bridge/client";
import type { Catalog, Game, Snapshot } from "./bridge/types";
import { translate, type Language, type MessageKey } from "./i18n";

export default function App() {
  const [lang, setLang] = useState<Language>("es");
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [catalog, setCatalog] = useState<Catalog | undefined>();
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [eventLine, setEventLine] = useState("");

  useEffect(() => {
    let alive = true;

    const offCatalog = Events.On("catalog.updated", (event) => {
      if (!alive) return;
      const value = event.data as Catalog;
      setCatalog({
        games: value.games ?? [],
        warnings: value.warnings ?? [],
      });
      setEventLine(`catalog.updated · ${value.games?.length ?? 0}`);
    });
    const offSession = Events.On("session.updated", (event) => {
      if (!alive) return;
      setEventLine(`session.updated · ${String((event.data as { phase?: string }).phase ?? "")}`);
    });

    async function load() {
      try {
        const value = await bridge.Snapshot();
        if (!alive) return;
        setSnapshot(value);
        setLang(value.language);
        setCatalog(value.catalog);
        if (!value.catalog) {
          const catalog = await bridge.RefreshCatalog();
          if (alive) setCatalog(catalog);
        }
      } catch (reason: unknown) {
        if (alive) setError(toBridgeError(reason).problem.message);
      }
    }

    void load();

    return () => {
      alive = false;
      offCatalog();
      offSession();
    };
  }, []);

  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);

  async function refresh() {
    setBusy(true);
    setError(null);
    try {
      setCatalog(await bridge.RefreshCatalog());
    } catch (reason: unknown) {
      setError(toBridgeError(reason).problem.message);
    } finally {
      setBusy(false);
    }
  }

  async function changeLanguage(next: Language) {
    if (next === lang) return;
    try {
      await bridge.SetLanguage(next);
      setLang(next);
      setSnapshot((current) =>
        current
          ? { ...current, language: next, config: { ...current.config, language: next } }
          : current,
      );
    } catch (reason: unknown) {
      setError(toBridgeError(reason).problem.message);
    }
  }

  return (
    <div className="shell">
      <header className="shell-header">
        <div className="brand">
          <img className="brand-mark" src="/brand/icon.png" alt="" />
          <div>
            <h1>{t("app.title")}</h1>
            <p>{t("app.subtitle")}</p>
          </div>
        </div>
        <nav className="language-switch" aria-label={t("app.language")}>
          {(["es", "en"] as const).map((code) => (
            <button
              key={code}
              type="button"
              data-active={lang === code}
              onClick={() => void changeLanguage(code)}
            >
              {t(`language.${code}`)}
            </button>
          ))}
        </nav>
      </header>

      <main className="shell-main">
        <section className="shell-card" aria-live="polite">
          <div className="shell-heading">
            <div>
              <h2>{t("library.title")}</h2>
              <p>{t("app.bootstrap")}</p>
            </div>
            <button type="button" className="refresh" onClick={() => void refresh()} disabled={busy}>
              {busy ? t("action.refreshing") : t("action.refresh")}
            </button>
          </div>

          {error && (
            <div className="shell-error" role="alert">
              <strong>{t("error.title")}</strong>
              <p>{error}</p>
            </div>
          )}

          {!snapshot && !error && <p className="shell-note">{t("library.loading")}</p>}

          {snapshot && (
            <>
              <dl className="shell-facts">
                <div>
                  <dt>{t("app.version")}</dt>
                  <dd>{snapshot.version}</dd>
                </div>
                <div>
                  <dt>{t("library.columnStatus")}</dt>
                  <dd>
                    {catalog
                      ? t("library.count", catalog.games.length)
                      : t("library.loading")}
                  </dd>
                </div>
              </dl>

              {catalog ? (
                catalog.games.length > 0 ? (
                  <ul className="game-list" aria-label={t("library.title")}>
                    {catalog.games.slice(0, 8).map((game) => (
                      <li key={game.appId}>
                        <span className="game-name">{game.name}</span>
                        <span className="game-meta">
                          {kindLabel(lang, game.kind)} ·{" "}
                          {t(`status.${game.status}` as MessageKey)}
                        </span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="shell-note">{t("library.empty")}</p>
                )
              ) : (
                <p className="shell-note">{t("library.loading")}</p>
              )}
            </>
          )}

          <p className="shell-event" aria-live="polite">
            {eventLine}
          </p>
        </section>
      </main>
    </div>
  );
}

function kindLabel(lang: Language, kind: Game["kind"]): string {
  if (kind === "native") return translate(lang, "kind.native");
  if (kind === "proton") return translate(lang, "kind.proton");
  return translate(lang, "common.unknown");
}

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type RefObject,
} from "react";
import { Events } from "@wailsio/runtime";

import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

import { artUrl } from "./art";
import { toBridgeError } from "./bridge/client";
import { bridge } from "./bridge/wails";
import type { Catalog, Game, GameKind, GameStatus, Problem, Snapshot } from "./bridge/types";
import { translate, type Language, type MessageKey } from "./i18n";

type Filter = "all" | "ready" | "unknown" | "incompatible";
type LibraryMode = "loading" | "ready" | "empty" | "no-results" | "error";

const FILTERS: Filter[] = ["all", "ready", "unknown", "incompatible"];
const ROW_HEIGHT = 42;

const STATUS_KEY: Record<GameStatus, MessageKey> = {
  ready: "status.ready",
  unknown: "status.unknown",
  incompatible: "status.incompatible",
  missing: "status.missing",
  error: "status.error",
  active: "status.active",
};

const STATUS_TONE: Record<GameStatus, string> = {
  ready: "success",
  unknown: "warning",
  incompatible: "error",
  missing: "neutral",
  error: "error",
  active: "info",
};

export default function App() {
  const [lang, setLang] = useState<Language>("es");
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [status, setStatus] = useState<LibraryMode>("loading");
  const [problem, setProblem] = useState<Problem | null>(null);
  const [headerError, setHeaderError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [selectedId, setSelectedId] = useState(0);
  const [busy, setBusy] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let alive = true;
    const offCatalog = Events.On("catalog.updated", (event) => {
      if (!alive) return;
      setCatalog(normalizeCatalog(event.data as Catalog));
      setStatus("ready");
      setProblem(null);
    });

    async function load() {
      try {
        const value = await bridge.Snapshot();
        if (!alive) return;
        setSnapshot(value);
        setLang(value.language);
        if (value.catalog) {
          setCatalog(normalizeCatalog(value.catalog));
          setStatus("ready");
          return;
        }
        const refreshed = await bridge.RefreshCatalog();
        if (!alive) return;
        setCatalog(normalizeCatalog(refreshed));
        setStatus("ready");
      } catch (reason: unknown) {
        if (!alive) return;
        setProblem(toBridgeError(reason).problem);
        setStatus("error");
      }
    }

    void load();

    return () => {
      alive = false;
      offCatalog();
    };
  }, []);

  useEffect(() => {
    if (!catalog) return;
    setSelectedId((current) => {
      if (catalog.games.some((game) => game.appId === current)) return current;
      return catalog.games.find((game) => game.appId !== 480)?.appId ?? 0;
    });
  }, [catalog]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "f") {
        event.preventDefault();
        searchRef.current?.focus();
        searchRef.current?.select();
      }
      if (event.key === "Escape" && document.activeElement === searchRef.current) {
        event.preventDefault();
        setQuery("");
        searchRef.current?.blur();
      }
      if (event.key === "F5") {
        event.preventDefault();
        void refresh();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [busy]);

  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);

  const games = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return (catalog?.games ?? [])
      .filter((game) => game.appId !== 480)
      .filter((game) => {
        if (filter === "all") return true;
        return game.status === filter;
      })
      .filter((game) => {
        if (!needle) return true;
        return (
          game.name.toLowerCase().includes(needle) ||
          String(game.appId).includes(needle)
        );
      })
      .sort((left, right) => left.name.localeCompare(right.name, lang));
  }, [catalog, filter, lang, query]);

  const selected = catalog?.games.find((game) => game.appId === selectedId);
  const mode: LibraryMode =
    busy || status === "loading"
      ? "loading"
      : status === "error"
        ? "error"
        : !catalog || catalog.games.length === 0
          ? "empty"
          : games.length === 0
            ? "no-results"
            : "ready";

  async function refresh() {
    if (busy) return;
    setBusy(true);
    setStatus("loading");
    setProblem(null);
    setHeaderError(null);
    try {
      const refreshed = await bridge.RefreshCatalog();
      setCatalog(normalizeCatalog(refreshed));
      setStatus("ready");
    } catch (reason: unknown) {
      setProblem(toBridgeError(reason).problem);
      setStatus("error");
    } finally {
      setBusy(false);
    }
  }

  async function changeLanguage(next: Language) {
    if (next === lang) return;
    setHeaderError(null);
    try {
      await bridge.SetLanguage(next);
      setLang(next);
      setSnapshot((current) =>
        current
          ? { ...current, language: next, config: { ...current.config, language: next } }
          : current,
      );
    } catch (reason: unknown) {
      setHeaderError(toBridgeError(reason).problem.message);
    }
  }

  async function openSteam() {
    setHeaderError(null);
    try {
      await bridge.OpenSteam();
    } catch (reason: unknown) {
      setHeaderError(toBridgeError(reason).problem.message);
    }
  }

  function resetFilters() {
    setQuery("");
    setFilter("all");
  }

  return (
    <div className="app-window">
      <header className="app-header">
        <div className="brand">
          <img className="brand-mark" src="/brand/icon.png" alt="" />
          <div>
            <h1 className="brand-title">{t("app.title")}</h1>
            <p className="brand-subtitle">{t("app.subtitle")}</p>
          </div>
        </div>
        <div className="header-right">
          <span className="app-version">
            {t("app.version")} {snapshot?.version ?? ""}
          </span>
          <nav className="language-switch" aria-label={t("app.language")}>
            {(["es", "en"] as const).map((code) => (
              <Button
                key={code}
                variant="ghost"
                className="language-button"
                data-active={lang === code}
                onClick={() => void changeLanguage(code)}
              >
                {t(`language.${code}`)}
              </Button>
            ))}
          </nav>
        </div>
      </header>

      {headerError && (
        <p className="header-error" role="alert">
          {headerError}
        </p>
      )}

      <main className="app-main">
        <LibraryPane
          lang={lang}
          mode={mode}
          games={games}
          catalog={catalog}
          query={query}
          filter={filter}
          selectedId={selectedId}
          problem={problem}
          searchRef={searchRef}
          onQuery={setQuery}
          onFilter={setFilter}
          onSelect={setSelectedId}
          onRefresh={() => void refresh()}
          onOpenSteam={() => void openSteam()}
          onReset={resetFilters}
        />
        <aside className="session-pane" aria-label={t("session.section")}>
          <div className="session-placeholder">
            <h2 className="session-placeholder-title">{t("session.section")}</h2>
            <p className="session-placeholder-name">
              {selected ? selected.name : t("selection.hint")}
            </p>
            <p className="hint">{t("selection.privacy")}</p>
          </div>
        </aside>
      </main>
    </div>
  );
}

function LibraryPane({
  lang,
  mode,
  games,
  catalog,
  query,
  filter,
  selectedId,
  problem,
  searchRef,
  onQuery,
  onFilter,
  onSelect,
  onRefresh,
  onOpenSteam,
  onReset,
}: {
  lang: Language;
  mode: LibraryMode;
  games: Game[];
  catalog: Catalog | null;
  query: string;
  filter: Filter;
  selectedId: number;
  problem: Problem | null;
  searchRef: RefObject<HTMLInputElement | null>;
  onQuery: (query: string) => void;
  onFilter: (filter: Filter) => void;
  onSelect: (appId: number) => void;
  onRefresh: () => void;
  onOpenSteam: () => void;
  onReset: () => void;
}) {
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const countLabel =
    games.length === 1
      ? t("library.countOne")
      : t("library.count", games.length);

  return (
    <section className="library-pane" aria-labelledby="library-title">
      <div className="library-heading">
        <h2 className="library-title" id="library-title">
          {t("library.title")}
        </h2>
        <span className="library-count">{countLabel}</span>
      </div>

      <Input
        ref={searchRef}
        className="search-field"
        type="search"
        value={query}
        placeholder={t("search.placeholder")}
        aria-label={t("search.placeholder")}
        onChange={(event) => onQuery(event.target.value)}
      />

      <div className="library-toolbar">
        <ToggleGroup
          type="single"
          value={filter}
          onValueChange={(value) => {
            if (value) onFilter(value as Filter);
          }}
          className="filter-tabs"
          spacing={0}
          aria-label={t("library.title")}
        >
          {FILTERS.map((item) => (
            <ToggleGroupItem
              key={item}
              value={item}
              className="filter-button"
              aria-label={t(`filter.${item}`)}
            >
              {t(`filter.${item}`)}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        <Button
          variant="ghost"
          className="refresh-button"
          onClick={onRefresh}
          disabled={mode === "loading"}
        >
          <RefreshIcon />
          {mode === "loading" ? t("action.refreshing") : t("action.refresh")}
        </Button>
      </div>

      {mode === "ready" && (
        <VirtualTable
          games={games}
          lang={lang}
          selectedId={selectedId}
          onSelect={onSelect}
        />
      )}

      {mode === "loading" && <LoadingTable lang={lang} />}

      {mode === "empty" && (
        <Empty className="library-empty">
          <h2>{t("library.empty")}</h2>
          <p>{t("library.emptyHint")}</p>
          <div className="inline-actions">
            <Button variant="ghost" className="refresh-button" onClick={onRefresh}>
              {t("action.refresh")}
            </Button>
            <Button variant="ghost" className="refresh-button" onClick={onOpenSteam}>
              {t("action.openSteam")}
            </Button>
          </div>
        </Empty>
      )}

      {mode === "no-results" && (
        <Empty className="library-empty">
          <h2>{t("search.empty")}</h2>
          <div className="inline-actions">
            <Button variant="ghost" className="refresh-button" onClick={onReset}>
              {t("action.resetFilters")}
            </Button>
          </div>
        </Empty>
      )}

      {mode === "error" && (
        <Empty className="library-empty" role="alert">
          <h2>{t("error.refresh")}</h2>
          <p className="mono">{problem?.message ?? t("error.generic")}</p>
          {problem?.detail && <p className="mono">{problem.detail}</p>}
          <div className="inline-actions">
            <Button variant="ghost" className="refresh-button" onClick={onRefresh}>
              {t("action.retry")}
            </Button>
          </div>
        </Empty>
      )}

      {catalog && catalog.warnings.length > 0 && (
        <details className="warning-panel">
          <summary>{t("library.warnings")}</summary>
          {catalog.warnings.map((warning, index) => (
            <p key={`${warning.path}-${index}`}>
              <span className="mono">{warning.path}</span> {warning.problem.message}
            </p>
          ))}
        </details>
      )}
    </section>
  );
}

function VirtualTable({
  games,
  lang,
  selectedId,
  onSelect,
}: {
  games: Game[];
  lang: Language;
  selectedId: number;
  onSelect: (appId: number) => void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [viewport, setViewport] = useState({ top: 0, height: 400 });

  useEffect(() => {
    const element = scrollRef.current;
    if (!element) return;
    const update = () =>
      setViewport({ top: element.scrollTop, height: element.clientHeight || 400 });
    update();
    element.addEventListener("scroll", update, { passive: true });
    let observer: ResizeObserver | undefined;
    if (typeof ResizeObserver !== "undefined") {
      observer = new ResizeObserver(update);
      observer.observe(element);
    }
    return () => {
      element.removeEventListener("scroll", update);
      observer?.disconnect();
    };
  }, []);

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = 0;
    setViewport((current) => ({ ...current, top: 0 }));
  }, [games]);

  const start = Math.max(0, Math.floor(viewport.top / ROW_HEIGHT) - 3);
  const end = Math.min(
    games.length,
    Math.ceil((viewport.top + viewport.height) / ROW_HEIGHT) + 3,
  );
  const visible = games.slice(start, end);

  return (
    <div className="library-table" role="table" aria-label={translate(lang, "library.title")}>
      <div className="table-header" role="row">
        <span role="columnheader">{translate(lang, "library.columnGame")}</span>
        <span role="columnheader">{translate(lang, "library.columnKind")}</span>
        <span role="columnheader">{translate(lang, "library.columnStatus")}</span>
      </div>
      <div className="table-scroll" ref={scrollRef}>
        <div
          className="table-spacer"
          role="rowgroup"
          style={{ height: games.length * ROW_HEIGHT }}
        >
          {visible.map((game, index) => (
            <GameRow
              key={game.appId}
              game={game}
              lang={lang}
              selected={game.appId === selectedId}
              onSelect={onSelect}
              style={{ top: (start + index) * ROW_HEIGHT }}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function GameRow({
  game,
  lang,
  selected,
  onSelect,
  style,
}: {
  game: Game;
  lang: Language;
  selected: boolean;
  onSelect: (appId: number) => void;
  style: CSSProperties;
}) {
  return (
    <div
      className="game-row"
      role="row"
      tabIndex={0}
      aria-selected={selected}
      data-selected={selected}
      style={{ position: "absolute", left: 0, right: 0, height: ROW_HEIGHT, ...style }}
      onClick={() => onSelect(game.appId)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect(game.appId);
        }
        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          event.preventDefault();
          const rows = Array.from(
            event.currentTarget.parentElement?.querySelectorAll<HTMLElement>(".game-row") ?? [],
          );
          const index = rows.indexOf(event.currentTarget);
          rows[index + (event.key === "ArrowDown" ? 1 : -1)]?.focus();
        }
      }}
    >
      <span className="game-identity" role="cell">
        <GameIcon game={game} />
        <span className="game-name">{game.name}</span>
      </span>
      <span className="game-kind" role="cell">
        {kindLabel(lang, game.kind)}
      </span>
      <span className="game-status" role="cell" data-tone={STATUS_TONE[game.status]}>
        {translate(lang, STATUS_KEY[game.status])}
      </span>
    </div>
  );
}

function GameIcon({ game }: { game: Game }) {
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [game.appId]);

  if (failed) {
    return (
      <span className="game-icon-fallback" aria-hidden="true">
        {game.name.slice(0, 1).toUpperCase()}
      </span>
    );
  }
  return (
    <img
      className="game-icon"
      src={artUrl(game.appId, "icon")}
      alt=""
      loading="lazy"
      decoding="async"
      onError={() => setFailed(true)}
    />
  );
}

function LoadingTable({ lang }: { lang: Language }) {
  return (
    <div className="library-table" aria-busy="true">
      <div className="table-header" role="row">
        <span role="columnheader">{translate(lang, "library.columnGame")}</span>
        <span role="columnheader">{translate(lang, "library.columnKind")}</span>
        <span role="columnheader">{translate(lang, "library.columnStatus")}</span>
      </div>
      <div className="table-scroll">
        {[0, 1, 2, 3].map((index) => (
          <div className="game-row" key={index} aria-hidden="true" style={{ height: ROW_HEIGHT }}>
            <span className="game-identity">
              <Skeleton className="game-icon" />
              <Skeleton className="h-4 w-48 max-w-full" />
            </span>
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-3 w-20" />
          </div>
        ))}
      </div>
    </div>
  );
}

function kindLabel(lang: Language, kind: GameKind): string {
  if (kind === "native") return translate(lang, "kind.native");
  if (kind === "proton") return translate(lang, "kind.proton");
  return translate(lang, "common.unknown");
}

function normalizeCatalog(catalog: Catalog): Catalog {
  return {
    games: catalog.games ?? [],
    warnings: catalog.warnings ?? [],
  };
}

function RefreshIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 16 16" width="16" height="16" fill="none">
      <path
        d="M13.5 6.5A5.5 5.5 0 1 0 14 9"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
      <path d="M13.5 2.5v4h-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  );
}

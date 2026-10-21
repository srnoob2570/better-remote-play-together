import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type RefObject,
} from "react";

import {
  LibraryIcon,
  MenuIcon,
  RefreshCwIcon,
  SearchXIcon,
  XIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Empty, EmptyMedia } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { LoadingButton } from "@/components/ui/loading-button";
import { Skeleton } from "@/components/ui/skeleton";
import { Toaster, toast } from "@/components/ui/toast";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { TooltipProvider } from "@/components/ui/tooltip";

import { artUrl } from "./art";
import { toBridgeError } from "./bridge/client";
import { bridge } from "./bridge/wails";
import type { Catalog, Game, Problem, Snapshot } from "./bridge/types";
import { translate, type Language, type MessageKey } from "./i18n";
import { SessionPanel } from "./components/SessionPanel";
import { SessionDialog } from "./components/SessionDialog";
import { DiagnosticsPane } from "./components/DiagnosticsPane";
import { SetupIndicator } from "./components/SetupIndicator";
import { LanguageMenu } from "./components/LanguageMenu";
import { UpdateControl } from "./components/UpdateControl";
import { GAME_STATUS_KEY, GAME_STATUS_TONE, gameKindLabel } from "./session/gameLabels";
import { problemText } from "./session/problem";
import { useSession } from "./session/useSession";
import { RepairWizard } from "./wizards/RepairWizard";
import { SetupWizard } from "./wizards/SetupWizard";
import { setupAutoOpen } from "./wizards/readiness";

type Filter = "all" | "ready" | "unknown" | "incompatible";
type LibraryMode = "loading" | "ready" | "empty" | "no-results" | "error";
type View = "library" | "diagnostics" | "setup" | "repair";
type SetupReturn = "library" | "diagnostics" | "repair";
type RepairReturn = "library" | "diagnostics" | "setup";

const FILTERS: Filter[] = ["all", "ready", "unknown", "incompatible"];
const ROW_HEIGHT = 42;

export default function App() {
  const [lang, setLang] = useState<Language>("es");
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [status, setStatus] = useState<LibraryMode>("loading");
  const [openingSteam, setOpeningSteam] = useState(false);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [selectedId, setSelectedId] = useState(0);
  const [busy, setBusy] = useState(false);
  const [view, setView] = useState<View>("library");
  const [menuOpen, setMenuOpen] = useState(false);
  const [setupReturn, setSetupReturn] = useState<SetupReturn>("library");
  const [repairReturn, setRepairReturn] = useState<RepairReturn>("library");
  const [repairCheckId, setRepairCheckId] = useState<string | undefined>();
  const [guideRequest, setGuideRequest] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const autoSetup = useRef(false);
  const lastView = useRef<View>(view);
  const headerMenu = useRef<HTMLDivElement>(null);
  const menuButton = useRef<HTMLButtonElement>(null);
  const compact = useCompactLayout();
  const session = useSession();
  const t = (key: MessageKey, ...args: Array<string | number>) =>
    translate(lang, key, ...args);
  const clearGuideRequest = useCallback(() => setGuideRequest(0), []);
  const modalOpen = view === "setup" || view === "repair";
  const underlayView =
    view === "setup"
      ? setupReturn === "diagnostics"
        ? "diagnostics"
        : "library"
      : view === "repair"
        ? repairReturn === "diagnostics"
          ? "diagnostics"
          : "library"
        : view;

  useEffect(() => {
    if (!session.notice) return;
    toast.add({
      type: session.notice.tone === "mint" ? "success" : "warning",
      title: translate(lang, session.notice.key, ...(session.notice.args ?? [])),
    });
  }, [lang, session.notice]);

  useEffect(() => {
    if (!session.problem) return;
    const message = problemText(lang, session.problem);
    toast.add({
      type: "error",
      title: translate(lang, "error.title"),
      description: session.problem.detail
        ? `${message}\n\n${session.problem.detail}`
        : message,
    });
  }, [lang, session.problem]);

  useEffect(() => {
    const error = session.state?.phase === "error" ? session.state.error : undefined;
    if (!error) return;
    toast.add({
      type: "error",
      title: translate(lang, "error.title"),
      description: problemText(lang, error),
    });
  }, [lang, session.state?.phase, session.state?.token]);

  useEffect(() => {
    let alive = true;

    async function load() {
      try {
        const value = await bridge.Snapshot();
        if (!alive) return;
        setSnapshot(value);
        setLang(value.language);
        if (value.catalog) {
          setCatalog(value.catalog);
          setStatus("ready");
          return;
        }
        const refreshed = await bridge.RefreshCatalog();
        if (!alive) return;
        setCatalog(refreshed);
        setStatus("ready");
      } catch (reason: unknown) {
        if (!alive) return;
        const next = toBridgeError(reason).problem;
        setProblem(next);
        setStatus("error");
        toast.add({
          type: "error",
          title: t("error.title"),
          description: next.message,
        });
      }
    }

    void load();

    return () => {
      alive = false;
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
    if (view !== "library") return;
    if (autoSetup.current || !session.ready || !catalog || status !== "ready") {
      return;
    }
    if (setupAutoOpen(session.setup?.status)) {
      autoSetup.current = true;
      setSetupReturn("library");
      setView("setup");
    }
  }, [catalog, session.ready, session.setup?.status, status, view]);

  useEffect(() => {
    setMenuOpen(false);
  }, [compact, view]);

  useEffect(() => {
    if (!menuOpen) return;
    function onPointerDown(event: PointerEvent) {
      if (headerMenu.current?.contains(event.target as Node)) return;
      setMenuOpen(false);
      menuButton.current?.focus();
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") return;
      setMenuOpen(false);
      menuButton.current?.focus();
    }
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [menuOpen]);

  useEffect(() => {
    if (lastView.current === view) return;
    lastView.current = view;
    if (view === "setup" || view === "repair") return;
    const id = view === "library" ? "library-title" : "diagnostics-title";
    document.getElementById(id)?.focus({ preventScroll: true });
  }, [view]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (modalOpen) return;
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
  }, [busy, modalOpen]);

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
  const donorInstalled = catalog?.games.some((game) => game.appId === 480) ?? false;
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
    try {
      const refreshed = await bridge.RefreshCatalog();
      setCatalog(refreshed);
      setStatus("ready");
    } catch (reason: unknown) {
      const next = toBridgeError(reason).problem;
      setProblem(next);
      setStatus("error");
      toast.add({
        type: "error",
        title: t("error.refresh"),
        description: next.message,
      });
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
      toast.add({
        type: "error",
        title: t("error.title"),
        description: toBridgeError(reason).problem.message,
      });
    }
  }

  async function openSteam() {
    if (openingSteam) return;
    setOpeningSteam(true);
    try {
      await bridge.OpenSteam();
    } catch (reason: unknown) {
      toast.add({
        type: "error",
        title: t("error.title"),
        description: toBridgeError(reason).problem.message,
      });
    } finally {
      setOpeningSteam(false);
    }
  }

  function resetFilters() {
    setQuery("");
    setFilter("all");
  }

  function openSetup() {
    setSetupReturn(
      view === "diagnostics" ? "diagnostics" : view === "repair" ? "repair" : "library",
    );
    setMenuOpen(false);
    setView("setup");
  }

  function openRepair(checkId?: string, origin: RepairReturn = "library") {
    setRepairReturn(origin);
    setMenuOpen(false);
    setRepairCheckId(checkId);
    setView("repair");
  }

  function openGuide() {
    setView("library");
    setGuideRequest((value) => value + 1);
  }

  const headerActions = (
    <>
      {view !== "setup" && (
        <SetupIndicator
          lang={lang}
          status={session.setup?.status}
          ready={session.ready}
          onOpen={openSetup}
        />
      )}
      <Button
        variant="ghost"
        className="diagnostics-button"
        data-active={view === "diagnostics"}
        onClick={() => setView("diagnostics")}
      >
        {t("diagnostics.open")}
      </Button>
    </>
  );

  return (
    <TooltipProvider delayDuration={300}>
      <div className="app-window">
        <header className="app-header">
          <div className="brand">
            <img className="brand-mark" src="/brand/icon.png" alt="" />
            <div>
              <p className="brand-title">{t("app.title")}</p>
              <p className="brand-subtitle">{t("app.subtitle")}</p>
            </div>
          </div>
          <div className="header-actions" ref={headerMenu}>
            <button
              type="button"
              ref={menuButton}
              className="compact-menu-button"
              aria-label={t("app.menu")}
              aria-expanded={menuOpen}
              aria-controls="app-header-menu"
              onClick={() => setMenuOpen((open) => !open)}
            >
              {menuOpen ? <XIcon aria-hidden="true" /> : <MenuIcon aria-hidden="true" />}
            </button>
            <div
              id="app-header-menu"
              className={compact ? "header-right header-menu-panel" : "header-right"}
              role={compact ? "group" : undefined}
              aria-label={compact ? t("app.menu") : undefined}
              hidden={compact && !menuOpen}
              onClick={(event) => {
                if ((event.target as HTMLElement).closest("button, a")) setMenuOpen(false);
              }}
            >
              {headerActions}
            </div>
          </div>
        </header>

        <main
          className={
            underlayView === "library" ? "app-main" : "app-main diagnostics-main"
          }
        >
          {underlayView === "diagnostics" ? (
            <DiagnosticsPane
              lang={lang}
              onBack={() => setView("library")}
              onRepair={(checkId) => openRepair(checkId, "diagnostics")}
            />
          ) : (
            <>
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
                openingSteam={openingSteam}
                onQuery={setQuery}
                onFilter={setFilter}
                onSelect={setSelectedId}
                onRefresh={() => void refresh()}
                onOpenSteam={() => void openSteam()}
                onReset={resetFilters}
              />
              <SessionPanel
                lang={lang}
                games={catalog?.games ?? []}
                selected={selected}
                donorInstalled={donorInstalled}
                session={session}
                onSelectLast={setSelectedId}
                guideRequest={guideRequest}
                onGuideRequestHandled={clearGuideRequest}
                onOpenSetup={openSetup}
                onOpenRepair={openRepair}
              />
            </>
          )}
        </main>

        <footer className="app-dock">
          <UpdateControl lang={lang} version={snapshot?.version ?? ""} />
          <LanguageMenu lang={lang} onSelect={(next) => void changeLanguage(next)} />
        </footer>

        <Dialog
          open={modalOpen}
          onOpenChange={(open) => {
            if (!open) setView(view === "setup" ? setupReturn : repairReturn);
          }}
        >
          <DialogContent
            className="wizard-dialog"
            showCloseButton={false}
            aria-labelledby={
              view === "setup" ? "setup-wizard-title" : "repair-wizard-title"
            }
            onCloseAutoFocus={(event) => {
              event.preventDefault();
              const returnView = view === "setup" ? setupReturn : repairReturn;
              if (returnView === "setup" || returnView === "repair") return;
              document
                .getElementById(
                  returnView === "diagnostics" ? "diagnostics-title" : "library-title",
                )
                ?.focus({ preventScroll: true });
            }}
          >
            {modalOpen &&
              (view === "setup" ? (
                <SetupWizard
                  lang={lang}
                  session={session}
                  onClose={() => setView(setupReturn)}
                  onOpenGuide={openGuide}
                  onOpenRepair={() => openRepair(undefined, "setup")}
                />
              ) : (
                <RepairWizard
                  lang={lang}
                  session={session}
                  initialCheckId={repairCheckId}
                  onClose={() => setView(repairReturn)}
                  onOpenGuide={openGuide}
                  onOpenSetup={openSetup}
                />
              ))}
          </DialogContent>
        </Dialog>

        <SessionDialog lang={lang} session={session} />
        <Toaster
          closeLabel={t("common.close")}
          viewportLabel={t("common.notifications")}
        />
      </div>
    </TooltipProvider>
  );
}

function useCompactLayout(): boolean {
  const [compact, setCompact] = useState(() => {
    if (typeof window === "undefined" || !window.matchMedia) return false;
    return window.matchMedia("(max-width: 900px)").matches;
  });

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return;
    const query = window.matchMedia("(max-width: 900px)");
    const update = () => setCompact(query.matches);
    update();
    if (query.addEventListener) {
      query.addEventListener("change", update);
      return () => query.removeEventListener("change", update);
    }
    query.addListener(update);
    return () => query.removeListener(update);
  }, []);

  return compact;
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
  openingSteam,
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
  openingSteam: boolean;
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
    mode === "loading"
      ? "—"
      : games.length === 1
        ? t("library.countOne")
        : t("library.count", games.length);

  return (
    <section className="library-pane" aria-labelledby="library-title">
      <div className="library-heading">
        <h1 className="library-title" id="library-title" tabIndex={-1}>
          {t("library.title")}
        </h1>
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
        <LoadingButton
          variant="ghost"
          className="refresh-button"
          onClick={onRefresh}
          loading={mode === "loading"}
        >
          {mode !== "loading" && (
            <RefreshCwIcon data-icon="inline-start" aria-hidden="true" />
          )}
          {t("action.refresh")}
        </LoadingButton>
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
          <EmptyMedia variant="icon">
            <LibraryIcon aria-hidden="true" />
          </EmptyMedia>
          <h2>{t("library.empty")}</h2>
          <p>{t("library.emptyHint")}</p>
          <div className="inline-actions">
            <LoadingButton
              className="primary-button empty-cta"
              onClick={onOpenSteam}
              loading={openingSteam}
            >
              {t("action.openSteam")}
            </LoadingButton>
            <Button
              variant="outline"
              className="secondary-button empty-cta"
              onClick={onRefresh}
            >
              {t("action.refresh")}
            </Button>
          </div>
        </Empty>
      )}

      {mode === "no-results" && (
        <Empty className="library-empty">
          <EmptyMedia variant="icon">
            <SearchXIcon aria-hidden="true" />
          </EmptyMedia>
          <h2>{t("search.empty")}</h2>
          <div className="inline-actions">
            <Button className="primary-button empty-cta" onClick={onReset}>
              {t("action.resetFilters")}
            </Button>
          </div>
        </Empty>
      )}

      {mode === "error" && (
        <Empty className="library-empty">
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
        {gameKindLabel(lang, game.kind)}
      </span>
      <span className="game-status" role="cell" data-tone={GAME_STATUS_TONE[game.status]}>
        {translate(lang, GAME_STATUS_KEY[game.status])}
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

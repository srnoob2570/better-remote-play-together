export function focusInvoker(invoker: HTMLElement | null): void {
  if (typeof document === "undefined") return;
  if (invoker && invoker !== document.body && invoker !== document.documentElement) {
    invoker.focus();
    if (document.activeElement === invoker) return;
  }
  const fallbacks = [
    ".session-pane [role='tab'][aria-selected='true']",
    ".session-pane .session-tab",
    ".diagnostics-title",
    ".library-title",
    ".wizard-title",
    ".diagnostics-button",
    ".compact-menu-button",
  ];
  for (const selector of fallbacks) {
    const target = document.querySelector<HTMLElement>(selector);
    if (!target) continue;
    target.focus();
    if (document.activeElement === target) return;
  }
}

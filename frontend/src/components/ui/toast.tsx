"use client";

import { useSyncExternalStore, useState, type ReactNode } from "react";
import { Toast as ToastPrimitive } from "radix-ui";
import {
  CircleCheckIcon,
  InfoIcon,
  Loader2Icon,
  OctagonXIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";

import { cn } from "@/lib/utils";

type ToastOptions = {
  id?: string;
  type?: string;
  title?: ReactNode;
  description?: ReactNode;
  timeout?: number;
};

type ToastItem = ToastOptions & {
  id: string;
  open: boolean;
};

let items: ToastItem[] = [];
const listeners = new Set<() => void>();
const removals = new Map<string, ReturnType<typeof setTimeout>>();
let nextID = 0;

function emit(next: ToastItem[]) {
  items = next;
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot() {
  return items;
}

function cancelRemoval(id: string) {
  const timer = removals.get(id);
  if (timer === undefined) return;
  clearTimeout(timer);
  removals.delete(id);
}

function dismiss(id: string) {
  const existing = items.find((item) => item.id === id);
  if (!existing || !existing.open) return;
  emit(items.map((item) => (item.id === id ? { ...item, open: false } : item)));
  if (removals.has(id)) return;
  removals.set(
    id,
    setTimeout(() => {
      removals.delete(id);
      emit(items.filter((item) => item.id !== id));
    }, 250),
  );
}

const toast = {
  add(options: ToastOptions) {
    const id = options.id ?? `toast-${++nextID}`;
    cancelRemoval(id);
    const next: ToastItem = { ...options, id, open: true };
    const index = items.findIndex((item) => item.id === id);
    if (index < 0) {
      emit([...items, next]);
    } else {
      const updated = items.slice();
      updated[index] = { ...updated[index], ...next };
      emit(updated);
    }
    return id;
  },
  close(id: string) {
    dismiss(id);
  },
};

const ICON_TONE: Record<string, string> = {
  success: "text-[var(--color-mint)]",
  info: "text-[var(--color-cyan)]",
  warning: "text-[var(--color-amber)]",
  error: "text-[var(--color-coral)]",
  loading: "text-[var(--color-muted-ink)]",
};

function ToastIcon({ type }: { type: string | undefined }) {
  let icon: ReactNode = null;

  if (type === "success") {
    icon = <CircleCheckIcon aria-hidden="true" />;
  }
  if (type === "info") {
    icon = <InfoIcon aria-hidden="true" />;
  }
  if (type === "warning") {
    icon = <TriangleAlertIcon aria-hidden="true" />;
  }
  if (type === "error") {
    icon = <OctagonXIcon aria-hidden="true" />;
  }
  if (type === "loading") {
    icon = <Loader2Icon className="animate-spin" aria-hidden="true" />;
  }

  if (!icon) return null;

  return (
    <span
      data-slot="toast-icon"
      className={cn(
        "shrink-0 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4",
        ICON_TONE[type ?? ""],
      )}
    >
      {icon}
    </span>
  );
}

function ToastRow({ item, closeLabel }: { item: ToastItem; closeLabel: string }) {
  const [active, setActive] = useState(false);

  return (
    <ToastPrimitive.Root
      data-slot="toast"
      open={item.open}
      duration={item.timeout ?? 5000}
      onOpenChange={(open) => {
        if (!open) dismiss(item.id);
      }}
      onMouseEnter={() => setActive(true)}
      onMouseLeave={() => setActive(false)}
      onFocus={() => setActive(true)}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setActive(false);
        }
      }}
      className={cn(
        "pointer-events-auto relative w-full origin-bottom rounded-[2px] border border-[var(--color-line)] bg-popover text-popover-foreground shadow-[0_8px_24px_rgba(0,0,0,0.28)] transition-[transform,opacity] duration-200 ease-out outline-none",
        "data-[state=open]:translate-y-0 data-[state=open]:opacity-100",
        "data-[state=closed]:translate-y-2 data-[state=closed]:opacity-0",
        "focus-visible:border-[var(--color-cyan)] focus-visible:ring-[3px] focus-visible:ring-[rgba(127,200,216,0.35)]",
      )}
    >
      <div
        data-slot="toast-content"
        className="flex h-full items-center gap-3 overflow-hidden p-3"
      >
        <ToastIcon type={item.type} />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          {item.title !== undefined && (
            <ToastPrimitive.Title
              data-slot="toast-title"
              className="text-sm leading-5 font-medium"
            >
              {item.title}
            </ToastPrimitive.Title>
          )}
          {item.description !== undefined && (
            <ToastPrimitive.Description
              data-slot="toast-description"
              className="text-sm leading-5 whitespace-pre-line text-[var(--color-muted-ink)]"
            >
              {item.description}
            </ToastPrimitive.Description>
          )}
        </div>
        <ToastPrimitive.Close
          data-slot="toast-close"
          aria-label={closeLabel}
          className={cn(
            "relative shrink-0 text-[var(--color-muted-ink)] transition-opacity after:absolute after:-inset-2 after:content-[''] hover:text-[var(--color-cream)]",
            active ? "opacity-100" : "opacity-0",
          )}
        >
          <XIcon aria-hidden="true" />
        </ToastPrimitive.Close>
      </div>
    </ToastPrimitive.Root>
  );
}

function Toaster({
  closeLabel,
  viewportLabel,
}: {
  closeLabel: string;
  viewportLabel: string;
}) {
  const toasts = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

  return (
    <ToastPrimitive.Provider swipeDirection="right" label={viewportLabel}>
      {toasts.map((item) => (
        <ToastRow key={item.id} item={item} closeLabel={closeLabel} />
      ))}
      <ToastPrimitive.Viewport
        data-slot="toast-viewport"
        label={viewportLabel}
        className="pointer-events-none fixed inset-x-4 bottom-4 z-[100] m-0 flex list-none flex-col gap-3 p-0 outline-none sm:right-4 sm:left-auto sm:mx-0 sm:w-full sm:max-w-sm"
      />
    </ToastPrimitive.Provider>
  );
}

export { Toaster, toast };

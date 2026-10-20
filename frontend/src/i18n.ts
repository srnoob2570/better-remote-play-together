import es from "../../internal/i18n/catalogs/es.json";
import en from "../../internal/i18n/catalogs/en.json";

const catalogs = { es, en } as const;

export type Language = keyof typeof catalogs;
export type MessageKey = keyof typeof es;

/** Resolves shared catalogue copy and fills Go-style %s/%d placeholders. */
export function translate(
  lang: Language,
  key: MessageKey,
  ...args: Array<string | number>
): string {
  let value: string = catalogs[lang][key] ?? catalogs.es[key] ?? key;
  let index = 0;
  value = value.replace(/%[sd]/g, () => String(args[index++] ?? ""));
  return value;
}

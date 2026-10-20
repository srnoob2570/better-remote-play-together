import { Browser } from "@wailsio/runtime";

export const DOCS_BASE =
  "https://github.com/srnoob2570/better-remote-play-together/blob/main/";

export function docsUrl(doc: string): string {
  return `${DOCS_BASE}${doc}`;
}

export async function openDoc(doc: string): Promise<void> {
  await Browser.OpenURL(docsUrl(doc));
}

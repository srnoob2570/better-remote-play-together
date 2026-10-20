export type ArtKind = "cover" | "icon";

/** Same-origin endpoint served by the Go asset middleware. */
export function artUrl(appId: number, kind: ArtKind): string {
  return `/api/art/${appId}/${kind}`;
}

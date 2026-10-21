import type { SetupStatus } from "./bridge/types";
import type { MessageKey } from "./i18n";

export interface GuideStep {
  title: MessageKey;
  body: MessageKey;
  image?: string;
  alt?: MessageKey;
}

export const GUIDE_STEPS: GuideStep[] = [
  {
    title: "guide.step1.title",
    body: "guide.step1.body",
    image: "/guide/library-properties.png",
    alt: "guide.image.library",
  },
  {
    title: "guide.step2.title",
    body: "guide.step2.body",
    image: "/guide/launch-options.png",
    alt: "guide.image.launchOptions",
  },
  {
    title: "guide.step3.title",
    body: "guide.step3.body",
    image: "/guide/command.png",
    alt: "guide.image.command",
  },
  {
    title: "guide.step4.title",
    body: "guide.step4.body",
  },
  {
    title: "guide.step5.title",
    body: "guide.step5.body",
    image: "/guide/popup.png",
    alt: "guide.image.popup",
  },
  {
    title: "guide.step6.title",
    body: "guide.step6.body",
    image: "/guide/invite.png",
    alt: "guide.image.invite",
  },
];

export function setupMessageKey(status: SetupStatus): MessageKey {
  switch (status) {
    case "installed":
      return "guide.setupInstalled";
    case "guide":
      return "guide.setupGuide";
    case "restored":
      return "guide.setupRestored";
    case "conflict":
      return "guide.setupConflict";
    default:
      return "guide.setupMissing";
  }
}

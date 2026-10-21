import { LanguagesIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import { translate, type Language, type MessageKey } from "../i18n";

const LANGUAGES = ["es", "en"] as const;

export function LanguageMenu({
  lang,
  onSelect,
}: {
  lang: Language;
  onSelect: (lang: Language) => void;
}) {
  const t = (key: MessageKey) => translate(lang, key);
  const label = t("app.language");

  return (
    <div className="language-menu">
      <Tooltip>
        <DropdownMenu>
          <TooltipTrigger asChild>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="language-trigger"
                aria-label={label}
              >
                <LanguagesIcon aria-hidden="true" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <DropdownMenuContent
            side="top"
            align="end"
            sideOffset={8}
            className="language-menu-content"
          >
            <DropdownMenuLabel>{label}</DropdownMenuLabel>
            <DropdownMenuRadioGroup
              value={lang}
              onValueChange={(value) => {
                if (value === "es" || value === "en") onSelect(value);
              }}
            >
              {LANGUAGES.map((code) => (
                <DropdownMenuRadioItem key={code} value={code}>
                  {t(`language.${code}`)}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        <TooltipContent side="top" align="end" sideOffset={8} className="app-tooltip">
          {label}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

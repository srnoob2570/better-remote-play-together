import * as React from "react";

import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

function LoadingButton({
  loading = false,
  disabled,
  children,
  ...props
}: React.ComponentProps<typeof Button> & {
  loading?: boolean;
}) {
  return (
    <Button
      {...props}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      data-loading={loading ? "true" : undefined}
    >
      {loading && <Spinner data-icon="inline-start" aria-hidden="true" />}
      {children}
    </Button>
  );
}

export { LoadingButton };

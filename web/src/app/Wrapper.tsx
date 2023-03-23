import type { ReactNode } from "react";
import { StrictMode } from "react";

import ErrorBoundary from "./ErrorBoundary";

export type Props = { children: ReactNode };

function Wrapper({ children }: Props) {
  return (
    <StrictMode>
      <ErrorBoundary>{children}</ErrorBoundary>
    </StrictMode>
  );
}

export default Wrapper;

import { createRoot } from "react-dom/client";

import Root from "./app/Root";

// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
const container = document.getElementById("app-root")!;
const root = createRoot(container);
root.render(<Root />);

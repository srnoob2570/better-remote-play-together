import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import App from "./App";
import "./styles.css";
import { installMockTransport } from "./e2e/mockTransport";

const params = new URLSearchParams(window.location.search);
const language = params.get("lang") === "en" ? "en" : "es";
installMockTransport(params.get("e2e") ?? "library", language);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

import React from "react";
import { createRoot } from "react-dom/client";
import "./style.css";
import App from "./App";

// ── Disable browser zoom (Ctrl +/-, Ctrl 0, Ctrl scroll) ──
document.addEventListener("keydown", (e) => {
  if (
    (e.ctrlKey || e.metaKey) &&
    (e.key === "+" || e.key === "-" || e.key === "=" || e.key === "0")
  ) {
    e.preventDefault();
  }
});

document.addEventListener(
  "wheel",
  (e) => {
    if (e.ctrlKey) {
      e.preventDefault();
    }
  },
  { passive: false },
);

// ── Disable browser context menu ──
document.addEventListener("contextmenu", (e) => {
  e.preventDefault();
});

const container = document.getElementById("root");

const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

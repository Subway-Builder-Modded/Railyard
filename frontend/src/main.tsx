import './style.css';

import React from 'react';
import { createRoot } from 'react-dom/client';

import App from './App';

// ── Disable browser zoom (Ctrl +/-, Ctrl 0, Ctrl scroll) ──
document.addEventListener('keydown', (e) => {
  if (
    (e.ctrlKey || e.metaKey) &&
    (e.key === '+' || e.key === '-' || e.key === '=' || e.key === '0')
  ) {
    e.preventDefault();
  }
});

document.addEventListener(
  'wheel',
  (e) => {
    if (e.ctrlKey) {
      e.preventDefault();
    }
  },
  { passive: false },
);

document.addEventListener(
  'touchmove',
  (e) => {
    if (e.touches.length > 1) {
      e.preventDefault();
      e.stopPropagation();
    }
  },
  { passive: false, capture: true },
);

const preventGestureZoom = (e: Event) => {
  e.preventDefault();
  e.stopPropagation();
};

document.addEventListener(
  'gesturestart',
  preventGestureZoom as EventListener,
  true,
);
document.addEventListener(
  'gesturechange',
  preventGestureZoom as EventListener,
  true,
);
document.addEventListener(
  'gestureend',
  preventGestureZoom as EventListener,
  true,
);

document.addEventListener(
  'contextmenu',
  (e) => {
    e.preventDefault();
    e.stopPropagation();
  },
  true,
);

document.addEventListener(
  'mousedown',
  (e) => {
    if (e.button === 2) {
      e.preventDefault();
      e.stopPropagation();
    }
  },
  true,
);

document.addEventListener(
  'auxclick',
  (e) => {
    if (e.button === 2) {
      e.preventDefault();
      e.stopPropagation();
    }
  },
  true,
);

const container = document.getElementById('root');

const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

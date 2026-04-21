import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { applyTheme, readStoredTheme } from './stores/themeStore'

// Apply the persisted theme before React mounts so the first paint matches
// the user's choice instead of flashing the default.
applyTheme(readStoredTheme())

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

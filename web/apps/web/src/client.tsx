import { StrictMode, startTransition } from 'react'
import { createRoot } from 'react-dom/client'
import { StartClient } from '@tanstack/react-start/client'

// Use createRoot instead of hydrateRoot — the Go server serves a prerendered
// shell that doesn't match client routes, so hydration always fails.
const root = createRoot(document as any)
startTransition(() => {
  root.render(
    <StrictMode>
      <StartClient />
    </StrictMode>,
  )
})

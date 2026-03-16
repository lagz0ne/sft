# SFT Pattern Coverage

UI interaction patterns and whether SFT's 14 keywords can express them.

## Covered (demonstrated in examples)

| # | Pattern | Example | How |
|---|---------|---------|-----|
| 1 | Browse → detail navigation | All 4 | `navigate(Screen)` in state transition |
| 2 | Multi-select / bulk actions | Gmail, Stripe, Shopify, Linear | `browsing → selecting` state machine |
| 3 | Confirmation dialogs | Stripe (refund, delete), Shopify (fulfillment) | Overlay region + parent state machine |
| 4 | Sub-machines (nested state) | Gmail (ReplyComposer), Stripe (EvidenceUploader) | `states` on region |
| 5 | Event bubbling + emit | Gmail (reply-sent), Stripe (evidence-complete) | `action: emit(event-name)` |
| 6 | Persistent overlays | Gmail (ComposeWindow) | App-level region with `[overlay]` tag |
| 7 | Cross-app composition | Shopify (StorefrontPreview) | `[contains:AppName]` tag |
| 8 | Data-conditional regions | Stripe (has-payments/no-payments/loading/error) | Tags: `[has-X]`, `[no-X]`, `[loading]`, `[error]` |
| 9 | Destructive actions with confirm | Stripe (delete customer), Shopify (refund) | `[destructive]` tag + confirmation overlay |
| 10 | History re-entry | Gmail, Stripe, Shopify, Linear | `(H)` in flow sequence |
| 11 | Ambient events (keyboard) | Gmail (Escape), Linear (C for create) | Event in state machine without region declaring it |
| 12 | Inline editing (expand/collapse) | Gmail (ReplyComposer), Linear (DescriptionEditor) | Sub-machine `viewing → editing` |
| 13 | Search + results | Gmail (SearchResults) | Screen with filter regions |
| 14 | Multi-app | Shopify | `app:` as list |
| 15 | Action weight | Stripe (RefundButton [primary]) | `[primary]`/`[destructive]` tags |
| 16 | Role-based visibility | Linear (WorkspaceSettings [admin]) | `[admin]`/`[team-admin]` tags |
| 17 | Overlay activation in flows | Gmail (ComposeWindow activates) | `activates` keyword in sequence |
| 18 | Unhappy path flows | Stripe (RefundFailed) | Separate flow with error branch |

## Untested — need to verify expressibility

| # | Pattern | Status |
|---|---------|--------|
| 19 | Wizard / multi-step form | ? |
| 20 | Tabs within a screen | ? |
| 21 | Drag-and-drop reordering | ? |
| 22 | Toast / snackbar notifications | ? |
| 23 | Optimistic updates + rollback | ? |
| 24 | Infinite scroll / pagination | ? |
| 25 | File upload with progress | ? |
| 26 | Authentication flow (login/MFA/forgot) | ? |
| 27 | Onboarding / guided tour | ? |
| 28 | Undo/redo | ? |
| 29 | Context menu / right-click | ? |
| 30 | Real-time updates (live data) | ? |
| 31 | Split/master-detail pane | ? |
| 32 | Theme / dark mode toggle | ? |
| 33 | Responsive breakpoints (mobile nav) | ? |
| 34 | Offline mode / sync | ? |
| 35 | Search-as-you-type / autocomplete | ? |
| 36 | Parallel independent sub-machines | ? |

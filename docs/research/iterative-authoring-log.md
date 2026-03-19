# Iterative SFT Authoring — Research Log

Captures requirement → spec → adjustment cycles to study how specs are authored layer by layer.

---

## Session 1: betamed — 2026-03-19

### Requirement

> This is a new project for a customer, let's call them betamed, they want to build a mobile application, that'll be used by end-user (patient), to help them do booking, and also is an opportunity to show partner service/offer. Target market: Germany. Patient should be able to book themselves, or book for their family members. The model is request/ack, so, the booking will be requested, sent and wait for feedback (up to 24 hours).

### Layer 0 — Sketch (screens, regions, events, flows)

```yaml
app:
  name: betamed
  description: patient booking + partner offers — Germany market, mobile-first

  screens:
    home:
      description: landing after login — upcoming bookings, partner offers
      regions:
        upcoming_bookings:
          description: next 3 appointments with date, doctor, status
          events: [select_booking]
        partner_offers:
          description: scrollable cards from partner services
          events: [select_offer]

    booking_flow:
      description: request an appointment — self or family member
      regions:
        patient_picker:
          description: toggle between self and family members
          events: [select_patient]
        specialty_picker:
          description: filter by medical specialty
          events: [select_specialty]
        slot_picker:
          description: available time slots from selected provider
          events: [select_slot]
        review_submit:
          description: confirm details and submit request
          events: [submit_booking]

    booking_detail:
      description: single booking — status, details, actions
      regions:
        booking_info:
          description: date, doctor, patient name, specialty
        status_badge:
          description: requested / confirmed / declined
        actions:
          description: cancel or reschedule — only when confirmed
          events: [cancel_booking, reschedule_booking]

    family:
      description: manage family members who can be booked for
      regions:
        member_list:
          description: list of family members with relationship
          events: [select_member, add_member]
        member_form:
          description: add/edit family member details
          events: [save_member]

    offer_detail:
      description: partner service detail — description, CTA
      regions:
        offer_content:
          description: partner branding, service description, terms
        offer_action:
          description: external link or in-app redemption
          events: [redeem_offer]

  flows:
    book_appointment:
      description: patient requests a booking and waits for confirmation
      sequence: "home → booking_flow → submit → booking_detail(requested)"

    check_booking_status:
      description: patient checks if booking was confirmed
      sequence: "home → booking_detail"
```

**What this answers:** screen inventory, user interactions, key journeys.
**What a designer can do with this:** start wireframing.
**What's deliberately missing:** state machines, data model, fixtures, validation.

### Adjustment 1 — Missing screens: auth, onboarding, profile

**Feedback:**
> What are missed here are: registrations, validations; then onboarding; then common user profiles, address managements, preferences (and can combine with some of existings)

**Changes:**
- Added `registration` (credentials + verification), `login`, `onboarding` (personal info + address + preferences)
- `family` screen absorbed into `profile` as a region — profile is the natural home for account settings
- `member_form` kept as own screen (full-screen focus on mobile)
- `profile` combines: personal info, address, preferences, family members
- German-specific: Versichertennummer, PLZ, address format in descriptions
- New flows: `new_user` (signup funnel), `manage_family` (add member → book)

**Observation:** Layer 0 iteration is about screen inventory completeness. The booking domain was right but the user lifecycle was missing — registration through onboarding is the entry funnel. Profile/settings is the management layer. Both are standard mobile app structure that should have been in v1.

```yaml
app:
  name: betamed
  description: patient booking + partner offers — Germany market, mobile-first

  screens:
    registration:
      description: email/phone sign-up with identity verification
      regions:
        credentials_form:
          description: email, password, phone — standard account creation
          events: [submit_registration]
        verification:
          description: OTP or email link confirmation
          events: [submit_code, resend_code]

    login:
      description: returning user authentication
      regions:
        login_form:
          description: email + password, biometric shortcut
          events: [submit_login, forgot_password]

    onboarding:
      description: first-time setup after registration — 2-3 steps
      regions:
        personal_info:
          description: name, date of birth, insurance number (Versichertennummer)
          events: [save_personal]
        address_form:
          description: street, PLZ, city — German address format
          events: [save_address]
        preferences:
          description: language, notification settings, default pharmacy
          events: [save_preferences]

    home:
      description: landing after login — upcoming bookings, partner offers
      regions:
        upcoming_bookings:
          description: next 3 appointments with date, doctor, status
          events: [select_booking]
        partner_offers:
          description: scrollable cards from partner services
          events: [select_offer]

    profile:
      description: user account — personal info, address, preferences, family
      regions:
        personal_section:
          description: name, DOB, insurance — editable
          events: [edit_personal]
        address_section:
          description: German address with PLZ validation
          events: [edit_address]
        preferences_section:
          description: language, notifications, default pharmacy
          events: [edit_preferences]
        family_members:
          description: list of family members who can be booked for
          events: [select_member, add_member]

    member_form:
      description: add/edit family member — name, DOB, relationship, insurance
      regions:
        member_fields:
          description: personal details for the family member
          events: [save_member, delete_member]

    booking_flow:
      description: request an appointment — self or family member
      regions:
        patient_picker:
          description: toggle between self and family members
          events: [select_patient]
        specialty_picker:
          description: filter by medical specialty
          events: [select_specialty]
        slot_picker:
          description: available time slots from selected provider
          events: [select_slot]
        review_submit:
          description: confirm details and submit request
          events: [submit_booking]

    booking_detail:
      description: single booking — status, details, actions
      regions:
        booking_info:
          description: date, doctor, patient name, specialty
        status_badge:
          description: requested / confirmed / declined
        actions:
          description: cancel or reschedule — only when confirmed
          events: [cancel_booking, reschedule_booking]

    offer_detail:
      description: partner service detail — description, CTA
      regions:
        offer_content:
          description: partner branding, service description, terms
        offer_action:
          description: external link or in-app redemption
          events: [redeem_offer]

  flows:
    new_user:
      description: sign up, verify, set up profile, land on home
      sequence: "registration → verification → onboarding → home"

    book_appointment:
      description: patient requests a booking and waits for confirmation
      sequence: "home → booking_flow → submit → booking_detail(requested)"

    check_booking_status:
      description: patient checks if booking was confirmed
      sequence: "home → booking_detail"

    manage_family:
      description: add a family member then book for them
      sequence: "profile → member_form → save → booking_flow"
```

### Adjustment 2 — Insurance, multi-step booking, German healthcare specifics

**Feedback:**
> Insurance information (public/private), booking_flow should be multiple steps since request-based (date range, time range, doctor optional, visit type: walk-in/follow-up/with referral), location/distance

**Research verified:**
- GKV (public ~88%) vs PKV (private ~10-13%), Hausarztmodell affects specialist routing
- Überweisung (referral) required for imaging, optional for most specialists
- Visit types: Erstbesuch, Folgetermin, Akutsprechstunde
- 2-48h confirmation window is realistic for request-based model

**Changes:**
- Insurance section added to onboarding, profile, member_form
- Booking flow exploded from 1 screen (4 regions) → 5 screens (wizard steps): patient → type → when/where → doctor → review
- Visit types reflect German system
- Referral upload in step 2 (conditional on visit type)
- Location uses PLZ + radius, time uses preference bands (not slots)
- New flow: `book_with_referral`

**Observation:** The single-screen booking_flow was wrong for mobile + request-based model. When you don't have real-time slot availability, you're collecting *preferences* not *selections*. That changes the UX from "pick a slot" to "tell us what works for you." Each step is a screen because mobile wizards need full-screen focus per decision.

### Adjustment 3 — Booking steps as regions in one screen

**Feedback:**
> Actually, we can combine booking screens to different regions

**Changes:**
- 5 booking screens collapsed to 1 `booking` screen with 7 regions
- Added `state_machine` on booking screen — first state machine in the spec, driven by the wizard pattern
- Each state declares `regions:` — `step_indicator` always visible, content region swaps per step
- `edit_step` from review returns to `picking_patient` (start over)
- `submitted` is terminal (navigates to booking_detail)
- Flows simplified: `book_with_referral` absorbed (referral_upload is just a region in the type step)

**Observation:** State-region visibility earned its place here. The booking wizard is the first case where Layer 0 (structure) demands Layer 2 (state machine) — you can't describe "which regions show in which step" without it. This is the natural escalation point: when a single screen has multiple modes, you need a state machine. The 5-screen version avoided this but felt wrong for mobile.

### Adjustment 4 — Navigation, state machines, app shaping

**Feedback:**
> Tiny bit of detail, now start shaping more like application, show navigations, statemachine of change states etc

**Changes:**
- App-level regions: `bottom_tabs` (Home/Bookings/Profile) + `top_bar` (back/title/notifications)
- New `bookings_list` screen for the Bookings tab (filter: upcoming/past/declined)
- State machines added to every multi-mode screen: registration (entering→verifying), login (credentials↔resetting), onboarding (3-step wizard), home (has/no bookings), profile (viewing↔editing per section), member_form (editing↔confirming_delete), booking_detail (requested→confirmed/declined/cancelled)
- All cross-screen transitions use explicit `navigate()`
- booking_detail state machine models request/ack lifecycle — `requested` is read-only (waiting), `confirmed` enables actions
- Reschedule flow added, logout added

**Observation:** Adding state machines to every screen revealed two patterns:
1. **Wizard screens** (registration, onboarding, booking) — states are steps, `regions:` swaps content per step
2. **Detail screens** (booking_detail, member_form) — states are lifecycle stages, `regions:` shows/hides actions based on status
3. **List screens** (bookings_list) — states are filter modes, same regions but different data

The booking_detail state machine is interesting — it models the async request/ack lifecycle purely through states. `requested` → (external: doctor responds) → `confirmed` or `declined`. The external trigger has no event in SFT (it's a push notification / polling result), which validates the round 1 finding that system events are outside SFT scope. The spec documents what states exist and what the user can do in each — how the state changes from `requested` to `confirmed` is implementation.

### Adjustment 5 — Layer 3: Data model, context, ambient

**Feedback:**
> Yes, we can go ahead (to data layer)

**Changes:**
- Added `data:` section with 6 types: patient, insurance_info, address, family_member, booking, doctor, offer
- `?` suffix on optional fields: confirmed_date, confirmed_doctor, referral_document, selected_patient, etc.
- App-level context: current_user + family (available everywhere)
- Screen-level context on every data-driven screen
- Ambient refs on regions declaring where data comes from

**Observation:** Layer 3 made certain things explicit that were vague before:
1. `member_form` context is `family_member?` — null when adding, populated when editing. The `?` suffix earned its place here.
2. The `booking` type has both request fields (date_range, time_preference) and response fields (confirmed_date?, confirmed_doctor?) — the `?` separates what the patient provides from what the system fills in later.
3. Ambient refs create a readable data dependency graph: `review_submit` reads from 3 different context fields, making the wizard's data accumulation visible.
4. `doctor_preference` region reads `data(booking, .selected_patient)` because available doctors may depend on the patient's insurance type — the ambient ref documents this dependency.

### Adjustment 6 — Layer 4: Fixtures

**Feedback:**
> Let's go (to fixtures layer)

**Changes:**
- Base `anna` fixture — shared German patient persona with son Max, all other fixtures extend it
- German-realistic data: TK insurer, Berlin PLZ, German specialties, German offer copy
- Fixture-per-state on home (2 states), bookings_list (3 filter states), booking_detail (4 lifecycle states), booking wizard (3 key steps), profile, member_form
- Inheritance chain: booking_cancelled → booking_confirmed → booking_requested → anna — each adds only the delta

**Observation:** Fixtures prove the spec works. Key discoveries:
1. The `booking_detail` lifecycle states now have concrete proof — `requested` shows no confirmed_date (optional field is absent), `confirmed` extends it with the doctor assignment. The `?` suffix on `confirmed_date` and `confirmed_doctor` in the data type suddenly has visual meaning: these fields appear only when the state transitions.
2. Fixture inheritance eliminates massive duplication — anna's data (user, family, address, insurance) is written once and inherited 15 times.
3. The booking wizard fixtures reveal data accumulation: step_patient has nothing selected, step_type has patient selected, review has everything. Each fixture shows exactly what the user has committed at that point.
4. German-specific data (PLZ, insurer names, specialty names) in fixtures makes the spec immediately testable with the target market's terminology.

### Adjustment 7 — Booking lifecycle: cancel, alter, rebook, rate, favorite

**Feedback:**
> Add cancel booking, alter booking, rebook (of rejected one). Rate the result, mark last doctor favorite.

**Changes:**
- booking_detail state machine: 5 → 8 states (added completed, rating, rated)
- Separate action regions per lifecycle state (actions_requested, actions_confirmed, actions_declined, actions_completed)
- `alter_booking` from confirmed → re-enters booking wizard prefilled
- `rebook` from declined → re-enters booking wizard prefilled
- `completed` state enables rate_visit + favorite_doctor
- rating_form region with 1-5 stars + optional comment
- patient.favorite_doctors added to data model
- Profile gains favorite_doctors region (select to quick-book, remove)
- doctor_preference in booking wizard surfaces favorites
- booking.source_booking context for prefilling alter/rebook
- 4 new flows: alter, rebook, rate_and_favorite, book_from_favorite
- Fixtures: booking_completed, booking_rated added to inheritance chain

**Observation:** The booking_detail screen is now the most complex screen — 8 states, 7 regions, each state showing a different action set. This is exactly the pattern state-region visibility was designed for. Without `regions:` on states, you'd need 7 tags ([requested], [confirmed], etc.) on the action regions with no formal connection to the state machine. With `regions:`, each state declares exactly which regions are active — the spec is self-documenting and validatable.

### Adjustment 8 — Requirements audit: critical gaps + inconsistencies

**Feedback:**
> Subagent requirements review identified 5 critical gaps, 4 inconsistencies, and risky assumptions. Remove pharmacy-related for now.

**Critical gaps closed:**
- Notifications: `notification` type, `notifications` screen, `unread_count` in app context, badges on nav
- GDPR/DSGVO: `terms_consent` in registration (Art. 9), `delete_account` + `export_data` on profile, flows for both
- Booking expiry: `expired` state, `expires_at` field, `expiry_notice` region, flow
- Password reset: `reset_sent` region + `reset_confirmation` state
- Akutsprechstunde: description clarifies fast-tracked path, separate flow

**Inconsistencies fixed:**
- 24h response window (per customer requirement, not 48h)
- `alter_booking` is the single term (reschedule removed)
- `jump_to_step` with guards replaces edit_step (jump to any step from review)
- Cancel available from both `requested` and `confirmed` states

**Other additions:**
- `offline_banner` + `error_toast` (app-level, tagged)
- eGK scanning (`scan_egk` event)
- Language picker (de, en, tr, ar, ru)
- Documents region on booking_detail (view uploaded Überweisung)
- `decline_reason` on booking type
- Help/FAQ screen
- Offer action clarified as external link only (no in-app redemption, removed pharmacy assumption)
- Specialty made optional (Hausarztmodell patients don't pick)

**Observation:** The requirements audit caught gaps that iterative domain-focused authoring misses — compliance (GDPR), infrastructure (notifications), and edge cases (expiry). The iterative painting approach works well for the happy path but needs a structured cross-check against non-functional requirements and regulatory context. The German market makes this especially sharp: DSGVO, eGK, BFSG are non-negotiable. A Layer 0 sketch can't catch these — they emerge when you ask "what's obviously needed but never stated?"

### Adjustment 9 — Verification audit: 5 errors + 6 warnings fixed

**Feedback:**
> Subagent verification found 5 ERRORs and 14 WARNINGs comparing spec against requirements.

**ERROR fixes:**

1. **booking wizard skip_doctor** — `picking_doctor` state now has `skip_doctor: reviewing` transition. Doctor was always optional; the state machine just didn't reflect it.

2. **decline_reason type** — confirmed as `decline_reason: string?` on booking type. Was already in the Adj 8 YAML.

3. **new_user flow** — corrected to `"registration -> onboarding -> home"`. Verification is a state within registration, not a separate screen.

4. **booking_expired fixture** — added:
```yaml
booking_expired:
  extends: booking_requested
  booking_detail:
    booking:
      status: expired
```

5. **booking_detail/rating fixture** — rating state reuses `booking_completed` fixture (rating form is empty overlay on completed data).

**WARNING fixes:**

- **W3: expired rebook** — `expired` state now has `rebook: navigate(booking)` transition, same as `declined`. Added `rebook_expired` flow.
- **W5: bookings_list expired filter** — expired bookings appear under "declined" tab (functionally the same — request didn't result in appointment). Description updated.
- **W7: notification type fields** — specified: `title: string, body: string, type: string, booking: booking?, read: boolean, created_at: datetime`
- **W8: delete/export flows** — added:
  - `delete_account: "profile -> delete_account -> confirm -> login"`
  - `export_data: "profile -> export_data -> download"`
- **W10: notification fixtures** — added `notifications_unread` (with booking confirmation notification) and `notifications_empty`
- **W2: delete_account state** — profile state machine now has `confirming_delete` state with `delete_account_confirm` region

**WARNINGs accepted (no change needed):**
- W1: terms_consent is a region in `entering` state, not a separate state — correct as designed
- W4: declined doesn't need viewed/acted-upon sub-states — over-engineering
- W6: source_booking is screen context, not a stored field — correct
- W9: auto-auth after registration is standard mobile UX — correct assumption
- W11-14: fixtures for registration/login/onboarding/help are nice-to-have, not errors — these screens are form-driven, not data-driven. Fixtures add value for data-display screens, not input screens.

**Observation:** Verification caught real structural gaps (skip mechanism, missing fixtures, incomplete state transitions) that authoring didn't. The pattern: state machines need to model EVERY path including skips and edge transitions, not just the happy path. Fixtures need to exist for every state that has a `fixture:` binding. Flows need to reference actual screen names, not internal states/regions. These are mechanical consistency checks that a validator would catch automatically — exactly what SFT's validation rules are designed for.

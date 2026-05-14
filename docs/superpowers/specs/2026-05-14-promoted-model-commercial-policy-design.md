# Promoted Model Commercial Policy Design

## Goal

Add a private-cloud To B commercial policy layer for promoted models. Admins can attach recommended models to an enterprise quota policy, set discounted selling prices, lock those commercial prices against bulk updates, and keep an auditable change history with rollback.

The feature is recommendation-first. It must not force routing to a specific model or channel unless a later feature explicitly adds that behavior.

## Current Context

The repo already separates several concepts:

- `ModelConfig.Price` is the base model price used by the system.
- `ModelConfig.Config["sync_price_locked"]` protects synced provider prices from PPIO/Novita sync overwrites.
- `GroupModelConfig.OverridePrice` overrides effective prices for a group.
- `QuotaPolicy` controls period quota, tier thresholds, RPM/TPM multipliers, blocked models, and price-based blocking.

Promoted model pricing should not be stored directly in `QuotaPolicy` and should not overwrite global `ModelConfig.Price` by default. It is a commercial selling-price layer tied to policy scope.

## Product Decisions

- This phase supports single-model promoted discount configuration as an operationally effective commercial policy.
- Single-model discounts apply only to new requests after the configuration becomes effective. Historical usage logs, settled usage amounts, reports, and bills are not recalculated.
- Promoted models remain recommendations. They do not force model selection, model access, channel routing, or channel priority.
- `channel_id` is a reference or price-source channel only. It is used for admin context and validation, not for request distribution.
- Commercial discounts are resolved dynamically at request pricing time. They are not materialized into `GroupModelConfig`.
- Price precedence is `GroupModelConfig.OverridePrice` first, then promoted commercial discount, then `ModelConfig.Price`. Existing group/customer override prices remain the strongest explicit contract price unless a future feature adds a separately reviewed "force policy commercial price" control.
- `base_price_snapshot` is for audit and rollback context. Current discount display can compare the override price with the latest `ModelConfig.Price` so admins can see drift after provider sync updates.
- Commercial `price_locked` is a visible admin-controlled business lock. It is separate from provider `sync_price_locked`.
- Bulk import, bulk discount recalculation, and historical billing recalculation are not included in this phase.
- Future bulk configuration updates must skip `price_locked=true` records by default and require an explicit force action to override them.
- Future historical billing corrections must be implemented as a separate adjustment workflow. They must not overwrite original request logs or settled usage facts.

## Product Semantics

The quota management page gains a `Promoted Models` tab. For each quota policy, admins can configure promoted model entries:

- model name
- optional source channel
- display name
- recommendation badge
- sort order
- enabled state
- base price snapshot
- override selling price
- computed discount rate
- commercial price lock
- effective and expiry time

Promoted models affect display order and billing price when the user is in scope of the policy. They do not restrict model access, block other models, or force traffic to a specific channel.

Operational behavior for the first phase:

- Admins can add, edit, enable, disable, lock, unlock, and roll back one promoted model entry at a time.
- An enabled entry with an active effective window is used for new request pricing when the requester is in scope of the quota policy and the requested model matches.
- Expired, disabled, or out-of-window entries are ignored and pricing falls back to the next source.

## Price Separation

The implementation must preserve these distinctions:

- Provider sync price: `ModelConfig.Price`
- Provider sync lock: `sync_price_locked`
- Group override price: `GroupModelConfig.OverridePrice`
- Commercial promoted price: new promoted model policy records

The promoted model `price_locked` field means "do not overwrite this commercial price through bulk imports, provider recommendation syncs, or auto-discount recalculation." It does not imply provider sync locking.

If the UI offers a separate "also lock provider sync price" option, that action may set `ModelConfig.Config["sync_price_locked"] = true`, but it must be explicit and audited separately.

Commercial price locking must be visible and controllable in the UI. It is a business-control feature, not a hidden backend safeguard. Admins must be able to see, filter, set, unset, and audit commercial locks so future bulk operations do not leave individual models unexpectedly frozen.

## Data Model

Add a new enterprise model, for example `PromotedModelPolicy`, stored in a separate table:

`enterprise_promoted_model_policies`

Recommended fields:

- `id`
- `quota_policy_id`
- `model`
- `channel_id`
- `display_name`
- `recommend_badge`
- `sort_order`
- `enabled`
- `base_price_snapshot`
- `override_price`
- `discount_rate`
- `price_locked`
- `effective_at`
- `expires_at`
- `version`
- `created_by`
- `updated_by`
- `deleted_at`

`base_price_snapshot` and `override_price` should use the existing `model.Price` shape so token, cache, image, and conditional prices can be represented without inventing a second price schema.

Add an audit table:

`enterprise_promoted_model_policy_audits`

Recommended fields:

- `id`
- `promoted_model_policy_id`
- `quota_policy_id`
- `action`
- `before`
- `after`
- `operator_id`
- `operator_name`
- `created_at`

The audit record stores JSON snapshots of changed entries. Rollback creates a new current version from an older audited snapshot instead of mutating history. Audit actions must distinguish create, update, enable, disable, price lock, price unlock, price change, forced locked-price override, delete, and rollback.

## Backend APIs

Add enterprise quota endpoints under the existing quota permission model:

- `GET /enterprise/quota/policies/:id/promoted-models`
- `POST /enterprise/quota/policies/:id/promoted-models`
- `PUT /enterprise/quota/policies/:id/promoted-models/:entry_id`
- `DELETE /enterprise/quota/policies/:id/promoted-models/:entry_id`
- `POST /enterprise/quota/policies/:id/promoted-models/:entry_id/rollback`
- `GET /enterprise/quota/policies/:id/promoted-models/audit`

View endpoints require `quota_manage_view`. Mutating endpoints require `quota_manage_manage`.

Validation rules:

- `quota_policy_id` must exist.
- `model` must exist in `ModelConfig`.
- `channel_id`, if provided, must exist and include the model in that channel's model list. It remains reference metadata and must not affect routing.
- `override_price` must pass existing `model.Price.ValidateConditionalPrices`.
- `expires_at` must be after `effective_at` when both are set.
- locked commercial prices cannot be changed by bulk update unless the request explicitly sets an override flag and writes an audit entry.
- new promoted model entries default to `price_locked=false`.
- single-entry edits may change locked prices or lock state only through authenticated `quota_manage_manage` actions with audit records.
- future bulk updates must skip `price_locked=true` records by default.
- future bulk updates may override locked prices only when the request explicitly sends a force flag such as `override_locked=true`.

## Billing Resolution

Add a pricing resolver step for enterprise requests:

1. Resolve the requester and effective quota policy.
2. Find an enabled promoted model entry for that policy and requested model.
3. Check effective time window.
4. If present, use `override_price` as the effective selling price.
5. Otherwise fall back to the base model price.

This step should live close to existing group-model price override resolution, not in frontend code. The client must never be trusted to provide the discounted price.

Use dynamic resolution during request pricing. Do not materialize promoted policy prices into `GroupModelConfig` when policies are bound, because materialization creates stale prices when department/user bindings, policy versions, or effective windows change.

Pricing precedence:

1. `GroupModelConfig.OverridePrice`
2. Active promoted commercial discount for the effective quota policy and requested model
3. `ModelConfig.Price`

This order protects existing group/customer price overrides from being unexpectedly replaced by a policy-level recommendation price.

## UI Design

In `web/src/pages/enterprise/quota.tsx`, add a quota-policy detail tab:

- `Basic Policy`
- `Binding Scope`
- `Promoted Models`
- `Change History`

The promoted models tab uses a dense management table:

- model
- source channel
- base price
- override price
- discount
- badge
- effective window
- status
- commercial lock
- actions

Actions:

- add promoted model
- edit price and metadata
- enable or disable
- lock or unlock commercial price
- rollback to previous version
- view audit history

The table should make clear that recommendation does not force routing. Avoid presenting source channel as an enforcement guarantee.

Commercial lock UI requirements:

- The table shows lock state for every row.
- The table can filter locked and unlocked rows.
- The add/edit form includes a `Lock commercial discount price` switch.
- Price inputs for locked rows show a lock indicator.
- Editing a locked price requires a confirmation step.
- Future bulk import and bulk discount operations must show `Skip locked prices` enabled by default.
- Future bulk operations can expose `Override locked prices`, but it must be an explicit opt-in and should be visually distinct from the default path.

Default behavior:

- Newly created promoted model rows are not locked by default.
- Manually entered contract prices may offer a "save and lock" option, but the admin must choose it.
- Future imported or auto-calculated discount prices must not become locked unless the import payload or admin action explicitly requests it.

Not in this phase:

- batch importing promoted models
- batch recalculating override prices from discount rates
- automatic commercial price refresh when provider prices change
- recalculating historical logs, usage, reports, or bills

## My Access Display

The enterprise user access page should show promoted models first when the current user has an effective quota policy with promoted entries:

- promoted models appear before normal models within the relevant provider group
- show recommendation badge and discounted price
- keep all existing available models visible unless blocked by existing policy rules

This preserves "recommended, not forced."

## Rollback Behavior

Rollback copies an older audited snapshot into a new current version:

- old state remains in audit history
- new version increments
- operator and timestamp are recorded
- cache invalidation runs after success

Rollback must not mutate provider sync prices or remove `sync_price_locked`.

## Cache And Consistency

Changes to promoted model entries should invalidate the same effective-access cache path used by model/group config changes, plus any new promoted-model cache key.

Request-time resolution should tolerate missing or expired promoted entries by falling back to current behavior.

## Testing

Backend tests:

- CRUD validation for promoted model entries.
- audit record creation for create, update, disable, delete, and rollback.
- locked commercial price resists normal bulk update.
- effective price resolver chooses promoted override for an in-window entry.
- resolver keeps `GroupModelConfig.OverridePrice` ahead of promoted commercial price.
- resolver falls back when policy, model, time window, or enabled state does not match.
- historical usage amounts are not recalculated by promoted model changes.
- provider `sync_price_locked` remains separate from commercial `price_locked`.

Frontend tests or focused verification:

- quota policy promoted tab loads entries.
- add/edit forms validate model, channel, price, and dates.
- locked entry communicates commercial lock semantics.
- My Access shows promoted entries first without hiding normal models.
- UI makes clear that reference channel does not force routing.

## Migration And Rollout

The migration is additive:

1. Create promoted model policy and audit tables.
2. Add APIs behind existing quota permissions.
3. Add resolver behind enterprise build path.
4. Add UI tab and My Access display.

Rollback is safe because no global model prices are overwritten. Disabling the resolver or hiding the tab leaves existing quota policy behavior unchanged.

Product acceptance criteria:

- An admin can configure one promoted model discount for a quota policy and see it become effective for new matching requests.
- A user in scope sees the promoted model before normal models in My Access, with badge and discounted price.
- A user out of scope, or a request for a non-promoted model, keeps current pricing behavior.
- Existing group override prices remain unchanged and take precedence.
- Commercial lock state is visible, editable, filterable, and audited.
- Audit history answers who changed which model's commercial price, when, and from/to which values.
- No historical usage amount changes when a promoted model entry is created, edited, disabled, locked, unlocked, or rolled back.

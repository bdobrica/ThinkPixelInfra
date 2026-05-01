# LLM Client Refactor Plan

This package currently concentrates core client behavior, transport internals, framework integration, example usage, and public exports inside [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py).

The main goal of this refactor is to split [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py) into smaller, more manageable components with single responsibilities, while preserving runtime behavior first and only then improving the public extraction boundary.

## Target Layout

Create the following files under [docker/models/external/external/llm_client](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client):

- `__init__.py`
- `client.py`
- `config.py`
- `deadline.py`
- `errors.py`
- `resources.py`
- `types.py`
- `transports/dns_cache.py`
- `integrations/fastapi.py`
- `examples/fastapi_app.py` or move the example to package docs instead of keeping it in runtime code

## Single-Responsibility Rule

The refactor must explicitly remove the current monolithic responsibilities from [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py).

After refactoring:

- `__init__.py` should only re-export stable public symbols.
- `client.py` should only own the HTTP client orchestration.
- `deadline.py` should only own deadline context helpers.
- `errors.py` should only own exception types.
- `config.py` should only own config dataclasses.
- `resources.py` should only own resource facade classes.
- `transports/dns_cache.py` should only own DNS cache and transport internals.
- `integrations/fastapi.py` should only own FastAPI-specific helpers and middleware.

## Phase 1: Mechanical Split

Goal: no semantic changes, only better file boundaries.

### 1. Move error classes

Create [errors.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/errors.py) with:

- `LLMClientError`
- `LLMConfigurationError`
- `LLMDeadlineExceeded`
- `LLMProviderError`

### 2. Move deadline helpers

Create [deadline.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/deadline.py) with:

- `_deadline_monotonic`
- `set_deadline_after_ms`
- `set_deadline_at`
- `reset_deadline`
- `current_deadline`
- `remaining_budget_seconds`

### 3. Move config dataclasses

Create [config.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/config.py) with:

- `RetryConfig`
- `LLMClientConfig`

### 4. Move resource facades

Create [resources.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/resources.py) with:

- `ChatCompletionsResource`
- `ResponsesResource`
- `EmbeddingsResource`

These classes may temporarily keep the current `request` argument shape during the first split.

### 5. Move DNS transport internals

Create [transports/dns_cache.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/transports/dns_cache.py) with:

- `DNSCacheEntry`
- `AsyncDNSCache`
- `CachingAsyncNetworkBackend`
- `DNSCachingAsyncHTTPTransport`

This module should be treated as advanced and version-sensitive because it depends on `httpcore` and `httpx` transport details.

### 6. Move FastAPI integration

Create [integrations/fastapi.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/integrations/fastapi.py) with:

- `DeadlineMiddleware`
- `llm_client_from_request`
- `translate_llm_exception`

### 7. Move core client

Create [client.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/client.py) with:

- `LLMClient`

`client.py` should import from:

- [config.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/config.py)
- [deadline.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/deadline.py)
- [errors.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/errors.py)
- [resources.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/resources.py)
- [transports/dns_cache.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/transports/dns_cache.py)

### 8. Shrink `__init__.py`

Rewrite [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py) so it becomes a thin public export layer only.

It should re-export stable core symbols such as:

- `LLMClient`
- `RetryConfig`
- `LLMClientConfig`
- `LLMClientError`
- `LLMConfigurationError`
- `LLMDeadlineExceeded`
- `LLMProviderError`
- `set_deadline_after_ms`
- `set_deadline_at`
- `reset_deadline`
- `current_deadline`
- `remaining_budget_seconds`

Do not keep transport internals, example code, or FastAPI-specific symbols in the root package by default.

## Phase 2: Clean Public API Boundary

Goal: remove framework coupling from the core client.

### 1. Replace `Request` in core signatures

Current core methods are framework-aware because they accept FastAPI `Request` objects directly.

Change the core client and resource facade methods to accept a framework-neutral cancellation abstraction instead.

Recommended shape:

```python
from collections.abc import Awaitable, Callable

DisconnectChecker = Callable[[], Awaitable[bool]]
```

Then update the core request path to use:

- `disconnect_checker: DisconnectChecker | None`

instead of:

- `request: Request | None`

### 2. Adapt FastAPI in the integration layer only

Create a helper in [integrations/fastapi.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/integrations/fastapi.py) that adapts `request.is_disconnected()` to `DisconnectChecker`.

The core package should no longer import FastAPI just to support disconnect cancellation.

### 3. Keep deadline context in core, header parsing in FastAPI integration

Retain the `ContextVar`-based deadline helpers in [deadline.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/deadline.py).

Keep header parsing middleware only in [integrations/fastapi.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/integrations/fastapi.py).

## Phase 3: Stabilize Transport and Typing

Goal: make the package extraction-ready.

### 1. Fix current type issues in transport internals

The current implementation shows typing mismatches around:

- socket option signatures
- `AsyncResponseStream` wrapping
- FastAPI fallback typing

Resolve these before treating the transport layer as reusable.

### 2. Make the DNS transport explicitly optional

The core client should work without importing transport internals unless DNS caching is enabled.

Keep DNS cache transport opt-in and documented as advanced behavior.

### 3. Remove runtime example from the package module

Move `EXAMPLE_FASTAPI_USAGE` out of runtime code and into either:

- package docs
- a dedicated examples module
- a small example application file

## Exact Symbol Move Checklist

### From `__init__.py` to `deadline.py`

- `_deadline_monotonic`
- `set_deadline_after_ms`
- `set_deadline_at`
- `reset_deadline`
- `current_deadline`
- `remaining_budget_seconds`

### From `__init__.py` to `errors.py`

- `LLMClientError`
- `LLMConfigurationError`
- `LLMDeadlineExceeded`
- `LLMProviderError`

### From `__init__.py` to `config.py`

- `RetryConfig`
- `LLMClientConfig`

### From `__init__.py` to `resources.py`

- `ChatCompletionsResource`
- `ResponsesResource`
- `EmbeddingsResource`

### From `__init__.py` to `transports/dns_cache.py`

- `DNSCacheEntry`
- `AsyncDNSCache`
- `CachingAsyncNetworkBackend`
- `DNSCachingAsyncHTTPTransport`

### From `__init__.py` to `integrations/fastapi.py`

- FastAPI imports
- `DeadlineMiddleware`
- `llm_client_from_request`
- `translate_llm_exception`

### Keep or move to `client.py`

- `LLMClient`
- helper methods used only by `LLMClient`

## Import Rewrite Checklist

After the split, update imports in the external gateway code.

### Current usage in the external model package

Code that currently imports from the package root can stay on root exports if those symbols remain public.

Examples:

```python
from .llm_client import LLMClient, RetryConfig
from .llm_client import DeadlineMiddleware, translate_llm_exception
```

### Recommended rewritten imports

Core imports:

```python
from .llm_client import LLMClient, RetryConfig
```

FastAPI integration imports:

```python
from .llm_client.integrations.fastapi import DeadlineMiddleware, translate_llm_exception
```

### Root package export policy

Avoid re-exporting the following from `__init__.py`:

- `DeadlineMiddleware`
- `llm_client_from_request`
- `translate_llm_exception`
- `AsyncDNSCache`
- `DNSCachingAsyncHTTPTransport`

Those should require explicit submodule imports.

## Validation Checklist

After the split:

1. Run type checking on the package.
2. Run compile checks for the package.
3. Validate current gateway imports still resolve.
4. Test request retries against retryable and non-retryable responses.
5. Test disconnect cancellation through the FastAPI integration layer.
6. Test deadline budget exhaustion across retries.
7. Test DNS cache behavior separately from the main client tests.

## Notes for Future Extraction

- Treat `transports/dns_cache.py` as internal or provisional until compatibility is pinned against specific `httpx` and `httpcore` versions.
- Treat `integrations/fastapi.py` as optional framework glue, not part of the minimal core dependency surface.
- Keep the root package narrow and boring. That is the biggest step toward making this a standalone library later.

## Execution Checklist

This section turns the refactor plan into the recommended execution order for this repository.

The priority is:

1. preserve runtime behavior
2. minimize import churn per step
3. keep each step locally verifiable
4. postpone signature changes until after the file split is stable

### Stage 0: Pre-flight snapshot

Goal: capture the current behavior before touching file boundaries.

- Read [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py) once end to end and mark each symbol with its target module.
- Record the current imports from the rest of the external gateway, especially the imports in [fastapi.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/fastapi.py) and [gateway.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/gateway.py) if they depend on the package root.
- Run a compile check for the package before starting.
- Keep the current public package imports working during Stage 1.

Stop condition:

- You have a symbol-to-file move map and a known-good pre-refactor compile result.

### Stage 1: Mechanical split with no API changes

Goal: split [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py) into smaller single-responsibility files without changing behavior.

#### Step 1. Create `errors.py`

- Move `LLMClientError`
- Move `LLMConfigurationError`
- Move `LLMDeadlineExceeded`
- Move `LLMProviderError`

Validation:

- Compile the package.
- Confirm imports in the still-monolithic `__init__.py` resolve from `errors.py`.

#### Step 2. Create `deadline.py`

- Move the deadline `ContextVar`
- Move deadline helper functions

Validation:

- Compile the package.
- Confirm the package still imports from the root unchanged.

#### Step 3. Create `config.py`

- Move `RetryConfig`
- Move `LLMClientConfig`

Validation:

- Compile the package.
- Confirm the core client still constructs normally.

#### Step 4. Create `resources.py`

- Move `ChatCompletionsResource`
- Move `ResponsesResource`
- Move `EmbeddingsResource`

Important constraint:

- Do not change method signatures yet.
- Keep the current `request` argument shape for now.

Validation:

- Compile the package.
- Confirm the resource facades can still be instantiated by `LLMClient`.

#### Step 5. Create `transports/dns_cache.py`

- Move `DNSCacheEntry`
- Move `AsyncDNSCache`
- Move `CachingAsyncNetworkBackend`
- Move `DNSCachingAsyncHTTPTransport`

Important constraint:

- Keep the transport behavior unchanged, even if the types are still imperfect.
- Do not try to solve transport typing and module splitting in the same step.

Validation:

- Compile the package.
- Confirm DNS-cached client construction still works when enabled.

#### Step 6. Create `integrations/fastapi.py`

- Move FastAPI imports and fallback handling
- Move `DeadlineMiddleware`
- Move `llm_client_from_request`
- Move `translate_llm_exception`

Important constraint:

- Keep behavior unchanged.
- Do not redesign the integration API yet.

Validation:

- Compile the package.
- Confirm the external gateway still imports the same symbols successfully.

#### Step 7. Create `client.py`

- Move `LLMClient`
- Move only helper methods used directly by `LLMClient`

Validation:

- Compile the package.
- Confirm the external gateway can still construct and close the client.

#### Step 8. Reduce `__init__.py`

- Replace the monolith with imports from the new modules.
- Keep the old root-level public imports working for now.

Important constraint:

- At the end of Stage 1, [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py) must be a thin export layer only.
- No business logic, no transport implementations, no middleware bodies, and no example app bodies should remain there.

Validation:

- Compile the package.
- Run diagnostics on the package.
- Confirm the current external gateway imports still resolve unchanged.

Stop condition:

- The file split is complete and the package still behaves the same from the caller’s point of view.

### Stage 2: Internal import cleanup

Goal: make the split readable and durable before changing any APIs.

- Remove any circular imports created by the split.
- Make imports directional: core modules should not depend on FastAPI integration modules.
- Ensure `resources.py` only depends on stable core symbols.
- Ensure `integrations/fastapi.py` depends on core, but core does not depend on integration.

Validation:

- Compile the package.
- Run diagnostics on the package.

Stop condition:

- Imports are one-directional and the split no longer relies on awkward back-imports from `__init__.py`.

### Stage 3: Narrow the root export surface

Goal: keep only stable public symbols in the package root.

Status: completed on 2026-05-01.

Completed work:

- Root exports were narrowed to a minimal stable entrypoint in [__init__.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/__init__.py): `LLMClient`, `RetryConfig`, and the public error types.
- FastAPI helpers remain available from [integrations/fastapi.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/integrations/fastapi.py) instead of the package root.
- Transport internals remain available from [transports/dns_cache.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/transports/dns_cache.py) instead of the package root.
- Example code was moved out of the root runtime module into [examples/fastapi_app.py](/home/bogdan/GitHub/ThinkPixelInfra/docker/models/external/external/llm_client/examples/fastapi_app.py).
- External gateway callers were already updated to use explicit FastAPI integration imports before this stage was closed.

Validation notes:

- Package compile validation passed after the root export cleanup.
- Runtime import validation passed after syncing the mirrored `/mnt/c/...` runtime tree used by the active interpreter in this environment.

- Keep root exports for:
	- `LLMClient`
	- `RetryConfig`
	- error classes
- Stop re-exporting:
	- FastAPI helpers
	- transport internals
	- example code
	- deadline helpers
	- `LLMClientConfig`

Update callers in the external gateway:

- Core imports stay at package root.
- FastAPI-specific imports move to `integrations.fastapi`.

Validation:

- Compile the package.
- Compile the external gateway package.
- Confirm no import sites still rely on removed root exports.

Stop condition:

- The root package is clean, small, and framework-neutral from an import perspective.

### Stage 4: Decouple core from FastAPI request types

Goal: remove framework-specific types from the core client API.

#### Step 1. Introduce a framework-neutral disconnect type

- Add `DisconnectChecker` in `types.py`.
- Use a minimal async callable type instead of `Request` in the core client.

Recommended shape:

```python
from collections.abc import Awaitable, Callable

DisconnectChecker = Callable[[], Awaitable[bool]]
```

#### Step 2. Update the core client

- Replace `request: Request | None` with `disconnect_checker: DisconnectChecker | None` in the core request path.
- Keep semantics unchanged.

#### Step 3. Update resource facades

- Replace their `request` argument with the framework-neutral form, or add a backwards-compatible transitional overload if needed.

#### Step 4. Adapt FastAPI only in the integration module

- Add a helper that converts `request.is_disconnected` to `DisconnectChecker`.
- Keep all FastAPI awareness inside `integrations/fastapi.py`.

Validation:

- Compile the package.
- Confirm gateway endpoints still cancel requests correctly on disconnect.
- Confirm no core module imports FastAPI or Starlette types.

Stop condition:

- The core package no longer depends on FastAPI types.

### Stage 5: Transport typing and hardening

Goal: stabilize the most fragile part of the package after the architecture is clean.

- Fix the socket option typing mismatches in the DNS transport module.
- Fix the `AsyncResponseStream` typing and behavior concerns.
- Review whether the transport layer should wrap or adapt sync and async response streams differently.
- Decide whether to pin `httpx` and `httpcore` versions for the transport module.

Important constraint:

- Do this only after the architectural split. Transport hardening is easier once it is isolated.

Validation:

- Run package diagnostics.
- Run compile checks.
- Exercise the client with and without DNS caching enabled.

Stop condition:

- Transport internals are isolated and type-clean enough to be treated as an optional advanced module.

### Stage 6: Remove runtime example from core package code

Goal: keep runtime modules focused on runtime behavior.

- Move `EXAMPLE_FASTAPI_USAGE` into one of:
	- `examples/fastapi_app.py`
	- package documentation
	- a separate usage example file outside the runtime package

Validation:

- Compile the package.
- Confirm no runtime code depends on the moved example.

Stop condition:

- The package contains implementation only, not embedded tutorial strings in the runtime modules.

### Stage 7: Extraction readiness pass

Goal: verify the package is ready to be treated as a standalone library candidate.

- Review root exports one final time.
- Confirm the package can be imported without FastAPI installed if only the core client is used.
- Confirm FastAPI integration is optional.
- Confirm transport internals are not required unless explicitly enabled.
- Confirm type errors are limited to acceptable internal exceptions, ideally zero.

Validation:

1. Compile the package.
2. Run type diagnostics.
3. Verify importability of core-only usage.
4. Verify importability of FastAPI integration when FastAPI is installed.
5. Verify DNS-cached and non-DNS-cached client creation.

Stop condition:

- The package can be extracted with minimal additional redesign.

## Lowest-Risk Order Summary

If you want the shortest practical execution order, do the work in this exact sequence:

1. `errors.py`
2. `deadline.py`
3. `config.py`
4. `resources.py`
5. `transports/dns_cache.py`
6. `integrations/fastapi.py`
7. `client.py`
8. shrink `__init__.py`
9. clean internal imports
10. narrow root exports
11. replace `Request` in core APIs
12. harden transport typing
13. move example out of runtime code

## What Not To Combine

To keep the refactor low risk, do not combine these in the same step:

- file splitting and API redesign
- file splitting and transport hardening
- root export cleanup and gateway import rewrites plus transport changes
- FastAPI decoupling and public extraction changes

Each of those combinations increases the blast radius unnecessarily.

# Provider UI Composition Design

## Software Design Document

## Revision History

| Version | Date | Modified By | Description |
| :---: | :---: | :---- | :---- |
| 0.1 | 2026-10-09 | Red Hat NCP Team | Initial draft |

---

# 1. Introduction

## 1.1 Purpose

This document describes how NICo's admin UI sidebar is extended dynamically
from provider-contributed resource type registrations. It covers the minimal
change required to `base.html` (Askama) and the recommended approach for
provider-owned page rendering.

## 1.2 Scope

- How `GetResourceTypes` responses drive the sidebar nav
- The recommended provider-owns-HTML approach and why it minimises NICo core changes
- How NICo routes proxied requests to provider-served pages
- Failure handling when a provider is unavailable

---

# 2. Design

## 2.1 `GetResourceTypes` in the provider lifecycle

NICo core calls `GetResourceTypes` once, after `Init` succeeds and after
`GetRoutes` routes are registered. The response is cached in a struct shared
with the Askama template context:

```rust
// In carbide-api-core, alongside configured_tools():
pub fn configured_provider_resource_types() -> &'static [ProviderNavEntry] { ... }

pub struct ProviderNavEntry {
    pub label: String,    // ResourceTypeDescriptor.plural, title-cased
    pub href: String,     // /org/:orgName/:apiName/{api_prefix}/ui/
}
```

NICo does not poll; if a provider restarts and its resource types change, the
cache is refreshed at the next provider re-registration (the discovery loop
re-calls `Init` → `GetResourceTypes`).

## 2.2 `base.html` change

One small addition to the existing Askama template adds the dynamic nav section:

```diff
 {% if !Self::tools().is_empty() %}
 <hr />
 <h3>Tools</h3>
 <ul>
   {% for tool in Self::tools() %}
   <li><a href="{{ tool.url }}">{{ tool.display_name }}</a></li>
   {% endfor %}
 </ul>
 {% endif %}
+{% if !Self::provider_resources().is_empty() %}
+<hr />
+<h3>Providers</h3>
+<ul>
+  {% for entry in Self::provider_resources() %}
+  <li><a href="{{ entry.href }}">{{ entry.label }}</a></li>
+  {% endfor %}
+</ul>
+{% endif %}
```

`Self::provider_resources()` is added to the `Base` trait in `api-web/src/lib.rs`
following the same pattern as `Self::tools()`:

```rust
fn provider_resources() -> &'static [ProviderNavEntry] {
    carbide_api_core::configured_provider_resource_types()
}
```

This is the only NICo core change required. No new templates, no generic
resource-list pages.

## 2.3 Provider-owns-HTML (recommended approach)

The provider serves complete HTML pages via `HandleRequest`. NICo's middleware
(auth, RBAC, org scoping) runs before the request reaches `HandleRequest`, so
the provider receives an already-authenticated request context in
`HTTPRequest.headers` (including the Keycloak session token).

**Advantages:**
- Zero NICo core template changes beyond the nav entry
- Provider controls its own page structure, styling, and data freshness
- Provider can use any server-side rendering approach (Go `html/template`,
  React SSR, static files)
- Provider upgrades don't require NICo core releases

**Request routing:**
NICo proxies requests whose path begins with `{api_prefix}` to `HandleRequest`.
The provider's internal Echo (or equivalent) handles routing within that
prefix. The nav entry links to `{api_prefix}/ui/` — the provider's root list
page.

```
Browser → GET /org/test-org/nico/provider/switch-config/ui/
  NICo API: path matches registered provider route prefix
  → HandleRequest(HTTPRequest{method: "GET", path: "/ui/", ...})
  → provider renders HTML → HTTPResponse{status: 200, body: <html>...}
  → NICo forwards response body to browser
```

## 2.4 Failure handling

If a provider pod is unavailable (connection refused, timeout on `HealthCheck`):

- Its nav entries are **hidden** — not shown as broken links or error states
- Requests to its proxied routes return HTTP 502 with a standard NICo error page
- No sidebar error is shown to operators; the entries simply disappear until the
  provider reconnects and re-registers

This matches the existing behavior for the `GetCapabilities` endpoint: features
are only listed when the provider is registered and healthy.

## 2.5 Action routing

Actions declared in `ResourceTypeDescriptor.actions` are rendered as buttons on
provider-served detail pages. The buttons call the action path (relative to
`api_prefix`) via JavaScript fetch, handled by the same `HandleRequest` proxy.
NICo does not need to know about individual action paths — they are fully
provider-owned.

Actions also appear in the OpenAPI fragment returned by `GetOpenAPIFragment`,
keeping the programmatic API and the UI consistent.

---

# 3. Non-goals

- **Generic resource-list templates in NICo core**: providers own their HTML.
  NICo does not render resource tables from provider-returned JSON.
- **Real-time sidebar updates**: the cache is refreshed at re-registration only.
  A provider cannot push a nav update without restarting.
- **Cross-provider resource references**: links between provider-served pages
  (e.g., a switch detail page linking to an instance) use absolute NICo URLs,
  not provider-local paths.

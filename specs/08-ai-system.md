# AI System

Models receive only typed, allowlisted tools backed by application services. They never receive SQL,
filesystem, arbitrary HTTP, unrestricted database, credential, membership, policy, lifecycle, or
deletion access.

Every analysis records provider/model identity, versioned prompt/schema/toolset, immutable inputs,
attempts, token/cost metadata, terminal state, evidence links, and evaluation version. Structured
output is schema-validated; every factual claim must cite accessible immutable evidence before a run
may succeed. Reruns and feedback append history rather than editing prior results.

Model unavailability degrades only declared probabilistic capabilities. Collection, deterministic
metrics, policies, health, comparisons, and operations remain available. Human confirmation tokens
are short-lived, action-bound, actor-bound, version-bound, one-use, and cannot authorize forbidden
actions.

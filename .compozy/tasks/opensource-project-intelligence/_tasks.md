---
schema_version: 'compozy.tasks/v2'
workflow: opensource-project-intelligence
graph:
  nodes:
    - id: task_01
      file: task_01.md
    - id: task_02
      file: task_02.md
    - id: task_03
      file: task_03.md
    - id: task_04
      file: task_04.md
    - id: task_05
      file: task_05.md
    - id: task_06
      file: task_06.md
    - id: task_07
      file: task_07.md
  edges:
    - from: task_01
      to: task_02
    - from: task_02
      to: task_03
    - from: task_03
      to: task_04
    - from: task_04
      to: task_05
    - from: task_05
      to: task_06
    - from: task_06
      to: task_07
---

# Open Source Project Intelligence delivery tasks

This graph decomposes the complete product vision into seven large, independently reviewable
vertical deliveries. The sequence is intentional: each task leaves a usable, verified foundation
for the next task, and no task may silently defer behavior that its assigned stories or tests require.

| Task                  | Delivery                                                                                | Type      | Complexity | Tests |
| --------------------- | --------------------------------------------------------------------------------------- | --------- | ---------- | ----: |
| [task_01](task_01.md) | Publish contracts and establish the generated platform foundation                       | infra     | critical   |    12 |
| [task_02](task_02.md) | Implement identity, local access governance, audit, and the localized application shell | fullstack | high       |   106 |
| [task_03](task_03.md) | Implement Projects, durable work, evidence ownership, and core source ingestion         | fullstack | critical   |   184 |
| [task_04](task_04.md) | Materialize deterministic metrics, health, contributors, and comparisons                | fullstack | high       |    61 |
| [task_05](task_05.md) | Add extended sources, knowledge retrieval, topics, releases, and immutable AI analyses  | fullstack | high       |   111 |
| [task_06](task_06.md) | Implement trends, policies, recommendations, radar, and alerts                          | fullstack | high       |    96 |
| [task_07](task_07.md) | Add bounded ADK/HITL, exports, and final operational hardening                          | fullstack | critical   |    57 |

Coverage totals: 274 unit tests, 298 integration tests, and 55 end-to-end tests; 627 unique cases.

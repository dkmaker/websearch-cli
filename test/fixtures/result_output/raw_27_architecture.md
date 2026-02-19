---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 7
cached: false
---

Implement a **saga orchestrator** that maintains an execution state machine and uses an **outbox pattern** with Change Data Capture (CDC) to ensure reliable message delivery, combined with **idempotent message processing** and a **dead-letter queue** for compensating actions that fail repeatedly.

## Core Architecture

The orchestrator should follow this structure:

- **Saga Coordinator Service**: Receives the initial request, creates a saga execution log in its local database, and maintains state transitions (e.g., PENDING → STEP_1_COMPLETE → STEP_2_COMPLETE → SUCCESS or COMPENSATING → COMPENSATED)
- **Outbox Table**: Each service writes both its local transaction results and outgoing messages to its own database. CDC tools like Debezium capture these changes and publish them to Apache Kafka
- **Participants**: Three microservices that receive commands from the orchestrator, execute local transactions, and emit reply messages via their own outbox tables

## Handling Compensating Action Failures

Since compensating actions themselves can fail, implement these mechanisms:

**Idempotency Guarantees:**
- Assign each command and compensating action a **unique message ID**
- Each service maintains a **journal table** recording consumed message IDs to detect and skip duplicates
- Design compensating transactions to be **idempotent**—executing them multiple times produces the same result as executing once (e.g., "set inventory to X" rather than "decrement inventory by Y")

**Retry Strategy with State Tracking:**
- When a compensating action fails, the orchestrator logs this in its saga execution state and schedules a retry
- Implement **exponential backoff** with predetermined retry counts
- Store retry attempts and failure reasons in the saga execution log for audit trails

**Dead-Letter Queue:**
- After exhausting retries, route failed compensating actions to a dead-letter queue for manual intervention
- This prevents the saga from becoming indefinitely stuck while ensuring visibility into failures

## Implementation Example Flow

For three services (Order, Payment, Inventory):

1. Order Service receives POST /orders and creates saga orchestrator with PENDING state
2. Orchestrator sends "Reserve Inventory" command to Inventory Service via outbox
3. Inventory Service processes command, updates database, sends reply via outbox
4. Orchestrator receives reply; if success, sends "Process Payment" command to Payment Service
5. If Payment Service fails, orchestrator triggers compensating action "Release Inventory" with the same message ID pattern
6. If "Release Inventory" fails, orchestrator retries with idempotency checks; if all retries fail, routes to dead-letter queue

## Key Implementation Considerations

- Use the **outbox pattern** (not dual writes) to ensure messages are sent atomically with local state updates
- Store the current saga step and compensation history in a state table to recover from orchestrator crashes
- Implement **at-least-once delivery semantics** for messages, relying on idempotent message processing to handle duplicates
- Consider frameworks like **Eventuate Tram Sagas** that provide built-in saga orchestration logic

The orchestrator approach is simpler than choreography because each service only responds to commands rather than listening to multiple event streams, providing a clearer audit trail, though it creates a central point of coordination that requires resilience mechanisms.
# E-Commerce Order Processing

Multi-language Scenario

## Overview

This scenario demonstrates a complete e-commerce order processing workflow using durable orchestrations with the Saga pattern for distributed transaction management.

### Workflow

```
Customer places order
    │
    ├─→ 1. Validate Order
    │       └─→ Check inventory, validate payment method
    │
    ├─→ 2. Reserve Inventory
    │       └─→ Decrement stock (compensate: release inventory)
    │
    ├─→ 3. Process Payment
    │       └─→ Charge customer (compensate: refund payment)
    │
    ├─→ 4. Create Shipment
    │       └─→ Generate shipping label (compensate: cancel shipment)
    │
    └─→ 5. Send Confirmation
            └─→ Email + push notification
```

If any step fails, all previous steps are automatically compensated (rolled back) in reverse order.

### Why Durable Execution?

Without durable execution, a failure during payment processing could leave inventory reserved but never purchased, or a charge applied without a shipment created. Durable orchestrations ensure:

- **Atomicity** — Either all steps complete or all are compensated
- **Visibility** — Full execution history viewable in the dashboard
- **Recoverability** — If the process crashes mid-flight, it resumes from the last checkpoint

## Implementations

| Language | Framework | Sample |
|----------|-----------|--------|
| .NET | Durable Functions | [Saga Pattern](../../durable-functions/dotnet/Saga/) |
| .NET | Durable Functions | [Order Processor](../../durable-functions/dotnet/OrderProcessor/) |

## Related Patterns

- [Function Chaining](../../durable-task-sdks/dotnet/FunctionChaining/) — Sequential activity execution
- [Human Interaction](../../durable-task-sdks/dotnet/HumanInteraction/) — Add approval steps to your workflow
- [Saga Pattern documentation →](https://learn.microsoft.com/azure/architecture/reference-architectures/saga/saga)

## Learn More

- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
- [Durable Functions Patterns](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview?tabs=csharp#application-patterns)

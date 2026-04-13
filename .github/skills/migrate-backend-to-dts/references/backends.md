# Existing Backend Reference

Detailed configuration reference for each existing Durable Functions backend, showing what to look for and what to remove during migration.

## Azure Storage (Default Backend)

The default backend when no `storageProvider` is configured.

### How to Identify

**host.json** — no `storageProvider` block, or `type` is `"azure"`:

```json
{
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub"
    }
  }
}
```

Or explicitly:

```json
{
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub",
      "storageProvider": {
        "type": "azure",
        "connectionStringName": "AzureWebJobsStorage"
      }
    }
  }
}
```

**Azure Storage-specific settings** you may see:

```json
{
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "azure",
        "connectionStringName": "AzureWebJobsStorage",
        "controlQueueBatchSize": 32,
        "controlQueueBufferThreshold": 128,
        "controlQueueVisibilityTimeout": "00:05:00",
        "maxQueuePollingInterval": "00:00:30",
        "partitionCount": 4,
        "trackingStoreConnectionStringName": "TrackingStorage",
        "trackingStoreNamePrefix": "DurableTask",
        "useLegacyPartitionManagement": false,
        "useTablePartitionManagement": true
      }
    }
  }
}
```

**Packages (.NET):**
- In-process: `Microsoft.Azure.WebJobs.Extensions.DurableTask`
- Isolated: `Microsoft.Azure.Functions.Worker.Extensions.DurableTask` (no AzureManaged suffix)

**Azure Resources Used:**
- Azure Storage account (Tables + Queues + Blobs)
  - Tables: `<HubName>Instances`, `<HubName>History`
  - Queues: `<HubName>-control-<N>`, `<HubName>-workitems`
  - Blobs: `<HubName>-leases`

### What to Remove

- Remove Storage-specific settings: `controlQueueBatchSize`, `controlQueueBufferThreshold`, `controlQueueVisibilityTimeout`, `maxQueuePollingInterval`, `partitionCount`, `trackingStoreConnectionStringName`, `trackingStoreNamePrefix`, `useLegacyPartitionManagement`, `useTablePartitionManagement`
- Remove `TrackingStorage` connection string (if separate tracking store was used)
- Keep `AzureWebJobsStorage` — still needed for the Azure Functions runtime

### What to Replace With

**.NET:**
```json
{
  "extensions": {
    "durableTask": {
      "hubName": "%TASKHUB_NAME%",
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING"
      }
    }
  }
}
```

**Python / JavaScript / Java:**
```json
{
  "extensions": {
    "durableTask": {
      "hubName": "default",
      "storageProvider": {
        "type": "durabletask-scheduler",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING"
      }
    }
  },
  "extensionBundle": {
    "id": "Microsoft.Azure.Functions.ExtensionBundle.Preview",
    "version": "[4.29.0, 5.0.0)"
  }
}
```

---

## Netherite Backend

High-throughput backend built on Azure Event Hubs and FASTER.

### How to Identify

**host.json:**

```json
{
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub",
      "storageProvider": {
        "type": "netherite",
        "storageConnectionName": "AzureWebJobsStorage",
        "eventHubsConnectionName": "EventHubsConnection",
        "partitionCount": 12
      }
    }
  }
}
```

**Netherite-specific settings** you may see:

```json
{
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "netherite",
        "storageConnectionName": "AzureWebJobsStorage",
        "eventHubsConnectionName": "EventHubsConnection",
        "partitionCount": 12,
        "CacheDebugger": false,
        "LogLevelLimit": "Information",
        "StorageLogLevelLimit": "Warning",
        "TransportLogLevelLimit": "Warning",
        "EventLogLevelLimit": "Warning",
        "WorkItemLogLevelLimit": "Warning",
        "TakeStateCheckpointWhenStoppingPartition": true,
        "MaxNumberBytesBetweenCheckpoints": 200000000,
        "MaxTimeMsBetweenCheckpoints": 60000,
        "IdleCheckpointFrequencyMs": 60000
      }
    }
  }
}
```

**Packages (.NET):**
- `Microsoft.Azure.DurableTask.Netherite.AzureFunctions`

**Connection Strings:**
- `EventHubsConnection` — Azure Event Hubs namespace connection string
- `AzureWebJobsStorage` — Azure Storage account

**Azure Resources Used:**
- Azure Event Hubs namespace (partition-per-partition mapping)
- Azure Storage account (for checkpoints and state via FASTER)

### What to Remove

- Remove `Microsoft.Azure.DurableTask.Netherite.AzureFunctions` NuGet package
- Remove `EventHubsConnection` from app settings
- Remove all Netherite-specific settings: `partitionCount`, `CacheDebugger`, `LogLevelLimit`, all `*LogLevelLimit` settings, `TakeStateCheckpointWhenStoppingPartition`, `MaxNumberBytesBetweenCheckpoints`, `MaxTimeMsBetweenCheckpoints`, `IdleCheckpointFrequencyMs`
- Consider deprovisioning Event Hubs namespace if only used by Netherite

### What to Replace With

**.NET:**
```json
{
  "extensions": {
    "durableTask": {
      "hubName": "%TASKHUB_NAME%",
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING"
      }
    }
  }
}
```

**Python / JavaScript / Java:**
```json
{
  "extensions": {
    "durableTask": {
      "hubName": "default",
      "storageProvider": {
        "type": "durabletask-scheduler",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING"
      }
    }
  },
  "extensionBundle": {
    "id": "Microsoft.Azure.Functions.ExtensionBundle.Preview",
    "version": "[4.29.0, 5.0.0)"
  }
}
```

---

## Microsoft SQL Server Backend

Backend using SQL Server for orchestration state storage.

### How to Identify

**host.json:**

```json
{
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub",
      "storageProvider": {
        "type": "mssql",
        "connectionStringName": "SQLDB_Connection",
        "createDatabaseIfNotExists": true
      }
    }
  }
}
```

**MSSQL-specific settings** you may see:

```json
{
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "mssql",
        "connectionStringName": "SQLDB_Connection",
        "taskEventLockTimeout": "00:02:00",
        "createDatabaseIfNotExists": true,
        "databaseName": "DurableFunctionsDB",
        "schemaName": "dt"
      }
    }
  }
}
```

**Packages (.NET):**
- `Microsoft.DurableTask.SqlServer.AzureFunctions`

**Connection Strings:**
- `SQLDB_Connection` — SQL Server connection string (e.g., `Server=...;Database=...;User ID=...;Password=...`)

**Azure Resources Used:**
- Azure SQL Database or SQL Server
  - Schema: `dt` (default)
  - Tables: `dt.Instances`, `dt.History`, `dt.Payloads`, `dt.NewEvents`, `dt.NewTasks`, `dt.Versions`, `dt.GlobalSettings`

### What to Remove

- Remove `Microsoft.DurableTask.SqlServer.AzureFunctions` NuGet package
- Remove `SQLDB_Connection` from app settings
- Remove all MSSQL-specific settings: `taskEventLockTimeout`, `createDatabaseIfNotExists`, `databaseName`, `schemaName`
- After confirming successful migration, drop the `dt.*` schema tables from your SQL database

### What to Replace With

**.NET:**
```json
{
  "extensions": {
    "durableTask": {
      "hubName": "%TASKHUB_NAME%",
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING"
      }
    }
  }
}
```

**Python / JavaScript / Java:**
```json
{
  "extensions": {
    "durableTask": {
      "hubName": "default",
      "storageProvider": {
        "type": "durabletask-scheduler",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING"
      }
    }
  },
  "extensionBundle": {
    "id": "Microsoft.Azure.Functions.ExtensionBundle.Preview",
    "version": "[4.29.0, 5.0.0)"
  }
}
```

---

## Backend Comparison

| Feature | Azure Storage | Netherite | MSSQL | Durable Task Scheduler |
|---------|--------------|-----------|-------|----------------------|
| **Transport** | Queue polling | Event Hubs streaming | SQL polling | gRPC push streaming |
| **State Store** | Table Storage | FASTER + Blob | SQL Server | Managed (built-in) |
| **Partitioning** | 4 (configurable) | Configurable (Event Hubs) | None | Managed (automatic) |
| **Auth Model** | Shared key / connection string | Shared key / connection string | SQL auth / Entra | Identity-only (Entra) |
| **Scaling** | Tied to partition count | Tied to Event Hubs partitions | Limited by SQL | Independent, automatic |
| **Dashboard** | None (use App Insights) | None | None | Built-in (dashboard.durabletask.io) |
| **Managed Service** | No (self-managed storage) | No (self-managed) | No (self-managed) | Yes (fully managed) |
| **Local Dev** | Azurite | Azurite + Event Hubs emulator | SQL Server | Docker emulator |
| **Large Payloads** | Blob overflow (automatic) | FASTER storage | SQL (limited) | Configurable blob offload |

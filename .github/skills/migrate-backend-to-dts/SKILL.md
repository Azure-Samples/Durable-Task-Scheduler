---
name: migrate-backend-to-dts
description: Migrate existing Azure Durable Functions apps from existing backend storage providers (Azure Storage, Netherite, MSSQL) to the Durable Task Scheduler. Use when switching backends, converting to azureManaged storage provider, upgrading from Azure Storage default provider, migrating from Netherite Event Hubs-based backend, migrating from Microsoft SQL Server backend, or modernizing Durable Functions infrastructure. Applies to .NET, Python, JavaScript/TypeScript, and Java Durable Functions apps that need to adopt the managed Durable Task Scheduler service.
---

# Migrate Durable Functions to Durable Task Scheduler

Step-by-step guide for migrating Azure Durable Functions apps from existing backend storage providers to the Durable Task Scheduler (DTS).

## Before You Start

### ⚠️ Critical Prerequisites

1. **Drain in-flight orchestrations.** DTS does not import state from other backends. All running orchestrations must complete or be terminated before switching.
2. **.NET apps must use isolated worker model.** DTS does not support the in-process hosting model. If your app uses in-process (`Microsoft.Azure.WebJobs.Extensions.DurableTask`), migrate to isolated worker first.
3. **Identity-based auth only.** DTS uses Microsoft Entra ID / managed identity — no shared keys or connection string secrets. Plan for RBAC setup.

## Step 1: Identify Your Current Backend

Inspect your `host.json` to determine which backend you're migrating from:

| Current `storageProvider.type` | Backend | Key Indicator |
|-------------------------------|---------|---------------|
| *(missing or empty)* | **Azure Storage** (default) | No explicit type; uses `AzureWebJobsStorage` connection |
| `"azure"` | **Azure Storage** (explicit) | Same as default |
| `"netherite"` | **Netherite** | Requires Event Hubs connection string |
| `"mssql"` | **Microsoft SQL** | Requires SQL Server connection string |

### Also check your packages for confirmation:

**.NET (.csproj):**

| Package | Backend |
|---------|---------|
| `Microsoft.Azure.WebJobs.Extensions.DurableTask` (no suffix) | Azure Storage (in-process — must also migrate to isolated) |
| `Microsoft.Azure.Functions.Worker.Extensions.DurableTask` (no suffix) | Azure Storage (isolated) |
| `Microsoft.Azure.DurableTask.Netherite.AzureFunctions` | Netherite |
| `Microsoft.DurableTask.SqlServer.AzureFunctions` | MSSQL |

**Python (requirements.txt):** `azure-functions-durable` — backend is configured in host.json only.

**JavaScript (package.json):** `durable-functions` — backend is configured in host.json only.

**Java (build.gradle/pom.xml):** `azure-functions-java-library` — backend is configured in host.json only.

## Step 2: Update host.json

Remove your old `storageProvider` block and replace it with the DTS configuration.

### Migrating from Azure Storage (default)

```json
// BEFORE — Azure Storage (default, no storageProvider block)
{
  "version": "2.0",
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub"
    }
  }
}

// AFTER — Durable Task Scheduler
{
  "version": "2.0",
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

### Migrating from Azure Storage (explicit)

```json
// BEFORE — Azure Storage (explicit type)
{
  "version": "2.0",
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

// AFTER — Durable Task Scheduler
{
  "version": "2.0",
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

### Migrating from Netherite

```json
// BEFORE — Netherite
{
  "version": "2.0",
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

// AFTER — Durable Task Scheduler
{
  "version": "2.0",
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

> **Netherite cleanup:** After migration, remove the `EventHubsConnection` setting and consider deprovisioning the Event Hubs namespace if no longer needed. DTS handles partitioning internally — `partitionCount` is not needed.

### Migrating from MSSQL

```json
// BEFORE — Microsoft SQL Server
{
  "version": "2.0",
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub",
      "storageProvider": {
        "type": "mssql",
        "connectionStringName": "SQLDB_Connection",
        "taskEventLockTimeout": "00:02:00",
        "createDatabaseIfNotExists": true,
        "schemaName": "dt"
      }
    }
  }
}

// AFTER — Durable Task Scheduler
{
  "version": "2.0",
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

> **MSSQL cleanup:** After migration, remove `SQLDB_Connection` from app settings. The `dt.*` schema tables in your SQL database can be dropped once you've confirmed the migration is successful.

### Non-.NET Languages (Python, JavaScript, Java)

For Python, JavaScript, and Java apps, migration is **configuration-only** — no code changes or package changes are required. You only need to update `host.json`.

There is one key difference from .NET: the extension bundle must be updated to the Preview bundle.

#### Python — Migrating from Azure Storage (default)

```json
// BEFORE — host.json (Azure Storage default)
{
  "version": "2.0",
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub"
    }
  },
  "extensionBundle": {
    "id": "Microsoft.Azure.Functions.ExtensionBundle",
    "version": "[4.0.0, 5.0.0)"
  }
}

// AFTER — host.json (Durable Task Scheduler)
{
  "version": "2.0",
  "logging": {
    "logLevel": {
      "DurableTask.Core": "Warning"
    }
  },
  "extensions": {
    "durableTask": {
      "hubName": "default",
      "storageProvider": {
        "type": "azureManaged",
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

**requirements.txt** — no changes needed:
```
azure-functions
azure-functions-durable
```

**local.settings.json:**
```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "python",
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"
  }
}
```

#### JavaScript / TypeScript — Migrating from Azure Storage (default)

```json
// BEFORE — host.json (Azure Storage default)
{
  "version": "2.0",
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub"
    }
  },
  "extensionBundle": {
    "id": "Microsoft.Azure.Functions.ExtensionBundle",
    "version": "[4.0.0, 5.0.0)"
  }
}

// AFTER — host.json (Durable Task Scheduler)
{
  "version": "2.0",
  "logging": {
    "logLevel": {
      "DurableTask.Core": "Warning"
    }
  },
  "extensions": {
    "durableTask": {
      "hubName": "default",
      "storageProvider": {
        "type": "azureManaged",
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

**package.json** — no changes needed:
```json
{
  "dependencies": {
    "@azure/functions": "^4.0.0",
    "durable-functions": "^3.0.0"
  }
}
```

**local.settings.json:**
```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "node",
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"
  }
}
```

#### Java — Migrating from Azure Storage (default)

```json
// BEFORE — host.json (Azure Storage default)
{
  "version": "2.0",
  "extensions": {
    "durableTask": {
      "hubName": "MyTaskHub"
    }
  },
  "extensionBundle": {
    "id": "Microsoft.Azure.Functions.ExtensionBundle",
    "version": "[4.0.0, 5.0.0)"
  }
}

// AFTER — host.json (Durable Task Scheduler)
{
  "version": "2.0",
  "logging": {
    "logLevel": {
      "DurableTask.Core": "Warning"
    }
  },
  "extensions": {
    "durableTask": {
      "hubName": "default",
      "storageProvider": {
        "type": "azureManaged",
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

**pom.xml** — no changes needed. Your existing dependencies stay the same:
```xml
<dependency>
    <groupId>com.microsoft.azure.functions</groupId>
    <artifactId>azure-functions-java-library</artifactId>
    <version>3.2.3</version>
</dependency>
<dependency>
    <groupId>com.microsoft</groupId>
    <artifactId>durabletask-azure-functions</artifactId>
    <version>1.7.0</version>
</dependency>
```

**local.settings.json:**
```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "java",
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"
  }
}
```

#### Non-.NET — Migrating from Netherite or MSSQL

The target configuration is the same regardless of which existing backend you're migrating from. Replace the old `storageProvider` block and update the extension bundle as shown above. The only additional step is removing the old backend's connection strings from your app settings:

- **From Netherite:** Remove `EventHubsConnection` (or your Event Hubs connection name)
- **From MSSQL:** Remove `SQLDB_Connection` (or your SQL connection name)

## Step 3: Update Packages (.NET Only)

Non-.NET languages do not need package changes — skip to Step 4.

### Remove old backend package

```xml
<!-- REMOVE one of these (whichever you have): -->
<PackageReference Include="Microsoft.Azure.DurableTask.Netherite.AzureFunctions" Version="*" />
<PackageReference Include="Microsoft.DurableTask.SqlServer.AzureFunctions" Version="*" />
```

### Add DTS backend package

```xml
<!-- ADD: -->
<PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.DurableTask.AzureManaged" Version="*" />
<PackageReference Include="Azure.Identity" Version="1.*" />
```

### If migrating from in-process to isolated worker

This is a larger migration. Replace the entire package set:

```xml
<!-- REMOVE all in-process packages: -->
<PackageReference Include="Microsoft.Azure.WebJobs.Extensions.DurableTask" Version="*" />
<PackageReference Include="Microsoft.NET.Sdk.Functions" Version="*" />

<!-- ADD isolated worker packages: -->
<PackageReference Include="Microsoft.Azure.Functions.Worker" Version="2.*" />
<PackageReference Include="Microsoft.Azure.Functions.Worker.Sdk" Version="2.*" />
<PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.Http" Version="3.*" />
<PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.Http.AspNetCore" Version="1.*" />
<PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.DurableTask" Version="1.*" />
<PackageReference Include="Microsoft.Azure.Functions.Worker.Extensions.DurableTask.AzureManaged" Version="*" />
<PackageReference Include="Azure.Identity" Version="1.*" />
```

> The in-process to isolated migration also requires code changes (new `Program.cs`, attribute changes, etc.). See [Microsoft's migration guide](https://learn.microsoft.com/azure/azure-functions/migrate-dotnet-to-isolated-model).

## Step 4: Update Connection Strings

### local.settings.json (local development)

```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None",
    "TASKHUB_NAME": "default"
  }
}
```

### Azure App Settings (production)

```
DURABLE_TASK_SCHEDULER_CONNECTION_STRING=Endpoint=https://<scheduler-name>.<region>.durabletask.io;Authentication=DefaultAzure
TASKHUB_NAME=<your-taskhub-name>
```

### Remove old connection strings

| Backend | Settings to Remove |
|---------|-------------------|
| Azure Storage | `AzureWebJobsStorage` is still needed for Functions runtime, but no longer used for Durable state |
| Netherite | `EventHubsConnection` (or your Event Hubs connection name) |
| MSSQL | `SQLDB_Connection` (or your SQL connection name) |

## Step 5: Set Up Authentication

DTS uses identity-based authentication. No shared keys.

### Local Development

No authentication needed — the emulator accepts unauthenticated requests:
```
Endpoint=http://localhost:8080;Authentication=None
```

### Azure Production

1. **Enable managed identity** on your Function App (system-assigned or user-assigned)
2. **Assign RBAC role** — grant the Function App's identity the `Durable Task Scheduler Task Hub Contributor` role on the DTS resource
3. **Use DefaultAzure authentication** in your connection string:
   ```
   Endpoint=https://<scheduler-name>.<region>.durabletask.io;Authentication=DefaultAzure
   ```

## Step 6: Handle Large Payloads (Optional)

If your orchestrations pass large inputs/outputs (>10 KB), enable large payload storage to avoid message size limits:

**.NET:**
```json
{
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING",
        "largePayloadStorageEnabled": true,
        "largePayloadStorageThresholdBytes": 10240
      },
      "hubName": "%TASKHUB_NAME%"
    }
  }
}
```

**Python / JavaScript / Java:**
```json
{
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING",
        "largePayloadStorageEnabled": true,
        "largePayloadStorageThresholdBytes": 10240
      },
      "hubName": "default"
    }
  }
}
```

This offloads large payloads to Azure Blob Storage via the `AzureWebJobsStorage` connection.

## Step 7: Validate Locally

```bash
# 1. Start the DTS emulator
docker run -d -p 8080:8080 -p 8082:8082 --name dts-emulator mcr.microsoft.com/dts/dts-emulator:latest

# 2. Start Azurite (required for Azure Functions runtime)
azurite start

# 3. Run your Function App
func start

# 4. Open the DTS dashboard to monitor orchestrations
open http://localhost:8082
```

Trigger your orchestrations and verify they complete successfully. Check the dashboard for execution history.

## Step 8: Deploy to Azure

1. **Provision a Durable Task Scheduler resource** in the Azure portal or via CLI
2. **Create a task hub** within the scheduler
3. **Configure managed identity** and RBAC (Step 5)
4. **Set app settings** with the production connection string (Step 4)
5. **Deploy your Function App**
6. **Verify** orchestrations run correctly in Azure using the [DTS dashboard](https://dashboard.durabletask.io)

## Migration Warnings

### ⚠️ No State Migration

Running orchestrations do **not** carry over between backends. Before switching:
- Wait for all in-flight orchestrations to complete, **or**
- Terminate remaining orchestrations and accept they won't resume

There is no tool to export/import orchestration state between backends.

### ⚠️ In-Process Model Not Supported

DTS only supports the **isolated worker model** for .NET. If your app uses `Microsoft.Azure.WebJobs.Extensions.DurableTask` (in-process), you must migrate to isolated worker first. See [Migrate to isolated worker](https://learn.microsoft.com/azure/azure-functions/migrate-dotnet-to-isolated-model).

### ⚠️ Task Hub Names

Task hub names from your old backend won't conflict with DTS — they are separate systems. You can use any name for your DTS task hub.

### ⚠️ Netherite-Specific

- Remove Event Hubs namespace connection strings
- Remove `partitionCount` configuration — DTS manages partitions automatically
- Event Hubs resources can be deprovisioned if only used for Netherite

### ⚠️ MSSQL-Specific

- Remove SQL connection strings from app settings
- The `dt.*` schema tables can be dropped after confirming successful migration
- `taskEventLockTimeout`, `createDatabaseIfNotExists`, and `schemaName` settings are not used by DTS

### ⚠️ Custom Status and External Events

These APIs (`SetCustomStatus`, `RaiseEventAsync`, `WaitForExternalEvent`) work identically on DTS. **No code changes needed.**

### ⚠️ Durable Entities

Durable Entities are supported on DTS with .NET isolated worker. No code changes needed — only the backend configuration changes.

## Migration Checklist

- [ ] All in-flight orchestrations drained or terminated
- [ ] .NET app uses isolated worker model (not in-process)
- [ ] `host.json` updated with `azureManaged` storage provider
- [ ] Extension bundle updated to Preview (non-.NET only)
- [ ] Old backend NuGet packages removed (.NET only)
- [ ] `Microsoft.Azure.Functions.Worker.Extensions.DurableTask.AzureManaged` added (.NET only)
- [ ] `Azure.Identity` package added (.NET only)
- [ ] Old connection strings removed from app settings
- [ ] DTS connection string added to app settings
- [ ] DTS emulator tested locally
- [ ] Managed identity configured in Azure
- [ ] RBAC role assigned (`Durable Task Scheduler Task Hub Contributor`)
- [ ] Large payload storage configured (if needed)
- [ ] Deployed and validated in Azure

## References

- [Durable Task Scheduler overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Develop with Durable Task Scheduler (migration guide)](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler-functions)
- [Identity-based authentication](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-identity)
- [Migrate .NET to isolated worker model](https://learn.microsoft.com/azure/azure-functions/migrate-dotnet-to-isolated-model)
- [references/backends.md](references/backends.md) — Detailed backend comparison and configuration
- [references/setup.md](references/setup.md) — DTS provisioning, emulator, and identity setup

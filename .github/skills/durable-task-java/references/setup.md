# Durable Task Java SDK Setup and Deployment

Comprehensive setup guide for Azure Durable Task Scheduler with Java applications.

## Local Development with Docker Emulator

### Quick Start

```bash
# Pull the emulator image
docker pull mcr.microsoft.com/dts/dts-emulator:latest

# Run the emulator
docker run -d \
    -p 8080:8080 \
    -p 8082:8082 \
    --name dts-emulator \
    mcr.microsoft.com/dts/dts-emulator:latest

# Dashboard available at http://localhost:8082
```

### Docker Compose Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  dts-emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"  # gRPC endpoint
      - "8082:8082"  # Dashboard
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8082"]
      interval: 5s
      timeout: 3s
      retries: 3

  worker:
    build: .
    depends_on:
      dts-emulator:
        condition: service_healthy
    environment:
      - DURABLE_TASK_CONNECTION_STRING=Endpoint=http://dts-emulator:8080;TaskHub=default;Authentication=None
```

### Multi-Hub Development

```yaml
# docker-compose-multi-hub.yml
version: '3.8'

services:
  dts-emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"
      - "8082:8082"

  order-worker:
    build: ./order-service
    environment:
      - DURABLE_TASK_CONNECTION_STRING=Endpoint=http://dts-emulator:8080;TaskHub=orders;Authentication=None

  notification-worker:
    build: ./notification-service
    environment:
      - DURABLE_TASK_CONNECTION_STRING=Endpoint=http://dts-emulator:8080;TaskHub=notifications;Authentication=None
```

## Azure Durable Task Scheduler Provisioning

### Prerequisites

```bash
# Install/update Azure CLI
brew install azure-cli  # macOS
# or
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash  # Linux

# Login to Azure
az login

# Set subscription
az account set --subscription "your-subscription-id"
```

### Create Durable Task Scheduler

```bash
# Variables
RESOURCE_GROUP="my-dts-rg"
LOCATION="eastus"
SCHEDULER_NAME="my-dts-scheduler"
TASKHUB_NAME="my-taskhub"

# Create resource group
az group create --name $RESOURCE_GROUP --location $LOCATION

# Create Durable Task Scheduler namespace
az durabletask scheduler create \
    --name $SCHEDULER_NAME \
    --resource-group $RESOURCE_GROUP \
    --location $LOCATION \
    --sku "standard"

# Create a Task Hub
az durabletask taskhub create \
    --scheduler-name $SCHEDULER_NAME \
    --resource-group $RESOURCE_GROUP \
    --name $TASKHUB_NAME

# Get the endpoint
ENDPOINT=$(az durabletask scheduler show \
    --name $SCHEDULER_NAME \
    --resource-group $RESOURCE_GROUP \
    --query "properties.endpoint" -o tsv)

echo "Connection String: Endpoint=$ENDPOINT;TaskHub=$TASKHUB_NAME;Authentication=DefaultAzure"
```

### Bicep Template

```bicep
// main.bicep
@description('Name of the Durable Task Scheduler')
param schedulerName string

@description('Location for resources')
param location string = resourceGroup().location

@description('Task Hub name')
param taskHubName string = 'default'

@description('SKU for the scheduler')
@allowed(['basic', 'standard', 'premium'])
param sku string = 'standard'

resource scheduler 'Microsoft.DurableTask/schedulers@2025-11-01' = {
  name: schedulerName
  location: location
  properties: {
    sku: {
      name: sku
    }
  }
}

resource taskHub 'Microsoft.DurableTask/schedulers/taskHubs@2025-11-01' = {
  parent: scheduler
  name: taskHubName
  properties: {}
}

output endpoint string = scheduler.properties.endpoint
output connectionString string = 'Endpoint=${scheduler.properties.endpoint};TaskHub=${taskHubName};Authentication=DefaultAzure'
```

Deploy with:
```bash
az deployment group create \
    --resource-group $RESOURCE_GROUP \
    --template-file main.bicep \
    --parameters schedulerName=$SCHEDULER_NAME taskHubName=$TASKHUB_NAME
```

## Authentication Configuration

### DefaultAzureCredential (Recommended)

Works locally with Azure CLI credentials and in Azure with Managed Identity:

```java
String connectionString = "Endpoint=https://my-scheduler.region.durabletask.io;TaskHub=my-hub;Authentication=DefaultAzure";

DurableTaskGrpcWorker worker = new DurableTaskGrpcWorkerBuilder()
    .connectionString(connectionString)
    .addOrchestration("MyOrchestration", ctx -> { /* ... */ })
    .build();
```

Dependencies required:
```xml
<dependency>
    <groupId>com.azure</groupId>
    <artifactId>azure-identity</artifactId>
    <version>1.11.0</version>
</dependency>
```

### Managed Identity

```java
// System-assigned managed identity
String connectionString = "Endpoint=https://my-scheduler.region.durabletask.io;TaskHub=my-hub;Authentication=ManagedIdentity";

// User-assigned managed identity
String connectionString = "Endpoint=https://my-scheduler.region.durabletask.io;TaskHub=my-hub;Authentication=ManagedIdentity;ClientId=<client-id>";
```

### Azure CLI (Local Development)

```bash
# Login to Azure CLI
az login

# Your Java app will automatically use Azure CLI credentials
# with Authentication=DefaultAzure
```

### Role Assignments

Grant the worker/client identity the `Durable Task Data Owner` role:

```bash
# Get the scheduler resource ID
SCHEDULER_ID=$(az durabletask scheduler show \
    --name $SCHEDULER_NAME \
    --resource-group $RESOURCE_GROUP \
    --query id -o tsv)

# Assign role to managed identity
az role assignment create \
    --assignee "<principal-id>" \
    --role "Durable Task Data Owner" \
    --scope $SCHEDULER_ID
```

## Application Integration

### Console Application

```java
// Main.java
import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.*;
import java.time.Duration;

public class Main {
    public static void main(String[] args) throws Exception {
        String connectionString = getConnectionString();
        
        // Create worker
        DurableTaskGrpcWorker worker = new DurableTaskGrpcWorkerBuilder()
            .connectionString(connectionString)
            .addOrchestration("ProcessOrder", Orchestrations::processOrder)
            .addActivity("ValidateOrder", Activities::validateOrder)
            .addActivity("ProcessPayment", Activities::processPayment)
            .addActivity("SendConfirmation", Activities::sendConfirmation)
            .build();
        
        // Start worker in background thread
        Thread workerThread = new Thread(() -> {
            try {
                worker.start();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        });
        workerThread.start();
        
        // Create client
        DurableTaskClient client = new DurableTaskGrpcClientBuilder()
            .connectionString(connectionString)
            .build();
        
        // Schedule orchestration
        OrderInput input = new OrderInput("order-123", 99.99);
        String instanceId = client.scheduleNewOrchestrationInstance("ProcessOrder", input);
        System.out.println("Started: " + instanceId);
        
        // Wait for result
        OrchestrationMetadata result = client.waitForInstanceCompletion(
            instanceId, Duration.ofMinutes(5), true);
        
        System.out.println("Status: " + result.getRuntimeStatus());
        System.out.println("Output: " + result.readOutputAs(String.class));
        
        // Cleanup
        worker.close();
        client.close();
    }
    
    private static String getConnectionString() {
        String cs = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        return cs != null ? cs : "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
    }
}
```

### Spring Boot Integration

```java
// DurableTaskConfig.java
import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.*;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.*;

@Configuration
public class DurableTaskConfig {
    
    @Value("${durable-task.connection-string}")
    private String connectionString;
    
    @Bean
    public DurableTaskClient durableTaskClient() {
        return new DurableTaskGrpcClientBuilder()
            .connectionString(connectionString)
            .build();
    }
    
    @Bean
    public DurableTaskGrpcWorker durableTaskWorker(
            List<OrchestrationDefinition> orchestrations,
            List<ActivityDefinition> activities) {
        
        DurableTaskGrpcWorkerBuilder builder = new DurableTaskGrpcWorkerBuilder()
            .connectionString(connectionString);
        
        for (OrchestrationDefinition orch : orchestrations) {
            builder.addOrchestration(orch.getName(), orch.getImplementation());
        }
        
        for (ActivityDefinition act : activities) {
            builder.addActivity(act.getName(), act.getImplementation());
        }
        
        return builder.build();
    }
}

// WorkerLifecycle.java
import org.springframework.stereotype.Component;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;

@Component
public class WorkerLifecycle {
    
    private final DurableTaskGrpcWorker worker;
    private Thread workerThread;
    
    public WorkerLifecycle(DurableTaskGrpcWorker worker) {
        this.worker = worker;
    }
    
    @PostConstruct
    public void start() {
        workerThread = new Thread(() -> {
            try {
                worker.start();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        });
        workerThread.setDaemon(true);
        workerThread.start();
    }
    
    @PreDestroy
    public void stop() {
        try {
            worker.close();
        } catch (Exception e) {
            // Log error
        }
    }
}

// WorkflowController.java
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/workflows")
public class WorkflowController {
    
    private final DurableTaskClient client;
    
    public WorkflowController(DurableTaskClient client) {
        this.client = client;
    }
    
    @PostMapping("/orders")
    public WorkflowResponse startOrder(@RequestBody OrderInput input) {
        String instanceId = client.scheduleNewOrchestrationInstance("ProcessOrder", input);
        return new WorkflowResponse(instanceId, "Started");
    }
    
    @GetMapping("/{instanceId}")
    public WorkflowStatus getStatus(@PathVariable String instanceId) throws Exception {
        OrchestrationMetadata metadata = client.getInstanceMetadata(instanceId, true);
        return new WorkflowStatus(
            instanceId,
            metadata.getRuntimeStatus().toString(),
            metadata.getCustomStatus()
        );
    }
    
    @PostMapping("/{instanceId}/events/{eventName}")
    public void raiseEvent(
            @PathVariable String instanceId,
            @PathVariable String eventName,
            @RequestBody Object eventData) {
        client.raiseEvent(instanceId, eventName, eventData);
    }
}
```

### application.yml for Spring Boot

```yaml
durable-task:
  connection-string: ${DURABLE_TASK_CONNECTION_STRING:Endpoint=http://localhost:8080;TaskHub=default;Authentication=None}
```

## Deployment Options

### Container Apps

```yaml
# container-app.yaml
apiVersion: apps/v1
kind: ContainerApp
metadata:
  name: dts-worker
spec:
  template:
    containers:
      - name: worker
        image: myregistry.azurecr.io/my-worker:latest
        env:
          - name: DURABLE_TASK_CONNECTION_STRING
            secretRef: dts-connection-string
        resources:
          cpu: 0.5
          memory: 1Gi
  scale:
    minReplicas: 1
    maxReplicas: 10
    rules:
      - name: queue-scaling
        custom:
          type: external
          metadata:
            scalerAddress: "azure-scheduler-scaler:5050"
```

### Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dts-worker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: dts-worker
  template:
    metadata:
      labels:
        app: dts-worker
    spec:
      serviceAccountName: dts-worker-sa
      containers:
        - name: worker
          image: myregistry.azurecr.io/my-worker:latest
          env:
            - name: DURABLE_TASK_CONNECTION_STRING
              valueFrom:
                secretKeyRef:
                  name: dts-secrets
                  key: connection-string
            - name: AZURE_CLIENT_ID  # For workload identity
              valueFrom:
                secretKeyRef:
                  name: dts-secrets
                  key: client-id
          resources:
            requests:
              cpu: 200m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 1Gi
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
```

### Dockerfile

```dockerfile
# Dockerfile
FROM eclipse-temurin:17-jdk as builder

WORKDIR /app
COPY pom.xml .
COPY src ./src

RUN apt-get update && apt-get install -y maven
RUN mvn clean package -DskipTests

FROM eclipse-temurin:17-jre

WORKDIR /app
COPY --from=builder /app/target/*.jar app.jar

ENV JAVA_OPTS="-XX:+UseContainerSupport -XX:MaxRAMPercentage=75.0"

EXPOSE 8080
ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]
```

## Logging and Monitoring

### SLF4J Configuration

```xml
<!-- logback.xml -->
<configuration>
    <appender name="CONSOLE" class="ch.qos.logback.core.ConsoleAppender">
        <encoder>
            <pattern>%d{HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
        </encoder>
    </appender>
    
    <logger name="com.microsoft.durabletask" level="INFO"/>
    <logger name="io.grpc" level="WARN"/>
    
    <root level="INFO">
        <appender-ref ref="CONSOLE"/>
    </root>
</configuration>
```

### Application Insights Integration

```xml
<dependency>
    <groupId>com.microsoft.azure</groupId>
    <artifactId>applicationinsights-runtime-attach</artifactId>
    <version>3.4.18</version>
</dependency>
```

```java
// Enable in main method
import com.microsoft.applicationinsights.attach.ApplicationInsights;

public class Main {
    public static void main(String[] args) {
        ApplicationInsights.attach();
        // ... rest of application
    }
}
```

### Health Check Endpoint

```java
// For containerized deployments, add health endpoints
import com.sun.net.httpserver.*;
import java.io.*;
import java.net.*;

public class HealthServer {
    private final HttpServer server;
    
    public HealthServer(int port) throws IOException {
        server = HttpServer.create(new InetSocketAddress(port), 0);
        
        server.createContext("/health", exchange -> {
            String response = "{\"status\":\"healthy\"}";
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(200, response.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(response.getBytes());
            }
        });
        
        server.createContext("/ready", exchange -> {
            String response = "{\"status\":\"ready\"}";
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(200, response.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(response.getBytes());
            }
        });
        
        server.setExecutor(null);
    }
    
    public void start() {
        server.start();
    }
    
    public void stop() {
        server.stop(0);
    }
}
```

## Maven Project Template

### pom.xml

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>com.example</groupId>
    <artifactId>durable-task-worker</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>

    <properties>
        <maven.compiler.source>17</maven.compiler.source>
        <maven.compiler.target>17</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
        <durabletask.version>1.6.2</durabletask.version>
    </properties>

    <dependencies>
        <!-- Durable Task SDK -->
        <dependency>
            <groupId>com.microsoft.durabletask</groupId>
            <artifactId>durabletask-client</artifactId>
            <version>${durabletask.version}</version>
        </dependency>
        <dependency>
            <groupId>com.microsoft.durabletask</groupId>
            <artifactId>durabletask-azuremanaged</artifactId>
            <version>${durabletask.version}</version>
        </dependency>
        
        <!-- Azure Identity -->
        <dependency>
            <groupId>com.azure</groupId>
            <artifactId>azure-identity</artifactId>
            <version>1.11.0</version>
        </dependency>
        
        <!-- Logging -->
        <dependency>
            <groupId>org.slf4j</groupId>
            <artifactId>slf4j-api</artifactId>
            <version>2.0.9</version>
        </dependency>
        <dependency>
            <groupId>ch.qos.logback</groupId>
            <artifactId>logback-classic</artifactId>
            <version>1.4.11</version>
        </dependency>
        
        <!-- JSON Processing -->
        <dependency>
            <groupId>com.fasterxml.jackson.core</groupId>
            <artifactId>jackson-databind</artifactId>
            <version>2.15.2</version>
        </dependency>
        
        <!-- Testing -->
        <dependency>
            <groupId>org.junit.jupiter</groupId>
            <artifactId>junit-jupiter</artifactId>
            <version>5.10.0</version>
            <scope>test</scope>
        </dependency>
    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-compiler-plugin</artifactId>
                <version>3.11.0</version>
            </plugin>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-jar-plugin</artifactId>
                <version>3.3.0</version>
                <configuration>
                    <archive>
                        <manifest>
                            <mainClass>com.example.Main</mainClass>
                        </manifest>
                    </archive>
                </configuration>
            </plugin>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-shade-plugin</artifactId>
                <version>3.5.0</version>
                <executions>
                    <execution>
                        <phase>package</phase>
                        <goals>
                            <goal>shade</goal>
                        </goals>
                        <configuration>
                            <createDependencyReducedPom>false</createDependencyReducedPom>
                        </configuration>
                    </execution>
                </executions>
            </plugin>
        </plugins>
    </build>
</project>
```

## Testing

### Unit Testing Orchestrations

```java
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class OrchestrationTests {
    
    @Test
    void testOrderWorkflowSuccess() {
        // Create mock context
        TaskOrchestrationContext ctx = mock(TaskOrchestrationContext.class);
        
        // Setup input
        OrderInput input = new OrderInput("order-123", 99.99);
        when(ctx.getInput(OrderInput.class)).thenReturn(input);
        
        // Setup activity calls
        when(ctx.callActivity(eq("ValidateOrder"), any(), eq(Boolean.class)))
            .thenReturn(completedTask(true));
        when(ctx.callActivity(eq("ProcessPayment"), any(), eq(PaymentResult.class)))
            .thenReturn(completedTask(new PaymentResult(true, "tx-123")));
        when(ctx.callActivity(eq("SendConfirmation"), any(), eq(Void.class)))
            .thenReturn(completedTask(null));
        
        // Execute orchestration
        Object result = Orchestrations.processOrder(ctx);
        
        // Verify
        assertNotNull(result);
        verify(ctx).callActivity(eq("ValidateOrder"), any(), eq(Boolean.class));
        verify(ctx).callActivity(eq("ProcessPayment"), any(), eq(PaymentResult.class));
        verify(ctx).callActivity(eq("SendConfirmation"), any(), eq(Void.class));
    }
    
    private <T> Task<T> completedTask(T value) {
        Task<T> task = mock(Task.class);
        when(task.await()).thenReturn(value);
        return task;
    }
}
```

### Integration Testing with Emulator

```java
import org.junit.jupiter.api.*;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

@Testcontainers
class IntegrationTests {
    
    @Container
    static GenericContainer<?> emulator = new GenericContainer<>("mcr.microsoft.com/dts/dts-emulator:latest")
        .withExposedPorts(8080, 8082);
    
    private DurableTaskClient client;
    private DurableTaskGrpcWorker worker;
    
    @BeforeEach
    void setup() {
        String connectionString = String.format(
            "Endpoint=http://%s:%d;TaskHub=test;Authentication=None",
            emulator.getHost(),
            emulator.getMappedPort(8080)
        );
        
        worker = new DurableTaskGrpcWorkerBuilder()
            .connectionString(connectionString)
            .addOrchestration("TestOrchestration", ctx -> {
                String input = ctx.getInput(String.class);
                return "Hello, " + input + "!";
            })
            .build();
        
        new Thread(() -> {
            try { worker.start(); } catch (Exception e) {}
        }).start();
        
        client = new DurableTaskGrpcClientBuilder()
            .connectionString(connectionString)
            .build();
    }
    
    @AfterEach
    void teardown() throws Exception {
        worker.close();
        client.close();
    }
    
    @Test
    void testSimpleOrchestration() throws Exception {
        String instanceId = client.scheduleNewOrchestrationInstance("TestOrchestration", "World");
        
        OrchestrationMetadata result = client.waitForInstanceCompletion(
            instanceId, Duration.ofSeconds(30), true);
        
        assertEquals(OrchestrationRuntimeStatus.COMPLETED, result.getRuntimeStatus());
        assertEquals("Hello, World!", result.readOutputAs(String.class));
    }
}
```

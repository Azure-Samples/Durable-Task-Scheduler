// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

/**
 * Shared helper for building DTS connection strings from environment variables.
 *
 * <p>Supports three modes:
 * <ul>
 *   <li><b>Local emulator</b> (default): {@code Endpoint=http://localhost:8080;TaskHub=default;Authentication=None}</li>
 *   <li><b>Azure with Managed Identity</b>: when {@code AZURE_MANAGED_IDENTITY_CLIENT_ID} is set</li>
 *   <li><b>Azure with DefaultAzure</b>: when {@code ENDPOINT} points to a non-localhost address</li>
 * </ul>
 */
final class ConnectionHelper {

    private ConnectionHelper() {
    }

    static String getConnectionString() {
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString != null) {
            return connectionString;
        }

        String endpoint = System.getenv("ENDPOINT");
        if (endpoint == null) {
            endpoint = "http://localhost:8080";
        }

        String taskHub = System.getenv("TASKHUB");
        if (taskHub == null) {
            taskHub = "default";
        }

        String managedIdentityClientId = System.getenv("AZURE_MANAGED_IDENTITY_CLIENT_ID");
        boolean isLocalEmulator = endpoint.equals("http://localhost:8080");

        if (isLocalEmulator) {
            return String.format("Endpoint=%s;TaskHub=%s;Authentication=None", endpoint, taskHub);
        } else if (managedIdentityClientId != null && !managedIdentityClientId.isEmpty()) {
            return String.format("Endpoint=%s;TaskHub=%s;Authentication=ManagedIdentity;ClientID=%s",
                    endpoint, taskHub, managedIdentityClientId);
        } else {
            return String.format("Endpoint=%s;TaskHub=%s;Authentication=DefaultAzure", endpoint, taskHub);
        }
    }
}

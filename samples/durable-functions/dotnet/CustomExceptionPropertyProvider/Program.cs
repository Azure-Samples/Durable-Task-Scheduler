// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

using CustomExceptionPropertyProvider;
using Microsoft.Azure.Functions.Worker.Builder;
using Microsoft.DurableTask.Worker;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

FunctionsApplicationBuilder builder = FunctionsApplication.CreateBuilder(args);

// Register the custom exception properties provider. When an activity throws,
// the Durable worker consults this provider to extract structured properties
// from the exception and attaches them to FailureDetails.Properties, which is
// then surfaced to the orchestration that observes the failure.
builder.Services.AddSingleton<IExceptionPropertiesProvider, BusinessExceptionPropertiesProvider>();

builder.Build().Run();

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Worker;

namespace CustomExceptionPropertyProvider;

// =============================================================================
// Orchestration + activity
// =============================================================================

/// <summary>
/// Demonstrates custom FailureDetails properties. An activity throws a
/// <see cref="BusinessValidationException"/>; the registered
/// <see cref="IExceptionPropertiesProvider"/> attaches its structured fields to
/// the failure. The orchestration catches the propagated
/// <see cref="TaskFailedException"/> and returns its <c>FailureDetails</c>,
/// including the custom <c>Properties</c>.
/// </summary>
public static class CustomExceptionOrchestration
{
    [Function(nameof(OrchestrationWithCustomException))]
    public static async Task<TaskFailureDetails?> OrchestrationWithCustomException(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        try
        {
#pragma warning disable DURABLE2001 // Activity has no input
            await context.CallActivityAsync(nameof(BusinessActivity));
#pragma warning restore DURABLE2001
        }
        catch (TaskFailedException ex)
        {
            // FailureDetails.Properties contains the values supplied by the provider.
            return ex.FailureDetails;
        }

        // Should never reach here — the activity always throws.
        return null;
    }

    [Function(nameof(BusinessActivity))]
    public static void BusinessActivity([ActivityTrigger] TaskActivityContext context)
    {
        throw new BusinessValidationException(
            message: "Business logic validation failed",
            stringProperty: "validation-error-123",
            intProperty: 100,
            longProperty: 999999999L,
            dateTimeProperty: new DateTime(2025, 10, 15, 14, 30, 0, DateTimeKind.Utc),
            dictionaryProperty: new Dictionary<string, object?>
            {
                ["error_code"] = "VALIDATION_FAILED",
                ["retry_count"] = 3,
                ["is_critical"] = true,
            },
            listProperty: new List<object?> { "error1", "error2", 500, null },
            nullProperty: null);
    }

    [Function(nameof(OrchestrationWithCustomException) + "_Start")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/customException")] HttpRequestData req,
        [DurableClient] DurableTaskClient client)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(nameof(OrchestrationWithCustomException));
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}

// =============================================================================
// Custom exception + provider
// =============================================================================

/// <summary>
/// Business exception carrying structured properties that should be surfaced on
/// <c>FailureDetails.Properties</c>.
/// </summary>
[Serializable]
public class BusinessValidationException : Exception
{
    public BusinessValidationException(
        string message,
        string stringProperty,
        int intProperty,
        long longProperty,
        DateTime dateTimeProperty,
        IDictionary<string, object?> dictionaryProperty,
        IList<object?> listProperty,
        object? nullProperty)
        : base(message)
    {
        this.StringProperty = stringProperty;
        this.IntProperty = intProperty;
        this.LongProperty = longProperty;
        this.DateTimeProperty = dateTimeProperty;
        this.DictionaryProperty = dictionaryProperty;
        this.ListProperty = listProperty;
        this.NullProperty = nullProperty;
    }

    public BusinessValidationException(string message) : base(message) { }

    public string? StringProperty { get; }
    public int? IntProperty { get; }
    public long? LongProperty { get; }
    public DateTime? DateTimeProperty { get; }
    public IDictionary<string, object?>? DictionaryProperty { get; }
    public IList<object?>? ListProperty { get; }
    public object? NullProperty { get; }
}

/// <summary>
/// Maps thrown exceptions to the custom properties attached to FailureDetails.
/// Returning <c>null</c> opts out of adding properties for a given exception.
/// </summary>
public class BusinessExceptionPropertiesProvider : IExceptionPropertiesProvider
{
    public IDictionary<string, object?>? GetExceptionProperties(Exception exception)
    {
        return exception switch
        {
            BusinessValidationException e => new Dictionary<string, object?>
            {
                ["StringProperty"] = e.StringProperty,
                ["IntProperty"] = e.IntProperty,
                ["LongProperty"] = e.LongProperty,
                ["DateTimeProperty"] = e.DateTimeProperty,
                ["DictionaryProperty"] = e.DictionaryProperty,
                ["ListProperty"] = e.ListProperty,
                ["NullProperty"] = e.NullProperty,
            },
            _ => null,
        };
    }
}

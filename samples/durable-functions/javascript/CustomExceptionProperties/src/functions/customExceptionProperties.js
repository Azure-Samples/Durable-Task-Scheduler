const { app } = require("@azure/functions");
const df = require("durable-functions");
const { TaskFailedError } = df;

// =============================================================================
// Custom exception carrying structured properties.
// =============================================================================
class BusinessValidationException extends Error {
  constructor(
    message,
    stringProperty,
    intProperty,
    longProperty,
    dateTimeProperty,
    dictionaryProperty,
    listProperty,
    nullProperty
  ) {
    super(message);
    this.name = "BusinessValidationException";
    // Fix the prototype chain (necessary when extending built-ins).
    Object.setPrototypeOf(this, new.target.prototype);
    this.stringProperty = stringProperty;
    this.intProperty = intProperty;
    this.longProperty = longProperty;
    this.dateTimeProperty = dateTimeProperty;
    this.dictionaryProperty = dictionaryProperty;
    this.listProperty = listProperty;
    this.nullProperty = nullProperty;
  }
}

// =============================================================================
// Register a global provider that surfaces custom properties from thrown
// exceptions into FailureDetails.Properties. Return `undefined` to opt out for
// a particular error.
// =============================================================================
df.app.setExceptionPropertiesProvider({
  getExceptionProperties(error) {
    if (error instanceof BusinessValidationException) {
      return {
        StringProperty: error.stringProperty,
        IntProperty: error.intProperty,
        LongProperty: error.longProperty,
        DateTimeProperty: error.dateTimeProperty,
        DictionaryProperty: error.dictionaryProperty,
        ListProperty: error.listProperty,
        NullProperty: error.nullProperty,
      };
    }
    return undefined;
  },
});

// Activity: throws an exception carrying custom properties.
df.app.activity("businessActivity", {
  handler: () => {
    throw new BusinessValidationException(
      "Business logic validation failed",
      "validation-error-123",
      100,
      999999999,
      "2025-10-15T14:30:00.000Z",
      {
        error_code: "VALIDATION_FAILED",
        retry_count: 3,
        is_critical: true,
      },
      ["error1", "error2", 500, null],
      null
    );
  },
});

// Orchestrator: calls the activity, catches the propagated TaskFailedError, and
// returns its FailureDetails (which includes the custom Properties).
df.app.orchestration("orchestrationWithCustomException", function* (context) {
  try {
    yield context.df.callActivity("businessActivity");
  } catch (e) {
    if (e instanceof TaskFailedError) {
      return e.failureDetails;
    }
    throw e;
  }
  // Should never reach here — the activity always throws.
  return null;
});

// HTTP trigger: start the orchestration.
app.http("StartCustomException", {
  route: "StartCustomException",
  methods: ["POST"],
  authLevel: "anonymous",
  extraInputs: [df.input.durableClient()],
  handler: async (request, context) => {
    const client = df.getClient(context);
    const instanceId = await client.startNew("orchestrationWithCustomException");
    context.log(`Started orchestration with ID = '${instanceId}'.`);
    return client.createCheckStatusResponse(request, instanceId);
  },
});

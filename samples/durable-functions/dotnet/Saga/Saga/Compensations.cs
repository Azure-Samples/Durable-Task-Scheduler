using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Saga
{
    public class Compensations
    {
        private readonly TaskOrchestrationContext _context;
        private readonly ILogger _logger;
        private readonly List<Func<Task>> _compensations = new();

        public Compensations(TaskOrchestrationContext context, ILogger logger)
        {
            _context = context ?? throw new ArgumentNullException(nameof(context));
            _logger = logger ?? throw new ArgumentNullException(nameof(logger));
        }

        /// <summary>
        /// Adds a compensation action to be executed if the workflow fails
        /// </summary>
        /// <typeparam name="T">The type of the compensation activity input</typeparam>
        /// <param name="activityName">The name of the activity to execute as compensation</param>
        /// <param name="input">The input to the compensation activity</param>
        public void AddCompensation<T>(string activityName, T input)
        {
            if (string.IsNullOrEmpty(activityName))
                throw new ArgumentNullException(nameof(activityName));

            _compensations.Add(async () => await _context.CallActivityAsync(activityName, input));
        }

        /// <summary>
        /// Executes all registered compensation actions in reverse order (LIFO)
        /// </summary>
        /// <param name="inParallel">If true, executes all compensations in parallel; otherwise, sequentially in LIFO order</param>
        public async Task CompensateAsync(bool inParallel = false)
        {
            if (inParallel)
            {
                // Execute all compensations in parallel
                var compensationTasks = new List<Task>();
                
                foreach (var compensation in _compensations)
                {
                    compensationTasks.Add(ExecuteCompensationWithErrorHandling(compensation));
                }

                await Task.WhenAll(compensationTasks);
            }
            else
            {
                // Execute compensations in LIFO order (reverse order of registration)
                for (int i = _compensations.Count - 1; i >= 0; i--)
                {
                    await ExecuteCompensationWithErrorHandling(_compensations[i]);
                }
            }
        }

        private async Task ExecuteCompensationWithErrorHandling(Func<Task> compensation)
        {
            try
            {
                await compensation();
            }
            catch (Exception ex)
            {
                // Log the error but continue with other compensations
                _logger.LogError(ex, "Compensation action failed but continuing with other compensations");
            }
        }
    }
}

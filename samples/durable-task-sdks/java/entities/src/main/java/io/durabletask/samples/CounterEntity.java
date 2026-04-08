// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.AbstractTaskEntity;
import com.microsoft.durabletask.TaskEntityOperation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * A durable entity that maintains a counter state.
 * <p>
 * Supports operations: add, subtract, get, reset.
 * Public methods are automatically dispatched based on operation name.
 */
public class CounterEntity extends AbstractTaskEntity<Integer> {
    private static final Logger logger = LoggerFactory.getLogger(CounterEntity.class);

    @Override
    protected Integer initializeState(TaskEntityOperation operation) {
        return 0;
    }

    @Override
    protected Class<Integer> getStateType() {
        return Integer.class;
    }

    public void add(int value) {
        this.state += value;
        logger.info("Counter '{}': Added {}, new value: {}", this.context.getId().getKey(), value, this.state);
    }

    public void subtract(int value) {
        this.state -= value;
        logger.info("Counter '{}': Subtracted {}, new value: {}", this.context.getId().getKey(), value, this.state);
    }

    public int get() {
        logger.info("Counter '{}': Current value: {}", this.context.getId().getKey(), this.state);
        return this.state;
    }

    public void reset() {
        this.state = 0;
        logger.info("Counter '{}': Reset to 0", this.context.getId().getKey());
    }
}

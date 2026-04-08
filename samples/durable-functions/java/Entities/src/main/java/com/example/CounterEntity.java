package com.example;

import com.microsoft.durabletask.AbstractTaskEntity;
import com.microsoft.durabletask.TaskEntityOperation;

import java.util.logging.Logger;

/**
 * A durable entity that maintains a counter state.
 * <p>
 * Supports operations: add, subtract, get, reset.
 * Public methods are automatically dispatched based on operation name.
 */
public class CounterEntity extends AbstractTaskEntity<Integer> {
    private static final Logger logger = Logger.getLogger(CounterEntity.class.getName());

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
        logger.info(String.format("Counter '%s': Added %d, new value: %d",
            this.context.getId().getKey(), value, this.state));
    }

    public void subtract(int value) {
        this.state -= value;
        logger.info(String.format("Counter '%s': Subtracted %d, new value: %d",
            this.context.getId().getKey(), value, this.state));
    }

    public int get() {
        logger.info(String.format("Counter '%s': Current value: %d",
            this.context.getId().getKey(), this.state));
        return this.state;
    }

    public void reset() {
        this.state = 0;
        logger.info(String.format("Counter '%s': Reset to 0",
            this.context.getId().getKey()));
    }
}

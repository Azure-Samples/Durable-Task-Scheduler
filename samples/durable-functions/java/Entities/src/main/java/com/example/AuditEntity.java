package com.example;

import com.microsoft.durabletask.AbstractTaskEntity;
import com.microsoft.durabletask.TaskEntityOperation;

import java.util.ArrayList;
import java.util.List;
import java.util.logging.Logger;

/**
 * A durable entity that records audit log entries.
 * <p>
 * This entity is signaled by other entities (e.g., AccountEntity) to record
 * events such as large transactions. It demonstrates entity-to-entity signaling
 * as the target of signals from other entities.
 */
public class AuditEntity extends AbstractTaskEntity<List<String>> {
    private static final Logger logger = Logger.getLogger(AuditEntity.class.getName());

    @Override
    @SuppressWarnings("unchecked")
    protected List<String> initializeState(TaskEntityOperation operation) {
        return new ArrayList<>();
    }

    @Override
    @SuppressWarnings("unchecked")
    protected Class<List<String>> getStateType() {
        return (Class<List<String>>) (Class<?>) List.class;
    }

    /**
     * Records an audit entry. Called via entity-to-entity signaling from AccountEntity.
     */
    public void record(String message) {
        this.state.add(message);
        logger.info(String.format("Audit '%s': Recorded entry #%d: %s",
            this.context.getId().getKey(), this.state.size(), message));
    }

    /**
     * Returns the list of recorded audit entries.
     */
    public List<String> getEntries() {
        logger.info(String.format("Audit '%s': Returning %d entries",
            this.context.getId().getKey(), this.state.size()));
        return this.state;
    }

    /**
     * Clears all audit entries.
     */
    public void clear() {
        this.state.clear();
        logger.info(String.format("Audit '%s': Cleared all entries",
            this.context.getId().getKey()));
    }
}

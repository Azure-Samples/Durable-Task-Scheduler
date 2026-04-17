// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.AbstractTaskEntity;
import com.microsoft.durabletask.EntityInstanceId;
import com.microsoft.durabletask.TaskEntityOperation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * A durable entity that maintains a bank account balance.
 * <p>
 * Supports operations: deposit, withdraw, getBalance, reset.
 * <p>
 * Demonstrates:
 * - Entities signaling other entities: when a deposit exceeds a threshold, the account
 *   signals an "audit" entity to log the transaction.
 * - Entities starting orchestrations: when a large withdrawal occurs, the account
 *   starts an audit orchestration.
 */
public class AccountEntity extends AbstractTaskEntity<Integer> {
    private static final Logger logger = LoggerFactory.getLogger(AccountEntity.class);
    private static final int LARGE_TRANSACTION_THRESHOLD = 500;

    @Override
    protected Integer initializeState(TaskEntityOperation operation) {
        return 0;
    }

    @Override
    protected Class<Integer> getStateType() {
        return Integer.class;
    }

    /**
     * Deposits funds into this account.
     * If the deposit amount exceeds the threshold, this entity signals an audit entity
     * to record the large transaction (entity-to-entity signaling).
     */
    public void deposit(int amount) {
        this.state += amount;
        String key = this.context.getId().getKey();
        logger.info("Account '{}': Deposited {}, new balance: {}", key, amount, this.state);

        // Entity signals another entity: notify the audit entity of large deposits
        if (amount >= LARGE_TRANSACTION_THRESHOLD) {
            EntityInstanceId auditEntityId = new EntityInstanceId("audit", "ledger");
            String message = String.format("Large deposit of %d into account '%s'", amount, key);
            this.context.signalEntity(auditEntityId, "record", message);
            logger.info("Account '{}': Signaled audit entity for large deposit of {}", key, amount);
        }
    }

    /**
     * Withdraws funds from this account.
     * If the withdrawal amount exceeds the threshold, this entity starts an audit
     * orchestration to process the large withdrawal (entity starting an orchestration).
     */
    public void withdraw(int amount) {
        this.state -= amount;
        String key = this.context.getId().getKey();
        logger.info("Account '{}': Withdrew {}, new balance: {}", key, amount, this.state);

        // Entity starts a new orchestration: trigger an audit workflow for large withdrawals
        if (amount >= LARGE_TRANSACTION_THRESHOLD) {
            String auditInput = String.format("Large withdrawal of %d from account '%s', remaining balance: %d",
                amount, key, this.state);
            String instanceId = this.context.startNewOrchestration("AuditWorkflow", auditInput);
            logger.info("Account '{}': Started AuditWorkflow '{}' for large withdrawal of {}", key, instanceId, amount);
        }
    }

    public int getBalance() {
        logger.info("Account '{}': Current balance: {}", this.context.getId().getKey(), this.state);
        return this.state;
    }

    public void reset() {
        this.state = 0;
        logger.info("Account '{}': Reset to 0", this.context.getId().getKey());
    }
}

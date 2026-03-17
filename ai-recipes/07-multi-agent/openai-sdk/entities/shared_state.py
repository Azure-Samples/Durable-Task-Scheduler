from __future__ import annotations

from durabletask.entities import DurableEntity


class sharedstate(DurableEntity):
    """Entity that tracks shared context across agents.

    Operations: add_finding, get_findings, set_status, get_status, snapshot.
    """

    def _load_state(self) -> dict:
        return self.get_state(dict, {"findings": [], "status": {}})

    def add_finding(self, finding: dict) -> dict:
        state = self._load_state()
        state["findings"].append(finding)
        self.set_state(state)
        return state

    def get_findings(self, _input=None) -> list:
        return self._load_state()["findings"]

    def set_status(self, status_update: dict) -> dict:
        state = self._load_state()
        agent = status_update.get("agent", "unknown")
        state["status"][agent] = status_update.get("status", "unknown")
        self.set_state(state)
        return state["status"]

    def get_status(self, agent=None):
        status = self._load_state()["status"]
        if agent:
            return status.get(agent)
        return status

    def snapshot(self, _input=None) -> dict:
        return self._load_state()


SharedStateEntity = sharedstate

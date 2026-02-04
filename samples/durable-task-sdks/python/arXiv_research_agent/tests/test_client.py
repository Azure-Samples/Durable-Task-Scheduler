"""Tests for FastAPI client endpoints."""

import pytest
from unittest.mock import patch, Mock, MagicMock
from fastapi.testclient import TestClient

# We need to mock the client before importing the app
with patch("arxiv_research_agent.client.DurableTaskSchedulerClient"):
    from arxiv_research_agent.client import app, AgentStartRequest


@pytest.fixture
def test_client():
    """Create a test client."""
    return TestClient(app)


@pytest.fixture
def mock_durable_client():
    """Mock the DurableTask client."""
    with patch("arxiv_research_agent.client.get_client") as mock_get:
        mock_client = MagicMock()
        mock_get.return_value = mock_client
        yield mock_client


class TestHealthEndpoint:
    """Tests for health check endpoint."""

    def test_health_check(self, test_client):
        """Test health check returns healthy."""
        response = test_client.get("/health")
        
        assert response.status_code == 200
        assert response.json() == {"status": "healthy"}


class TestStartAgentEndpoint:
    """Tests for POST /agents endpoint."""

    def test_start_agent_success(self, test_client, mock_durable_client):
        """Test starting an agent successfully."""
        mock_durable_client.schedule_new_orchestration.return_value = "instance-123"
        
        response = test_client.post(
            "/agents",
            json={"topic": "deep learning transformers", "max_iterations": 3}
        )
        
        assert response.status_code == 200
        data = response.json()
        assert data["ok"] is True
        assert data["instance_id"] == "instance-123"

    def test_start_agent_default_iterations(self, test_client, mock_durable_client):
        """Test starting an agent with default iterations."""
        mock_durable_client.schedule_new_orchestration.return_value = "instance-456"
        
        response = test_client.post(
            "/agents",
            json={"topic": "machine learning"}
        )
        
        assert response.status_code == 200
        call_args = mock_durable_client.schedule_new_orchestration.call_args
        assert call_args.kwargs["input"]["max_iterations"] == 3

    def test_start_agent_empty_topic(self, test_client, mock_durable_client):
        """Test starting an agent with empty topic fails."""
        response = test_client.post(
            "/agents",
            json={"topic": ""}
        )
        
        assert response.status_code == 400
        assert "Topic cannot be empty" in response.json()["detail"]

    def test_start_agent_whitespace_topic(self, test_client, mock_durable_client):
        """Test starting an agent with whitespace topic fails."""
        response = test_client.post(
            "/agents",
            json={"topic": "   "}
        )
        
        assert response.status_code == 400

    def test_start_agent_error(self, test_client, mock_durable_client):
        """Test starting an agent handles errors."""
        mock_durable_client.schedule_new_orchestration.side_effect = Exception("Connection failed")
        
        response = test_client.post(
            "/agents",
            json={"topic": "neural networks"}
        )
        
        assert response.status_code == 500


class TestListAgentsEndpoint:
    """Tests for GET /agents endpoint."""

    def test_list_agents(self, test_client, mock_durable_client):
        """Test listing agents returns empty list (current implementation)."""
        response = test_client.get("/agents")
        
        assert response.status_code == 200
        assert response.json() == []


class TestGetAgentStatusEndpoint:
    """Tests for GET /agents/{instance_id} endpoint."""

    def test_get_status_pending(self, test_client, mock_durable_client):
        """Test getting status of pending agent."""
        from durabletask import client as durable_client
        
        mock_state = Mock()
        mock_state.runtime_status = durable_client.OrchestrationStatus.PENDING
        mock_state.serialized_input = '{"topic": "deep learning"}'
        mock_state.serialized_output = None
        mock_state.created_at = None
        mock_durable_client.get_orchestration_state.return_value = mock_state
        
        response = test_client.get("/agents/instance-123")
        
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "PENDING"
        assert data["topic"] == "deep learning"

    def test_get_status_completed(self, test_client, mock_durable_client):
        """Test getting status of completed agent."""
        from durabletask import client as durable_client
        
        mock_state = Mock()
        mock_state.runtime_status = durable_client.OrchestrationStatus.COMPLETED
        mock_state.serialized_input = '{"topic": "neural networks"}'
        mock_state.serialized_output = '{"topic": "neural networks", "iterations": 3, "report": "Final literature review"}'
        mock_state.created_at = None
        mock_durable_client.get_orchestration_state.return_value = mock_state
        
        response = test_client.get("/agents/instance-123")
        
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "COMPLETED"
        assert data["report"] == "Final literature review"
        assert data["iterations"] == 3

    def test_get_status_not_found(self, test_client, mock_durable_client):
        """Test getting status of non-existent agent."""
        mock_durable_client.get_orchestration_state.return_value = None
        
        response = test_client.get("/agents/nonexistent")
        
        assert response.status_code == 404


class TestWaitForAgentEndpoint:
    """Tests for GET /agents/{instance_id}/wait endpoint."""

    def test_wait_completed(self, test_client, mock_durable_client):
        """Test waiting for completed agent."""
        from durabletask import client as durable_client
        
        mock_state = Mock()
        mock_state.runtime_status = durable_client.OrchestrationStatus.COMPLETED
        mock_state.serialized_output = '{"topic": "deep learning", "iterations": 2, "report": "Report", "findings_count": 2}'
        mock_durable_client.wait_for_orchestration_completion.return_value = mock_state
        
        response = test_client.get("/agents/instance-123/wait")
        
        assert response.status_code == 200
        data = response.json()
        assert data["topic"] == "deep learning"
        assert data["iterations"] == 2
        assert data["report"] == "Report"
        assert data["findings_count"] == 2

    def test_wait_timeout(self, test_client, mock_durable_client):
        """Test waiting for agent that times out."""
        mock_durable_client.wait_for_orchestration_completion.return_value = None
        
        response = test_client.get("/agents/instance-123/wait?timeout=10")
        
        assert response.status_code == 408

    def test_wait_failed(self, test_client, mock_durable_client):
        """Test waiting for failed agent."""
        from durabletask import client as durable_client
        
        mock_state = Mock()
        mock_state.runtime_status = durable_client.OrchestrationStatus.FAILED
        mock_durable_client.wait_for_orchestration_completion.return_value = mock_state
        
        response = test_client.get("/agents/instance-123/wait")
        
        assert response.status_code == 500


class TestTerminateAgentEndpoint:
    """Tests for DELETE /agents/{instance_id} endpoint."""

    def test_terminate_success(self, test_client, mock_durable_client):
        """Test terminating an agent successfully."""
        response = test_client.delete("/agents/instance-123")
        
        assert response.status_code == 200
        data = response.json()
        assert data["ok"] is True
        mock_durable_client.terminate_orchestration.assert_called_once()

    def test_terminate_error(self, test_client, mock_durable_client):
        """Test terminating an agent handles errors."""
        mock_durable_client.terminate_orchestration.side_effect = Exception("Error")
        
        response = test_client.delete("/agents/instance-123")
        
        assert response.status_code == 500


class TestAgentStartRequestValidation:
    """Tests for AgentStartRequest validation."""

    def test_validated_max_iterations_normal(self):
        """Test validated_max_iterations with normal value."""
        request = AgentStartRequest(topic="test", max_iterations=5)
        assert request.validated_max_iterations == 5

    def test_validated_max_iterations_too_low(self):
        """Test validated_max_iterations clamps low values."""
        request = AgentStartRequest(topic="test", max_iterations=0)
        assert request.validated_max_iterations == 1

    def test_validated_max_iterations_too_high(self):
        """Test validated_max_iterations clamps high values."""
        request = AgentStartRequest(topic="test", max_iterations=20)
        assert request.validated_max_iterations == 10

    def test_validated_max_iterations_negative(self):
        """Test validated_max_iterations handles negative values."""
        request = AgentStartRequest(topic="test", max_iterations=-5)
        assert request.validated_max_iterations == 1

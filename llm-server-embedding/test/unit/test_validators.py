"""
Tests unitaires pour app/utils/validators.py
"""

import pytest
from pydantic import ValidationError
from app.utils.validators import Message, ChatCompletionRequest, ErrorResponse


class TestMessage:
    """Tests pour le modèle Message"""

    def test_valid_message(self):
        """Vérifie création message valide"""
        msg = Message(role="user", content="Hello")
        assert msg.role == "user"
        assert msg.content == "Hello"

    def test_valid_roles(self):
        """Vérifie acceptation des 3 rôles valides"""
        for role in ["user", "assistant", "system"]:
            msg = Message(role=role, content="test")
            assert msg.role == role

    def test_invalid_role_rejected(self):
        """Vérifie rejet rôle invalide"""
        with pytest.raises(ValidationError, match="Role must be one of"):
            Message(role="invalid", content="test")

    def test_empty_content_rejected(self):
        """Vérifie rejet contenu vide"""
        with pytest.raises(ValidationError, match="Content cannot be empty"):
            Message(role="user", content="")

    def test_whitespace_only_content_rejected(self):
        """Vérifie rejet contenu whitespace uniquement"""
        with pytest.raises(ValidationError, match="Content cannot be empty"):
            Message(role="user", content="   ")

    def test_very_long_content_rejected(self):
        """Vérifie rejet contenu > 100k caractères"""
        long_content = "a" * 100001
        with pytest.raises(ValidationError, match="Content too long"):
            Message(role="user", content=long_content)

    def test_max_length_content_accepted(self):
        """Vérifie acceptation contenu = 100k caractères"""
        max_content = "a" * 100000
        msg = Message(role="user", content=max_content)
        assert len(msg.content) == 100000


class TestChatCompletionRequest:
    """Tests pour ChatCompletionRequest"""

    def test_valid_request_minimal(self):
        """Vérifie requête minimale valide"""
        req = ChatCompletionRequest(
            messages=[Message(role="user", content="test")]
        )
        assert len(req.messages) == 1
        assert req.model == "olmocr"
        assert req.max_tokens == 2048
        assert req.temperature == 1.0
        assert req.top_p == 1.0
        assert req.stream is False
        assert req.stop is None

    def test_valid_request_full(self):
        """Vérifie requête complète valide"""
        req = ChatCompletionRequest(
            model="custom-model",
            messages=[
                Message(role="system", content="You are helpful"),
                Message(role="user", content="Hello")
            ],
            max_tokens=1024,
            temperature=0.7,
            top_p=0.9,
            stream=True,
            stop=["<|im_end|>"]
        )
        assert req.model == "custom-model"
        assert len(req.messages) == 2
        assert req.max_tokens == 1024
        assert req.temperature == 0.7
        assert req.top_p == 0.9
        assert req.stream is True
        assert req.stop == ["<|im_end|>"]

    def test_empty_messages_rejected(self):
        """Vérifie rejet liste messages vide"""
        with pytest.raises(ValidationError, match="at least 1 item"):
            ChatCompletionRequest(messages=[])

    def test_max_tokens_out_of_range(self):
        """Vérifie rejet max_tokens hors limites"""
        with pytest.raises(ValidationError):
            ChatCompletionRequest(
                messages=[Message(role="user", content="test")],
                max_tokens=10000
            )

    def test_max_tokens_zero_rejected(self):
        """Vérifie rejet max_tokens=0"""
        with pytest.raises(ValidationError):
            ChatCompletionRequest(
                messages=[Message(role="user", content="test")],
                max_tokens=0
            )

    def test_temperature_out_of_range_low(self):
        """Vérifie rejet temperature < 0"""
        with pytest.raises(ValidationError):
            ChatCompletionRequest(
                messages=[Message(role="user", content="test")],
                temperature=-0.1
            )

    def test_temperature_out_of_range_high(self):
        """Vérifie rejet temperature > 2"""
        with pytest.raises(ValidationError):
            ChatCompletionRequest(
                messages=[Message(role="user", content="test")],
                temperature=2.1
            )

    def test_top_p_out_of_range(self):
        """Vérifie rejet top_p hors [0, 1]"""
        with pytest.raises(ValidationError):
            ChatCompletionRequest(
                messages=[Message(role="user", content="test")],
                top_p=1.5
            )

    def test_edge_values_accepted(self):
        """Vérifie acceptation valeurs limites"""
        req = ChatCompletionRequest(
            messages=[Message(role="user", content="test")],
            max_tokens=1,
            temperature=0.0,
            top_p=0.0
        )
        assert req.max_tokens == 1
        assert req.temperature == 0.0
        assert req.top_p == 0.0


class TestErrorResponse:
    """Tests pour ErrorResponse"""

    def test_create_error_response(self):
        """Vérifie création réponse d'erreur"""
        error = ErrorResponse.create(
            code="test_error",
            message="Test message",
            error_type="test_type"
        )
        assert error["error"]["code"] == "test_error"
        assert error["error"]["message"] == "Test message"
        assert error["error"]["type"] == "test_type"

    def test_create_error_response_default_type(self):
        """Vérifie type par défaut"""
        error = ErrorResponse.create(
            code="test_error",
            message="Test message"
        )
        assert error["error"]["type"] == "invalid_request_error"

"""
Tests unitaires pour app/config.py
"""

import pytest
import os
from app.config import (
    ServerConfig,
    ModelConfig,
    SecurityConfig,
    RateLimitConfig,
    LoggingConfig,
    reset_config,
    get_config
)


class TestServerConfig:
    """Tests pour ServerConfig"""

    def test_default_values(self):
        """Vérifie les valeurs par défaut"""
        config = ServerConfig()
        assert config.host == "127.0.0.1"
        assert config.port == 11999

    def test_custom_values(self):
        """Vérifie l'acceptation de valeurs personnalisées"""
        config = ServerConfig(host="0.0.0.0", port=12000)
        assert config.host == "0.0.0.0"
        assert config.port == 12000

    def test_invalid_port_too_low(self):
        """Vérifie rejet port < 1024"""
        with pytest.raises(ValueError):
            ServerConfig(port=80)

    def test_invalid_port_too_high(self):
        """Vérifie rejet port > 65535"""
        with pytest.raises(ValueError):
            ServerConfig(port=70000)

    def test_empty_host_rejected(self):
        """Vérifie rejet host vide"""
        with pytest.raises(ValueError, match="Host cannot be empty"):
            ServerConfig(host="")


class TestModelConfig:
    """Tests pour ModelConfig"""

    def test_default_values(self):
        """Vérifie les valeurs par défaut"""
        config = ModelConfig()
        assert config.model_path == "./model/olmOCR"
        assert config.adapter_file is None
        assert config.max_tokens == 2048
        assert config.temperature == 1.0
        assert config.top_p == 1.0

    def test_valid_ranges(self):
        """Vérifie les limites valides"""
        config = ModelConfig(
            max_tokens=4096,
            temperature=0.5,
            top_p=0.9
        )
        assert config.max_tokens == 4096
        assert config.temperature == 0.5
        assert config.top_p == 0.9

    def test_max_tokens_out_of_range(self):
        """Vérifie rejet max_tokens hors limites"""
        with pytest.raises(ValueError):
            ModelConfig(max_tokens=10000)

    def test_temperature_out_of_range(self):
        """Vérifie rejet temperature hors limites"""
        with pytest.raises(ValueError):
            ModelConfig(temperature=3.0)

    def test_empty_model_path_rejected(self):
        """Vérifie rejet model_path vide"""
        with pytest.raises(ValueError, match="Model path cannot be empty"):
            ModelConfig(model_path="")


class TestSecurityConfig:
    """Tests pour SecurityConfig"""

    def test_valid_api_key(self):
        """Vérifie acceptation clé valide"""
        key = "a" * 32  # 32 caractères minimum
        config = SecurityConfig(api_key=key)
        assert config.api_key == key

    def test_placeholder_api_key_rejected(self):
        """Vérifie rejet du placeholder"""
        with pytest.raises(ValueError, match="API_KEY must be set"):
            SecurityConfig(api_key="your-secret-key-here")

    def test_short_api_key_rejected(self):
        """Vérifie rejet clé trop courte"""
        with pytest.raises(ValueError, match="at least 32 characters"):
            SecurityConfig(api_key="short")

    def test_empty_api_key_rejected(self):
        """Vérifie rejet clé vide"""
        with pytest.raises(ValueError):
            SecurityConfig(api_key="")


class TestRateLimitConfig:
    """Tests pour RateLimitConfig"""

    def test_default_values(self):
        """Vérifie les valeurs par défaut"""
        config = RateLimitConfig()
        assert config.requests == 60
        assert config.window == 3600

    def test_custom_values(self):
        """Vérifie acceptation valeurs personnalisées"""
        config = RateLimitConfig(requests=100, window=7200)
        assert config.requests == 100
        assert config.window == 7200

    def test_zero_requests_rejected(self):
        """Vérifie rejet requests=0"""
        with pytest.raises(ValueError):
            RateLimitConfig(requests=0)


class TestLoggingConfig:
    """Tests pour LoggingConfig"""

    def test_default_values(self):
        """Vérifie les valeurs par défaut"""
        config = LoggingConfig()
        assert config.level == "DEBUG"
        assert config.file == "./logs/server.log"
        assert config.max_bytes == 10485760
        assert config.backup_count == 5

    def test_valid_log_levels(self):
        """Vérifie acceptation niveaux de log valides"""
        for level in ["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]:
            config = LoggingConfig(level=level)
            assert config.level == level

    def test_case_insensitive_log_level(self):
        """Vérifie normalisation niveau de log"""
        config = LoggingConfig(level="debug")
        assert config.level == "DEBUG"

    def test_invalid_log_level_rejected(self):
        """Vérifie rejet niveau invalide"""
        with pytest.raises(ValueError, match="Log level must be one of"):
            LoggingConfig(level="INVALID")


class TestConfigSingleton:
    """Tests pour le singleton Config"""

    def setup_method(self):
        """Réinitialiser avant chaque test"""
        reset_config()
        # Définir des variables d'environnement de test
        os.environ["API_KEY"] = "test_key_with_more_than_32_characters_long"

    def teardown_method(self):
        """Nettoyer après chaque test"""
        reset_config()
        if "API_KEY" in os.environ:
            del os.environ["API_KEY"]

    def test_singleton_returns_same_instance(self):
        """Vérifie que get_config retourne la même instance"""
        config1 = get_config()
        config2 = get_config()
        assert config1 is config2

    def test_reset_config_creates_new_instance(self):
        """Vérifie que reset_config permet nouvelle instance"""
        config1 = get_config()
        reset_config()
        config2 = get_config()
        assert config1 is not config2

    def test_config_is_frozen(self):
        """Vérifie que la config est immutable"""
        config = get_config()
        from pydantic import ValidationError
        with pytest.raises(ValidationError):
            config.server = ServerConfig(host="0.0.0.0", port=9999)

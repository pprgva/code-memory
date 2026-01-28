"""
Configuration centralisée pour MLX Multilingual-E5-Large API Server
Utilise Pydantic pour la validation et python-dotenv pour charger .env
Pattern Singleton pour instance unique de configuration
"""

import os
from typing import Optional
from pydantic import BaseModel, Field, field_validator, ConfigDict
from dotenv import load_dotenv

# Charger les variables d'environnement depuis .env
load_dotenv()


class ServerConfig(BaseModel):
    """Configuration du serveur"""

    host: str = Field(default="127.0.0.1", description="Host du serveur")
    port: int = Field(default=12000, ge=1024, le=65535, description="Port du serveur")

    @field_validator('host')
    @classmethod
    def validate_host(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("Host cannot be empty")
        return v


class ModelConfig(BaseModel):
    """Configuration du modèle MLX"""

    model_path: str = Field(default="./model/multilingual-e5-large", description="Chemin vers le modèle")
    adapter_file: Optional[str] = Field(default=None, description="Fichier adapter optionnel")
    max_tokens: int = Field(default=2048, ge=1, le=8192, description="Nombre max de tokens")
    temperature: float = Field(default=1.0, ge=0.0, le=2.0, description="Température de génération")
    top_p: float = Field(default=1.0, ge=0.0, le=1.0, description="Top-p sampling")
    unload_timeout: int = Field(
        default=300,
        ge=10,
        description="Timeout en secondes avant déchargement automatique du modèle"
    )

    @field_validator('model_path')
    @classmethod
    def validate_model_path(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("Model path cannot be empty")
        return v


class SecurityConfig(BaseModel):
    """Configuration de sécurité"""

    api_key: str = Field(description="API Key pour authentification Bearer")

    @field_validator('api_key')
    @classmethod
    def validate_api_key(cls, v: str) -> str:
        if not v or v == "your-secret-key-here":
            raise ValueError(
                "API_KEY must be set to a secure value. "
                "Generate one with: python -c \"import secrets; print(secrets.token_urlsafe(32))\""
            )
        if len(v) < 32:
            raise ValueError("API_KEY must be at least 32 characters long")
        return v


class RateLimitConfig(BaseModel):
    """Configuration du rate limiting"""

    requests: int = Field(default=60, ge=1, description="Nombre de requêtes autorisées")
    window: int = Field(default=3600, ge=1, description="Fenêtre temporelle en secondes")


class LoggingConfig(BaseModel):
    """Configuration du logging"""

    level: str = Field(default="DEBUG", description="Niveau de log")
    file: str = Field(default="./logs/server.log", description="Fichier de log")
    max_bytes: int = Field(default=10485760, ge=1024, description="Taille max du fichier (bytes)")
    backup_count: int = Field(default=5, ge=1, le=20, description="Nombre de fichiers de backup")

    @field_validator('level')
    @classmethod
    def validate_level(cls, v: str) -> str:
        valid_levels = ["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]
        v_upper = v.upper()
        if v_upper not in valid_levels:
            raise ValueError(f"Log level must be one of {valid_levels}")
        return v_upper


class Config(BaseModel):
    """Configuration globale - Singleton"""

    model_config = ConfigDict(frozen=True)  # Immutable après création

    server: ServerConfig
    model: ModelConfig
    security: SecurityConfig
    rate_limit: RateLimitConfig
    logging: LoggingConfig


# Singleton instance
_config_instance: Optional[Config] = None


def get_config() -> Config:
    """
    Récupère l'instance singleton de la configuration.
    Charge depuis les variables d'environnement à la première invocation.

    Returns:
        Config: Instance de configuration

    Raises:
        ValueError: Si des variables d'environnement requises sont manquantes
    """
    global _config_instance

    if _config_instance is None:
        _config_instance = Config(
            server=ServerConfig(
                host=os.getenv("HOST", "127.0.0.1"),
                port=int(os.getenv("PORT", "12000"))
            ),
            model=ModelConfig(
                model_path=os.getenv("MODEL_PATH", "./model/multilingual-e5-large"),
                adapter_file=os.getenv("ADAPTER_FILE") or None,
                max_tokens=int(os.getenv("MAX_TOKENS", "2048")),
                temperature=float(os.getenv("TEMPERATURE", "1.0")),
                top_p=float(os.getenv("TOP_P", "1.0")),
                unload_timeout=int(os.getenv("MODEL_UNLOAD_TIMEOUT", "300"))
            ),
            security=SecurityConfig(
                api_key=os.getenv("API_KEY", "your-secret-key-here")
            ),
            rate_limit=RateLimitConfig(
                requests=int(os.getenv("RATE_LIMIT_REQUESTS", "60")),
                window=int(os.getenv("RATE_LIMIT_WINDOW", "3600"))
            ),
            logging=LoggingConfig(
                level=os.getenv("LOG_LEVEL", "DEBUG"),
                file=os.getenv("LOG_FILE", "./logs/server.log"),
                max_bytes=int(os.getenv("LOG_MAX_BYTES", "10485760")),
                backup_count=int(os.getenv("LOG_BACKUP_COUNT", "5"))
            )
        )

    return _config_instance


def reset_config() -> None:
    """
    Réinitialise l'instance de configuration.
    Utile pour les tests.
    """
    global _config_instance
    _config_instance = None

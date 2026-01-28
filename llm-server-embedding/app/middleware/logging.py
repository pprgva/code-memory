"""
Configuration du logging pour MLX-OlmOCR API Server
Niveau DEBUG avec rotation automatique des fichiers
"""

import logging
import os
from logging.handlers import RotatingFileHandler
from pathlib import Path
from app.config import get_config


def setup_logging() -> None:
    """
    Configure le système de logging avec:
    - Niveau DEBUG (pour développement et Claude Code)
    - Handler fichier avec rotation (10MB, 5 backups)
    - Handler console pour affichage temps réel
    - Format détaillé avec timestamp, niveau, module, fonction, ligne

    La fonction est idempotente: appels multiples ne créent pas de handlers dupliqués.
    """
    config = get_config()

    # Format détaillé pour debugging
    log_format = (
        "%(asctime)s | %(levelname)-8s | "
        "%(name)s:%(funcName)s:%(lineno)d | "
        "%(message)s"
    )

    # Format pour la date (ISO 8601)
    date_format = "%Y-%m-%d %H:%M:%S"

    formatter = logging.Formatter(log_format, datefmt=date_format)

    # Créer le répertoire logs s'il n'existe pas
    log_file_path = Path(config.logging.file)
    log_file_path.parent.mkdir(parents=True, exist_ok=True)

    # Handler fichier avec rotation
    file_handler = RotatingFileHandler(
        config.logging.file,
        maxBytes=config.logging.max_bytes,
        backupCount=config.logging.backup_count,
        encoding='utf-8'
    )
    file_handler.setLevel(logging.DEBUG)
    file_handler.setFormatter(formatter)

    # Handler console (INFO pour ne pas polluer)
    console_handler = logging.StreamHandler()
    console_handler.setLevel(logging.INFO)
    console_handler.setFormatter(formatter)

    # Configurer le root logger
    root_logger = logging.getLogger()
    root_logger.setLevel(getattr(logging, config.logging.level))

    # Éviter les doublons si setup_logging est appelé plusieurs fois
    root_logger.handlers.clear()

    root_logger.addHandler(file_handler)
    root_logger.addHandler(console_handler)

    # Message de démarrage
    root_logger.info("=" * 80)
    root_logger.info("MLX-OlmOCR API Server - Logging initialisé")
    root_logger.info(f"Niveau de log: {config.logging.level}")
    root_logger.info(f"Fichier de log: {config.logging.file}")
    root_logger.info(f"Rotation: {config.logging.max_bytes} bytes, {config.logging.backup_count} backups")
    root_logger.info("=" * 80)


def get_logger(name: str) -> logging.Logger:
    """
    Récupère un logger avec le nom spécifié.
    Pratique pour avoir des loggers par module.

    Args:
        name: Nom du logger (généralement __name__)

    Returns:
        logging.Logger: Instance de logger configuré

    Example:
        >>> logger = get_logger(__name__)
        >>> logger.debug("Message de debug")
    """
    return logging.getLogger(name)

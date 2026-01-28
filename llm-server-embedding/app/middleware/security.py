"""
Middleware de sécurité pour l'authentification API Key
Vérifie le header Authorization: Bearer {token}
"""

from functools import wraps
from typing import Callable, Any
from flask import request, jsonify
import logging

from app.config import get_config

logger = logging.getLogger(__name__)


def require_api_key(f: Callable) -> Callable:
    """
    Décorateur pour protéger les endpoints avec authentification API Key.
    Vérifie le header Authorization: Bearer {token}

    Args:
        f: Fonction à décorer (endpoint Flask)

    Returns:
        Fonction décorée avec vérification API Key

    Responses:
        401: API key manquante, format invalide, ou clé incorrecte

    Example:
        @app.route('/api/protected')
        @require_api_key
        def protected_endpoint():
            return jsonify({"message": "Success"})
    """

    @wraps(f)
    def decorated(*args: Any, **kwargs: Any) -> Any:
        config = get_config()

        # Récupérer le header Authorization
        auth_header = request.headers.get('Authorization')

        if not auth_header:
            logger.warning(
                f"API key manquante - "
                f"IP: {request.remote_addr}, "
                f"Path: {request.path}, "
                f"Method: {request.method}"
            )
            return jsonify({
                "error": {
                    "code": "missing_api_key",
                    "message": "Authorization header is required",
                    "type": "authentication_error"
                }
            }), 401

        # Vérifier le format Bearer token
        try:
            parts = auth_header.split()
            if len(parts) != 2:
                raise ValueError("Invalid format")

            scheme, token = parts

            if scheme.lower() != 'bearer':
                raise ValueError("Invalid scheme")

        except ValueError as e:
            logger.warning(
                f"Format Authorization invalide - "
                f"IP: {request.remote_addr}, "
                f"Path: {request.path}, "
                f"Header: {auth_header[:20]}..."
            )
            return jsonify({
                "error": {
                    "code": "invalid_auth_format",
                    "message": "Authorization header must be 'Bearer {token}'",
                    "type": "authentication_error"
                }
            }), 401

        # Vérifier la validité de la clé
        if token != config.security.api_key:
            logger.warning(
                f"API key invalide - "
                f"IP: {request.remote_addr}, "
                f"Path: {request.path}, "
                f"Token: {token[:8]}..."
            )
            return jsonify({
                "error": {
                    "code": "invalid_api_key",
                    "message": "Invalid API key",
                    "type": "authentication_error"
                }
            }), 401

        # Authentification réussie
        logger.debug(
            f"Authentification réussie - "
            f"IP: {request.remote_addr}, "
            f"Path: {request.path}"
        )

        return f(*args, **kwargs)

    return decorated

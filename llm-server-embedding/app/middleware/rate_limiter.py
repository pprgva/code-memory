"""
Rate Limiter pour MLX-OlmOCR API Server
Limite le nombre de requêtes par IP/identifier avec sliding window
"""

from collections import defaultdict
from time import time
from typing import Dict, List, Tuple
import logging

from app.config import get_config

logger = logging.getLogger(__name__)


class RateLimiter:
    """
    Rate limiter avec sliding window algorithm.
    Limite les requêtes par identifier (généralement IP) sur une fenêtre temporelle.

    Attributes:
        requests: Dict stockant les timestamps des requêtes par identifier
    """

    def __init__(self) -> None:
        """Initialise le rate limiter avec un dict vide"""
        self.requests: Dict[str, List[float]] = defaultdict(list)
        config = get_config()
        logger.info(
            f"Rate Limiter initialisé: "
            f"{config.rate_limit.requests} requêtes / {config.rate_limit.window}s"
        )

    def is_allowed(self, identifier: str) -> Tuple[bool, int]:
        """
        Vérifie si une requête est autorisée pour l'identifier donné.

        Args:
            identifier: Identifiant unique (généralement IP)

        Returns:
            Tuple[bool, int]: (autorisé, retry_after_seconds)
                - autorisé: True si la requête est autorisée
                - retry_after: Secondes avant de pouvoir réessayer (0 si autorisé)

        Example:
            >>> limiter = RateLimiter()
            >>> allowed, retry_after = limiter.is_allowed("192.168.1.1")
            >>> if not allowed:
            ...     print(f"Réessayez dans {retry_after} secondes")
        """
        config = get_config()
        now = time()
        window_start = now - config.rate_limit.window

        # Nettoyer les anciennes requêtes (hors fenêtre)
        self.requests[identifier] = [
            ts for ts in self.requests[identifier]
            if ts > window_start
        ]

        # Vérifier la limite
        current_count = len(self.requests[identifier])

        if current_count >= config.rate_limit.requests:
            # Limite atteinte - calculer retry_after
            oldest_request = min(self.requests[identifier])
            retry_after = int(oldest_request + config.rate_limit.window - now) + 1

            logger.warning(
                f"Rate limit dépassé - "
                f"Identifier: {identifier}, "
                f"Count: {current_count}/{config.rate_limit.requests}, "
                f"Retry after: {retry_after}s"
            )

            return False, retry_after

        # Requête autorisée - ajouter timestamp
        self.requests[identifier].append(now)

        logger.debug(
            f"Rate limit OK - "
            f"Identifier: {identifier}, "
            f"Count: {current_count + 1}/{config.rate_limit.requests}"
        )

        return True, 0

    def reset(self, identifier: str) -> None:
        """
        Réinitialise le compteur pour un identifier.
        Utile pour les tests ou la gestion manuelle.

        Args:
            identifier: Identifiant à réinitialiser
        """
        if identifier in self.requests:
            del self.requests[identifier]
            logger.info(f"Rate limit réinitialisé pour: {identifier}")

    def get_stats(self, identifier: str) -> Dict[str, int]:
        """
        Récupère les statistiques pour un identifier.

        Args:
            identifier: Identifiant à analyser

        Returns:
            Dict contenant current_count et remaining
        """
        config = get_config()
        now = time()
        window_start = now - config.rate_limit.window

        # Compter les requêtes valides
        valid_requests = [
            ts for ts in self.requests.get(identifier, [])
            if ts > window_start
        ]

        current_count = len(valid_requests)
        remaining = max(0, config.rate_limit.requests - current_count)

        return {
            "current_count": current_count,
            "limit": config.rate_limit.requests,
            "remaining": remaining,
            "window_seconds": config.rate_limit.window
        }


# Instance globale singleton
_rate_limiter_instance: RateLimiter = None


def get_rate_limiter() -> RateLimiter:
    """
    Récupère l'instance singleton du rate limiter.

    Returns:
        RateLimiter: Instance globale
    """
    global _rate_limiter_instance

    if _rate_limiter_instance is None:
        _rate_limiter_instance = RateLimiter()

    return _rate_limiter_instance

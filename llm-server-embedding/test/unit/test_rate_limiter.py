"""
Tests unitaires pour app/middleware/rate_limiter.py
"""

import pytest
import time
import os
from app.middleware.rate_limiter import RateLimiter, get_rate_limiter
from app.config import reset_config


class TestRateLimiter:
    """Tests pour RateLimiter"""

    def setup_method(self):
        """Setup avant chaque test"""
        reset_config()
        # Configuration de test avec petites valeurs
        os.environ["RATE_LIMIT_REQUESTS"] = "5"
        os.environ["RATE_LIMIT_WINDOW"] = "2"  # 2 secondes
        os.environ["API_KEY"] = "test_key_with_more_than_32_characters_long"
        reset_config()

    def teardown_method(self):
        """Nettoyage après chaque test"""
        reset_config()
        for key in ["RATE_LIMIT_REQUESTS", "RATE_LIMIT_WINDOW", "API_KEY"]:
            if key in os.environ:
                del os.environ[key]

    def test_first_request_allowed(self):
        """Vérifie que la première requête est autorisée"""
        limiter = RateLimiter()
        allowed, retry_after = limiter.is_allowed("test-ip")
        assert allowed is True
        assert retry_after == 0

    def test_within_limit_allowed(self):
        """Vérifie requêtes dans la limite"""
        limiter = RateLimiter()
        for i in range(5):  # Limite = 5
            allowed, retry_after = limiter.is_allowed("test-ip")
            assert allowed is True
            assert retry_after == 0

    def test_exceed_limit_rejected(self):
        """Vérifie rejet au-delà de la limite"""
        limiter = RateLimiter()

        # Faire 5 requêtes (limite)
        for i in range(5):
            limiter.is_allowed("test-ip")

        # 6ème requête devrait être rejetée
        allowed, retry_after = limiter.is_allowed("test-ip")
        assert allowed is False
        assert retry_after > 0

    def test_different_identifiers_independent(self):
        """Vérifie que les identifiers sont indépendants"""
        limiter = RateLimiter()

        # Remplir limite pour ip1
        for i in range(5):
            limiter.is_allowed("ip1")

        # ip2 devrait être autorisée
        allowed, retry_after = limiter.is_allowed("ip2")
        assert allowed is True
        assert retry_after == 0

    def test_window_expiration(self):
        """Vérifie que la fenêtre expire correctement"""
        limiter = RateLimiter()

        # Faire 5 requêtes (limite)
        for i in range(5):
            limiter.is_allowed("test-ip")

        # Vérifier rejet
        allowed, _ = limiter.is_allowed("test-ip")
        assert allowed is False

        # Attendre expiration de la fenêtre (2 secondes + marge)
        time.sleep(2.5)

        # Devrait être autorisée maintenant
        allowed, retry_after = limiter.is_allowed("test-ip")
        assert allowed is True
        assert retry_after == 0

    def test_retry_after_calculation(self):
        """Vérifie calcul de retry_after"""
        limiter = RateLimiter()

        # Remplir limite
        for i in range(5):
            limiter.is_allowed("test-ip")

        # Vérifier retry_after
        allowed, retry_after = limiter.is_allowed("test-ip")
        assert allowed is False
        assert 0 < retry_after <= 2  # Window = 2 secondes

    def test_reset_identifier(self):
        """Vérifie réinitialisation d'un identifier"""
        limiter = RateLimiter()

        # Remplir limite
        for i in range(5):
            limiter.is_allowed("test-ip")

        # Vérifier rejet
        allowed, _ = limiter.is_allowed("test-ip")
        assert allowed is False

        # Réinitialiser
        limiter.reset("test-ip")

        # Devrait être autorisée
        allowed, retry_after = limiter.is_allowed("test-ip")
        assert allowed is True
        assert retry_after == 0

    def test_get_stats(self):
        """Vérifie récupération des statistiques"""
        limiter = RateLimiter()

        # Faire 3 requêtes
        for i in range(3):
            limiter.is_allowed("test-ip")

        stats = limiter.get_stats("test-ip")
        assert stats["current_count"] == 3
        assert stats["limit"] == 5
        assert stats["remaining"] == 2
        assert stats["window_seconds"] == 2

    def test_get_stats_unknown_identifier(self):
        """Vérifie stats pour identifier inconnu"""
        limiter = RateLimiter()
        stats = limiter.get_stats("unknown-ip")
        assert stats["current_count"] == 0
        assert stats["remaining"] == 5


class TestGetRateLimiter:
    """Tests pour get_rate_limiter singleton"""

    def setup_method(self):
        """Setup avant chaque test"""
        reset_config()
        os.environ["API_KEY"] = "test_key_with_more_than_32_characters_long"
        # Force nouvelle instance
        import app.middleware.rate_limiter as rl_module
        rl_module._rate_limiter_instance = None

    def teardown_method(self):
        """Nettoyage"""
        reset_config()
        if "API_KEY" in os.environ:
            del os.environ["API_KEY"]

    def test_singleton_returns_same_instance(self):
        """Vérifie que get_rate_limiter retourne la même instance"""
        limiter1 = get_rate_limiter()
        limiter2 = get_rate_limiter()
        assert limiter1 is limiter2

"""
Gestionnaire de modèle avec auto-unload pour économiser la mémoire.

Ce module implémente un Singleton thread-safe qui gère le cycle de vie
du modèle d'embedding : chargement lazy, surveillance d'activité,
et déchargement automatique après inactivité.

Pattern: Singleton avec watchdog thread daemon
"""

import gc
import threading
import time
from typing import Optional, Tuple

import torch
from transformers import AutoModel, AutoTokenizer, PreTrainedModel, PreTrainedTokenizer

from app.config import get_config
from app.middleware.logging import get_logger

logger = get_logger(__name__)


class ModelManager:
    """
    Gestionnaire Singleton pour le modèle d'embedding avec auto-unload.

    Cette classe gère le cycle de vie du modèle :
    - Chargement lazy (à la première utilisation)
    - Surveillance de l'activité via watchdog thread
    - Déchargement automatique après période d'inactivité
    - Libération mémoire avec gc.collect() et torch.mps.empty_cache()

    Thread-safe via threading.Lock pour toutes les opérations critiques.

    Attributes:
        model: Le modèle d'embedding (None si déchargé)
        tokenizer: Le tokenizer associé (None si déchargé)
        last_activity: Timestamp de la dernière activité
        timeout: Durée d'inactivité avant déchargement (secondes)

    Example:
        >>> manager = get_model_manager()
        >>> model, tokenizer = manager.get_model()
        >>> # Utiliser model/tokenizer...
        >>> manager.touch()  # Mettre à jour l'activité
    """

    _instance: Optional['ModelManager'] = None
    _lock: threading.Lock = threading.Lock()

    def __new__(cls) -> 'ModelManager':
        """
        Crée ou retourne l'instance singleton de ModelManager.

        Thread-safe via double-checked locking pattern.

        Returns:
            ModelManager: L'instance unique du gestionnaire
        """
        if cls._instance is None:
            with cls._lock:
                # Double-check après acquisition du lock
                if cls._instance is None:
                    instance = super().__new__(cls)
                    instance._initialize()
                    cls._instance = instance
        return cls._instance

    def _initialize(self) -> None:
        """
        Initialise les attributs de l'instance.

        Appelé une seule fois lors de la création du singleton.
        Configure le timeout depuis la config et démarre le watchdog.
        """
        self._model_lock = threading.Lock()
        self.model: Optional[PreTrainedModel] = None
        self.tokenizer: Optional[PreTrainedTokenizer] = None
        self.last_activity: float = 0.0

        # Charger le timeout depuis la configuration
        config = get_config()
        self.timeout: int = config.model.unload_timeout
        self._model_path: str = config.model.model_path

        # Thread watchdog (daemon = se termine avec le processus principal)
        self._watchdog_thread: Optional[threading.Thread] = None
        self._watchdog_running: bool = False

        logger.info(
            f"ModelManager initialisé - "
            f"timeout: {self.timeout}s, "
            f"modèle: {self._model_path}"
        )

    def load_model(self) -> bool:
        """
        Charge le modèle et le tokenizer en mémoire.

        Utilise MPS (Metal Performance Shaders) sur Apple Silicon,
        ou CPU en fallback. Le modèle est mis en mode évaluation.

        Returns:
            bool: True si chargement réussi, False sinon

        Note:
            Cette méthode est thread-safe. Si le modèle est déjà
            chargé, retourne True immédiatement.
        """
        with self._model_lock:
            if self.model is not None:
                logger.debug("Modèle déjà chargé, skip load_model()")
                return True

            try:
                load_start = time.time()
                logger.info(f"Chargement du modèle depuis {self._model_path}...")

                # Charger tokenizer
                self.tokenizer = AutoTokenizer.from_pretrained(self._model_path)

                # Charger modèle (MPS sur Apple Silicon, sinon CPU)
                device = "mps" if torch.backends.mps.is_available() else "cpu"
                self.model = AutoModel.from_pretrained(self._model_path).to(device)
                self.model.eval()  # Mode évaluation (désactive dropout, etc.)

                # Mettre à jour l'activité et démarrer le watchdog
                self.last_activity = time.time()
                self._start_watchdog()

                load_duration = time.time() - load_start
                logger.info(
                    f"Modèle chargé sur {device} en {load_duration:.2f}s - "
                    f"watchdog actif (check toutes les 30s)"
                )
                return True

            except Exception as e:
                logger.error(f"Erreur chargement modèle: {e}", exc_info=True)
                self.model = None
                self.tokenizer = None
                return False

    def unload_model(self) -> None:
        """
        Décharge le modèle et libère la mémoire.

        Effectue les opérations suivantes :
        1. Met model et tokenizer à None
        2. Appelle gc.collect() pour le garbage collector Python
        3. Appelle torch.mps.empty_cache() pour libérer la mémoire GPU MPS

        Note:
            Cette méthode est thread-safe et peut être appelée
            même si le modèle n'est pas chargé (no-op dans ce cas).
        """
        with self._model_lock:
            if self.model is None:
                logger.debug("Modèle déjà déchargé, skip unload_model()")
                return

            unload_start = time.time()
            logger.info("Déchargement du modèle pour libérer la mémoire...")

            # Libérer les références
            self.model = None
            self.tokenizer = None

            # Libérer la mémoire Python
            gc.collect()

            # Libérer la mémoire MPS (Apple Silicon)
            if torch.backends.mps.is_available():
                torch.mps.empty_cache()

            unload_duration = time.time() - unload_start
            logger.info(
                f"Modèle déchargé en {unload_duration:.3f}s - "
                f"mémoire libérée (gc.collect + mps.empty_cache)"
            )

    def get_model(self) -> Tuple[PreTrainedModel, PreTrainedTokenizer]:
        """
        Retourne le modèle et tokenizer, en les chargeant si nécessaire.

        Implémente le pattern lazy loading : le modèle n'est chargé
        qu'à la première utilisation ou après un déchargement.

        Returns:
            Tuple[PreTrainedModel, PreTrainedTokenizer]: Le modèle et son tokenizer

        Raises:
            RuntimeError: Si le chargement du modèle échoue

        Example:
            >>> manager = get_model_manager()
            >>> model, tokenizer = manager.get_model()
        """
        # Vérifier d'abord sans lock (fast path)
        if self.model is not None and self.tokenizer is not None:
            return self.model, self.tokenizer

        # Charger le modèle si nécessaire
        if not self.load_model():
            raise RuntimeError(
                "Impossible de charger le modèle d'embedding. "
                "Vérifiez les logs pour plus de détails."
            )

        return self.model, self.tokenizer

    def touch(self) -> None:
        """
        Met à jour le timestamp de dernière activité.

        Doit être appelé à chaque requête pour maintenir
        le modèle en mémoire pendant la période d'activité.

        Cette opération est thread-safe et très rapide (O(1)).
        """
        self.last_activity = time.time()
        logger.debug(f"Activité mise à jour: {self.last_activity}")

    def _start_watchdog(self) -> None:
        """
        Démarre le thread watchdog qui surveille l'inactivité.

        Le watchdog vérifie toutes les 30 secondes si le modèle
        doit être déchargé (inactivité > timeout).

        Le thread est un daemon : il se termine automatiquement
        quand le processus principal se termine.
        """
        if self._watchdog_running:
            logger.debug("Watchdog déjà en cours d'exécution")
            return

        self._watchdog_running = True
        self._watchdog_thread = threading.Thread(
            target=self._watchdog_loop,
            name="ModelManager-Watchdog",
            daemon=True  # Se termine avec le processus principal
        )
        self._watchdog_thread.start()
        logger.debug("Thread watchdog démarré (daemon)")

    def _watchdog_loop(self) -> None:
        """
        Boucle principale du watchdog.

        Vérifie toutes les 30 secondes si le temps depuis la dernière
        activité dépasse le timeout configuré. Si oui, décharge le modèle.

        La boucle s'arrête quand le modèle est déchargé (watchdog_running=False).
        """
        check_interval = 30  # Vérifier toutes les 30 secondes

        logger.debug(
            f"Watchdog démarré - interval: {check_interval}s, timeout: {self.timeout}s"
        )

        while self._watchdog_running:
            time.sleep(check_interval)

            # Vérifier si le modèle est chargé
            if self.model is None:
                logger.debug("Watchdog: modèle non chargé, arrêt du watchdog")
                self._watchdog_running = False
                break

            # Calculer le temps d'inactivité
            inactive_time = time.time() - self.last_activity

            logger.debug(
                f"Watchdog check - inactivité: {inactive_time:.1f}s / {self.timeout}s"
            )

            # Décharger si timeout dépassé
            if inactive_time > self.timeout:
                logger.info(
                    f"Timeout d'inactivité dépassé ({inactive_time:.1f}s > {self.timeout}s) - "
                    f"déchargement automatique du modèle"
                )
                self.unload_model()
                self._watchdog_running = False
                break

    def is_model_loaded(self) -> bool:
        """
        Vérifie si le modèle est actuellement chargé en mémoire.

        Returns:
            bool: True si le modèle est chargé, False sinon
        """
        return self.model is not None and self.tokenizer is not None


# Fonction utilitaire pour accès simplifié
def get_model_manager() -> ModelManager:
    """
    Retourne l'instance singleton du ModelManager.

    Fonction utilitaire pour un accès simplifié au gestionnaire.

    Returns:
        ModelManager: L'instance unique du gestionnaire de modèle

    Example:
        >>> from app.middleware.model_manager import get_model_manager
        >>> manager = get_model_manager()
        >>> model, tokenizer = manager.get_model()
    """
    return ModelManager()

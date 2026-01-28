"""
Gestion du modèle MLX-OlmOCR
Chargement et inférence avec mlx-vlm
"""

import logging
from typing import Optional, Dict, Any, List
from pathlib import Path
import mlx.core as mx
from mlx_vlm import load, generate
from mlx_vlm.prompt_utils import apply_chat_template
from mlx_vlm.utils import load_config

from app.config import get_config

logger = logging.getLogger(__name__)


class OlmOCRModel:
    """
    Singleton wrapper pour le modèle MLX-OlmOCR
    Gère le chargement et l'inférence
    """

    def __init__(self):
        """Initialise sans charger le modèle (lazy loading)"""
        self.model = None
        self.processor = None
        self.config = None
        self._loaded = False

    def load_model(self) -> None:
        """
        Charge le modèle MLX-OlmOCR depuis le chemin configuré
        Appelé une seule fois au démarrage du serveur
        """
        if self._loaded:
            logger.info("Modèle déjà chargé")
            return

        config = get_config()
        model_path = Path(config.model.model_path)

        logger.info("=" * 80)
        logger.info(f"Chargement du modèle MLX-OlmOCR depuis: {model_path}")

        if not model_path.exists():
            raise FileNotFoundError(f"Modèle non trouvé: {model_path}")

        try:
            # Charger le modèle avec mlx-vlm
            logger.info("Chargement du modèle et du processeur...")
            self.model, self.processor = load(
                str(model_path),
                lazy=True  # Lazy loading pour optimiser la mémoire
            )

            # Charger la config
            self.config = load_config(str(model_path))

            self._loaded = True

            logger.info("✅ Modèle MLX-OlmOCR chargé avec succès")
            logger.info(f"   Architecture: {self.config.get('model_type', 'unknown')}")
            logger.info(f"   Vision: {self.config.get('vision_config', {}).get('model_type', 'unknown')}")
            logger.info("=" * 80)

        except Exception as e:
            logger.error(f"❌ Erreur lors du chargement du modèle: {str(e)}", exc_info=True)
            raise

    def generate_response(
        self,
        prompt: str,
        image_path: Optional[str] = None,
        max_tokens: int = 2048,
        temperature: float = 1.0,
        top_p: float = 1.0
    ) -> Dict[str, Any]:
        """
        Génère une réponse avec le modèle OlmOCR

        Args:
            prompt: Texte de la requête utilisateur
            image_path: Chemin vers l'image (optionnel, pour OCR)
            max_tokens: Nombre max de tokens à générer
            temperature: Température de sampling
            top_p: Top-p sampling

        Returns:
            Dict contenant la réponse et les métadatas

        Raises:
            RuntimeError: Si le modèle n'est pas chargé
        """
        if not self._loaded:
            raise RuntimeError("Modèle non chargé. Appelez load_model() d'abord.")

        logger.debug(
            f"Génération - "
            f"Prompt: {prompt[:100]}..., "
            f"Image: {image_path is not None}, "
            f"Max tokens: {max_tokens}"
        )

        try:
            # Préparer le message au format chat
            messages = [{"role": "user", "content": prompt}]

            # Appliquer le template de chat
            formatted_prompt = apply_chat_template(
                self.processor,
                self.config,
                messages,
                num_images=1 if image_path else 0
            )

            # Générer la réponse
            output = generate(
                self.model,
                self.processor,
                formatted_prompt,
                image=image_path,
                max_tokens=max_tokens,
                temp=temperature,
                top_p=top_p,
                verbose=False
            )

            # Extraire le texte généré
            # generate() peut retourner soit une string, soit un objet GenerationResult
            if hasattr(output, 'text'):
                generated_text = output.text
            elif hasattr(output, '__str__'):
                generated_text = str(output)
            else:
                generated_text = output

            # Compter les tokens (approximatif)
            prompt_tokens = len(prompt.split()) * 2  # Approximation
            completion_tokens = len(str(generated_text).split()) * 2  # Approximation

            logger.debug(
                f"Génération terminée - "
                f"Tokens: ~{prompt_tokens + completion_tokens}"
            )

            return {
                "text": generated_text,
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens
            }

        except Exception as e:
            logger.error(f"Erreur lors de la génération: {str(e)}", exc_info=True)
            raise

    def is_loaded(self) -> bool:
        """Vérifie si le modèle est chargé"""
        return self._loaded


# Singleton instance
_model_instance: Optional[OlmOCRModel] = None


def get_model() -> OlmOCRModel:
    """
    Récupère l'instance singleton du modèle

    Returns:
        OlmOCRModel: Instance du modèle
    """
    global _model_instance

    if _model_instance is None:
        _model_instance = OlmOCRModel()

    return _model_instance

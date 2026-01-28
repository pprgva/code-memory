"""
Serveur Flask pour MLX Multilingual-E5-Large Embeddings API
API compatible OpenAI Embeddings
"""

from flask import Flask, request, jsonify
from flask_cors import CORS
import logging
import time
import numpy as np
import torch
from typing import List
import setproctitle

from app.config import get_config
from app.middleware.logging import setup_logging, get_logger
from app.middleware.security import require_api_key
from app.middleware.model_manager import get_model_manager
# from app.middleware.rate_limiter import get_rate_limiter  # Rate limiting désactivé

# Renommer le processus pour une meilleure lisibilité
setproctitle.setproctitle("llm-server-embedding")

# Initialiser le logging
setup_logging()
logger = get_logger(__name__)

# Créer l'application Flask
app = Flask(__name__)
CORS(app)

# Configuration
config = get_config()
# rate_limiter = get_rate_limiter()  # Rate limiting désactivé

# Gestionnaire de modèle avec auto-unload
model_manager = get_model_manager()


def mean_pooling(model_output, attention_mask):
    """Mean Pooling pour les embeddings"""
    token_embeddings = model_output[0]
    input_mask_expanded = attention_mask.unsqueeze(-1).expand(token_embeddings.size()).float()
    return torch.sum(token_embeddings * input_mask_expanded, 1) / torch.clamp(input_mask_expanded.sum(1), min=1e-9)


def add_e5_prefix(texts: List[str], input_type: str = "passage") -> List[str]:
    """
    Ajoute les préfixes E5 aux textes pour des embeddings optimaux.

    Le modèle multilingual-e5-large nécessite des préfixes spécifiques:
    - "query: " pour les requêtes de recherche
    - "passage: " pour les documents/passages à indexer

    Args:
        texts: Liste de textes
        input_type: "query" ou "passage" (défaut: "passage")

    Returns:
        Liste de textes avec préfixes
    """
    prefix = "query: " if input_type == "query" else "passage: "
    return [prefix + text for text in texts]


def generate_embeddings(texts: List[str], input_type: str = "passage") -> np.ndarray:
    """
    Génère des embeddings pour une liste de textes.

    Utilise le ModelManager pour récupérer le modèle et tokenizer.
    Le modèle est chargé automatiquement si nécessaire (lazy loading).

    Args:
        texts: Liste de textes
        input_type: "query" pour recherches, "passage" pour indexation (défaut)

    Returns:
        np.ndarray: Embeddings (shape: [len(texts), 1024])

    Raises:
        RuntimeError: Si le modèle ne peut pas être chargé
    """
    # Récupérer le modèle et tokenizer via le ModelManager
    model, tokenizer = model_manager.get_model()
    device = next(model.parameters()).device

    # Ajouter les préfixes E5 pour des embeddings optimaux
    prefixed_texts = add_e5_prefix(texts, input_type)
    logger.debug(f"E5 prefix applied: input_type={input_type}, sample='{prefixed_texts[0][:50]}...'")

    # Tokenize
    encoded_input = tokenizer(
        prefixed_texts,
        padding=True,
        truncation=True,
        max_length=512,
        return_tensors='pt'
    ).to(device)

    # Generate embeddings
    with torch.no_grad():
        model_output = model(**encoded_input)

    # Mean pooling
    embeddings = mean_pooling(model_output, encoded_input['attention_mask'])

    # Normalize (pour similarité cosine)
    embeddings = torch.nn.functional.normalize(embeddings, p=2, dim=1)

    # Retourner en numpy
    return embeddings.cpu().numpy()


@app.before_request
def log_request():
    """Log chaque requête"""
    logger.info(
        f"Requête - Method: {request.method}, "
        f"Path: {request.path}, IP: {request.remote_addr}"
    )


@app.after_request
def log_response(response):
    """Log chaque réponse"""
    logger.info(
        f"Réponse - Status: {response.status_code}, "
        f"Path: {request.path}"
    )
    return response


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({
        "status": "healthy",
        "service": "mlx-embedding-api",
        "version": "1.0.0",
        "model_loaded": model_manager.is_model_loaded()
    }), 200


@app.route('/v1/embeddings', methods=['POST'])
@require_api_key
def create_embeddings():
    """
    Endpoint principal pour générer des embeddings.

    Compatible OpenAI Embeddings API.
    Support requêtes simultanées (pas de lock contrairement à l'OCR).

    Le modèle est chargé automatiquement via lazy loading si nécessaire.
    Chaque requête met à jour le timestamp d'activité pour éviter
    le déchargement automatique du modèle.
    """
    start_time = time.time()

    # Mettre à jour l'activité pour éviter le déchargement automatique
    model_manager.touch()

    # Rate limiting désactivé
    # identifier = request.remote_addr
    # allowed, retry_after = rate_limiter.is_allowed(identifier)
    #
    # if not allowed:
    #     logger.warning(f"Rate limit - IP: {identifier}")
    #     return jsonify({
    #         "error": {
    #             "message": f"Rate limit exceeded. Retry after {retry_after}s",
    #             "type": "rate_limit_error",
    #             "code": "rate_limit_exceeded"
    #         }
    #     }), 429, {'Retry-After': str(retry_after)}

    # Validation de la requête
    try:
        data = request.get_json()
        if not data:
            raise ValueError("Empty request body")

        # Extraire input (str ou List[str])
        input_text = data.get('input')
        if not input_text:
            raise ValueError("Missing 'input' field")

        # Normaliser en liste
        if isinstance(input_text, str):
            texts = [input_text]
        elif isinstance(input_text, list):
            texts = input_text
            if not texts:
                raise ValueError("Empty input list")
            if len(texts) > 100:
                raise ValueError("Too many inputs (max 100)")
        else:
            raise ValueError("Input must be string or list of strings")

        # Valider chaque texte
        for i, text in enumerate(texts):
            if not isinstance(text, str):
                raise ValueError(f"Item {i} is not a string")
            if not text.strip():
                raise ValueError(f"Item {i} is empty")
            if len(text) > 8192:
                raise ValueError(f"Item {i} too long (max 8192 chars)")

        # Extraire input_type pour les préfixes E5 (query ou passage)
        # Par défaut: "passage" pour l'indexation
        input_type = data.get('input_type', 'passage')
        if input_type not in ('query', 'passage'):
            raise ValueError("input_type must be 'query' or 'passage'")

        logger.info(f"Generating embeddings for {len(texts)} text(s) with input_type='{input_type}'")

        # Générer les embeddings avec préfixes E5
        embeddings = generate_embeddings(texts, input_type=input_type)

        # Récupérer le tokenizer pour calculer le nombre de tokens
        _, tokenizer = model_manager.get_model()

        # Construire la réponse au format OpenAI
        response = {
            "object": "list",
            "data": [
                {
                    "object": "embedding",
                    "embedding": emb.tolist(),
                    "index": i
                }
                for i, emb in enumerate(embeddings)
            ],
            "model": data.get("model", "multilingual-e5-large"),
            "usage": {
                "prompt_tokens": sum(len(tokenizer.encode(t)) for t in texts),
                "total_tokens": sum(len(tokenizer.encode(t)) for t in texts)
            },
            "input_type": input_type  # E5 prefix type used
        }

        elapsed = time.time() - start_time
        logger.info(
            f"Embeddings générés - "
            f"Count: {len(texts)}, "
            f"Dimension: {embeddings.shape[1]}, "
            f"Durée: {elapsed:.2f}s"
        )

        return jsonify(response), 200

    except ValueError as e:
        logger.warning(f"Validation error: {e}")
        return jsonify({
            "error": {
                "message": str(e),
                "type": "invalid_request_error",
                "code": "invalid_request"
            }
        }), 400

    except Exception as e:
        logger.error(f"Internal error: {e}", exc_info=True)
        return jsonify({
            "error": {
                "message": "Internal server error",
                "type": "server_error",
                "code": "internal_error"
            }
        }), 500


@app.errorhandler(404)
def not_found(error):
    """Routes non trouvées"""
    return jsonify({
        "error": {
            "message": f"Route not found: {request.path}",
            "type": "not_found",
            "code": "not_found"
        }
    }), 404


@app.errorhandler(500)
def internal_error(error):
    """Erreurs internes"""
    logger.error(f"Internal error: {error}", exc_info=True)
    return jsonify({
        "error": {
            "message": "Internal server error",
            "type": "server_error",
            "code": "internal_error"
        }
    }), 500


def run_server():
    """
    Lance le serveur Flask avec chargement lazy du modèle.

    Le modèle n'est plus chargé au démarrage mais à la première requête
    (lazy loading via ModelManager). Cela permet un démarrage plus rapide
    et une gestion automatique de la mémoire avec auto-unload.
    """
    logger.info("=" * 80)
    logger.info("Démarrage du serveur MLX Multilingual-E5-Large API")
    logger.info(f"Host: {config.server.host}")
    logger.info(f"Port: {config.server.port}")
    logger.info(f"Model: {config.model.model_path}")
    logger.info(f"Auto-unload timeout: {config.model.unload_timeout}s")
    # logger.info(f"Rate limit: {config.rate_limit.requests} req/{config.rate_limit.window}s")  # Désactivé
    logger.info("Modèle: chargement lazy à la première requête (auto-unload activé)")
    logger.info("=" * 80)

    # Le modèle est chargé automatiquement à la première requête via ModelManager
    # Pas besoin de pré-charger ici

    app.run(
        host=config.server.host,
        port=config.server.port,
        debug=False,
        threaded=True  # Permet requêtes simultanées
    )


if __name__ == '__main__':
    run_server()

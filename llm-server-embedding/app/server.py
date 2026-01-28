"""
Serveur wrapper Flask pour MLX-OlmOCR API
Intègre mlx-llm-server avec sécurité, rate limiting et logging
"""

from flask import Flask, request, jsonify
from flask_cors import CORS
import logging
import time
import base64
import tempfile
import os
from pathlib import Path
from typing import Dict, Any, Optional, Tuple
import fitz  # PyMuPDF
import threading

from app.config import get_config
from app.middleware.logging import setup_logging, get_logger
from app.middleware.security import require_api_key
from app.middleware.rate_limiter import get_rate_limiter
from app.model import get_model
from app.utils.validators import (
    ChatCompletionRequest,
    ChatCompletionResponse,
    ErrorResponse
)

# Initialiser le logging en premier
setup_logging()
logger = get_logger(__name__)

# Créer l'application Flask
app = Flask(__name__)
CORS(app)  # Activer CORS pour usage local

# Récupérer la configuration
config = get_config()

# Récupérer le rate limiter
rate_limiter = get_rate_limiter()

# Récupérer le modèle (sera chargé au démarrage)
model = get_model()

# Lock pour garantir une seule requête OCR à la fois (évite crash GPU Metal)
ocr_lock = threading.Lock()


@app.before_request
def log_request():
    """Log chaque requête entrante"""
    logger.info(
        f"Requête entrante - "
        f"Method: {request.method}, "
        f"Path: {request.path}, "
        f"IP: {request.remote_addr}"
    )


@app.after_request
def log_response(response):
    """Log chaque réponse sortante"""
    logger.info(
        f"Réponse sortante - "
        f"Status: {response.status_code}, "
        f"Path: {request.path}, "
        f"IP: {request.remote_addr}"
    )
    return response


def extract_text_and_image(content) -> Tuple[str, Optional[str]]:
    """
    Extrait le texte et l'image (si présente) du contenu du message.
    Gère les types: text, image_url, et pdf.

    Args:
        content: str ou List[dict] (format multimodal OpenAI)

    Returns:
        Tuple[str, Optional[str]]: (texte, chemin_image_temporaire)
    """
    text = ""
    image_path = None

    if isinstance(content, str):
        # Format simple: texte seulement
        text = content
    elif isinstance(content, list):
        # Format multimodal: extraire texte, image et/ou PDF
        for item in content:
            if item.get('type') == 'text':
                text += item.get('text', '')

            elif item.get('type') == 'image_url':
                # Extraire l'URL de l'image (format data:image/...;base64,...)
                image_url = item.get('image_url', {}).get('url', '')
                if image_url.startswith('data:'):
                    # Décoder l'image base64
                    try:
                        # Format: data:image/png;base64,iVBORw0KG...
                        header, encoded = image_url.split(',', 1)
                        image_data = base64.b64decode(encoded)

                        # Déterminer l'extension depuis le header
                        if 'png' in header.lower():
                            ext = '.png'
                        elif 'jpeg' in header.lower() or 'jpg' in header.lower():
                            ext = '.jpg'
                        elif 'pdf' in header.lower():
                            ext = '.pdf'
                        else:
                            ext = '.png'  # par défaut

                        # Créer un fichier temporaire
                        fd, image_path = tempfile.mkstemp(suffix=ext, prefix='olmocr_')
                        with os.fdopen(fd, 'wb') as f:
                            f.write(image_data)

                        logger.debug(f"Image décodée et sauvegardée: {image_path}")
                    except Exception as e:
                        logger.error(f"Erreur décodage image base64: {e}")

            elif item.get('type') == 'pdf':
                # Traiter le PDF selon le mode spécifié
                try:
                    text, image_path = process_pdf_content(item, text)
                except Exception as e:
                    logger.error(f"Erreur traitement PDF: {e}")
                    raise

    return text, image_path


def pdf_to_image_path(pdf_data: bytes, page_num: int = 0, dpi: int = 150) -> str:
    """
    Convertit une page PDF en image PNG temporaire.

    Args:
        pdf_data: Données binaires du PDF
        page_num: Numéro de page (0-indexed)
        dpi: Résolution de l'image

    Returns:
        str: Chemin du fichier image temporaire
    """
    # Sauvegarder le PDF temporairement
    fd_pdf, pdf_path = tempfile.mkstemp(suffix='.pdf', prefix='olmocr_input_')
    try:
        with os.fdopen(fd_pdf, 'wb') as f:
            f.write(pdf_data)

        # Ouvrir le PDF
        pdf_doc = fitz.open(pdf_path)

        if page_num >= pdf_doc.page_count:
            page_num = 0

        # Convertir en image
        page = pdf_doc[page_num]
        zoom = dpi / 72
        mat = fitz.Matrix(zoom, zoom)
        pix = page.get_pixmap(matrix=mat)

        # Sauvegarder en PNG temporaire
        fd_img, img_path = tempfile.mkstemp(suffix='.png', prefix='olmocr_')
        with os.fdopen(fd_img, 'wb') as f:
            f.write(pix.tobytes("png"))

        pdf_doc.close()

        logger.debug(f"PDF page {page_num} convertie en image: {img_path}")
        return img_path

    finally:
        # Nettoyer le PDF temporaire
        if os.path.exists(pdf_path):
            os.unlink(pdf_path)


def extract_text_from_pdf_bytes(pdf_data: bytes, page_num: int = 0) -> str:
    """
    Extrait le texte natif d'une page PDF.

    Args:
        pdf_data: Données binaires du PDF
        page_num: Numéro de page

    Returns:
        str: Texte extrait
    """
    # Sauvegarder temporairement
    fd, pdf_path = tempfile.mkstemp(suffix='.pdf', prefix='olmocr_')
    try:
        with os.fdopen(fd, 'wb') as f:
            f.write(pdf_data)

        pdf_doc = fitz.open(pdf_path)

        if page_num >= pdf_doc.page_count:
            page_num = 0

        page = pdf_doc[page_num]
        text = page.get_text()

        pdf_doc.close()

        return text.strip()

    finally:
        if os.path.exists(pdf_path):
            os.unlink(pdf_path)


def detect_pdf_type(pdf_data: bytes, page_num: int = 0) -> str:
    """
    Détecte le type de PDF: 'text' ou 'scanned'.

    Returns:
        str: 'text' si texte natif suffisant, 'scanned' sinon
    """
    text = extract_text_from_pdf_bytes(pdf_data, page_num)

    if not text or len(text) < 50:
        return "scanned"
    else:
        return "text"


def process_pdf_content(
    item: dict,
    prompt: str
) -> Tuple[str, Optional[str]]:
    """
    Traite un élément PDF selon le mode spécifié.

    Args:
        item: Dict avec 'data', 'page', 'mode'
        prompt: Texte du prompt

    Returns:
        Tuple[str, Optional[str]]: (prompt_mis_à_jour, chemin_image_temporaire)

    Raises:
        ValueError: Si mode 'text' utilisé sur PDF scanné
    """
    # Décoder le PDF avec validation
    pdf_base64 = item.get('data', '')

    if not pdf_base64:
        raise ValueError("Données PDF manquantes dans 'data'")

    logger.debug(f"PDF base64 reçu: {len(pdf_base64)} caractères")

    try:
        pdf_data = base64.b64decode(pdf_base64)
        logger.debug(f"PDF décodé: {len(pdf_data)} bytes")

        if len(pdf_data) < 100:
            raise ValueError(f"PDF trop petit après décodage: {len(pdf_data)} bytes")

        # Vérifier le magic number PDF (%PDF-)
        if not pdf_data.startswith(b'%PDF-'):
            raise ValueError("Les données décodées ne sont pas un PDF valide (magic number manquant)")

    except Exception as e:
        logger.error(f"Erreur décodage PDF base64: {e}")
        raise ValueError(f"Impossible de décoder les données PDF: {e}")

    page_num = item.get('page', 0)
    mode = item.get('mode', 'ocr')

    logger.info(f"Traitement PDF - Page: {page_num}, Mode: {mode}, Taille: {len(pdf_data)} bytes")

    if mode == 'text':
        # Extraction native uniquement
        text = extract_text_from_pdf_bytes(pdf_data, page_num)

        if not text or len(text) < 50:
            raise ValueError(
                "Mode 'text' utilisé mais le PDF semble scanné. "
                "Utilisez mode 'ocr' ou 'auto' à la place."
            )

        logger.info(f"Extraction native: {len(text)} caractères")
        # Pas d'image, on retourne juste le texte extrait dans le prompt
        updated_prompt = f"{prompt}\n\nTexte du document:\n{text}" if prompt else text
        return updated_prompt, None

    elif mode == 'ocr':
        # Force OCR - convertir en image
        image_path = pdf_to_image_path(pdf_data, page_num)
        logger.info(f"Mode OCR forcé - Image: {image_path}")
        return prompt, image_path

    elif mode == 'auto':
        # Détection automatique
        pdf_type = detect_pdf_type(pdf_data, page_num)
        logger.info(f"Détection auto: {pdf_type}")

        if pdf_type == 'text':
            # Extraction native
            text = extract_text_from_pdf_bytes(pdf_data, page_num)
            logger.info(f"Auto→Native: {len(text)} caractères")
            updated_prompt = f"{prompt}\n\nTexte du document:\n{text}" if prompt else text
            return updated_prompt, None
        else:
            # OCR
            image_path = pdf_to_image_path(pdf_data, page_num)
            logger.info(f"Auto→OCR - Image: {image_path}")
            return prompt, image_path

    else:
        raise ValueError(f"Mode PDF invalide: {mode}")


@app.route('/health', methods=['GET'])
def health_check():
    """
    Endpoint de santé (health check)
    Accessible sans authentification
    """
    return jsonify({
        "status": "healthy",
        "service": "mlx-olmocr-api",
        "version": "1.0.0"
    }), 200


@app.route('/v1/chat/completions', methods=['POST'])
@require_api_key
def chat_completions():
    """
    Endpoint principal compatible OpenAI Chat Completions
    Nécessite authentification et rate limiting
    """
    start_time = time.time()

    # 1. Rate Limiting
    identifier = request.remote_addr
    allowed, retry_after = rate_limiter.is_allowed(identifier)

    if not allowed:
        logger.warning(
            f"Rate limit dépassé - "
            f"IP: {identifier}, "
            f"Retry after: {retry_after}s"
        )
        return jsonify(
            ErrorResponse.create(
                code="rate_limit_exceeded",
                message=f"Rate limit exceeded. Retry after {retry_after} seconds.",
                error_type="rate_limit_error"
            )
        ), 429, {'Retry-After': str(retry_after)}

    # 2. Vérification du lock OCR (une seule requête à la fois)
    if not ocr_lock.acquire(blocking=False):
        logger.warning(
            f"Service occupé - Une requête OCR est déjà en cours - IP: {identifier}"
        )
        return jsonify(
            ErrorResponse.create(
                code="service_busy",
                message="OCR service is currently processing another request. Please try again in a few seconds.",
                error_type="service_unavailable"
            )
        ), 503, {'Retry-After': '5'}

    # 3. Validation de la requête
    try:
        data = request.get_json()
        if not data:
            raise ValueError("Empty request body")

        # Valider avec Pydantic
        chat_request = ChatCompletionRequest(**data)

        logger.debug(
            f"Requête validée - "
            f"Messages: {len(chat_request.messages)}, "
            f"Max tokens: {chat_request.max_tokens}, "
            f"Temperature: {chat_request.temperature}"
        )

    except Exception as e:
        logger.error(f"Validation error: {str(e)}")
        ocr_lock.release()  # Libérer le lock avant de retourner
        logger.debug("Lock OCR libéré (validation error)")
        return jsonify(
            ErrorResponse.create(
                code="invalid_request",
                message=f"Invalid request: {str(e)}",
                error_type="invalid_request_error"
            )
        ), 400

    # 3. Appel au modèle MLX-OlmOCR
    image_path = None
    try:
        # Extraire le prompt (dernier message utilisateur)
        user_messages = [m for m in chat_request.messages if m.role == "user"]
        if not user_messages:
            raise ValueError("No user message found")

        last_message = user_messages[-1]

        # Extraire texte et image du message
        prompt, image_path = extract_text_and_image(last_message.content)

        if not prompt:
            prompt = "Analyse ce document."  # Prompt par défaut si vide

        logger.info(f"Appel au modèle MLX-OlmOCR - Texte: {len(prompt)} chars, Image: {image_path is not None}")

        # Générer la réponse avec le vrai modèle
        result = model.generate_response(
            prompt=prompt,
            image_path=image_path,
            max_tokens=chat_request.max_tokens,
            temperature=chat_request.temperature,
            top_p=chat_request.top_p
        )

        # Construire la réponse au format OpenAI
        response = {
            "id": f"chatcmpl-{int(time.time())}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": chat_request.model or "olmocr",
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": result["text"]
                    },
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": result["prompt_tokens"],
                "completion_tokens": result["completion_tokens"],
                "total_tokens": result["total_tokens"]
            }
        }

        elapsed = time.time() - start_time
        logger.info(
            f"Requête complétée - "
            f"Durée: {elapsed:.2f}s, "
            f"Tokens: {response['usage']['total_tokens']}"
        )

        return jsonify(response), 200

    except ValueError as e:
        # Erreur de validation utilisateur (ex: mode 'text' sur PDF scanné)
        logger.warning(f"Erreur validation: {str(e)}")
        return jsonify(
            ErrorResponse.create(
                code="invalid_request",
                message=str(e),
                error_type="invalid_request_error"
            )
        ), 400

    except Exception as e:
        logger.error(f"Erreur lors du traitement: {str(e)}", exc_info=True)
        return jsonify(
            ErrorResponse.create(
                code="internal_error",
                message="Internal server error occurred",
                error_type="internal_server_error"
            )
        ), 500
    finally:
        # Libérer le lock OCR
        ocr_lock.release()
        logger.debug("Lock OCR libéré")

        # Nettoyer le fichier temporaire si créé
        if image_path and os.path.exists(image_path):
            try:
                os.unlink(image_path)
                logger.debug(f"Fichier temporaire supprimé: {image_path}")
            except Exception as e:
                logger.warning(f"Impossible de supprimer {image_path}: {e}")


@app.errorhandler(404)
def not_found(error):
    """Gestion des routes non trouvées"""
    logger.warning(f"Route non trouvée: {request.path}")
    return jsonify(
        ErrorResponse.create(
            code="not_found",
            message=f"Route not found: {request.path}",
            error_type="not_found_error"
        )
    ), 404


@app.errorhandler(500)
def internal_error(error):
    """Gestion des erreurs internes"""
    logger.error(f"Erreur interne: {str(error)}", exc_info=True)
    return jsonify(
        ErrorResponse.create(
            code="internal_error",
            message="Internal server error",
            error_type="internal_server_error"
        )
    ), 500


def run_server():
    """
    Lance le serveur Flask avec la configuration
    """
    config = get_config()

    logger.info("=" * 80)
    logger.info("Démarrage du serveur MLX-OlmOCR API")
    logger.info(f"Host: {config.server.host}")
    logger.info(f"Port: {config.server.port}")
    logger.info(f"Model: {config.model.model_path}")
    logger.info(f"Rate limit: {config.rate_limit.requests} req/{config.rate_limit.window}s")
    logger.info("=" * 80)

    # Charger le modèle avant de démarrer le serveur
    logger.info("🔄 Chargement du modèle MLX-OlmOCR...")
    try:
        model.load_model()
        logger.info("✅ Modèle chargé, serveur prêt")
    except Exception as e:
        logger.error(f"❌ Impossible de charger le modèle: {e}")
        logger.error("Le serveur va démarrer mais les requêtes échoueront")

    app.run(
        host=config.server.host,
        port=config.server.port,
        debug=False,  # Pas de debug en production
        threaded=True
    )


if __name__ == '__main__':
    run_server()

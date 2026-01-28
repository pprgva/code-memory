"""
Validateurs Pydantic pour les requêtes API
Compatible avec le format OpenAI Chat Completions (avec support Vision)
"""

from typing import List, Optional, Union
from pydantic import BaseModel, Field, field_validator


class Message(BaseModel):
    """
    Message dans une conversation.
    Format compatible OpenAI (avec support Vision pour images).

    Le content peut être:
    - str: texte simple
    - List[dict]: format multimodal avec texte, images et/ou PDFs
      Ex: [
        {"type": "text", "text": "Quel est ce document?"},
        {"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}},
        {"type": "pdf", "data": "base64...", "page": 0, "mode": "auto|ocr|text"}
      ]

      Modes PDF:
      - "ocr" (défaut) : Force OCR (toujours marche, texte + images)
      - "text" : Extraction native PyMuPDF (rapide mais fail si scanné)
      - "auto" : Détection automatique (optimise selon le type)
    """

    role: str = Field(description="Rôle du message: user, assistant, ou system")
    content: Union[str, List[dict]] = Field(description="Contenu du message (texte ou multimodal)")

    @field_validator('role')
    @classmethod
    def validate_role(cls, v: str) -> str:
        """Valide que le rôle est l'un des 3 autorisés"""
        valid_roles = ['user', 'assistant', 'system']
        if v not in valid_roles:
            raise ValueError(f"Role must be one of {valid_roles}, got '{v}'")
        return v

    @field_validator('content')
    @classmethod
    def validate_content(cls, v: Union[str, List[dict]]) -> Union[str, List[dict]]:
        """Valide que le contenu n'est pas vide"""
        if isinstance(v, str):
            if not v or not v.strip():
                raise ValueError("Content cannot be empty")
            if len(v) > 100000:  # 100k caractères max
                raise ValueError(f"Content too long: {len(v)} characters (max 100000)")
        elif isinstance(v, list):
            if not v:
                raise ValueError("Content list cannot be empty")
            # Valider la structure des éléments multimodaux
            for item in v:
                if not isinstance(item, dict) or 'type' not in item:
                    raise ValueError("Multimodal content must have 'type' field")

                content_type = item['type']
                if content_type not in ['text', 'image_url', 'pdf']:
                    raise ValueError(f"Invalid content type: {content_type}")

                # Validation spécifique pour PDF
                if content_type == 'pdf':
                    if 'data' not in item:
                        raise ValueError("PDF content must have 'data' field with base64 encoded PDF")

                    # Valider le mode si spécifié
                    mode = item.get('mode', 'ocr')
                    if mode not in ['auto', 'ocr', 'text']:
                        raise ValueError(f"Invalid PDF mode: {mode}. Must be 'auto', 'ocr', or 'text'")
        else:
            raise ValueError("Content must be string or list")

        return v


class ChatCompletionRequest(BaseModel):
    """
    Requête pour l'endpoint /v1/chat/completions.
    Format compatible OpenAI.
    """

    model: Optional[str] = Field(
        default="olmocr",
        description="Nom du modèle (olmocr par défaut)"
    )

    messages: List[Message] = Field(
        description="Liste des messages de la conversation",
        min_length=1
    )

    max_tokens: Optional[int] = Field(
        default=2048,
        ge=1,
        le=8192,
        description="Nombre maximum de tokens à générer"
    )

    temperature: Optional[float] = Field(
        default=1.0,
        ge=0.0,
        le=2.0,
        description="Température de sampling (0=déterministe, 2=très créatif)"
    )

    top_p: Optional[float] = Field(
        default=1.0,
        ge=0.0,
        le=1.0,
        description="Nucleus sampling (top-p)"
    )

    stream: Optional[bool] = Field(
        default=False,
        description="Activer le streaming des réponses"
    )

    stop: Optional[List[str]] = Field(
        default=None,
        description="Séquences d'arrêt de génération"
    )

    @field_validator('messages')
    @classmethod
    def validate_messages(cls, v: List[Message]) -> List[Message]:
        """Valide qu'il y a au moins un message"""
        if not v:
            raise ValueError("At least one message is required")
        return v


class ChatCompletionResponse(BaseModel):
    """
    Réponse de l'endpoint /v1/chat/completions.
    Format compatible OpenAI.
    """

    id: str = Field(description="ID unique de la complétion")
    object: str = Field(default="chat.completion", description="Type d'objet")
    created: int = Field(description="Timestamp de création")
    model: str = Field(description="Modèle utilisé")
    choices: List[dict] = Field(description="Choix de réponses")
    usage: dict = Field(description="Statistiques d'utilisation")


class ErrorResponse(BaseModel):
    """
    Réponse d'erreur standardisée.
    Format compatible OpenAI.
    """

    error: dict = Field(
        description="Objet erreur contenant code, message, type"
    )

    @staticmethod
    def create(code: str, message: str, error_type: str = "invalid_request_error") -> dict:
        """
        Crée une réponse d'erreur formatée.

        Args:
            code: Code d'erreur (ex: "invalid_api_key")
            message: Message d'erreur lisible
            error_type: Type d'erreur OpenAI

        Returns:
            Dict formaté pour réponse JSON
        """
        return {
            "error": {
                "code": code,
                "message": message,
                "type": error_type
            }
        }

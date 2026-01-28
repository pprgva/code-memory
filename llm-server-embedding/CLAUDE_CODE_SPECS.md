# Projet MLX Multilingual-E5-Large API Server - Spécifications Complètes

## 🎯 Objectif du Projet

Créer une API REST compatible OpenAI pour la génération d'embeddings en utilisant le modèle multilingual-e5-large avec MLX (Apple Silicon).

**Contexte** : Projet PRIVÉ sur Mac Studio M2 Ultra (usage local uniquement)
**Développeur** : Claude Code (développement automatisé)

---

## 📋 Contexte Technique

### Technologie de Base
- **Serveur** : `mlx-llm-server` (https://pypi.org/project/mlx-llm-server/)
- **Framework ML** : MLX (optimisé Apple Silicon)
- **Modèle** : multilingual-e5-large (dans `model/multilingual-e5-large/`)
- **API** : Compatible OpenAI Embeddings API

### Environnement
- **Machine** : Mac Studio M2 Ultra (192GB RAM)
- **Conda** : `llm-server-embedding` (environnement dédié)
- **Python** : 3.10
- **Port** : 12000 (défaut, modifiable)
- **Host** : 127.0.0.1 (localhost uniquement)

---

## 🏗️ Architecture du Projet

```
llm-server-embedding/
├── model/                      # Modèle multilingual-e5-large
│   └── multilingual-e5-large/ # Poids, tokenizer, config
│
├── app/                        # Application (organisation selon best practices)
│   ├── __init__.py
│   ├── config.py              # Configuration centralisée (Pydantic)
│   ├── server.py              # Wrapper mlx-llm-server
│   ├── middleware/
│   │   ├── __init__.py
│   │   ├── logging.py         # Logging (DEBUG pour Claude Code)
│   │   ├── security.py        # Authentification API Key
│   │   └── rate_limiter.py    # 60 req/h (strict)
│   └── utils/
│       ├── __init__.py
│       └── validators.py      # Validation Pydantic
│
├── test/                      # Tests (ordre naturel)
│   ├── unit/                  # Phase 1: Tests unitaires
│   ├── integration/           # Phase 2: Tests d'intégration
│   ├── e2e/                   # Phase 3: Tests E2E
│   └── fixtures/              # Données de test
│
├── logs/                      # Logs (généré auto)
├── .claude/                   # Config Claude Code
│
├── start.sh                   # Démarrage (avec checks)
├── stop.sh                    # Arrêt gracieux
├── requirements.txt           # Dépendances
├── environment.yml            # Config conda
├── .env.example               # Template config
├── .env                       # Config locale (gitignored)
├── .gitignore
├── pytest.ini                 # Config tests
├── README.md
├── API_USAGE.md
└── CHANGELOG.md
```

### Principes Architecturaux

**Clean Architecture** : Séparation Présentation / Business / Infrastructure
**SOLID** : Chaque module une responsabilité unique
**Patterns** : Singleton (config), Facade (server), Middleware Chain, Strategy (validators)

---

## 🔧 Configuration (Réponses de Paul Intégrées)

### Fichier `.env.example`

```bash
# ==============================================
# MLX Multilingual-E5-Large API Configuration
# ==============================================

# Server
HOST=127.0.0.1
PORT=12000

# Model
MODEL_PATH=./model/multilingual-e5-large
ADAPTER_FILE=

# API Defaults
MAX_TOKENS=512
TEMPERATURE=1.0
TOP_P=1.0

# Security (Option 1: API Key simple - projet privé)
API_KEY=your-secret-key-here  # Générer: python -c "import secrets; print(secrets.token_urlsafe(32))"

# Rate Limiting (60/h strict - usage privé)
RATE_LIMIT_REQUESTS=60
RATE_LIMIT_WINDOW=3600  # secondes

# Logging (DEBUG - pour Claude Code)
LOG_LEVEL=DEBUG
LOG_FILE=./logs/server.log
LOG_MAX_BYTES=10485760  # 10MB
LOG_BACKUP_COUNT=5
```

### Points Clés de Configuration

1. **Sécurité** : API Key (Option 1 choisie)
   - Projet privé → pas besoin JWT
   - Clé de 32+ caractères

2. **Rate Limiting** : 60 requêtes/heure
   - Variable dans .env (ajustable)
   - Protection contre boucles infinies

3. **Logging** : DEBUG
   - Pour que Claude Code ait toutes les infos
   - Rotation automatique des logs

4. **Tests** : Ordre naturel
   - Unit → Integration → E2E

---

## 📝 API Specification

### Endpoint : `POST /v1/embeddings`

**Headers requis** :
```
Content-Type: application/json
Authorization: Bearer {API_KEY}
```

**Request Body** :
```json
{
  "model": "multilingual-e5-large",
  "input": "Text to embed",
  "encoding_format": "float"
}
```

**Notes** :
- `input` peut être une chaîne unique ou un array de chaînes (batch processing)
- `encoding_format` : "float" (défaut) ou "base64"
- Taille maximale input : configurable (défaut: 8192 tokens par texte)

**Response Success (200)** :
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.123, -0.456, 0.789, ...],
      "index": 0
    }
  ],
  "model": "multilingual-e5-large",
  "usage": {
    "prompt_tokens": 10,
    "total_tokens": 10
  }
}
```

**Responses** :
- `200` : Success
- `400` : Bad Request (validation failed)
- `401` : Unauthorized (API key missing/invalid)
- `429` : Too Many Requests (rate limit exceeded)
- `500` : Internal Server Error

---

## 🛡️ Sécurité

### 1. Authentification (API Key)

```python
# app/middleware/security.py

from functools import wraps
from flask import request, jsonify
from app.config import config
import logging

logger = logging.getLogger(__name__)

def require_api_key(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        auth = request.headers.get('Authorization')

        if not auth:
            logger.warning(f"Missing API key - IP: {request.remote_addr}")
            return jsonify({"error": {"code": "missing_api_key"}}), 401

        try:
            scheme, token = auth.split()
            if scheme.lower() != 'bearer':
                raise ValueError()
        except:
            logger.warning(f"Invalid auth format - IP: {request.remote_addr}")
            return jsonify({"error": {"code": "invalid_auth_format"}}), 401

        if token != config.security.api_key:
            logger.warning(f"Invalid API key - IP: {request.remote_addr}")
            return jsonify({"error": {"code": "invalid_api_key"}}), 401

        return f(*args, **kwargs)
    return decorated
```

### 2. Rate Limiting (60/h)

```python
# app/middleware/rate_limiter.py

from collections import defaultdict
from time import time
from app.config import config

class RateLimiter:
    def __init__(self):
        self.requests = defaultdict(list)

    def is_allowed(self, identifier: str) -> tuple[bool, int]:
        now = time()
        window_start = now - config.rate_limit.window

        # Nettoyer anciennes requêtes
        self.requests[identifier] = [
            ts for ts in self.requests[identifier] if ts > window_start
        ]

        # Vérifier limite
        if len(self.requests[identifier]) >= config.rate_limit.requests:
            oldest = min(self.requests[identifier])
            retry_after = int(oldest + config.rate_limit.window - now)
            return False, retry_after

        self.requests[identifier].append(now)
        return True, 0
```

### 3. Validation (Pydantic)

```python
# app/utils/validators.py

from pydantic import BaseModel, Field, validator
from typing import List, Optional, Union

class EmbeddingRequest(BaseModel):
    model: Optional[str] = "multilingual-e5-large"
    input: Union[str, List[str]]
    encoding_format: Optional[str] = "float"

    @validator('input')
    def validate_input(cls, v):
        if isinstance(v, str):
            if not v.strip():
                raise ValueError("Empty input string")
            if len(v) > 100000:
                raise ValueError("Input too long (max 100k chars)")
        elif isinstance(v, list):
            if not v:
                raise ValueError("Empty input list")
            if len(v) > 100:
                raise ValueError("Too many inputs (max 100)")
            for item in v:
                if not isinstance(item, str) or not item.strip():
                    raise ValueError("Invalid input item")
        else:
            raise ValueError("Input must be string or list of strings")
        return v

    @validator('encoding_format')
    def validate_encoding_format(cls, v):
        if v not in ['float', 'base64']:
            raise ValueError("encoding_format must be 'float' or 'base64'")
        return v
```

---

## 🔍 Logging (Optimisé pour Claude Code)

```python
# app/middleware/logging.py

import logging
from logging.handlers import RotatingFileHandler
from app.config import config

def setup_logging():
    log_format = (
        "%(asctime)s | %(levelname)-8s | "
        "%(name)s:%(funcName)s:%(lineno)d | "
        "%(message)s"
    )

    formatter = logging.Formatter(log_format)

    # Fichier avec rotation
    file_handler = RotatingFileHandler(
        config.logging.file,
        maxBytes=config.logging.max_bytes,
        backupCount=config.logging.backup_count
    )
    file_handler.setLevel(logging.DEBUG)
    file_handler.setFormatter(formatter)

    # Console
    console_handler = logging.StreamHandler()
    console_handler.setLevel(logging.INFO)
    console_handler.setFormatter(formatter)

    # Root logger
    root = logging.getLogger()
    root.setLevel(logging.DEBUG)
    root.addHandler(file_handler)
    root.addHandler(console_handler)
```

**Pourquoi DEBUG ?**
- Claude Code a besoin de toutes les infos
- Facilite le debugging automatisé
- Rotation évite saturation disque

---

## 🧪 Stratégie de Tests (Ordre Naturel)

### Phase 1 : Tests Unitaires

```python
# test/unit/test_config.py
def test_server_config_defaults():
    config = ServerConfig()
    assert config.host == "127.0.0.1"
    assert config.port == 12000

# test/unit/test_validators.py
def test_empty_input_rejected():
    with pytest.raises(ValueError):
        EmbeddingRequest(input="")

def test_batch_input_valid():
    req = EmbeddingRequest(input=["text1", "text2"])
    assert len(req.input) == 2

# test/unit/test_rate_limiter.py
def test_exceed_limit_rejected():
    limiter = RateLimiter()
    for _ in range(60):
        limiter.is_allowed("test-ip")

    allowed, _ = limiter.is_allowed("test-ip")
    assert allowed is False
```

### Phase 2 : Tests d'Intégration

```python
# test/integration/test_api.py
def test_valid_request_success():
    response = requests.post(
        f"{BASE_URL}/v1/embeddings",
        json={"input": "test text", "model": "multilingual-e5-large"},
        headers={"Authorization": f"Bearer {API_KEY}"}
    )
    assert response.status_code == 200
    data = response.json()
    assert "data" in data
    assert len(data["data"]) == 1
    assert "embedding" in data["data"][0]

def test_batch_request_success():
    response = requests.post(
        f"{BASE_URL}/v1/embeddings",
        json={"input": ["text1", "text2"], "model": "multilingual-e5-large"},
        headers={"Authorization": f"Bearer {API_KEY}"}
    )
    assert response.status_code == 200
    data = response.json()
    assert len(data["data"]) == 2

def test_missing_api_key_rejected():
    response = requests.post(
        f"{BASE_URL}/v1/embeddings",
        json={"input": "test"}
    )
    assert response.status_code == 401
```

### Phase 3 : Tests E2E

```python
# test/e2e/test_embedding_flow.py
def test_multilingual_embedding():
    """Test embedding avec différentes langues"""
    texts = [
        "Hello world",
        "Bonjour le monde",
        "Hola mundo",
        "你好世界"
    ]

    response = requests.post(
        f"{BASE_URL}/v1/embeddings",
        json={"input": texts, "model": "multilingual-e5-large"},
        headers={"Authorization": f"Bearer {API_KEY}"}
    )

    assert response.status_code == 200
    data = response.json()
    assert len(data["data"]) == len(texts)

    # Vérifier dimensions embeddings
    for item in data["data"]:
        assert len(item["embedding"]) == 1024  # E5-large dimension

def test_performance_under_load():
    """Test performance avec charge"""
    # Test 50 requêtes séquentielles
    pass
```

### Configuration

```ini
# pytest.ini
[pytest]
testpaths = test
addopts = -v --cov=app --cov-report=html --cov-fail-under=80
```

**Commande unique** : `pytest test/`

---

## 📦 Scripts de Gestion

### start.sh

```bash
#!/bin/bash
set -e

echo "🚀 Démarrage MLX Multilingual-E5-Large API Server..."

# 1. Activer conda
source $(conda info --base)/etc/profile.d/conda.sh
conda activate llm-server-embedding

# 2. Charger .env
export $(cat .env | grep -v '^#' | xargs)

# 3. Vérifier modèle
[ ! -d "$MODEL_PATH" ] && echo "❌ Modèle non trouvé: $MODEL_PATH" && exit 1

# 4. Vérifier API key
[ "$API_KEY" == "your-secret-key-here" ] && echo "❌ API key non configurée" && exit 1

# 5. Créer logs/
mkdir -p logs

# 6. Démarrer serveur
mlx-llm-server --model "$MODEL_PATH" --port $PORT >> logs/server.log 2>&1 &
echo $! > .server.pid

echo "✅ Serveur démarré (PID: $(cat .server.pid))"
echo "   URL: http://${HOST}:${PORT}"
echo "   Logs: tail -f logs/server.log"
```

### stop.sh

```bash
#!/bin/bash

echo "🛑 Arrêt MLX Multilingual-E5-Large API Server..."

if [ -f .server.pid ]; then
    PID=$(cat .server.pid)
    kill $PID 2>/dev/null
    sleep 2
    ps -p $PID > /dev/null && kill -9 $PID
    rm .server.pid
    echo "✅ Serveur arrêté"
else
    pkill -f "mlx-llm-server"
fi
```

---

## ✅ Definition of Done

Un module est "terminé" quand :

1. ✅ Code PEP 8 + type hints
2. ✅ Tests unitaires (>80% couverture)
3. ✅ Tests d'intégration passants
4. ✅ Docstrings (Google style)
5. ✅ Gestion d'erreurs + logging
6. ✅ Validation sécurité
7. ✅ Code review
8. ✅ Documentation à jour
9. ✅ Testé manuellement

---

## 🎯 Plan de Développement pour Claude Code

### Phase 1 : Setup (~30min)
- [ ] Créer env conda `llm-server-embedding`
- [ ] Installer mlx-llm-server
- [ ] Tester serveur basique
- [ ] Créer requirements.txt
- [ ] Créer .gitignore

### Phase 2 : Infrastructure (~1h)
- [ ] Créer .env.example
- [ ] Créer app/config.py (Pydantic)
- [ ] Créer start.sh et stop.sh
- [ ] Créer app/middleware/logging.py
- [ ] Tester scripts

### Phase 3 : Sécurité (~1h30)
- [ ] Créer app/middleware/security.py
- [ ] Créer app/middleware/rate_limiter.py
- [ ] Créer app/utils/validators.py
- [ ] Intégrer middleware
- [ ] Tester avec curl

### Phase 4 : Tests (~2h)
- [ ] Créer pytest.ini
- [ ] Tests unitaires (config, validators, rate_limiter)
- [ ] Tests d'intégration (API)
- [ ] Tests E2E (embedding flow multilingue)
- [ ] Atteindre >80% couverture

### Phase 5 : Documentation (~1h)
- [ ] README.md
- [ ] API_USAGE.md
- [ ] CHANGELOG.md
- [ ] Docstrings complètes

---

## 📞 Commandes Utiles

```bash
# Démarrage/Arrêt
./start.sh
./stop.sh

# Tests
pytest test/ -v
pytest test/ --cov=app --cov-report=html

# Debugging
tail -f logs/server.log
curl http://localhost:12000/v1/embeddings \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input":"Hello world","model":"multilingual-e5-large"}'

# API Key
python -c "import secrets; print(secrets.token_urlsafe(32))"
```

---

## ⚠️ Troubleshooting

**Serveur ne démarre pas** :
1. Vérifier conda : `conda activate llm-server-embedding`
2. Vérifier modèle : `ls model/multilingual-e5-large/`
3. Vérifier logs : `cat logs/server.log`

**Tests échouent** :
1. Réinstaller : `pip install -r requirements.txt --force-reinstall`
2. Nettoyer cache : `rm -rf .pytest_cache`

**401 Unauthorized** :
1. Vérifier .env : `cat .env | grep API_KEY`
2. Format header : `Authorization: Bearer {key}`

---

## 📚 Ressources

- MLX-LLM-Server : https://pypi.org/project/mlx-llm-server/
- Multilingual-E5 : https://huggingface.co/intfloat/multilingual-e5-large
- Pydantic : https://docs.pydantic.dev/
- Pytest : https://docs.pytest.org/
- PEP 8 : https://pep8.org/
- OpenAI Embeddings API : https://platform.openai.com/docs/api-reference/embeddings

---

## 📝 Notes pour Claude Code

### Décisions Validées par Paul

1. **Sécurité** : API Key simple (projet privé)
2. **Rate Limiting** : 60/h (strict, variable)
3. **Logging** : DEBUG (pour Claude Code)
4. **Tests** : Ordre naturel (Unit → Integration → E2E)

### Contexte Critique

- **Environnement dédié** : `llm-server-embedding`
- **Développement automatisé** par Claude Code
- **Usage privé uniquement** (localhost)
- **Mac Studio M2 Ultra** (192GB RAM)
- **Port** : 12000 (différent du projet OCR sur 11999)

### Priorités

1. Robustesse
2. Testabilité (>80%)
3. Logging complet
4. Sécurité basique
5. Documentation claire

### Différences avec Projet OCR

- **API** : `/v1/embeddings` au lieu de `/v1/chat/completions`
- **Input** : Texte uniquement (pas d'images)
- **Output** : Vecteurs d'embeddings (pas de texte généré)
- **Modèle** : multilingual-e5-large (embedding) vs OlmOCR (vision)
- **Use case** : Recherche sémantique, similarité, RAG

---

**Version** : 1.0.0
**Créé pour** : Claude Code
**Machine cible** : Mac Studio M2 Ultra
**Environnement** : llm-server-embedding
**Projet source** : Adapté de llm-server-olmOCR

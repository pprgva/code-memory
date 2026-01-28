# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 🎯 Aperçu du Projet

**MLX Multilingual-E5-Large API Server** : API REST compatible OpenAI pour embeddings utilisant le modèle multilingual-e5-large avec MLX (Apple Silicon).

- **Environnement** : Mac Studio M2 Ultra (192GB RAM)
- **Conda** : `llm-server-embedding` ⚠️ Environnement dédié à ce projet
- **Python** : 3.10
- **Port** : 12000 (localhost uniquement)
- **Serveur** : mlx-llm-server (https://pypi.org/project/mlx-llm-server/)
- **Modèle** : multilingual-e5-large dans `model/multilingual-e5-large/`

## 🏗️ Architecture

### Structure Principale

```
llm-server-embedding/
├── model/multilingual-e5-large/  # Modèle MLX (poids + tokenizer + config)
├── app/                           # Application (à créer selon spécifications)
│   ├── config.py                 # Configuration Pydantic
│   ├── server.py                 # Wrapper mlx-llm-server
│   ├── middleware/               # Logging, sécurité, rate limiting
│   └── utils/                    # Validateurs Pydantic
├── test/                         # Tests (ordre naturel : unit → integration → e2e)
├── logs/                         # Logs rotatifs (auto-généré)
├── start.sh                      # Démarrage avec checks
├── stop.sh                       # Arrêt gracieux
└── .env                          # Configuration locale
```

### Principes Architecturaux

- **Clean Architecture** : Séparation Présentation / Business / Infrastructure
- **SOLID** : Une responsabilité par module
- **Patterns** : Singleton (config), Facade (server), Middleware Chain, Strategy (validators)

## 🔧 Commandes de Développement

### Environnement Conda

```bash
# Activer l'environnement (IMPORTANT : llm-server-embedding)
conda activate llm-server-embedding

# Installer dépendances
pip install -r requirements.txt
```

### Serveur

```bash
# Démarrer le serveur
./start.sh

# Arrêter le serveur
./stop.sh

# Voir les logs en temps réel
tail -f logs/server.log

# Test manuel de l'API
curl http://localhost:12000/v1/embeddings \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input":"Hello world","model":"multilingual-e5-large"}'
```

### Tests

```bash
# Exécuter tous les tests (ordre naturel : unit → integration → e2e)
pytest test/ -v

# Tests avec couverture de code
pytest test/ --cov=app --cov-report=html

# Tests unitaires uniquement
pytest test/unit/ -v

# Tests d'intégration
pytest test/integration/ -v

# Tests E2E
pytest test/e2e/ -v
```

**Objectif de couverture** : >80%

### Utilitaires

```bash
# Générer une API key sécurisée
python -c "import secrets; print(secrets.token_urlsafe(32))"

# Vérifier que le modèle est présent
ls -lh model/multilingual-e5-large/

# Nettoyer les caches
rm -rf .pytest_cache __pycache__
```

## 🛡️ Configuration et Sécurité

### Fichier .env

Le fichier `.env` doit contenir (voir `.env.example` pour template complet) :

```bash
# Server
HOST=127.0.0.1
PORT=12000

# Model
MODEL_PATH=./model/multilingual-e5-large

# Security (API Key simple - projet privé)
API_KEY=your-secret-key-here  # Générer avec secrets.token_urlsafe(32)

# Rate Limiting (60 requêtes/heure - strict)
RATE_LIMIT_REQUESTS=60
RATE_LIMIT_WINDOW=3600

# Logging (DEBUG pour développement automatisé)
LOG_LEVEL=DEBUG
LOG_FILE=./logs/server.log
```

### Sécurité

- **Authentification** : API Key (Bearer token) - Option 1 choisie (projet privé)
- **Rate Limiting** : 60 requêtes/heure (configurable via .env)
- **Validation** : Pydantic pour tous les inputs
- **Logging** : Niveau DEBUG avec rotation automatique

## 📝 API Specification

### Endpoint Principal

**POST** `/v1/embeddings` (compatible OpenAI)

**Headers** :
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

**Note** : `input` peut être une chaîne unique ou un array de chaînes pour batch processing.

**Response** :
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.123, -0.456, ...],
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
- `400` : Bad Request (validation échouée)
- `401` : Unauthorized (API key invalide)
- `429` : Too Many Requests (rate limit dépassé)
- `500` : Internal Server Error

## 🧪 Stratégie de Tests

### Ordre Naturel (Recommandé)

1. **Phase 1 : Tests Unitaires** (`test/unit/`)
   - Configuration (config.py)
   - Validateurs (validators.py)
   - Rate limiter (rate_limiter.py)

2. **Phase 2 : Tests d'Intégration** (`test/integration/`)
   - Endpoints API complets
   - Middleware chain
   - Gestion d'erreurs

3. **Phase 3 : Tests E2E** (`test/e2e/`)
   - Flow embedding complet avec vrais textes
   - Performance sous charge
   - Batch processing

### Configuration pytest

```ini
# pytest.ini
[pytest]
testpaths = test
addopts = -v --cov=app --cov-report=html --cov-fail-under=80
```

## ✅ Definition of Done

Un module est considéré terminé quand :

1. ✅ Code conforme PEP 8 avec type hints
2. ✅ Tests unitaires (>80% couverture)
3. ✅ Tests d'intégration passants
4. ✅ Docstrings (Google style)
5. ✅ Gestion d'erreurs + logging complet
6. ✅ Validation sécurité implémentée
7. ✅ Documentation à jour

## 🔍 Logging (Optimisé pour Claude Code)

### Niveau DEBUG

Le logging est configuré en **DEBUG** pour faciliter le développement automatisé :
- Tous les détails disponibles pour debugging
- Rotation automatique (10MB par fichier, 5 backups)
- Format détaillé : `timestamp | level | module:function:line | message`

### Consultation des Logs

```bash
# Logs en temps réel
tail -f logs/server.log

# Logs complets
cat logs/server.log

# Filtrer par niveau
grep "ERROR" logs/server.log
```

## ⚠️ Points Critiques

### Environnement Conda

**ATTENTION** : L'environnement conda s'appelle `llm-server-embedding` (dédié à ce projet).

### Modèle

- Chemin : `./model/multilingual-e5-large/`
- Type : Text embedding model (multilingual)
- Vérifier présence avant démarrage

### Usage Privé

- Serveur sur localhost uniquement (127.0.0.1)
- Pas de déploiement externe prévu
- Sécurité basique suffisante (API Key)

## 📚 Ressources

- **MLX-LLM-Server** : https://pypi.org/project/mlx-llm-server/
- **Pydantic** : https://docs.pydantic.dev/
- **Pytest** : https://docs.pytest.org/
- **PEP 8** : https://pep8.org/
- **Multilingual-E5** : https://huggingface.co/intfloat/multilingual-e5-large
- **Spécifications complètes** : Voir `CLAUDE_CODE_SPECS.md`

## 🎯 Contexte de Développement

- **Développement automatisé** par Claude Code
- **Machine cible** : Mac Studio M2 Ultra (192GB RAM)
- **Priorités** : Robustesse > Testabilité (>80%) > Logging complet > Sécurité basique
- **Documentation de référence** : `CLAUDE_CODE_SPECS.md` contient toutes les spécifications détaillées
- **Projet source** : Adapté de llm-server-olmOCR (OCR) vers embedding

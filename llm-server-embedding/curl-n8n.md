# Guide API MLX Multilingual-E5-Large - Exemples CURL et n8n

Documentation complète pour utiliser l'API MLX Multilingual-E5-Large Embeddings avec des exemples curl fonctionnels et intégration n8n.

**Serveur local** : `http://127.0.0.1:12000`
**Serveur n8n** : `http://192.168.0.7:12000`
**API Key** : Configurée dans `.env`

---

## 📋 Table des matières

1. [Health Check](#1-health-check)
2. [Embedding texte simple](#2-embedding-texte-simple)
3. [Batch Embeddings (plusieurs textes)](#3-batch-embeddings-plusieurs-textes)
4. [Embeddings multilingues](#4-embeddings-multilingues)
5. [Intégration n8n](#5-intégration-n8n)
6. [Gestion d'erreurs](#6-gestion-derreurs)
7. [Performance et limites](#7-performance-et-limites)

---

## 1. Health Check

Vérifier que le serveur est opérationnel (pas d'authentification requise).

### Local
```bash
curl http://127.0.0.1:12000/health
```

### Depuis n8n (réseau)
```bash
curl http://192.168.0.7:12000/health
```

### Réponse
```json
{
  "status": "healthy",
  "service": "mlx-embedding-api",
  "version": "1.0.0",
  "model_loaded": true
}
```

---

## 2. Embedding texte simple

Générer l'embedding pour un seul texte.

### Local
```bash
curl -X POST http://127.0.0.1:12000/v1/embeddings \
  -H "Authorization: Bearer DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multilingual-e5-large",
    "input": "Hello world"
  }'
```

### Depuis n8n
```bash
curl -X POST http://192.168.0.7:12000/v1/embeddings \
  -H "Authorization: Bearer YOUR_API_KEY_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multilingual-e5-large",
    "input": "Hello world"
  }'
```

### Réponse
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [
        -0.003422415,
        -0.009711522,
        0.015150093,
        ...
        0.007097971
      ],
      "index": 0
    }
  ],
  "model": "multilingual-e5-large",
  "usage": {
    "prompt_tokens": 2,
    "total_tokens": 2
  }
}
```

**Caractéristiques** :
- ✅ Dimension: 1024 (standard multilingual-e5-large)
- ✅ Normalisé: Vecteurs normalisés pour similarité cosine
- ⏱️ Performance: ~2s pour premier embedding, ~0.3s ensuite

---

## 3. Batch Embeddings (plusieurs textes)

**⭐ RECOMMANDÉ** : Générer plusieurs embeddings en une seule requête pour optimiser les performances.

### Caractéristiques
- ✅ **Efficace** : Traitement optimisé en batch
- ✅ **Concurrent** : Support des requêtes simultanées
- ⏱️ **Performance** : ~0.06s par texte en batch
- 📊 **Limite** : 100 textes maximum par requête

### Exemple local

```bash
curl -X POST http://127.0.0.1:12000/v1/embeddings \
  -H "Authorization: Bearer DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multilingual-e5-large",
    "input": [
      "Hello world",
      "Bonjour le monde",
      "Hola mundo"
    ]
  }'
```

### Depuis n8n (avec variables)

```bash
curl -X POST http://192.168.0.7:12000/v1/embeddings \
  -H "Authorization: Bearer YOUR_API_KEY_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multilingual-e5-large",
    "input": ["{{$json.text1}}", "{{$json.text2}}", "{{$json.text3}}"]
  }'
```

### Réponse
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [-0.0034224146511405706, -0.009711521677672863, ...],
      "index": 0
    },
    {
      "object": "embedding",
      "embedding": [0.0015932329697534442, 0.011301237158477306, ...],
      "index": 1
    },
    {
      "object": "embedding",
      "embedding": [0.00603358494117856, 0.0008348542032763362, ...],
      "index": 2
    }
  ],
  "model": "multilingual-e5-large",
  "usage": {
    "prompt_tokens": 13,
    "total_tokens": 13
  }
}
```

---

## 4. Embeddings multilingues

Le modèle multilingual-e5-large supporte **100+ langues** avec qualité uniforme.

### Caractéristiques
- 🌍 **Multilingue** : Anglais, Français, Espagnol, Allemand, Japonais, Chinois, Arabe, etc.
- ✅ **Qualité uniforme** : Performance similaire entre langues
- 🔄 **Cross-lingue** : Recherche sémantique entre langues différentes
- 💡 **Usage** : Applications internationales, recherche multilingue

### Exemple local (5 langues)

```bash
curl -X POST http://127.0.0.1:12000/v1/embeddings \
  -H "Authorization: Bearer DmdPeuln8ox8k8CFfphuKscUW5xNvSPhtNSh-CT2ieI" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multilingual-e5-large",
    "input": [
      "The quick brown fox jumps over the lazy dog",
      "Le renard brun rapide saute par-dessus le chien paresseux",
      "El rápido zorro marrón salta sobre el perro perezoso",
      "Der schnelle braune Fuchs springt über den faulen Hund",
      "速い茶色のキツネが怠け者の犬を飛び越える"
    ]
  }'
```

### Depuis n8n

```bash
curl -X POST http://192.168.0.7:12000/v1/embeddings \
  -H "Authorization: Bearer YOUR_API_KEY_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multilingual-e5-large",
    "input": ["{{$json.text_en}}", "{{$json.text_fr}}", "{{$json.text_es}}"]
  }'
```

### Performance observée

| Textes | Durée totale | Par texte |
|--------|--------------|-----------|
| 1 texte | ~2.00s | 2.00s |
| 3 textes | ~2.40s | 0.80s |
| 5 textes | ~0.31s | 0.06s |

**Note** : Le premier appel charge le modèle (~2s), les suivants sont rapides.

---

## 5. Intégration n8n

### Configuration HTTP Request Node

**URL** : `http://192.168.0.7:12000/v1/embeddings`

**Method** : `POST`

**Authentication** : `Generic Credential Type`
- Header Name: `Authorization`
- Value: `Bearer YOUR_API_KEY_HERE`

**Body** :
```json
{
  "model": "multilingual-e5-large",
  "input": "{{$json.text}}"
}
```

### Workflow n8n typique - Embedding unique

```
1. [Webhook/Trigger] → Réception du texte
   ↓
2. [HTTP Request] → API MLX Embeddings
   ↓
3. [Function] → Extraire data[0].embedding
   ↓
4. [Save to Vector DB] → Stocker l'embedding (Pinecone, Weaviate, Qdrant)
```

### Workflow n8n - Batch embeddings

```
1. [Read from Database] → Récupérer plusieurs textes
   ↓
2. [Function] → Grouper en tableau (max 100)
   ↓
3. [HTTP Request] → API MLX Embeddings (batch)
   ↓
4. [Loop] → Traiter chaque embedding
   ↓
5. [Save to Vector DB] → Stocker tous les embeddings
```

### Variables n8n utiles

| Variable | Description |
|----------|-------------|
| `{{$json.text}}` | Texte unique à embedder |
| `{{$json.texts}}` | Tableau de textes pour batch |
| `{{$json.language}}` | Langue du texte (optionnel) |
| `{{$json.id}}` | ID du document |

### Exemple complet n8n - Embedding avec métadonnées

```json
{
  "model": "multilingual-e5-large",
  "input": "{{$json.content || $json.text || $json.description}}"
}
```

**Traitement de la réponse** (Function Node) :
```javascript
// Extraire l'embedding et ajouter métadonnées
const embedding = $input.item.json.data[0].embedding;
const tokens = $input.item.json.usage.total_tokens;

return {
  id: $json.id,
  embedding: embedding,
  dimension: embedding.length,
  tokens: tokens,
  text: $json.text,
  timestamp: new Date().toISOString()
};
```

### Exemple complet n8n - Batch avec boucle

```json
{
  "model": "multilingual-e5-large",
  "input": {{$json.texts}}
}
```

**Traitement de la réponse** (Function Node) :
```javascript
// Traiter tous les embeddings du batch
const embeddings = $input.item.json.data;
const results = [];

for (let i = 0; i < embeddings.length; i++) {
  results.push({
    index: i,
    text: $json.texts[i],
    embedding: embeddings[i].embedding,
    dimension: embeddings[i].embedding.length
  });
}

return results;
```

---

## 6. Gestion d'erreurs

### Erreurs possibles

#### 401 - Unauthorized (API Key manquante/invalide)
```json
{
  "error": {
    "message": "Invalid API key",
    "type": "authentication_error",
    "code": "invalid_api_key"
  }
}
```

**Solution** : Vérifier l'en-tête `Authorization: Bearer YOUR_KEY`

#### 400 - Bad Request (Input manquant)
```json
{
  "error": {
    "message": "Missing 'input' field",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}
```

**Solution** : Ajouter le champ `input` avec un texte ou tableau de textes

#### 400 - Bad Request (Input vide)
```json
{
  "error": {
    "message": "Empty input list",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}
```

**Solution** : S'assurer que `input` contient au moins un texte non vide

#### 400 - Bad Request (Trop d'inputs)
```json
{
  "error": {
    "message": "Too many inputs (max 100)",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}
```

**Solution** : Limiter à 100 textes maximum par requête, diviser en plusieurs batchs

#### 400 - Bad Request (Texte trop long)
```json
{
  "error": {
    "message": "Item 0 too long (max 8192 chars)",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}
```

**Solution** : Tronquer les textes à 8192 caractères maximum

#### 429 - Rate Limit Exceeded
```json
{
  "error": {
    "message": "Rate limit exceeded. Retry after 3600s",
    "type": "rate_limit_error",
    "code": "rate_limit_exceeded"
  }
}
```

**Solution** : Attendre et réessayer après le délai indiqué (60 requêtes/heure par défaut)

#### 500 - Internal Server Error
```json
{
  "error": {
    "message": "Internal server error",
    "type": "server_error",
    "code": "internal_error"
  }
}
```

**Solution** : Vérifier les logs serveur (`tail -f logs/server.log`)

---

## 7. Performance et limites

### Limites de l'API

| Paramètre | Limite | Notes |
|-----------|--------|-------|
| **Textes par requête** | 100 | Batch maximum |
| **Longueur par texte** | 8192 chars | Auto-tronqué à 512 tokens |
| **Dimension embedding** | 1024 | Fixe (multilingual-e5-large) |
| **Rate limit** | 60 req/h | Configurable dans .env |
| **Requêtes simultanées** | ✅ Supporté | Pas de lock |

### Performance observée (M2 Ultra)

| Scénario | Durée | Optimisation |
|----------|-------|--------------|
| Premier appel | ~2.00s | Chargement modèle |
| Single text | ~0.30s | Après warm-up |
| Batch 3 texts | ~2.40s | 0.80s/texte |
| Batch 5 texts | ~0.31s | 0.06s/texte |
| Concurrent 5 req | ~0.34s | Parallèle |

### Recommandations d'optimisation

1. **Utiliser batch autant que possible**
   - ❌ Mauvais : 10 requêtes × 1 texte = ~3s
   - ✅ Bon : 1 requête × 10 textes = ~0.6s

2. **Garder le serveur warm**
   - Premier appel : ~2s (chargement)
   - Appels suivants : ~0.3s
   - Solution : Health check périodique toutes les 5 minutes

3. **Utiliser des requêtes simultanées**
   - Le serveur supporte le traitement parallèle
   - Envoyer plusieurs requêtes en parallèle si besoin

4. **Optimiser la longueur des textes**
   - Textes courts : plus rapides
   - Tronquer si >8192 caractères
   - Le modèle limite à 512 tokens de toute façon

### Calcul des tokens

Le modèle utilise un tokenizer spécifique. Estimation approximative :
- Anglais : ~4 caractères = 1 token
- Français : ~5 caractères = 1 token
- Japonais/Chinois : ~2 caractères = 1 token

---

## 🎯 Cas d'usage typiques

### 1. Recherche sémantique

**Indexation** :
```bash
# Embedder tous les documents
curl -X POST http://127.0.0.1:12000/v1/embeddings \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input": ["Doc 1 text", "Doc 2 text", "Doc 3 text"]}'
```

**Recherche** :
```bash
# Embedder la requête
curl -X POST http://127.0.0.1:12000/v1/embeddings \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input": "user query"}'

# Calculer similarité cosine avec embeddings indexés
```

### 2. Similarité de documents

```bash
# Comparer 2 documents
curl -X POST http://127.0.0.1:12000/v1/embeddings \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "input": [
      "Document A complet...",
      "Document B complet..."
    ]
  }'

# Calculer cosine similarity entre embedding[0] et embedding[1]
```

### 3. Clustering/Classification

```bash
# Embedder tous les textes à classifier
curl -X POST http://127.0.0.1:12000/v1/embeddings \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "input": [
      "Text to classify 1",
      "Text to classify 2",
      "...",
      "Text to classify 100"
    ]
  }'

# Appliquer K-means, DBSCAN, ou autre algorithme de clustering
```

---

## 📝 Paramètres API complets

### Requête

```json
{
  "model": "multilingual-e5-large",    // Optionnel, défaut: "multilingual-e5-large"
  "input": "..." ou [...],             // Requis: string ou array de strings
  "encoding_format": "float"           // Optionnel, défaut: "float" (seule option)
}
```

### Réponse

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [...],               // Array de 1024 floats
      "index": 0                        // Index dans le batch
    }
  ],
  "model": "multilingual-e5-large",
  "usage": {
    "prompt_tokens": 10,
    "total_tokens": 10
  }
}
```

---

## 🚀 Scripts de test fournis

```bash
# Test embedding simple
python test_embedding.py

# Test batch processing
python test_batch_processing.py

# Test multilingue
python test_multilingual.py

# Test concurrent
./test_concurrent.sh

# Tests unitaires
pytest test/unit/

# Tests d'intégration
pytest test/integration/
```

---

## 🔧 Configuration serveur

**Fichier** : `.env`

```bash
HOST=127.0.0.1
PORT=12000
MODEL_PATH=./model/multilingual-e5-large
API_KEY=votre-clé-secrète
RATE_LIMIT_REQUESTS=60
RATE_LIMIT_WINDOW=3600
LOG_LEVEL=DEBUG
```

---

## 📞 Support

**Logs** : `tail -f logs/server.log`
**Health check** : `http://127.0.0.1:12000/health` ou `http://192.168.0.7:12000/health`
**Démarrer** : `./start.sh`
**Arrêter** : `./stop.sh`

---

## 🌐 Comparaison avec OpenAI Embeddings API

Notre API est **100% compatible** avec l'API OpenAI Embeddings. Pour migrer :

```python
# OpenAI
import openai
response = openai.Embedding.create(
    model="text-embedding-ada-002",
    input="Hello world"
)

# MLX Multilingual-E5-Large (même format)
import requests
response = requests.post(
    "http://127.0.0.1:12000/v1/embeddings",
    headers={"Authorization": "Bearer YOUR_KEY"},
    json={
        "model": "multilingual-e5-large",
        "input": "Hello world"
    }
)
```

**Différences** :
- OpenAI : 1536 dimensions (ada-002)
- MLX : 1024 dimensions (multilingual-e5-large)
- OpenAI : Cloud, payant
- MLX : Local, gratuit, privé

---

**🎉 API MLX Multilingual-E5-Large prête pour production avec n8n !**

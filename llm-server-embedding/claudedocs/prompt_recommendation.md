# Recommandations de Prompt pour n8n → OlmOCR

## Problème actuel

Le prompt actuel (2958 chars) est trop verbeux et utilise des quotes simples (non-JSON).

## Solution recommandée : Prompt simplifié

```json
{
  "model": "olmocr",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Extract ALL text from this PDF and return valid JSON:\n\n{\"text_content\": [{\"type\": \"paragraph|title|quote|caption\", \"text\": \"full text\", \"author\": \"name or null\"}], \"visual_elements\": [{\"type\": \"image|table|chart|diagram\", \"description\": \"what you see\", \"contains_text\": \"any text in the image or null\"}], \"language\": \"fr|en|pl|other\"}\n\nRules:\n- Return ONLY valid JSON (use double quotes)\n- Extract ALL text exhaustively\n- If image contains text, extract it in contains_text\n- Detect document language"
        },
        {
          "type": "pdf",
          "data": "base64_data",
          "page": 0,
          "mode": "ocr"
        }
      ]
    }
  ],
  "max_tokens": 8192,
  "temperature": 0.3
}
```

**Avantages** :
- ✅ 10x plus court (290 chars vs 2958)
- ✅ Focus sur ce que l'OCR fait bien (texte)
- ✅ Doubles quotes (JSON valide)
- ✅ Temperature 0.3 (moins rigide)
- ✅ Instructions claires

**Inconvénients** :
- ❌ Pas de classification (tags/entities/domains)
- ❌ Pas de métadonnées visuelles détaillées

## Solution alternative : 2 étapes

### Étape 1 : OCR pur (appel 1 au serveur OlmOCR)
```json
{
  "text": "Extrait exhaustivement TOUT le texte de ce PDF. Retourne un JSON simple : {\"text\": \"tout le contenu textuel\", \"language\": \"fr|en|pl|other\", \"has_images\": true|false}"
}
```

### Étape 2 : Classification (appel 2 à Claude/GPT)
Envoie le texte extrait à Claude/GPT avec :
```
Voici le texte d'un document :
[texte extrait]

Classifie ce document selon ces catégories :
TAGS disponibles : facture, contrat, lettre...
ENTITIES : Orange, EDF, SFR...
DOMAINS : énergie, téléphonie...

Retourne : {"selected_tags": [...], "new_tags": [...], ...}
```

**Avantages** :
- ✅ OlmOCR fait ce qu'il fait le mieux (OCR)
- ✅ Claude/GPT fait la classification (meilleur en analyse sémantique)
- ✅ Résultats plus fiables
- ✅ Prompts plus courts et ciblés

**Inconvénients** :
- ❌ 2 appels API au lieu d'un
- ❌ Coût légèrement supérieur
- ❌ Latence augmentée

## Choix recommandé

**Si performance > coût** : Utilise l'approche 2 étapes
**Si coût > performance** : Simplifie le prompt actuel (version courte)

## Modifications minimales au prompt actuel

Si tu veux garder ton prompt actuel, fais AU MINIMUM ces changements :

1. **Remplace toutes les quotes simples par doubles** :
   ```
   sed "s/'/\"/g" prompt.txt
   ```

2. **Supprime les métadonnées visuelles difficiles** :
   ```json
   // ENLEVER :
   "fonts_detected": [],
   "dominant_colors": [],
   "visual_details": {"colors": [], "frame": "string"}
   ```

3. **Augmente la température à 0.3** (moins rigide)

4. **Raccourcis les listes** :
   Au lieu de lister TOUS les tags, dis :
   ```
   TAGS disponibles : [voir liste complète en base]
   Choisis les tags pertinents pour selected_tags.
   Si aucun ne correspond, propose de nouveaux tags dans proposed_new_tags.
   ```

## Test rapide

Pour valider ton prompt, teste avec ce curl :
```bash
curl http://127.0.0.1:11999/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d @ton_payload_n8n.json \
  | jq -r '.choices[0].message.content' \
  | python -m json.tool  # Valide que c'est du JSON
```

Si `python -m json.tool` échoue, le modèle retourne du JSON invalide !

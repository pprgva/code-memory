// ============================================================================
// FONCTIONS N8N POUR GÉNÉRATION DE PROMPT OlmOCR
// ============================================================================

// ============================================================================
// VERSION 1 : CORRECTION MINIMALE (garde ton approche actuelle)
// ============================================================================
// ✅ Corrige les quotes simples → doubles quotes (JSON valide)
// ✅ Simplifie visual_details et layout_info (moins d'hallucinations)
// ✅ Garde toute la logique de classification
// ⚠️  Toujours long (2600 chars)

function buildOlmOCRPrompt_CorrectionMinimale() {
  const tags = $('Merge-Data').first().json.all_tags.map(item => item).join(',');
  const entities = $('Merge-Data').first().json.all_domains.map(item => item).join(',');
  const domains = $('Merge-Data').first().json.all_entities.map(item => item).join(',');

  const prompt = `Analyse exhaustivement ce document PDF. Retourne un JSON valide avec cette structure exacte :

{
  "text_content": [
    {"type": "poem|quote|paragraph|title|caption", "author": "string or null", "text": "contenu textuel complet"}
  ],
  "visual_elements": [
    {"type": "image|diagram|chart|table", "description": "description complète", "position": "string", "contains_text": "texte extrait or null"}
  ],
  "classification": {
    "document_language": "fr|en|pl|other",
    "selected_tags": [],
    "new_tags": false,
    "proposed_new_tags": [],
    "selected_entities": [],
    "new_entities": false,
    "proposed_new_entities": [],
    "selected_domains": [],
    "new_domains": false,
    "proposed_new_domains": []
  }
}

CLASSIFICATION - RÈGLES IMPORTANTES :

TAGS disponibles : ${tags}
- Choisis les tags pertinents dans cette liste pour selected_tags
- Si AUCUN tag ne correspond, mets new_tags à true et propose de nouveaux tags dans proposed_new_tags
- Si les tags de la liste sont suffisants, new_tags reste false

ENTITIES disponibles : ${entities}
- Choisis les entités mentionnées dans le document parmi cette liste pour selected_entities
- Si tu détectes des entités qui ne sont PAS dans la liste, mets new_entities à true et propose-les dans proposed_new_entities
- Si toutes les entités sont déjà dans la liste, new_entities reste false

DOMAINS disponibles : ${domains}
- Choisis les domaines pertinents dans cette liste pour selected_domains
- Si AUCUN domaine ne correspond, mets new_domains à true et propose de nouveaux domaines dans proposed_new_domains
- Si les domaines de la liste sont suffisants, new_domains reste false

RÈGLES CRITIQUES :
- Retourne UNIQUEMENT du JSON valide avec doubles quotes, aucun texte avant ou après
- Sois exhaustif : capture TOUT le texte et TOUS les éléments visuels
- Si une info n'existe pas, utilise null (pas de string vide)
- Si une image contient du texte, extrais-le dans contains_text
- Détecte la langue principale du document pour document_language`;

  // Construis le body complet
  for (const item of $input.all()) {
    item.json.requestBody = {
      model: "olmocr",
      messages: [{
        role: "user",
        content: [
          { type: "text", text: prompt },
          { type: "pdf", data: item.json.data, page: 0, mode: "ocr" }
        ]
      }],
      max_tokens: 8192,
      temperature: 0.3  // Augmenté de 0.1 → 0.3 (moins rigide)
    };
  }

  return $input.all();
}


// ============================================================================
// VERSION 2 : OPTIMISÉE (recommandée pour meilleure performance)
// ============================================================================
// ✅ 60% plus court (1000 chars vs 2600)
// ✅ Focus sur ce que l'OCR fait bien (texte)
// ✅ Moins d'hallucinations sur visual_details
// ✅ Instructions plus claires et ciblées
// ⚠️  Toujours toute la logique de classification

function buildOlmOCRPrompt_Optimisee() {
  const tags = $('Merge-Data').first().json.all_tags.map(item => item).join(',');
  const entities = $('Merge-Data').first().json.all_domains.map(item => item).join(',');
  const domains = $('Merge-Data').first().json.all_entities.map(item => item).join(',');

  const prompt = `Extract ALL text from this PDF and classify. Return ONLY valid JSON:

{
  "text_content": [{"type": "paragraph|title|quote|caption", "text": "full text", "author": "name or null"}],
  "visual_elements": [{"type": "image|table|chart|diagram", "description": "what you see", "contains_text": "any text in image or null"}],
  "classification": {
    "document_language": "fr|en|pl|other",
    "selected_tags": [],
    "new_tags": false,
    "proposed_new_tags": [],
    "selected_entities": [],
    "new_entities": false,
    "proposed_new_entities": [],
    "selected_domains": [],
    "new_domains": false,
    "proposed_new_domains": []
  }
}

CLASSIFICATION:
TAGS: ${tags}
ENTITIES: ${entities}
DOMAINS: ${domains}

Rules:
- Use doubles quotes for valid JSON
- Extract ALL text exhaustively
- Select existing tags/entities/domains when they match
- Set new_tags/new_entities/new_domains to true ONLY if none match, then propose new ones
- If image contains text, extract it in contains_text`;

  // Construis le body complet
  for (const item of $input.all()) {
    item.json.requestBody = {
      model: "olmocr",
      messages: [{
        role: "user",
        content: [
          { type: "text", text: prompt },
          { type: "pdf", data: item.json.data, page: 0, mode: "ocr" }
        ]
      }],
      max_tokens: 8192,
      temperature: 0.3
    };
  }

  return $input.all();
}


// ============================================================================
// VERSION 3 : ULTRA-SIMPLE (pour tests rapides)
// ============================================================================
// ✅ 90% plus court (300 chars)
// ✅ Extraction texte pure
// ❌ PAS de classification (à faire après avec Claude/GPT)

function buildOlmOCRPrompt_UltraSimple() {
  const prompt = `Extract ALL text from this PDF. Return ONLY valid JSON:

{
  "text": "complete extracted text",
  "language": "fr|en|pl|other",
  "has_images": true|false,
  "image_descriptions": ["desc1", "desc2"]
}

Use double quotes for valid JSON. Be exhaustive.`;

  // Construis le body complet
  for (const item of $input.all()) {
    item.json.requestBody = {
      model: "olmocr",
      messages: [{
        role: "user",
        content: [
          { type: "text", text: prompt },
          { type: "pdf", data: item.json.data, page: 0, mode: "ocr" }
        ]
      }],
      max_tokens: 4096,
      temperature: 0.3
    };
  }

  return $input.all();
}


// ============================================================================
// VERSION 4 : TEMPLATE MODULAIRE (facilite modifications humaines)
// ============================================================================
// ✅ Prompt en sections séparées (facile à modifier)
// ✅ Template JSON externe (réutilisable)
// ✅ Instructions claires par section

function buildOlmOCRPrompt_Modulaire() {
  // Configuration (facile à modifier)
  const config = {
    tags: $('Merge-Data').first().json.all_tags.map(item => item).join(','),
    entities: $('Merge-Data').first().json.all_domains.map(item => item).join(','),
    domains: $('Merge-Data').first().json.all_entities.map(item => item).join(','),
    max_tokens: 8192,
    temperature: 0.3
  };

  // Template JSON (facile à modifier la structure)
  const jsonTemplate = {
    text_content: [
      {type: "paragraph|title|quote|caption", text: "full text", author: "name or null"}
    ],
    visual_elements: [
      {type: "image|table|chart|diagram", description: "what you see", contains_text: "text or null"}
    ],
    classification: {
      document_language: "fr|en|pl|other",
      selected_tags: [],
      new_tags: false,
      proposed_new_tags: [],
      selected_entities: [],
      new_entities: false,
      proposed_new_entities: [],
      selected_domains: [],
      new_domains: false,
      proposed_new_domains: []
    }
  };

  // Instructions par section (facile à modifier)
  const instructions = {
    main: "Extract ALL text from this PDF and classify it.",
    output: `Return ONLY valid JSON with this structure:\n${JSON.stringify(jsonTemplate, null, 2)}`,
    classification: `
CLASSIFICATION LISTS:
- TAGS: ${config.tags}
- ENTITIES: ${config.entities}
- DOMAINS: ${config.domains}

CLASSIFICATION RULES:
- Select matching items from lists above
- Set new_* to true ONLY if NO items match
- Propose new items in proposed_new_* when new_* is true`,
    critical: `
CRITICAL RULES:
- Use double quotes for valid JSON
- Extract ALL text exhaustively
- Return ONLY JSON, no extra text
- Use null for missing values (not empty strings)`
  };

  // Assembler le prompt (facile à réorganiser)
  const prompt = [
    instructions.main,
    instructions.output,
    instructions.classification,
    instructions.critical
  ].join('\n\n');

  // Construis le body
  for (const item of $input.all()) {
    item.json.requestBody = {
      model: "olmocr",
      messages: [{
        role: "user",
        content: [
          { type: "text", text: prompt },
          { type: "pdf", data: item.json.data, page: 0, mode: "ocr" }
        ]
      }],
      max_tokens: config.max_tokens,
      temperature: config.temperature
    };
  }

  return $input.all();
}


// ============================================================================
// GUIDE D'UTILISATION
// ============================================================================

/*
QUELLE VERSION CHOISIR ?

VERSION 1 - Correction minimale :
✅ Si tu veux garder ton approche actuelle
✅ Si tu veux juste corriger le bug des quotes
✅ Si tu as besoin de toute la verbosité
❌ Risque d'hallucinations sur visual_details

VERSION 2 - Optimisée :
✅ Meilleure performance (prompt plus court)
✅ Moins d'hallucinations
✅ Garde toute la logique de classification
✅ RECOMMANDÉE pour production

VERSION 3 - Ultra-simple :
✅ Tests rapides
✅ OCR pur (classification après avec Claude/GPT)
❌ Nécessite 2 étapes (OCR puis classification)

VERSION 4 - Modulaire :
✅ Facile à modifier par un humain
✅ Sections séparées et commentées
✅ Configuration centralisée
✅ RECOMMANDÉE si tu modifies souvent le prompt


COMMENT TESTER ?

1. Remplace ta fonction actuelle par une des versions ci-dessus
2. Renomme la fonction en "buildOlmOCRPrompt" (sans suffixe)
3. Lance un test n8n
4. Vérifie dans logs/last_request.json que le prompt est correct
5. Vérifie que la réponse est du JSON valide :

   cat logs/last_request.json | jq -r '.messages[0].content[0].text'


VALIDATION DU JSON RETOURNÉ :

Pour tester si le modèle retourne du JSON valide :

1. Envoie une requête depuis n8n
2. Récupère la réponse
3. Teste avec :

   echo '<response>' | python3 -m json.tool

   Si pas d'erreur → JSON valide ✅
   Si erreur → JSON invalide ❌ (probablement quotes simples)


MONITORING :

Surveille les logs pour voir le prompt exact :
tail -f logs/server.log | grep "🔍 Requête complète"

Le prompt complet sera dans logs/last_request.json
*/

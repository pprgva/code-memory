#!/bin/bash
# Test de requêtes OCR concurrentes pour vérifier le lock

echo "🧪 Test de requêtes concurrentes..."
echo "📝 La 1ère requête doit réussir (200)"
echo "📝 La 2ème requête doit retourner HTTP 503 (Service Busy)"
echo ""

# Charger l'API key depuis .env
export $(grep API_KEY .env | xargs)

# Créer un payload de test simple
PAYLOAD='{"model":"olmocr","messages":[{"role":"user","content":"Test"}],"max_tokens":100}'

# Lancer 2 requêtes en parallèle
echo "🚀 Lancement de 2 requêtes en parallèle..."
echo ""

# Requête 1 (en arrière-plan)
(
  echo "📤 Requête 1 démarrée..."
  RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" \
    http://127.0.0.1:11999/v1/chat/completions)

  STATUS=$(echo "$RESPONSE" | grep HTTP_STATUS | cut -d':' -f2)
  BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

  echo "✅ Requête 1 terminée - Status: $STATUS"
  if [ "$STATUS" = "200" ]; then
    echo "   → Succès attendu ✓"
  else
    echo "   → Erreur inattendue ✗"
    echo "   Body: $BODY"
  fi
) &

# Petit délai pour s'assurer que la 1ère requête démarre en premier
sleep 0.5

# Requête 2 (devrait recevoir 503)
(
  echo "📤 Requête 2 démarrée (devrait recevoir 503)..."
  RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" \
    http://127.0.0.1:11999/v1/chat/completions)

  STATUS=$(echo "$RESPONSE" | grep HTTP_STATUS | cut -d':' -f2)
  BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS/d')

  echo "✅ Requête 2 terminée - Status: $STATUS"
  if [ "$STATUS" = "503" ]; then
    echo "   → Service Busy (attendu) ✓"
    echo "   Message: $(echo $BODY | jq -r '.error.message')"
  elif [ "$STATUS" = "200" ]; then
    echo "   → Succès (lock non fonctionnel?) ✗"
  else
    echo "   → Status inattendu: $STATUS ✗"
    echo "   Body: $BODY"
  fi
) &

# Attendre que les deux requêtes se terminent
wait

echo ""
echo "🏁 Test terminé - Vérifier les résultats ci-dessus"

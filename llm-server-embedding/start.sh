#!/bin/bash
# ==============================================
# MLX Multilingual-E5-Large API Server - Script de Démarrage
# ==============================================

set -e  # Arrêter en cas d'erreur

echo "🚀 Démarrage MLX Multilingual-E5-Large API Server..."

# 1. Vérifier que nous sommes dans le bon répertoire
if [ ! -f ".env.example" ]; then
    echo "❌ Erreur: Exécutez ce script depuis la racine du projet"
    exit 1
fi

# 2. Activer l'environnement conda
echo "📦 Activation de l'environnement conda llm-server-embedding..."
source /opt/anaconda3/etc/profile.d/conda.sh
conda activate llm-server-embedding

if [ $? -ne 0 ]; then
    echo "❌ Erreur: Impossible d'activer l'environnement conda"
    echo "   Créez-le avec: conda create -n llm-server-embedding python=3.10"
    exit 1
fi

# 3. Vérifier que .env existe
if [ ! -f ".env" ]; then
    echo "❌ Erreur: Fichier .env manquant"
    echo "   Copiez .env.example vers .env et configurez les valeurs"
    echo "   cp .env.example .env"
    exit 1
fi

# 4. Charger les variables d'environnement
echo "⚙️  Chargement de la configuration (.env)..."
export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)

# 5. Vérifier que le modèle existe
if [ ! -d "$MODEL_PATH" ]; then
    echo "❌ Erreur: Modèle non trouvé à $MODEL_PATH"
    echo "   Vérifiez MODEL_PATH dans .env"
    exit 1
fi

echo "✅ Modèle trouvé: $MODEL_PATH"

# 6. Vérifier que l'API key est configurée
if [ "$API_KEY" == "your-secret-key-here" ] || [ -z "$API_KEY" ]; then
    echo "❌ Erreur: API_KEY non configurée dans .env"
    echo "   Générez une clé avec:"
    echo "   python -c \"import secrets; print(secrets.token_urlsafe(32))\""
    exit 1
fi

echo "✅ API Key configurée"

# 7. Créer le répertoire logs s'il n'existe pas
mkdir -p logs

# 8. Vérifier si le serveur est déjà en cours d'exécution
if [ -f ".server.pid" ]; then
    PID=$(cat .server.pid)
    if ps -p $PID > /dev/null 2>&1; then
        echo "⚠️  Le serveur est déjà en cours d'exécution (PID: $PID)"
        echo "   Utilisez ./stop.sh pour l'arrêter d'abord"
        exit 1
    else
        echo "🧹 Nettoyage d'un ancien fichier PID..."
        rm .server.pid
    fi
fi

# 9. Démarrer le serveur Flask wrapper en arrière-plan
echo "🚀 Démarrage du serveur sur $HOST:$PORT..."

python -m app.server_embedding >> logs/server.log 2>&1 &

SERVER_PID=$!

# Sauvegarder le PID
echo $SERVER_PID > .server.pid

# Attendre un peu pour vérifier que le serveur démarre
sleep 3

if ps -p $SERVER_PID > /dev/null; then
    echo ""
    echo "✅ Serveur démarré avec succès!"
    echo "   PID: $SERVER_PID"
    echo "   URL: http://${HOST}:${PORT}"
    echo "   Logs: tail -f logs/server.log"
    echo ""
    echo "📝 Pour tester:"
    echo "   curl http://${HOST}:${PORT}/v1/embeddings \\"
    echo "     -H \"Authorization: Bearer \$API_KEY\" \\"
    echo "     -H \"Content-Type: application/json\" \\"
    echo "     -d '{\"input\":\"Hello world\",\"model\":\"multilingual-e5-large\"}'"
    echo ""
    echo "🛑 Pour arrêter: ./stop.sh"
else
    echo "❌ Erreur: Le serveur n'a pas démarré correctement"
    echo "   Vérifiez les logs: cat logs/server.log"
    rm .server.pid
    exit 1
fi

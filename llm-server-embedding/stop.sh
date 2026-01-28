#!/bin/bash
# ==============================================
# MLX Multilingual-E5-Large API Server - Script d'Arrêt
# ==============================================

echo "🛑 Arrêt MLX Multilingual-E5-Large API Server..."

# 1. Vérifier si le fichier PID existe
if [ -f ".server.pid" ]; then
    PID=$(cat .server.pid)

    # Vérifier si le processus existe
    if ps -p $PID > /dev/null 2>&1; then
        echo "📍 Processus trouvé (PID: $PID)"

        # Arrêt gracieux avec SIGTERM
        echo "⏸️  Envoi de SIGTERM..."
        kill $PID 2>/dev/null

        # Attendre jusqu'à 5 secondes
        for i in {1..5}; do
            if ! ps -p $PID > /dev/null 2>&1; then
                echo "✅ Serveur arrêté proprement"
                rm .server.pid
                exit 0
            fi
            echo "⏳ Attente de l'arrêt ($i/5)..."
            sleep 1
        done

        # Si toujours en cours, forcer avec SIGKILL
        if ps -p $PID > /dev/null 2>&1; then
            echo "⚠️  Arrêt forcé avec SIGKILL..."
            kill -9 $PID 2>/dev/null
            sleep 1

            if ps -p $PID > /dev/null 2>&1; then
                echo "❌ Impossible d'arrêter le processus $PID"
                exit 1
            fi
        fi

        echo "✅ Serveur arrêté (forcé)"
        rm .server.pid

    else
        echo "⚠️  Le processus $PID n'existe plus"
        echo "🧹 Nettoyage du fichier PID..."
        rm .server.pid
    fi
else
    # Pas de fichier PID - tenter d'arrêter via pkill
    echo "⚠️  Aucun fichier .server.pid trouvé"
    echo "🔍 Recherche de processus server_embedding..."

    if pkill -f "server_embedding" 2>/dev/null; then
        echo "✅ Processus server_embedding arrêté"
    else
        echo "ℹ️  Aucun processus server_embedding en cours"
    fi
fi

echo "✅ Arrêt terminé"

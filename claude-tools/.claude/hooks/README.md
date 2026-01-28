# Hooks grepai

Ce dossier contient les hooks Claude Code pour forcer l'utilisation de grepai.

## Installation

### 1. Rendre le script exécutable

```bash
chmod +x .claude/hooks/force-grepai.sh
```

### 2. Vérifier les dépendances

Le script nécessite `jq` pour parser le JSON :

```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt install jq
```

### 3. Tester le hook

```bash
echo '{"tool_name": "bash", "tool_input": {"command": "grep -r test ."}}' | .claude/hooks/force-grepai.sh
```

Devrait retourner un `decision: block` si `.grepai/config.yaml` existe.

## Comportement

| Outil/Commande | Action du hook |
|----------------|----------------|
| `grep`, `rg`, `ag`, `ack` | ❌ Bloqué → utiliser `grepai search` |
| `find -name`, `locate`, `fd` | ❌ Bloqué → utiliser `grepai search` |
| `glob` | ❌ Bloqué → utiliser `grepai search` |
| `grepai *` | ✅ Autorisé |
| Autres commandes | ✅ Autorisé |

## Personnalisation

### Ajouter des commandes à bloquer

Éditer `force-grepai.sh`, modifier les patterns :

```bash
SEARCH_COMMANDS="grep|egrep|fgrep|rg|ripgrep|ag|ack|pt|sift|NOUVELLE_COMMANDE"
```

### Désactiver temporairement

Commenter la section dans `.claude/settings.json` :

```json
{
  "hooks": {
    "PreToolUse": [
      // { "matcher": "bash", ... }
    ]
  }
}
```

## Debug

Si le hook ne fonctionne pas :

1. Vérifier qu'il est exécutable : `ls -la .claude/hooks/`
2. Tester manuellement avec un JSON d'entrée
3. Vérifier que `jq` est installé
4. Vérifier que `.grepai/config.yaml` existe dans le projet cible

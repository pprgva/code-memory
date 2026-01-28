# grepai - Guide d'installation et d'utilisation

## Qu'est-ce que grepai ?

Un moteur de recherche sémantique pour ton code. Au lieu de chercher du texte exact (`grep`), tu poses une question en langage naturel et il trouve le code pertinent.

Il est aussi utilisable automatiquement par les agents AI (Claude Code, Cursor) pour explorer ton code plus intelligemment.

---

## 1. Installation

### Compiler le binaire

```bash
cd /Users/ppr/Documents/development/code-memory
make build
```

Le binaire se retrouve dans `bin/grepai`.

### Rendre accessible partout

```bash
mkdir -p ~/.local/bin
cp bin/grepai ~/.local/bin/
```

Assure-toi que `~/.local/bin` est dans ton PATH. Dans `~/.zshrc` :

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Recharge le shell :

```bash
source ~/.zshrc
```

Vérifie :

```bash
grepai --help
```

---

## 2. Prérequis

- **Python 3** installé sur la machine (pour le worker d'embeddings)
- Le modèle E5 est téléchargé **automatiquement** au premier `grepai init`

---

## 3. Initialiser un projet

Va dans le dossier de ton projet et lance :

```bash
cd /chemin/vers/ton-projet
grepai init --yes
```

Ce que ça fait :
- Crée le dossier `.grepai/` avec le fichier `config.yaml`
- Crée un environnement Python virtuel dans `.grepai/venv/`
- Installe automatiquement `torch` et `transformers` dans le venv
- Télécharge le modèle `intfloat/multilingual-e5-large` dans `~/.grepai/models/` (une seule fois, partagé entre tous les projets)
- Ajoute `.grepai/` au `.gitignore` si présent

Le `--yes` évite les questions interactives. Sans ce flag, l'outil te demande le backend de stockage.

Si tu as déjà un modèle E5 sur ta machine, tu peux le spécifier :

```bash
grepai init --model-path /chemin/vers/mon-modele --yes
```

### Options de stockage

Par défaut, l'index est stocké dans un fichier local (GOB). Pour les gros projets :

```bash
# PostgreSQL avec pgvector
grepai init --model-path /chemin/modele --backend postgres

# Qdrant (base vectorielle Docker)
grepai init --model-path /chemin/modele --backend qdrant
```

---

## 4. Indexer et surveiller

### Mode arrière-plan (recommandé)

```bash
grepai watch --background
```

Le watcher tourne en fond et maintient l'index à jour quand tu modifies des fichiers.

```bash
# Vérifier que ça tourne
grepai watch --status

# Arrêter
grepai watch --stop
```

Les logs sont dans `~/Library/Logs/grepai/grepai-watch.log`.

### Mode premier plan (debug)

```bash
grepai watch
```

Affiche la progression en direct. Ctrl+C pour arrêter.

---

## 5. Chercher dans le code

```bash
# Recherche sémantique en langage naturel
grepai search "gestion des erreurs HTTP"
grepai search "user authentication flow"
grepai search "database connection pooling"

# Limiter le nombre de résultats
grepai search "validation des formulaires" -n 5

# Sortie JSON (pour scripts ou agents AI)
grepai search "error handling" --json

# JSON compact sans contenu (économise des tokens pour les agents)
grepai search "error handling" --json --compact
```

### Conseils pour les requêtes

- **Utilise l'anglais** pour de meilleurs résultats (le modèle E5 est entraîné en anglais)
- **Décris l'intention**, pas le code : "handles user login" plutôt que "func Login"
- **Sois spécifique** : "JWT token validation" plutôt que "token"

---

## 6. Tracer les appels de fonctions

```bash
# Qui appelle cette fonction ?
grepai trace callers "HandleRequest"

# Qu'est-ce que cette fonction appelle ?
grepai trace callees "ProcessOrder"

# Graphe complet (callers + callees)
grepai trace graph "ValidateToken" --depth 3

# Sortie JSON
grepai trace callers "HandleRequest" --json
```

---

## 7. Exclusions automatiques

Par défaut, grepai n'indexe **pas** ces dossiers :

| Catégorie | Exclusions |
|-----------|-----------|
| VCS | `.git`, `.svn`, `.hg` |
| IDE | `.idea`, `.vscode`, `.vs`, `.eclipse` |
| JS/TS | `node_modules`, `.next`, `.nuxt`, `bower_components` |
| Build | `build`, `out`, `dist`, `bin`, `obj`, `coverage` |
| Python | `__pycache__`, `.venv`, `venv`, `.tox`, `.mypy_cache` |
| Go | `vendor` |
| Rust | `target` |
| Java | `.gradle`, `.m2` |
| PHP | `var` |
| iOS/macOS | `Pods`, `DerivedData`, `.build` |
| Dart | `.dart_tool`, `.pub-cache` |
| Infra | `.terraform`, `.vagrant` |
| OS | `.DS_Store`, `Thumbs.db`, `tmp`, `temp`, `logs` |

Le scanner respecte aussi les `.gitignore` du projet.

Pour ajouter des exclusions, édite `.grepai/config.yaml` :

```yaml
ignore:
  - .git
  - node_modules
  - vendor
  - mon_dossier_custom    # ← ajouter ici
```

---

## 8. Utilisation avec Claude Code

grepai est automatiquement utilisé par Claude Code quand le `CLAUDE.md` du projet contient la directive appropriée (déjà configurée dans ce repo).

Claude Code utilise `grepai search` au lieu de `grep` pour explorer le code, ce qui donne des résultats plus pertinents sur les questions sémantiques.

---

## 9. Résumé des commandes

| Commande | Description |
|----------|-------------|
| `grepai init` | Initialiser un projet (une seule fois) |
| `grepai watch --background` | Lancer l'indexation en arrière-plan |
| `grepai watch --status` | Vérifier si le watcher tourne |
| `grepai watch --stop` | Arrêter le watcher |
| `grepai search "requête"` | Recherche sémantique |
| `grepai search "query" --json` | Recherche en JSON |
| `grepai trace callers "Func"` | Trouver les appelants |
| `grepai trace callees "Func"` | Trouver les fonctions appelées |
| `grepai trace graph "Func"` | Graphe d'appels complet |

---

## 10. Dépannage

**"no grepai project found"** : tu n'es pas dans un dossier initialisé. Lance `grepai init`.

**Le worker Python plante** : vérifie que le venv est OK :
```bash
.grepai/venv/bin/python3 -c "import torch, transformers; print('OK')"
```

Si le venv est cassé, supprime-le et relance init :
```bash
rm -rf .grepai/venv
grepai init --yes
```

**L'index est vide** : le watcher n'a pas tourné. Lance `grepai watch` en premier plan pour voir la progression.

# grepai - Roadmap

Ce document capture toutes les améliorations planifiées pour grepai.
Dernière mise à jour : 2026-01-28

---

## 1. Recherche Cross-Projet

**Problème** : On ne peut chercher que dans le projet courant. Parfois la solution existe dans un autre projet (Swift, Python, etc.).

**Solution** : Système de projets nommés avec recherche fédérée.

### Commandes
```bash
# Enregistrer des projets
grepai project add                      # Projet courant
grepai project add --name whisperclip   # Avec nom explicite
grepai project list                     # Lister tous les projets
grepai project remove <name>            # Supprimer

# Recherche cross-projet
grepai search "query" -p projet1 -p projet2
grepai search "query" --all             # Tous les projets

# Groupes (optionnel)
grepai group create fullstack proj1 proj2
grepai search "query" --group fullstack
```

### Stockage
```yaml
# ~/.grepai/projects.yaml
version: 1
projects:
  whisperclip:
    path: "~/Documents/development/SuperwhisperaPaul"
  grepai:
    path: "~/Documents/development/code-memory"
```

### Caractéristiques
- Chemins portables avec `~` (pas de chemins absolus)
- Recherche fédérée sur les index GOB existants (pas besoin de PostgreSQL)
- Fusion des résultats avec Reciprocal Rank Fusion (RRF)

---

## 2. Aide Intégrée Intelligente

**Problème** : Difficile de se souvenir de toutes les options et commandes.

**Solution** : Système d'aide complet et interrogeable.

### Commandes
```bash
# Aide par commande
grepai help                    # Vue d'ensemble
grepai help search             # Aide complète sur search
grepai help search --type      # Aide sur un flag spécifique
grepai help config             # Comment configurer grepai

# Aide contextuelle / exemples
grepai help examples           # Exemples concrets d'utilisation
grepai help troubleshooting    # Problèmes courants et solutions

# Aide interrogeable (pour Claude/LLMs)
grepai explain "comment chercher dans plusieurs projets"
grepai explain "filtrer les résultats par date"
```

### Caractéristiques
- Chaque commande et option documentée en détail
- Exemples concrets pour chaque cas d'usage
- Format lisible par les humains ET par les LLMs
- Sortie JSON possible pour intégration IA

---

## 3. Restructuration Base de Données + Entité Projet

**Problème** : Actuellement les chunks sont stockés avec des chemins absolus. Pas d'entité "projet" centrale. Si on déplace un projet, tout casse.

**Solution** : Ajouter une table `projects` comme entité centrale.

### Nouveau schéma PostgreSQL
```sql
-- Table projets (NOUVELLE)
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,           -- "whisperclip", "grepai"
    path TEXT,                           -- Chemin actuel (peut changer)
    languages TEXT[],                    -- ["swift", "python"]
    framework TEXT,                      -- "swiftui", "fastapi", etc.
    description TEXT,                    -- Généré par Claude
    metadata JSONB,                      -- Infos supplémentaires
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Table chunks (MODIFIÉE)
CREATE TABLE chunks (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id),  -- Lié au projet
    file_path TEXT NOT NULL,                  -- Chemin RELATIF au projet
    content TEXT,
    embedding vector(1024),
    metadata JSONB,
    created_at TIMESTAMP
);
```

### Avantages
- **Chemin déplaçable** : On change `projects.path`, les chunks restent valides
- **Métadonnées riches** : Langages, framework, description
- **Requêtes par projet** : Facile de filtrer, supprimer, migrer

---

## 4. Initialisation Automatique par Claude

**Problème** : L'initialisation actuelle demande trop de questions. Claude doit pouvoir setup un projet en une commande.

**Solution** : Commande `grepai setup` intelligente pour les LLMs.

### Commande
```bash
# Claude exécute ça et c'est tout
grepai setup --auto

# Ce que ça fait automatiquement :
# 1. Détecte le type de projet (Swift, Python, Go, etc.)
# 2. Détecte le framework (SwiftUI, FastAPI, Gin, etc.)
# 3. Crée le projet dans la base avec toutes les métadonnées
# 4. Lance l'indexation
# 5. Retourne un JSON avec les infos du projet
```

### Détection automatique
```yaml
# Ce que Claude/grepai détecte automatiquement :
- Package.swift → Swift + SPM
- project.yml → Swift + XcodeGen
- pyproject.toml → Python + Poetry/PDM
- go.mod → Go
- package.json → JavaScript/TypeScript
- Cargo.toml → Rust
- etc.
```

### Commandes LLM-friendly
```bash
# Setup complet automatique
grepai setup --auto --json

# Refresh/upgrade (ré-indexation complète)
grepai refresh
grepai refresh --project whisperclip

# Mise à jour du chemin si projet déplacé
grepai project move whisperclip ~/new/path

# Info projet (pour que Claude sache ce qu'il a)
grepai project info --json
```

### Output JSON pour Claude
```json
{
  "project": {
    "id": "uuid-xxx",
    "name": "whisperclip",
    "path": "~/Documents/development/SuperwhisperaPaul",
    "languages": ["swift"],
    "framework": "swiftui",
    "files_indexed": 42,
    "chunks": 1203
  },
  "status": "ready",
  "commands": {
    "search": "grepai search \"query\" --json",
    "trace": "grepai trace callers \"symbol\" --json",
    "refresh": "grepai refresh"
  }
}
```

### Philosophie
- **Zero config pour Claude** : `grepai setup --auto` fait tout
- **Tout est JSON** : Claude parse facilement
- **Projet = entité centrale** : Tout est lié au projet, pas aux chemins

---

## 5. Configuration Machine vs Projet

**Problème** : Actuellement on mélange config machine (DSN Supabase) et config projet (chunking). Claude doit poser des questions qu'il ne devrait pas poser.

**Solution** : Séparer clairement les deux niveaux.

### Niveau 1 : Configuration Machine (une seule fois)
```yaml
# ~/.grepai/machine.yaml
version: 1

# Base de données (ne change jamais sauf si nouvelle machine)
database:
  dsn: "postgresql://postgres.xxx:password@localhost:5432/postgres"

# Modèle E5 (installé une fois)
embedder:
  model_path: "~/.grepai/models/multilingual-e5-large"

# Préférences utilisateur
preferences:
  default_limit: 10
  json_output: false

# Chemins système
paths:
  venv: "~/.grepai/venv"
  cache: "~/.grepai/cache"
```

### Niveau 2 : Configuration Projet (automatique)
```yaml
# .grepai/config.yaml (généré automatiquement par grepai setup)
version: 1
project:
  name: "whisperclip"
  id: "uuid-xxx"

# Détecté automatiquement, pas besoin de toucher
detected:
  languages: ["swift"]
  framework: "swiftui"
  build_tool: "xcodegen"

# Peut être customisé si besoin (rare)
chunking:
  size: 512
  overlap: 100
```

### Workflow
```
NOUVELLE MACHINE (une seule fois) :
1. brew install grepai
2. grepai machine setup
   → Demande DSN Supabase
   → Télécharge modèle E5
   → Sauvegarde dans ~/.grepai/machine.yaml
3. C'est fini. Plus jamais besoin d'y toucher.

NOUVEAU PROJET (automatique) :
1. Claude fait: grepai setup --auto
   → Détecte langages, framework
   → Crée le projet en base
   → Lance le watcher
2. Claude code, les fichiers sont indexés automatiquement
3. L'humain ne fait RIEN
```

### Avantages
- **Humain** : Configure sa machine une fois, oublie
- **Claude** : `grepai setup --auto` et c'est prêt
- **Pas d'erreurs humaines** : Claude gère ce qu'il maîtrise
- **Portable** : Nouvelle machine = `grepai machine setup` puis c'est reparti

---

## 6. Commandes de Maintenance Globales

**Problème** : Si on met à jour grepai (nouveau schéma, nouvelle version), il faut ré-indexer tous les projets. Aller dans 20 dossiers faire `new-project` un par un = impossible.

**Solution** : Commandes qui agissent sur TOUS les projets enregistrés.

### Commandes
```bash
# Ré-indexer tous les projets (après une mise à jour de grepai)
grepai upgrade-all
# → Parcourt ~/.grepai/projects.yaml
# → Pour chaque projet : supprime l'ancien index, ré-indexe
# → Affiche la progression

# Variantes
grepai upgrade-all --parallel 4    # 4 projets en parallèle
grepai upgrade-all --project X Y   # Seulement certains projets
grepai upgrade-all --dry-run       # Voir ce qui serait fait

# Vérifier l'état de tous les projets
grepai status-all
# Output:
# PROJECT       STATUS      FILES    LAST UPDATE
# whisperclip   watching    42       2 min ago
# grepai        indexed     156      1 hour ago
# backend       outdated    89       3 days ago ⚠️

# Lancer watch sur tous les projets
grepai watch-all
# → Démarre un daemon qui surveille tous les projets

# Arrêter tous les watchers
grepai stop-all
```

### Cas d'usage
1. **Mise à jour grepai** : `grepai upgrade-all` après `brew upgrade grepai`
2. **Nouvelle machine** : `grepai machine setup` puis `grepai upgrade-all`
3. **Vérification quotidienne** : `grepai status-all` pour voir si tout va bien
4. **Mode développeur** : `grepai watch-all` pour tout surveiller

### Progression visuelle
```
$ grepai upgrade-all

Upgrading 5 projects...

[1/5] whisperclip     ████████████████████ 100%  42 files
[2/5] grepai          ████████████████████ 100%  156 files
[3/5] backend-api     ████████░░░░░░░░░░░░  40%  36/89 files
[4/5] frontend        ░░░░░░░░░░░░░░░░░░░░  waiting...
[5/5] shared-lib      ░░░░░░░░░░░░░░░░░░░░  waiting...

Done! 5 projects upgraded.
```

---

## 7. Nouveau Schéma PostgreSQL

**Problème** : Le schéma actuel est minimal (juste `chunks` avec chemins absolus). Pas d'entité projet, pas de portabilité, symboles stockés en GOB séparé.

**Solution** : Schéma complet avec projets comme entité centrale.

### Tables Core
```sql
-- Projets : entité centrale
CREATE TABLE grepai_projects (
    id              UUID PRIMARY KEY,
    name            TEXT UNIQUE NOT NULL,
    local_path      TEXT,                    -- Chemin actuel (seul endroit à changer si déplacé)
    languages       TEXT[],
    framework       TEXT,
    file_count      INTEGER DEFAULT 0,       -- Stats dénormalisées
    chunk_count     INTEGER DEFAULT 0,
    symbol_count    INTEGER DEFAULT 0,
    index_status    TEXT DEFAULT 'pending',
    created_at      TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ
);

-- Fichiers : chemins RELATIFS au projet
CREATE TABLE grepai_files (
    id              UUID PRIMARY KEY,
    project_id      UUID REFERENCES grepai_projects(id),
    relative_path   TEXT NOT NULL,           -- Portable !
    language        TEXT,
    content_hash    TEXT NOT NULL,
    line_count      INTEGER,
    UNIQUE(project_id, relative_path)
);

-- Chunks : liés aux fichiers, pas aux chemins
CREATE TABLE grepai_chunks (
    id              UUID PRIMARY KEY,
    project_id      UUID REFERENCES grepai_projects(id),
    file_id         UUID REFERENCES grepai_files(id),
    start_line      INTEGER,
    end_line        INTEGER,
    content         TEXT,
    embedding       vector(1024)
);
```

### Tables Call Graph
```sql
-- Symboles (fonctions, classes, etc.)
CREATE TABLE grepai_symbols (
    id              UUID PRIMARY KEY,
    project_id      UUID REFERENCES grepai_projects(id),
    file_id         UUID REFERENCES grepai_files(id),
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL,           -- function, method, class...
    line            INTEGER,
    signature       TEXT,
    is_exported     BOOLEAN
);

-- Références (appels, usages)
CREATE TABLE grepai_references (
    id              UUID PRIMARY KEY,
    project_id      UUID REFERENCES grepai_projects(id),
    file_id         UUID REFERENCES grepai_files(id),
    symbol_name     TEXT NOT NULL,
    symbol_id       UUID REFERENCES grepai_symbols(id),
    caller_symbol_id UUID REFERENCES grepai_symbols(id),
    line            INTEGER,
    context         TEXT
);

-- Graphe d'appels pré-calculé
CREATE TABLE grepai_call_edges (
    id              UUID PRIMARY KEY,
    project_id      UUID REFERENCES grepai_projects(id),
    caller_id       UUID REFERENCES grepai_symbols(id),
    callee_id       UUID REFERENCES grepai_symbols(id),
    caller_name     TEXT,
    callee_name     TEXT
);
```

### Tables Auxiliaires
```sql
-- Historique des recherches (analytics, debug)
CREATE TABLE grepai_search_history (
    id              UUID PRIMARY KEY,
    project_id      UUID,
    query           TEXT,
    result_count    INTEGER,
    duration_ms     INTEGER,
    created_at      TIMESTAMPTZ
);

-- Logs d'indexation
CREATE TABLE grepai_index_events (
    id              UUID PRIMARY KEY,
    project_id      UUID,
    event_type      TEXT,
    files_processed INTEGER,
    duration_ms     INTEGER,
    started_at      TIMESTAMPTZ
);
```

### Avantages
- **Projet déplacé** → change `grepai_projects.local_path`, c'est tout
- **Symboles en PostgreSQL** → plus de fichier GOB séparé
- **Cross-projet** → simple JOIN sur project_id
- **Analytics** → historique des recherches pour debug

---

## Priorités

| # | Fonctionnalité | Priorité | Complexité | Status |
|---|----------------|----------|------------|--------|
| 1 | Cross-projet | Haute | Moyenne | Planifié |
| 2 | Aide intégrée | Haute | Faible | Planifié |
| 3 | Table projets DB | Haute | Moyenne | Planifié |
| 4 | Setup auto Claude | Haute | Faible | Planifié |
| 5 | Config machine/projet | Moyenne | Faible | Planifié |
| 6 | Commandes globales | Moyenne | Faible | Planifié |
| 7 | Nouveau schéma complet | Haute | Haute | Planifié |

---

## 8. Mode interactif Claude-first

**Problème** : Actuellement, soit l'app pose des questions à l'humain (friction), soit elle fait tout en auto (mais rate des infos que Claude connaît).

**Solution** : L'app retourne une liste de questions, Claude répond à ce qu'il sait, et demande à l'humain le reste.

### Flux
```
1. Claude: grepai setup --interactive-json

2. grepai retourne:
{
  "questions": [
    {"id": "backend", "question": "Quel backend ?", "options": ["gob", "postgres"]},
    {"id": "dsn", "question": "DSN PostgreSQL ?", "depends_on": "backend=postgres"},
    {"id": "name", "question": "Nom du projet ?", "default": "code-memory"}
  ]
}

3. Claude analyse:
   - "Backend ?" → Je sais: postgres (Supabase)
   - "DSN ?" → Je check machine.yaml → existe → j'utilise
   - "Nom ?" → Je déduis du dossier

4. Claude: grepai setup --answers '{"backend":"postgres","dsn":"...","name":"code-memory"}'
```

### Avantages
- **Claude répond intelligemment** aux questions qu'il connaît
- **L'humain n'est sollicité** que pour ce que Claude ne sait pas
- **L'app reste simple** : elle expose ses besoins, Claude gère l'intelligence

### Implémentation
```go
// grepai setup --interactive-json
type SetupQuestion struct {
    ID        string   `json:"id"`
    Question  string   `json:"question"`
    Type      string   `json:"type"`      // text, choice, bool
    Options   []string `json:"options,omitempty"`
    Default   string   `json:"default,omitempty"`
    DependsOn string   `json:"depends_on,omitempty"` // "backend=postgres"
    Hint      string   `json:"hint,omitempty"`
}

// grepai setup --answers '{"key":"value",...}'
```

---

## 9. UUID Projet comme Source de Vérité

**Problème** : Si on renomme ou déplace un projet, comment savoir que c'est le même ? Le nom et le chemin peuvent changer.

**Solution** : Un UUID généré à la création du projet qui **fait foi**.

### Concept
```yaml
# .grepai/config.yaml (fichier local)
project_id: "550e8400-e29b-41d4-a716-446655440000"
name: "whisperclip"  # peut changer
```

```sql
-- PostgreSQL
grepai_projects.id = "550e8400-e29b-41d4-a716-446655440000"
grepai_projects.name = "whisperclip"  -- peut changer
grepai_projects.local_path = "~/Documents/..."  -- peut changer
```

### L'UUID fait foi
- **Renommage** : Le dossier s'appelle maintenant "whisper-v2" → L'UUID reste, on sait que c'est le même projet
- **Déplacement** : Le projet est dans un autre dossier → L'UUID reste
- **Synchronisation** : Claude vérifie `local UUID == base UUID` → Match = même projet
- **Conflit** : Deux dossiers avec le même nom mais UUID différents → Ce sont deux projets distincts

### Workflow
```
1. grepai setup --auto
   → Génère UUID: "550e8400-..."
   → Écrit dans .grepai/config.yaml
   → Crée en base avec cet UUID

2. Utilisateur déplace le projet
   → Le chemin change
   → L'UUID dans .grepai/config.yaml reste

3. grepai search (dans le nouveau chemin)
   → Lit l'UUID local
   → Trouve le projet en base par UUID
   → Met à jour le chemin en base si différent
   → Fonctionne normalement
```

### Avantages
- **Découplage total** : L'identité du projet ne dépend plus du chemin
- **Robustesse** : Pas de confusion même avec des noms identiques
- **Auto-réparation** : Le chemin en base se met à jour automatiquement

---

## 10. Ignorer tous les dotfiles/dotfolders par défaut

**Problème** : La liste d'ignore est longue et explicite. Les dossiers/fichiers commençant par `.` sont presque toujours des configs, caches, ou données non pertinentes.

**Solution** : Ajouter une règle générique pour ignorer tous les `.*`

### Avantages
- Plus simple à maintenir
- Les utilisateurs peuvent créer `.notes/`, `.docs/`, `.private/` pour leurs données non indexées
- Pas besoin de lister chaque `.xxx` individuellement

### Exceptions possibles
- `.github/` → Workflows CI (peut être utile parfois)
- Configurable via `.grepai/config.yaml` si besoin

### Implémentation
Dans `config/config.go`, la liste `Ignore` par défaut inclura `".*"` en premier.

---

## Notes de discussion

(Espace pour capturer les décisions et contexte des discussions)

- 2026-01-28 : Discussion initiale sur cross-projet et aide intégrée
- 2026-01-28 : Décision d'ignorer tous les dotfiles par défaut (exemple : mettre ROADMAP.md dans `.notes/`)

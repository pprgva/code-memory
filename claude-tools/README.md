# claude-tools — Configuration Claude Code pour grepai

Ce dossier contient les fichiers de configuration pour qu'une session Claude Code puisse utiliser grepai.

## Installation

Copier les fichiers dans votre projet :

```bash
# Copier le CLAUDE.md à la racine du projet
cp claude-tools/CLAUDE.md /chemin/vers/mon-projet/CLAUDE.md

# Copier les skills et commandes
cp -r claude-tools/.claude /chemin/vers/mon-projet/.claude
```

Ou pour une configuration globale (tous les projets) :

```bash
# Ajouter le contenu au CLAUDE.md global
cat claude-tools/CLAUDE.md >> ~/.claude/CLAUDE.md

# Copier skills et commandes globalement
cp -r claude-tools/.claude/skills/* ~/.claude/skills/
cp -r claude-tools/.claude/commands/* ~/.claude/commands/
```

## Contenu

- `CLAUDE.md` — Instructions principales pour Claude Code (commandes, quand les utiliser)
- `.claude/skills/grepai-search.md` — Skill de recherche sémantique
- `.claude/skills/grepai-trace.md` — Skill d'analyse du call graph
- `.claude/commands/doctor.md` — Commande /doctor
- `.claude/commands/search.md` — Commande /search

## Prérequis

Le projet cible doit avoir été initialisé avec `grepai init` et indexé avec `grepai watch`.

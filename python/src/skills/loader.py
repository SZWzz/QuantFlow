"""Skill loader — parses Markdown files with YAML frontmatter."""

import logging
import os
import re
from dataclasses import dataclass, field
from typing import List, Optional

import yaml

logger = logging.getLogger(__name__)


@dataclass
class Skill:
    """A skill document with frontmatter metadata and markdown content."""
    name: str           # filename without .md
    category: str       # from frontmatter
    title: str          # from frontmatter
    tags: List[str] = field(default_factory=list)
    difficulty: str = "intermediate"
    content: str = ""   # full markdown body (without frontmatter)


def _parse_frontmatter(text: str) -> tuple[dict, str]:
    """Parse YAML frontmatter from markdown text. Returns (metadata, content)."""
    match = re.match(r'^---\s*\n(.*?)\n---\s*\n', text, re.DOTALL)
    if not match:
        return {}, text
    try:
        meta = yaml.safe_load(match.group(1)) or {}
    except yaml.YAMLError:
        meta = {}
    content = text[match.end():]
    return meta, content


def load_skills(skills_dir: str) -> List[Skill]:
    """Load all skill files from a directory tree.

    Walks the directory recursively. Each .md file is parsed for
    YAML frontmatter + markdown content.
    """
    skills = []
    for root, dirs, files in os.walk(skills_dir):
        for fname in files:
            if not fname.endswith('.md'):
                continue
            fpath = os.path.join(root, fname)
            try:
                with open(fpath, 'r', encoding='utf-8') as f:
                    text = f.read()
            except Exception as e:
                logger.warning(f"Failed to read skill file {fpath}: {e}")
                continue

            meta, content = _parse_frontmatter(text)
            name = fname.replace('.md', '')
            category = meta.get('category', os.path.basename(root))

            skill = Skill(
                name=name,
                category=category,
                title=meta.get('title', name.replace('_', ' ').title()),
                tags=meta.get('tags', []),
                difficulty=meta.get('difficulty', 'intermediate'),
                content=content.strip(),
            )
            skills.append(skill)

    logger.info(f"Loaded {len(skills)} skills from {skills_dir}")
    return skills


def search_skills(skills: List[Skill], query: str, category: Optional[str] = None) -> List[Skill]:
    """Search skills by keyword and optional category filter.

    Searches in: title, tags, content (case-insensitive).
    """
    q = query.lower()
    results = []
    for s in skills:
        if category and s.category != category:
            continue
        if (q in s.title.lower() or
            q in s.content.lower() or
            any(q in tag.lower() for tag in s.tags)):
            results.append(s)
    return results


def get_categories(skills: List[Skill]) -> List[str]:
    """Return sorted list of unique categories."""
    return sorted(set(s.category for s in skills))

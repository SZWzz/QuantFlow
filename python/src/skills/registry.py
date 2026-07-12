"""Skill registry — exposes skills as structured, callable tools.

Extends the existing skill loader (loader.py) with a registry that can:
1. List all skills with metadata (for LLM tool discovery).
2. Retrieve full skill content (for LLM knowledge injection).
3. Search skills by query/category.
4. Export skills in LLM function-calling format.
"""

from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

from .loader import Skill, load_skills, search_skills, get_categories


@dataclass
class SkillRegistry:
    """A structured registry of all loaded skills with tool-format export."""

    skills: list[Skill] = field(default_factory=list)
    _skills_dir: Optional[Path] = None

    def load(self, skills_dir: Optional[Path] = None) -> "SkillRegistry":
        """Load all skills from the given directory (default: ../skills relative to this file)."""
        if skills_dir is None:
            skills_dir = Path(__file__).resolve().parent.parent.parent / "skills"
        self._skills_dir = Path(skills_dir)
        self.skills = load_skills(str(self._skills_dir))
        return self

    def list_all(self) -> list[dict]:
        """Return all skills as metadata dicts (id, title, category, tags, difficulty)."""
        return [
            {
                "id": s.name,
                "title": s.title,
                "category": s.category,
                "tags": s.tags,
                "difficulty": s.difficulty,
            }
            for s in self.skills
        ]

    def get(self, name: str) -> Optional[Skill]:
        """Get a skill by name (filename without .md)."""
        for s in self.skills:
            if s.name == name:
                return s
        return None

    def get_content(self, name: str) -> Optional[str]:
        """Get the full markdown content of a skill."""
        skill = self.get(name)
        return skill.content if skill else None

    def search(self, query: str, category: Optional[str] = None) -> list[dict]:
        """Search skills by query and optional category filter."""
        results = search_skills(self.skills, query, category)
        return [
            {
                "id": s.name,
                "title": s.title,
                "category": s.category,
                "tags": s.tags,
                "difficulty": s.difficulty,
                "snippet": s.content[:200] + "..." if len(s.content) > 200 else s.content,
            }
            for s in results
        ]

    def categories(self) -> list[str]:
        """Return sorted unique category names."""
        return get_categories(self.skills)

    def list_for_llm(self) -> list[dict]:
        """Export skills as LLM function-calling tool definitions.

        Each skill becomes a function that the LLM can "call" to retrieve
        domain expertise. The function name is "skill_<name>", the description
        comes from the skill title and tags.
        """
        tools = []
        for s in self.skills:
            tools.append(
                {
                    "type": "function",
                    "function": {
                        "name": f"skill_{s.name}",
                        "description": (
                            f"[{s.category}] {s.title}. "
                            f"Difficulty: {s.difficulty}. "
                            f"Tags: {', '.join(s.tags[:5])}"
                        ),
                        "parameters": {
                            "type": "object",
                            "properties": {},
                            "required": [],
                        },
                    },
                }
            )
        return tools


# Singleton for lazy loading.
_registry: Optional[SkillRegistry] = None


def get_registry() -> SkillRegistry:
    """Return the global skill registry (loads on first call)."""
    global _registry
    if _registry is None:
        _registry = SkillRegistry().load()
    return _registry

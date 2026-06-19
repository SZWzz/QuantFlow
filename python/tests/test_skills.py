"""Tests for the Skill Knowledge Base loader."""
import os
import tempfile
import pytest

from src.skills.loader import Skill, load_skills, search_skills, get_categories, _parse_frontmatter


class TestFrontmatterParsing:
    def test_parse_with_frontmatter(self):
        text = """---
title: Test Skill
category: test_cat
tags: [a, b]
difficulty: beginner
---

# Hello
Content here."""
        meta, content = _parse_frontmatter(text)
        assert meta["title"] == "Test Skill"
        assert meta["category"] == "test_cat"
        assert meta["tags"] == ["a", "b"]
        assert "Content here" in content

    def test_parse_without_frontmatter(self):
        text = "# Just content\nNo frontmatter."
        meta, content = _parse_frontmatter(text)
        assert meta == {}
        assert "Just content" in content


class TestSkillLoader:
    def test_load_skills_from_dir(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            os.makedirs(os.path.join(tmpdir, "cat1"))
            with open(os.path.join(tmpdir, "cat1", "skill_a.md"), "w") as f:
                f.write("""---
title: Skill A
category: cat1
tags: [tag1]
---
# Skill A Content
This is skill A.""")
            with open(os.path.join(tmpdir, "cat1", "skill_b.md"), "w") as f:
                f.write("""---
title: Skill B
category: cat1
tags: [tag2]
---
# Skill B Content
Momentum trading content.""")
            # Non-.md file should be ignored
            with open(os.path.join(tmpdir, "readme.txt"), "w") as f:
                f.write("not a skill")

            skills = load_skills(tmpdir)
            assert len(skills) == 2
            assert {s.name for s in skills} == {"skill_a", "skill_b"}

    def test_search_skills(self):
        skills = [
            Skill(name="a", category="cat1", title="Momentum Strategy", tags=["momentum"], content="Buy winners."),
            Skill(name="b", category="cat2", title="Mean Reversion", tags=["reversion"], content="Buy dips."),
        ]
        results = search_skills(skills, "momentum")
        assert len(results) == 1
        assert results[0].name == "a"

    def test_search_with_category_filter(self):
        skills = [
            Skill(name="a", category="cat1", title="Momentum", tags=["momentum"], content="x"),
            Skill(name="b", category="cat2", title="Momentum B", tags=["momentum"], content="x"),
        ]
        results = search_skills(skills, "momentum", category="cat1")
        assert len(results) == 1

    def test_get_categories(self):
        skills = [
            Skill(name="a", category="cat1", title="A", content=""),
            Skill(name="b", category="cat2", title="B", content=""),
            Skill(name="c", category="cat1", title="C", content=""),
        ]
        cats = get_categories(skills)
        assert cats == ["cat1", "cat2"]

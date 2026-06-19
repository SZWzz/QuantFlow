"""PromptTemplate engine — assembles system prompts with tools and skills."""

import json
import logging
from dataclasses import dataclass, field
from typing import List, Optional

logger = logging.getLogger(__name__)

# Rough token budget constants (character-based heuristic: ~4 chars per token)
DEFAULT_MAX_CONTEXT_TOKENS = 128_000
TOKEN_BUDGET_SYSTEM_FRACTION = 0.3  # 30% for system prompt + skills + tools
MAX_SKILL_CHARS = 20_000  # Cap skill injection at ~5k tokens


@dataclass
class ToolDef:
    """A tool definition for LLM function calling."""
    name: str
    description: str
    parameters_json: str  # JSON Schema as string


@dataclass
class PromptContext:
    """All the pieces needed to assemble a system prompt."""
    base_system_prompt: str = ""
    tools: List[ToolDef] = field(default_factory=list)
    skills: List[str] = field(default_factory=list)  # Skill content strings
    few_shot_examples: List[str] = field(default_factory=list)
    model_context_window: int = DEFAULT_MAX_CONTEXT_TOKENS


def estimate_tokens(text: str) -> int:
    """Rough token estimate: ~4 characters per token for English text."""
    return max(1, len(text) // 4)


class PromptTemplate:
    """Assembles the final system prompt from components within token budget."""

    def __init__(self, context: PromptContext):
        self.context = context
        self.token_budget = int(context.model_context_window * TOKEN_BUDGET_SYSTEM_FRACTION)

    def assemble_system_prompt(self) -> str:
        """Build the system prompt, injecting skills and tools within budget."""
        parts = []
        tokens_used = 0

        # 1. Base system prompt (always included, up to half of budget)
        base = self.context.base_system_prompt
        base_tokens = estimate_tokens(base)
        half_budget = self.token_budget // 2
        if base_tokens > half_budget:
            logger.warning(f"Base system prompt ({base_tokens} tokens) exceeds half budget ({half_budget}), truncating")
            base = base[:half_budget * 4]
        parts.append(base)
        tokens_used += base_tokens

        # 2. Tool descriptions
        if self.context.tools:
            tool_block = "\n\n## Available Tools\n\nYou have access to the following tools:\n\n"
            for tool in self.context.tools:
                tool_block += f"### {tool.name}\n{tool.description}\n"
                params = json.loads(tool.parameters_json) if tool.parameters_json else {}
                if params.get("properties"):
                    tool_block += f"Parameters: {json.dumps(params['properties'], indent=2)}\n"
                tool_block += "\n"
            tool_tokens = estimate_tokens(tool_block)
            if tokens_used + tool_tokens <= self.token_budget:
                parts.append(tool_block)
                tokens_used += tool_tokens
            else:
                # Truncate: only include tool names
                short_block = "\n\n## Available Tools\n\n" + ", ".join(t.name for t in self.context.tools)
                parts.append(short_block)
                tokens_used += estimate_tokens(short_block)

        # 3. Skill knowledge (injected as domain expertise, NOT as tools)
        if self.context.skills:
            skill_block = "\n\n## Domain Knowledge\n\n"
            budget_remaining = min(MAX_SKILL_CHARS, (self.token_budget - tokens_used) * 4)
            for skill in self.context.skills:
                if len(skill_block) + len(skill) > budget_remaining:
                    skill_block += skill[:budget_remaining - len(skill_block)] + "\n[...truncated]\n"
                    break
                skill_block += skill + "\n\n"
            parts.append(skill_block)

        # 4. Few-shot examples
        if self.context.few_shot_examples:
            example_block = "\n\n## Examples\n\n" + "\n\n".join(self.context.few_shot_examples)
            example_tokens = estimate_tokens(example_block)
            if tokens_used + example_tokens <= self.token_budget:
                parts.append(example_block)

        return "\n".join(parts)

    def format_tools_for_request(self) -> list:
        """Format tools into the structure expected by gRPC LLMChatRequest."""
        from src.proto.llm_pb2 import LLMTool

        result = []
        for tool in self.context.tools:
            lt = LLMTool()
            lt.name = tool.name
            lt.description = tool.description
            lt.parameters_json = tool.parameters_json
            result.append(lt)
        return result

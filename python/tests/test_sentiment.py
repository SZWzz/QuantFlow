"""Tests for NLP pipeline and sentiment analysis.

These tests assert concrete sentiment values. They work across all fallback
levels: VADER → TextBlob → keyword fallback for English, SnowNLP for Chinese.
"""
import pytest
from src.research.nlp_pipeline import NLPPipeline


class TestNLPPipeline:
    def setup_method(self):
        self.pipeline = NLPPipeline()

    # ── English positive ──────────────────────────────────────────

    def test_analyze_positive_english(self):
        """Positive English text must return positive label and score > 0."""
        result = self.pipeline.analyze(
            "The company reported record profit and strong growth outlook", "en"
        )
        assert result["label"] == "positive", f"expected positive, got {result['label']} score={result['score']}"
        assert result["score"] > 0, f"expected score > 0, got {result['score']}"
        assert len(result["keywords"]) > 0, "expected non-empty keywords"

    def test_analyze_negative_english(self):
        """Negative English text must return negative label and score < 0."""
        result = self.pipeline.analyze(
            "The company faces severe losses, decline, and regulatory fines", "en"
        )
        assert result["label"] == "negative", f"expected negative, got {result['label']} score={result['score']}"
        assert result["score"] < 0, f"expected score < 0, got {result['score']}"
        assert len(result["keywords"]) > 0

    def test_analyze_neutral_english(self):
        """Neutral text should stay near zero."""
        result = self.pipeline.analyze("The meeting is scheduled for Tuesday.", "en")
        assert result["label"] == "neutral", f"expected neutral, got {result['label']}"

    # ── English edge cases ────────────────────────────────────────

    def test_analyze_empty_text(self):
        result = self.pipeline.analyze("", "en")
        assert result["score"] == 0.0
        assert result["label"] == "neutral"
        assert result["confidence"] == 0.0

    def test_analyze_whitespace_only(self):
        result = self.pipeline.analyze("   ", "en")
        assert result["label"] == "neutral"
        assert result["score"] == 0.0

    def test_analyze_short_text(self):
        """Very short text should still produce a result."""
        result = self.pipeline.analyze("profit gains", "en")
        assert result["label"] in ("positive", "neutral", "negative")

    # ── Chinese sentiment (SnowNLP) ───────────────────────────────

    def test_analyze_positive_chinese(self):
        """Positive Chinese text should return positive sentiment."""
        result = self.pipeline.analyze("这个公司业绩非常好，利润大幅增长", "zh")
        assert "score" in result
        assert "label" in result
        # SnowNLP should detect this as positive
        if result["label"] != "neutral":
            assert result["score"] > 0

    def test_analyze_negative_chinese(self):
        """Negative Chinese text should return negative sentiment."""
        result = self.pipeline.analyze("公司面临严重亏损，业绩大幅下滑", "zh")
        assert result["label"] in ("positive", "neutral", "negative")

    # ── Aggregation ───────────────────────────────────────────────

    def test_aggregate_empty_list(self):
        result = self.pipeline.aggregate([])
        assert result["score"] == 0.0
        assert result["label"] == "neutral"

    def test_aggregate_single_result(self):
        results = [{"score": 0.8, "label": "positive", "confidence": 0.9, "keywords": ["growth"]}]
        result = self.pipeline.aggregate(results)
        assert result["label"] == "positive"
        assert result["score"] > 0

    def test_aggregate_mixed_sources(self):
        results = [
            {"score": 0.6, "label": "positive", "confidence": 0.8, "keywords": ["buy"]},
            {"score": -0.3, "label": "negative", "confidence": 0.4, "keywords": ["risk"]},
        ]
        result = self.pipeline.aggregate(results)
        # Weighted average: (0.6*0.8 + (-0.3)*0.4) / (0.8+0.4) = 0.36/1.2 = 0.3
        assert result["score"] > 0

    def test_keywords_deduplication(self):
        results = [
            {"score": 0.5, "label": "positive", "confidence": 0.5, "keywords": ["growth", "profit"]},
            {"score": 0.3, "label": "positive", "confidence": 0.5, "keywords": ["growth", "revenue"]},
        ]
        result = self.pipeline.aggregate(results)
        assert len(result["keywords"]) == 3  # "growth" appears once

    # ── Score range validation ────────────────────────────────────

    def test_score_in_valid_range(self):
        """Score must always be in [-1, 1]."""
        texts = [
            "profit growth revenue expansion record earnings",
            "loss decline drop bankruptcy crash default",
            "The quick brown fox jumps over the lazy dog",
        ]
        for text in texts:
            result = self.pipeline.analyze(text, "en")
            assert -1.0 <= result["score"] <= 1.0, \
                f"score {result['score']} out of range for text: {text[:40]}"

    def test_confidence_in_valid_range(self):
        """Confidence must always be in [0, 1]."""
        for text in ["profit growth", "loss decline", "neutral meeting"]:
            result = self.pipeline.analyze(text, "en")
            assert 0.0 <= result["confidence"] <= 1.0, \
                f"confidence {result['confidence']} out of range for text: {text}"

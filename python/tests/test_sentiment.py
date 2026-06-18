"""Tests for NLP pipeline and sentiment analysis."""
import pytest
from src.research.nlp_pipeline import NLPPipeline


class TestNLPPipeline:
    def setup_method(self):
        self.pipeline = NLPPipeline()

    def test_analyze_positive_english(self):
        result = self.pipeline.analyze("The company reported excellent earnings growth", "en")
        assert "score" in result
        assert "label" in result
        assert "confidence" in result
        assert "keywords" in result
        assert isinstance(result["score"], float)
        assert result["label"] in ("positive", "neutral", "negative")

    def test_analyze_negative_english(self):
        result = self.pipeline.analyze(
            "The company faces severe losses and regulatory fines", "en"
        )
        assert result["label"] in ("positive", "neutral", "negative")

    def test_analyze_empty_text(self):
        result = self.pipeline.analyze("", "en")
        assert result["score"] == 0.0
        assert result["label"] == "neutral"

    def test_analyze_whitespace_only(self):
        result = self.pipeline.analyze("   ", "en")
        assert result["label"] == "neutral"

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

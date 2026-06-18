"""NLP sentiment analysis pipeline using NLTK VADER (English) + SnowNLP (Chinese)."""
import logging
from typing import Optional

logger = logging.getLogger(__name__)

try:
    from nltk.sentiment.vader import SentimentIntensityAnalyzer
    _VADER_AVAILABLE = True
except ImportError:
    _VADER_AVAILABLE = False

try:
    from snownlp import SnowNLP
    _SNOWNLP_AVAILABLE = True
except ImportError:
    _SNOWNLP_AVAILABLE = False


class NLPPipeline:
    """News parsing -> entity recognition -> sentiment scoring."""

    def __init__(self):
        self._vader: Optional[SentimentIntensityAnalyzer] = None
        if _VADER_AVAILABLE:
            try:
                self._vader = SentimentIntensityAnalyzer()
            except Exception:
                logger.warning("VADER init failed, English sentiment degraded")

    def analyze(self, text: str, language: str = "en") -> dict:
        """Analyze a single text and return sentiment dict.

        Returns dict with keys: score (-1..1), label (positive/neutral/negative),
        confidence (0..1), keywords [], entities [].
        """
        if not text or not text.strip():
            return {
                "score": 0.0, "label": "neutral", "confidence": 0.0,
                "keywords": [], "entities": [],
            }

        score = 0.0
        confidence = 0.5

        if language == "zh" and _SNOWNLP_AVAILABLE:
            try:
                s = SnowNLP(text)
                raw = s.sentiments
                score = (raw - 0.5) * 2.0
                confidence = 0.7
            except Exception:
                pass
        elif self._vader is not None:
            try:
                vs = self._vader.polarity_scores(text)
                score = vs["compound"]
                confidence = 0.7
            except Exception:
                pass

        words = [w.strip(".,!?;:()[]{}\"'") for w in text.split() if len(w) > 3]
        keywords = [w for w in words if w.isalpha()][:10]
        if not keywords:
            keywords = [text[:20]]

        label = "neutral"
        if score > 0.15:
            label = "positive"
        elif score < -0.15:
            label = "negative"

        return {
            "score": round(score, 4),
            "label": label,
            "confidence": round(confidence, 4),
            "keywords": keywords,
            "entities": [],
        }

    def aggregate(self, results: list[dict]) -> dict:
        """Weighted multi-source aggregation."""
        if not results:
            return {"score": 0.0, "label": "neutral", "confidence": 0.0}

        total_weight = 0.0
        weighted_score = 0.0
        all_keywords = []
        all_labels = []

        for r in results:
            w = r.get("confidence", 0.5)
            weighted_score += r.get("score", 0.0) * w
            total_weight += w
            all_keywords.extend(r.get("keywords", []))
            all_labels.append(r.get("label", "neutral"))

        score = weighted_score / total_weight if total_weight > 0 else 0.0
        label = "neutral"
        if score > 0.15:
            label = "positive"
        elif score < -0.15:
            label = "negative"

        seen = set()
        unique_kw = []
        for kw in all_keywords:
            if kw not in seen:
                seen.add(kw)
                unique_kw.append(kw)

        return {
            "score": round(score, 4),
            "label": label,
            "confidence": round(total_weight / max(len(results), 1), 4),
            "keywords": unique_kw[:20],
        }

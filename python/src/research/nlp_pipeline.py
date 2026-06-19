"""NLP sentiment analysis pipeline using NLTK VADER (English) + SnowNLP (Chinese).

Degrades gracefully: VADER < TextBlob < simple keyword fallback for English,
SnowNLP for Chinese. All dependencies are optional.
"""
import logging
import re
import socket
from typing import Optional

logger = logging.getLogger(__name__)

# ----- VADER (English, best) -----
try:
    from nltk.sentiment.vader import SentimentIntensityAnalyzer
    _VADER_AVAILABLE = True
except ImportError:
    _VADER_AVAILABLE = False

# ----- SnowNLP (Chinese) -----
try:
    from snownlp import SnowNLP
    _SNOWNLP_AVAILABLE = True
except ImportError:
    _SNOWNLP_AVAILABLE = False

# ----- TextBlob (English fallback) -----
try:
    from textblob import TextBlob
    _TEXTBLOB_AVAILABLE = True
except ImportError:
    _TEXTBLOB_AVAILABLE = False

# ----- Simple English word lists for keyword-based fallback -----
_POSITIVE_WORDS = {
    "profit", "growth", "gain", "positive", "strong", "upgrade", "beat",
    "exceed", "rally", "bull", "bullish", "outperform", "buy", "opportunity",
    "increase", "record", "expansion", "improved", "improving", "recovery",
    "dividend", "breakthrough", "innovation", "leading", "momentum",
}
_NEGATIVE_WORDS = {
    "loss", "decline", "drop", "negative", "weak", "downgrade", "miss",
    "fall", "bear", "bearish", "underperform", "sell", "risk", "warning",
    "decrease", "layoff", "lawsuit", "fine", "penalty", "investigation",
    "bankruptcy", "default", "crash", "plunge", "downturn", "recession",
}

# Common threshold for label classification (aligned with Go sentimentToSignal).
_LABEL_THRESHOLD = 0.15

# Lazy, one-time VADER readiness check (module-level to avoid repeated network hangs).
_vader_ready: Optional[bool] = None


def _is_vader_ready() -> bool:
    """Check if vader_lexicon is available. Tries to download once with short timeout."""
    global _vader_ready
    if _vader_ready is not None:
        return _vader_ready
    if not _VADER_AVAILABLE:
        _vader_ready = False
        return False
    try:
        import nltk
        # socket timeout unreliable on Windows; use thread join timeout instead
        try:
            nltk.data.find('sentiment/vader_lexicon.zip')
            _vader_ready = True
        except LookupError:
            try:
                logger.info("Downloading vader_lexicon (one-time, 5s timeout)...")
                import threading
                result = {'done': False, 'error': None}
                def _dl():
                    try:
                        nltk.download('vader_lexicon', quiet=True)
                        result['done'] = True
                    except Exception as e:
                        result['error'] = e
                t = threading.Thread(target=_dl, daemon=True)
                t.start()
                t.join(timeout=3)
                if not result['done']:
                    if result['error']:
                        raise result['error']
                    raise TimeoutError('vader download timed out')
                nltk.data.find('sentiment/vader_lexicon.zip')
                _vader_ready = True
            except Exception as e:
                logger.warning(
                    "Failed to download vader_lexicon: %s. English sentiment uses fallback.", e
                )
                _vader_ready = False

    except Exception as e:
        logger.warning("VADER check failed: %s", e)
        _vader_ready = False
    return _vader_ready


def _simple_english_score(text: str) -> float:
    """Keyword-based English sentiment fallback. Returns score in [-1, 1]."""
    words = set(re.findall(r'\b[a-zA-Z]{3,}\b', text.lower()))
    pos = len(words & _POSITIVE_WORDS)
    neg = len(words & _NEGATIVE_WORDS)
    total = pos + neg
    if total == 0:
        return 0.0
    return round((pos - neg) / total, 4)


class NLPPipeline:
    """News parsing -> entity recognition -> sentiment scoring.

    English: VADER > TextBlob > keyword fallback
    Chinese: SnowNLP
    """

    def __init__(self):
        self._vader: Optional[SentimentIntensityAnalyzer] = None
        self._vader_ready = False
        self._textblob_ready = _TEXTBLOB_AVAILABLE
        self._snownlp_ready = _SNOWNLP_AVAILABLE

        if _is_vader_ready():
            try:
                self._vader = SentimentIntensityAnalyzer()
                self._vader_ready = True
            except Exception:
                logger.warning("VADER init failed, English sentiment degraded")

        if not self._vader_ready and not self._textblob_ready:
            logger.warning(
                "No English sentiment library available (VADER/TextBlob missing). "
                "Using simple keyword fallback."
            )

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
        engine = "none"

        if language == "zh" and self._snownlp_ready:
            try:
                s = SnowNLP(text)
                raw = s.sentiments
                score = (raw - 0.5) * 2.0
                confidence = 0.7
                engine = "snownlp"
            except Exception:
                logger.debug("SnowNLP analysis failed", exc_info=True)
        elif language == "zh":
            # Chinese text without SnowNLP — tokenize by characters as fallback
            logger.debug("SnowNLP unavailable, Chinese sentiment returns neutral")
        elif self._vader_ready and self._vader is not None:
            try:
                vs = self._vader.polarity_scores(text)
                score = vs["compound"]
                confidence = 0.75
                engine = "vader"
            except Exception:
                logger.debug("VADER analysis failed", exc_info=True)
        elif self._textblob_ready:
            try:
                tb = TextBlob(text)
                # TextBlob polarity is [-1, 1], subject is [0, 1]
                score = tb.sentiment.polarity
                confidence = 0.6
                engine = "textblob"
            except Exception:
                logger.debug("TextBlob analysis failed", exc_info=True)
        else:
            # Final fallback: simple keyword matching (no external deps needed)
            score = _simple_english_score(text)
            confidence = 0.3
            engine = "keyword"

        # Extract keywords from text
        words = [w.strip(".,!?;:()[]{}\"'") for w in text.split() if len(w) > 3]
        keywords = [w for w in words if w.isalpha()][:10]
        if not keywords:
            keywords = [text[:20]]

        label = "neutral"
        if score > _LABEL_THRESHOLD:
            label = "positive"
        elif score < -_LABEL_THRESHOLD:
            label = "negative"

        logger.debug(
            "Sentiment analyzed engine=%s lang=%s score=%.4f label=%s words=%d",
            engine, language, score, label, len(words),
        )

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

        for r in results:
            w = r.get("confidence", 0.5)
            weighted_score += r.get("score", 0.0) * w
            total_weight += w
            all_keywords.extend(r.get("keywords", []))

        score = weighted_score / total_weight if total_weight > 0 else 0.0
        label = "neutral"
        if score > _LABEL_THRESHOLD:
            label = "positive"
        elif score < -_LABEL_THRESHOLD:
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

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

# ----- jieba (Chinese word segmentation) -----
try:
    import jieba
    _JIEBA_AVAILABLE = True
except ImportError:
    _JIEBA_AVAILABLE = False

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

# Common Chinese stopwords to filter from keywords.
_CN_STOPWORDS = {
    "的", "了", "在", "是", "我", "有", "和", "就", "不", "人", "都", "一",
    "一个", "上", "也", "很", "到", "说", "要", "去", "你", "会", "着",
    "没有", "看", "好", "自己", "这", "他", "她", "它", "们", "那", "些",
    "什么", "怎么", "如何", "为什么", "可以", "这个", "那个", "已经",
    "因为", "所以", "但是", "如果", "虽然", "而且", "以及", "或者",
    "从", "对", "与", "对于", "关于", "通过", "被", "把", "将",
    "等", "其", "之", "所", "为", "以", "年", "月", "日", "时",
    "元", "万", "亿", "只", "股", "股票", "公司", "市场", "投资",
    "投资者", "资金", "交易", "价格", "走势", "行情", "板块", "指数",
    "主力", "机构", "散户", "涨停", "跌停", "上涨", "下跌",
}

def _is_stopword(word: str) -> bool:
    """Check if a word is a stopword."""
    return word in _CN_STOPWORDS or len(word) <= 1


def _is_numeric_token(word: str) -> bool:
    """Check if a token is purely numeric (should not be a keyword)."""
    if not word:
        return True
    # Strip common punctuation from boundaries
    w = word.strip(".,:;%+-")
    if not w:
        return True
    # Check if it's a number (int or float)
    try:
        float(w)
        return True
    except ValueError:
        pass
    # Check if it contains mostly digits (e.g. "2.78", "515.32")
    digit_count = sum(1 for c in w if c.isdigit() or c in ".-")
    if len(w) > 0 and digit_count / len(w) > 0.5:
        return True
    return False

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
                # Dampen extreme scores: SnowNLP is trained on product reviews,
                # not financial news, so 0.0/1.0 outputs are unreliable.
                if raw <= 0.05 or raw >= 0.95:
                    raw = 0.5 + (raw - 0.5) * 0.3  # pull toward neutral
                    confidence = 0.35  # low: extreme output from out-of-domain model
                elif raw <= 0.15 or raw >= 0.85:
                    confidence = 0.5  # moderate: somewhat confident
                else:
                    confidence = 0.65  # reasonable: within typical range
                score = (raw - 0.5) * 2.0
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

        # Confidence penalty for very short text (likely noise, not meaningful news)
        text_len = len(text.strip())
        if text_len < 50:
            confidence *= 0.5
        elif text_len < 100:
            confidence *= 0.8

        # Extract keywords from text (filter numerics to avoid garbage like "2.78")
        if language == "zh" and _JIEBA_AVAILABLE:
            words = jieba.lcut(text)
            keywords = [w for w in words if len(w) >= 2 and not _is_stopword(w) and not _is_numeric_token(w)][:10]
        else:
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

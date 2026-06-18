"""gRPC SentimentService — delegates to NLPPipeline."""
import asyncio
import logging
import time

from src.proto import sentiment_pb2, sentiment_pb2_grpc
from src.research.nlp_pipeline import NLPPipeline

logger = logging.getLogger(__name__)


class SentimentService(sentiment_pb2_grpc.SentimentServiceServicer):
    """gRPC service for sentiment analysis."""

    def __init__(self):
        self.pipeline = NLPPipeline()

    async def AnalyzeSentiment(self, request, context):
        t0 = time.time()
        symbol = request.symbol
        text = request.text_content
        language = request.language or "en"
        text_type = request.text_type or "news"

        try:
            if text:
                result = self.pipeline.analyze(text, language)
                results = [sentiment_pb2.SentimentResult(
                    score=result["score"],
                    label=result["label"],
                    confidence=result["confidence"],
                    keywords=result["keywords"],
                    entities=result.get("entities", []),
                    source=text_type,
                )]
            else:
                results = [sentiment_pb2.SentimentResult(
                    score=0.0, label="neutral", confidence=0.0,
                    keywords=[], entities=[], source=text_type,
                )]

            overall = self.pipeline.aggregate([
                {"score": r.score, "label": r.label, "confidence": r.confidence,
                 "keywords": list(r.keywords)}
                for r in results
            ])

            elapsed_ms = round((time.time() - t0) * 1000, 2)
            return sentiment_pb2.AnalyzeSentimentResponse(
                symbol=symbol,
                overall_score=overall["score"],
                overall_label=overall["label"],
                results=results,
                compute_time_ms=elapsed_ms,
            )
        except Exception as e:
            logger.exception("AnalyzeSentiment failed for %s", symbol)
            return sentiment_pb2.AnalyzeSentimentResponse(
                symbol=symbol,
                error=str(e),
            )

    async def BatchAnalyzeSentiment(self, request, context):
        try:
            tasks = []
            for symbol in request.symbols:
                req = sentiment_pb2.AnalyzeSentimentRequest(
                    symbol=symbol,
                    text_type=request.text_type or "news",
                    language=request.language or "en",
                )
                tasks.append(self.AnalyzeSentiment(req, context))
            responses = await asyncio.gather(*tasks, return_exceptions=True)
            results = []
            for r in responses:
                if isinstance(r, Exception):
                    results.append(sentiment_pb2.AnalyzeSentimentResponse(
                        error=str(r)))
                else:
                    results.append(r)
            return sentiment_pb2.BatchAnalyzeResponse(responses=results)
        except Exception as e:
            logger.exception("BatchAnalyzeSentiment failed")
            return sentiment_pb2.BatchAnalyzeResponse(error=str(e))

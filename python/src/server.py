"""QuantFlow Python gRPC Sidecar — main entry point.

Start the server:
    python -m src.server

Or with a custom port:
    python -m src.server --port 50052

The Go backend connects via gRPC on localhost:<port>.
Communication is unauthenticated (localhost-only).
"""

import os
# Enable gRPC fork support to suppress "skipping fork() handlers" warnings
# when subprocess is spawned from within a gRPC server process.
os.environ["GRPC_ENABLE_FORK_SUPPORT"] = "1"

from importlib.metadata import version as _pkg_version
try:
    __version__ = _pkg_version("quantflow-python")
except Exception:
    __version__ = "0.0.0"

import asyncio
import logging
import signal
import time
import argparse
from concurrent import futures

import grpc

from src.proto import (
    factor_pb2_grpc,
    health_pb2,
    health_pb2_grpc,
    llm_pb2_grpc,
    ml_pb2_grpc,
    data_pb2_grpc,
    sentiment_pb2_grpc,  # NEW
)
from src.factor.engine import FactorService
from src.ml.engine import MLService
from src.data.fetcher import DataService
from src.llm.engine import LLMService
from src.research.sentiment_service import SentimentService  # NEW
from src.health.health_server import HealthServer

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)

DEFAULT_PORT = 50051
DEFAULT_HEALTH_PORT = 50052


class HealthService(health_pb2_grpc.HealthServiceServicer):
    """Health check service for liveness and status queries."""

    def __init__(self):
        self.start_time = time.time()

    async def Ping(self, request, context):
        uptime = int(time.time() - self.start_time)
        return health_pb2.PingResponse(
            healthy=True,
            version=__version__,
            uptime_seconds=uptime,
        )

    async def GetStatus(self, request, context):
        mem_mb = 0
        try:
            import resource
            usage = resource.getrusage(resource.RUSAGE_SELF)
            mem_mb = usage.ru_maxrss // (1024 * 1024)  # macOS returns bytes
        except Exception:
            pass

        uptime = int(time.time() - self.start_time)
        return health_pb2.StatusResponse(
            healthy=True,
            version=__version__,
            uptime_seconds=uptime,
            active_requests=0,
            memory_mb=mem_mb,
        )


async def shutdown_coro(server, stop_event):
    """Handle shutdown signal — stop the server gracefully."""
    logger.info("Shutdown signal received, stopping gracefully...")
    stop_event.set()


async def serve(port: int = DEFAULT_PORT, health_port: int = DEFAULT_HEALTH_PORT, max_workers: int = 10):
    """Start the gRPC server and block until termination."""
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=max_workers))

    # Register all service implementations
    factor_pb2_grpc.add_FactorServiceServicer_to_server(FactorService(), server)
    ml_pb2_grpc.add_MLServiceServicer_to_server(MLService(), server)
    health_pb2_grpc.add_HealthServiceServicer_to_server(HealthService(), server)
    data_pb2_grpc.add_DataServiceServicer_to_server(DataService(), server)
    llm_pb2_grpc.add_LLMServiceServicer_to_server(LLMService(), server)
    sentiment_pb2_grpc.add_SentimentServiceServicer_to_server(SentimentService(), server)

    server.add_insecure_port(f"localhost:{port}")
    logger.info(f"QuantFlow Python sidecar listening on localhost:{port}")
    logger.info("Registered services: FactorService, MLService, HealthService, DataService, LLMService, SentimentService")

    await server.start()

    # Start standard gRPC health checking protocol server (GRPC-101)
    health_server = HealthServer(port=health_port)
    await health_server.start()
    logger.info(f"Health check server (GRPC-101) listening on 0.0.0.0:{health_port}")

    # Set up signal handling for graceful shutdown
    loop = asyncio.get_event_loop()
    stop_event = asyncio.Event()

    for sig in (signal.SIGTERM, signal.SIGINT):
        try:
            loop.add_signal_handler(
                sig,
                lambda: asyncio.create_task(shutdown_coro(server, stop_event))
            )
        except NotImplementedError:
            # Windows or restricted platforms
            pass

    try:
        await stop_event.wait()
    finally:
        await health_server.close()
        await server.stop(5)


def main():
    parser = argparse.ArgumentParser(description="QuantFlow Python gRPC Sidecar")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help=f"Port to listen on (default: {DEFAULT_PORT})")
    parser.add_argument("--health-port", type=int, default=DEFAULT_HEALTH_PORT, help=f"Health check port (default: {DEFAULT_HEALTH_PORT})")
    parser.add_argument("--workers", type=int, default=10, help="Max thread pool workers (default: 10)")
    args = parser.parse_args()

    asyncio.run(serve(port=args.port, health_port=args.health_port, max_workers=args.workers))


if __name__ == "__main__":
    main()

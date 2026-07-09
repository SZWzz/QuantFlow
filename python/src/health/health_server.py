"""
Health gRPC service for QuantFlow Python sidecar.

Implements the standard gRPC Health Checking Protocol (GRPC-101) so
that the Go app can poll the Python sidecar's readiness via a single
Unary RPC. Uses the official grpc_health.v1 package.
"""

import asyncio
from concurrent import futures
from dataclasses import dataclass, field
from typing import Dict

import grpc
from grpc import aio

try:
    from grpc_health.v1 import health_pb2, health_pb2_grpc
    _HAS_GRPC_HEALTH = True
except ImportError:
    _HAS_GRPC_HEALTH = False

if not _HAS_GRPC_HEALTH:

    class HealthServicer:  # type: ignore
        def __init__(self):
            raise RuntimeError("grpcio-health-checking package not installed. Run: pip install grpcio-health-checking")

    @dataclass
    class HealthServer:  # type: ignore
        port: int = 50052
        _server: aio.Server = field(default=None, init=False, repr=False)
        _servicer: HealthServicer = field(default=None, init=False, repr=False)

        async def start(self) -> None:
            raise RuntimeError("grpcio-health-checking package not installed")
else:
    SERVING = health_pb2.HealthCheckResponse.SERVING
    NOT_SERVING = health_pb2.HealthCheckResponse.NOT_SERVING
    UNKNOWN = health_pb2.HealthCheckResponse.UNKNOWN

    _status_int = int

    class HealthServicer(health_pb2_grpc.HealthServicer):
        def __init__(self):
            self._services: Dict[str, _status_int] = {
                "": SERVING,
                "quantflow.data": SERVING,
                "quantflow.ml": SERVING,
                "quantflow.llm": SERVING,
            }

        def set_status(self, service: str, status: int) -> None:
            self._services[service] = status

        async def Check(self, request, context):
            status = self._services.get(request.service, UNKNOWN)
            return health_pb2.HealthCheckResponse(status=status)

        async def Watch(self, request, context):
            status = self._services.get(request.service, UNKNOWN)
            yield health_pb2.HealthCheckResponse(status=status)

    @dataclass
    class HealthServer:
        port: int = 50052
        _server: aio.Server = field(default=None, init=False, repr=False)
        _servicer: HealthServicer = field(default=None, init=False, repr=False)

        async def start(self) -> None:
            self._servicer = HealthServicer()
            self._server = aio.server(futures.ThreadPoolExecutor(max_workers=1))
            health_pb2_grpc.add_HealthServicer_to_server(self._servicer, self._server)
            self._server.add_insecure_port(f"0.0.0.0:{self.port}")
            await self._server.start()

        async def stop(self, grace: float = 5) -> None:
            if self._server:
                await self._server.stop(grace)

        async def __aenter__(self):
            await self.start()
            return self

        async def __aexit__(self, *args):
            await self.stop()

        def __del__(self):
            if self._server:
                try:
                    loop = asyncio.get_event_loop()
                    if loop.is_running():
                        loop.create_task(self.stop(0))
                    else:
                        loop.run_until_complete(self.stop(0))
                except Exception:
                    pass

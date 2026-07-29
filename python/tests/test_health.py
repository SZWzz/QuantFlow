import pytest
pytest.importorskip("grpc_health", reason="grpcio-health-checking not installed")

import grpc
from grpc_health.v1 import health_pb2, health_pb2_grpc


@pytest.mark.asyncio
async def test_health_server_constructs():
    from src.health.health_server import HealthServer

    server = HealthServer(port=50551)
    assert server.port == 50551
    assert server._server is None


@pytest.mark.asyncio
async def test_health_server_start_stop():
    from src.health.health_server import HealthServer

    server = HealthServer(port=50552)
    try:
        await server.start()
        assert server._server is not None
    finally:
        await server.stop(0)


@pytest.mark.asyncio
async def test_health_check_responds_serving():
    from src.health.health_server import HealthServer

    server = HealthServer(port=50553)
    try:
        await server.start()
        channel = grpc.aio.insecure_channel("localhost:50553")
        stub = health_pb2_grpc.HealthStub(channel)

        resp = await stub.Check(health_pb2.HealthCheckRequest(service=""))
        assert resp.status == health_pb2.HealthCheckResponse.SERVING

        resp2 = await stub.Check(
            health_pb2.HealthCheckRequest(service="quantflow.data")
        )
        assert resp2.status == health_pb2.HealthCheckResponse.SERVING

        await channel.close()
    finally:
        await server.stop(0)


@pytest.mark.asyncio
async def test_health_check_unknown_service():
    from src.health.health_server import HealthServer

    server = HealthServer(port=50554)
    try:
        await server.start()
        channel = grpc.aio.insecure_channel("localhost:50554")
        stub = health_pb2_grpc.HealthStub(channel)

        resp = await stub.Check(
            health_pb2.HealthCheckRequest(service="nonexistent.service")
        )
        assert resp.status == health_pb2.HealthCheckResponse.UNKNOWN

        await channel.close()
    finally:
        await server.stop(0)


@pytest.mark.asyncio
async def test_set_status():
    from src.health.health_server import HealthServer

    server = HealthServer(port=50555)
    try:
        await server.start()
        channel = grpc.aio.insecure_channel("localhost:50555")
        stub = health_pb2_grpc.HealthStub(channel)

        server._servicer.set_status(
            "quantflow.data", health_pb2.HealthCheckResponse.NOT_SERVING
        )
        resp = await stub.Check(
            health_pb2.HealthCheckRequest(service="quantflow.data")
        )
        assert resp.status == health_pb2.HealthCheckResponse.NOT_SERVING

        await channel.close()
    finally:
        await server.stop(0)

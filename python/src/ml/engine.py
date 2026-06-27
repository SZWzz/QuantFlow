"""MLService gRPC implementation — routes to sub-engines."""
import os
import time
import logging
import uuid
from typing import AsyncIterator

import grpc
import numpy as np
import pyarrow as pa

from src.proto import ml_pb2, ml_pb2_grpc
from src.ml.tree_engine import TreeEngine
from src.ml.deep_engine import DeepEngine
from src.ml.serialization import MODEL_DIR
from src.ml.alpha_mining.genetic import AlphaMiningEngine

logger = logging.getLogger(__name__)


class MLService(ml_pb2_grpc.MLServiceServicer):
    """gRPC service for ML model training and inference."""

    def __init__(self):
        self._tree_engine = TreeEngine()
        self._deep_engine = None  # Lazy init to avoid torch import on startup
        self._models = {}  # model_id -> {"path": str, "type": str}
        self.rl_models = {}  # model_id -> trained RL trainer instance

    def _get_deep_engine(self):
        if self._deep_engine is None:
            self._deep_engine = DeepEngine()
        return self._deep_engine

    def _decode_arrow(self, data: bytes) -> pa.Table:
        reader = pa.ipc.open_stream(data)
        return reader.read_all()

    async def Train(self, request, context):
        try:
            features = self._decode_arrow(request.features)
            targets = self._decode_arrow(request.targets)
            params = dict(request.hyperparams)
            params["model_type"] = request.model_type
            params["target_type"] = request.target_type
            params["model_dir"] = MODEL_DIR

            model_type = request.model_type
            if model_type in ("xgboost", "lightgbm"):
                result = self._tree_engine.train(features, targets, params)
            elif model_type in ("lstm", "transformer"):
                engine = self._get_deep_engine()
                result = engine.train(features, targets, params)
            else:
                raise ValueError(f"unsupported model_type: {model_type}")

            model_id = str(uuid.uuid4())
            self._models[model_id] = {"path": result["model_path"], "type": model_type}

            with open(result["model_path"], "rb") as f:
                model_bytes = f.read()

            resp = ml_pb2.TrainResponse(
                model_id=model_id,
                model_bytes=model_bytes,
                model_file_path=result["model_path"],
                train_time_ms=result["train_time_ms"],
            )
            for k, v in result["metrics"].items():
                resp.metrics[k] = v

            # Feature importance (TreeEngine only)
            if model_type in ("xgboost", "lightgbm"):
                fi = self._tree_engine.feature_importance(result["model_path"], features)
                for f in fi:
                    resp.feature_importance.append(ml_pb2.FeatureImportance(
                        feature_name=f["feature"], importance=f["importance"]
                    ))

            return resp
        except ValueError as e:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(e))
            raise
        except KeyError as e:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details(str(e))
            raise
        except TimeoutError as e:
            context.set_code(grpc.StatusCode.DEADLINE_EXCEEDED)
            context.set_details(str(e))
            raise
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            raise

    async def Predict(self, request, context):
        try:
            model_info = self._models.get(request.model_id)
            if not model_info:
                raise KeyError(f"model not found: {request.model_id}")

            features = self._decode_arrow(request.features)

            import time
            start = time.time()
            if model_info["type"] in ("xgboost", "lightgbm"):
                preds = self._tree_engine.predict(model_info["path"], features)
            elif model_info["type"] in ("lstm", "transformer"):
                engine = self._get_deep_engine()
                preds = engine.predict(model_info["path"], features)
            else:
                raise ValueError(f"unknown model type: {model_info['type']}")

            elapsed = int((time.time() - start) * 1000)
            return ml_pb2.PredictResponse(
                predictions=list(preds.to_pylist()),
                predict_time_ms=elapsed,
            )
        except ValueError as e:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(e))
            raise
        except KeyError as e:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details(str(e))
            raise
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            raise

    async def Evaluate(self, request, context):
        try:
            model_info = self._models.get(request.model_id)
            if not model_info:
                raise KeyError(f"model not found: {request.model_id}")

            features = self._decode_arrow(request.features)
            actuals = self._decode_arrow(request.actuals)

            import time
            start = time.time()

            if model_info["type"] in ("xgboost", "lightgbm"):
                metrics = self._tree_engine.evaluate(model_info["path"], features, actuals)
            else:
                raise ValueError("evaluate not yet supported for deep models")

            elapsed = int((time.time() - start) * 1000)
            resp = ml_pb2.EvaluateResponse(evaluate_time_ms=elapsed)
            for k, v in metrics.items():
                resp.metrics[k] = v
            return resp
        except ValueError as e:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(e))
            raise
        except KeyError as e:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details(str(e))
            raise
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            raise

    async def AlphaMining(self, request, context):
        """Run genetic programming to discover predictive alpha factors."""
        try:
            start = time.time()

            factor_data = self._decode_arrow(request.factor_data)
            returns_data = self._decode_arrow(request.returns_data)

            params = {
                "population_size": str(request.population_size or 200),
                "generations": str(request.generations or 50),
                "top_k": str(request.top_k or 20),
                "crossover_rate": str(request.crossover_rate or 0.7),
                "mutation_rate": str(request.mutation_rate or 0.1),
                "fitness_metric": request.fitness_metric or "ic",
            }

            engine = AlphaMiningEngine()
            results = engine.evolve(factor_data, returns_data, params)

            elapsed = int((time.time() - start) * 1000)

            resp = ml_pb2.AlphaMiningResponse(mining_time_ms=elapsed)
            for r in results:
                resp.factors.append(ml_pb2.DiscoveredFactor(
                    formula=r["formula"],
                    ic=r["ic"],
                    ir=r["ir"],
                    sharpe=r["sharpe"],
                ))

            return resp
        except ValueError as e:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(e))
            raise
        except KeyError as e:
            context.set_code(grpc.StatusCode.NOT_FOUND)
            context.set_details(str(e))
            raise
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            raise

    async def RLTrain(self, request, context) -> AsyncIterator[ml_pb2.RLTrainUpdate]:
        """Phase 10.3: Stream RL training progress per episode."""
        try:
            ohlcv = self._decode_arrow(request.ohlcv_data)
            ohlcv_np = ohlcv.to_pandas().values.astype(np.float32)

            from src.ml.rl.env import TradingEnv

            window_size = int(request.hyperparams.get("window_size", "20"))
            action_space = request.action_space or "discrete"
            env = TradingEnv(ohlcv_np, window_size=window_size, action_type=action_space)

            algorithm = request.algorithm
            if algorithm == "ppo":
                from src.ml.rl.algorithms.ppo import PPOTrainer
                trainer = PPOTrainer(
                    state_dim=int(env.observation_space.shape[0]),
                    action_dim=int(env.action_space.n),
                )
            elif algorithm == "dqn":
                from src.ml.rl.algorithms.dqn import DQNTrainer
                trainer = DQNTrainer(
                    state_dim=int(env.observation_space.shape[0]),
                    action_dim=int(env.action_space.n),
                )
            elif algorithm == "sac":
                from src.ml.rl.algorithms.sac import SACTrainer
                trainer = SACTrainer(
                    state_dim=int(env.observation_space.shape[0]),
                    action_dim=1 if action_space == "continuous" else 3,
                )
            else:
                raise ValueError(f"unsupported RL algorithm: {algorithm}")

            total = request.total_episodes or 100
            for ep in range(total):
                result = trainer.train_episode(env)
                env.reset()
                yield ml_pb2.RLTrainUpdate(
                    episode=result["episode"],
                    reward=float(result["reward"]),
                    sharpe=float(result["sharpe"]),
                    steps=int(result["steps"]),
                    epsilon=float(getattr(trainer, "epsilon", 0.0)),
                )

            # Store trained model for later RLPredict inference
            self.rl_models[algorithm] = trainer
        except ImportError as e:
            logger.warning("RLTrain: %s", e)
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details(str(e))
            return
        except Exception as e:
            logger.exception("RLTrain failed")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return

    async def RLPredict(self, request, context):
        """Predict action using a trained RL model."""
        try:
            model_id = request.model_id
            if not model_id or model_id not in self.rl_models:
                context.set_code(grpc.StatusCode.NOT_FOUND)
                context.set_details(f"Model '{model_id}' not loaded")
                return ml_pb2.RLPredictResponse(action=1, action_value=0.0)

            model = self.rl_models[model_id]
            obs = np.array(request.observation).reshape(1, -1).astype(np.float32)
            action, _ = model.predict(obs, deterministic=True)

            # action is typically numpy array of shape (1,)
            act_val = int(np.clip(action[0], 0, 2)) if isinstance(action, np.ndarray) else 1
            return ml_pb2.RLPredictResponse(action=act_val, action_value=0.8)
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            # Fallback: hold
            return ml_pb2.RLPredictResponse(action=1, action_value=0.0)

    async def RiskModel(self, request, context):
        """Phase 10.4: Compute risk model (GARCH or covariance)."""
        try:
            returns = self._decode_arrow(request.returns_data)

            if request.model_type in ("garch", "gjr_garch", "egarch"):
                from src.ml.risk.garch import GARCHEngine
                engine = GARCHEngine()
                result = engine.fit(returns, {
                    "model_type": request.model_type,
                    "p": request.params.get("p", "1"),
                    "q": request.params.get("q", "1"),
                })
            else:
                from src.ml.risk.covariance import CovarianceEngine
                engine = CovarianceEngine()
                result = engine.estimate(returns, {
                    "method": request.params.get("method", "ledoit_wolf"),
                })

            import time as _time

            # Convert result dict to Arrow
            rows = {}
            for k, v in result.items():
                if isinstance(v, list):
                    rows[k] = v
            if rows:
                import pandas as pd
                result_table = pa.Table.from_pandas(
                    pd.DataFrame({k: v for k, v in rows.items() if len(rows)})
                )
                sink = pa.BufferOutputStream()
                with pa.ipc.new_stream(sink, result_table.schema) as w:
                    w.write_table(result_table)
                result_data = sink.getvalue().to_pybytes()
            else:
                result_data = b""

            resp = ml_pb2.RiskModelResponse(result_data=result_data)
            for k, v in result.items():
                if isinstance(v, (int, float)):
                    resp.metrics[k] = float(v)
            return resp
        except ImportError as e:
            logger.warning("RiskModel: %s", e)
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details(str(e))
            return ml_pb2.RiskModelResponse()
        except Exception as e:
            logger.exception("RiskModel failed")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return ml_pb2.RiskModelResponse()

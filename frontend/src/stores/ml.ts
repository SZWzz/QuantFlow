import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface MLModel {
  id: string
  name: string
  model_type: string
  category: string
  hyperparams: Record<string, string>
  metrics: Record<string, number>
  file_path: string
  status: 'training' | 'ready' | 'failed' | 'archived'
  created_at: string
  updated_at: string
}

export interface TrainingJob {
  id: string
  model_id: string
  model_type: string
  status: 'running' | 'completed' | 'failed'
  progress: number
  started_at: string
}

export interface Prediction {
  id: string
  model_id: string
  symbol: string
  date: string
  prediction: number
  actual: number | null
}

export interface DiscoveredFactor {
  formula: string
  ic: number
  ir: number
  sharpe: number
}

export interface RLTrainUpdate {
  episode: number
  reward: number
  sharpe: number
  steps: number
  epsilon: number
}

export interface RiskModelResult {
  model_type: string
  volatility?: number[]
  covariance?: number[][]
  aic?: number
  bic?: number
}

export const useMLStore = defineStore('ml', () => {
  const models = ref<MLModel[]>([])
  const selectedModel = ref<MLModel | null>(null)
  const trainingJobs = ref<TrainingJob[]>([])
  const trainingProgress = ref<Record<string, number>>({})
  const predictions = ref<Prediction[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const discoveredFactors = ref<DiscoveredFactor[]>([])
  const miningRunning = ref(false)

  // Phase 10.3: RL Training
  const rlTrainingEpisodes = ref<RLTrainUpdate[]>([])
  const rlTrainingRunning = ref(false)
  const rlAlgorithm = ref<string>('ppo')

  // Phase 10.4: Risk Modeling
  const riskModelResult = ref<RiskModelResult | null>(null)

  const readyModels = computed(() => models.value.filter(m => m.status === 'ready'))
  const predictionModels = computed(() => models.value.filter(m => m.category === 'prediction'))

  async function fetchModels() {
    loading.value = true
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      if (app.ListMLModels) {
        models.value = (await app.ListMLModels()) || []
      } else {
        models.value = []
      }
    } catch (e) {
      error.value = String(e)
    } finally {
      loading.value = false
    }
  }

  async function archiveModel(id: string) {
    // ArchiveMLModel not yet implemented in Go backend
  }

  async function deleteModel(id: string) {
    // DeleteMLModel not yet implemented in Go backend
  }

  function selectModel(model: MLModel | null) {
    selectedModel.value = model
  }

  async function fetchPredictions(modelId: string, symbol: string) {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      if (app.GetPredictions) {
        predictions.value = (await app.GetPredictions(modelId, symbol)) || []
      } else {
        predictions.value = []
      }
    } catch (e) {
      error.value = String(e)
    }
  }

  async function fetchEvaluations(modelId: string) {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      // GetEvaluations not yet implemented in Go backend
    } catch (e) {
      error.value = String(e)
    }
  }

  async function runAlphaMining(params: {
    factorNames: string[]; factorData: any; returnsData: any;
    populationSize: number; generations: number; topK: number;
  }) {
    miningRunning.value = true
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app?.RunAlphaMining) { error.value = 'Backend not available'; miningRunning.value = false; return }
      discoveredFactors.value = (await app.RunAlphaMining({
        factor_names: params.factorNames,
        population_size: params.populationSize,
        generations: params.generations,
        crossover_rate: 0.7,
        mutation_rate: 0.1,
        top_k: params.topK,
        fitness_metric: 'ic',
      })) || []
    } catch (e) {
      error.value = String(e)
    } finally {
      miningRunning.value = false
    }
  }

  // Phase 10.3: RL Training actions
  function startRLTraining(algorithm: string) {
    rlAlgorithm.value = algorithm
    rlTrainingRunning.value = true
    rlTrainingEpisodes.value = []
  }

  function addRLUpdate(update: RLTrainUpdate) {
    rlTrainingEpisodes.value.push(update)
  }

  function stopRLTraining() {
    rlTrainingRunning.value = false
  }

  async function trainRL(algorithm: string) {
    error.value = null
    rlTrainingEpisodes.value = []
    rlTrainingRunning.value = true
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      // TrainRL not yet implemented in Go backend
      startRLTraining(algorithm)
    } catch (e) {
      error.value = String(e)
      rlTrainingRunning.value = false
    }
  }

  // Phase 10.4: Risk Modeling actions
  function setRiskModelResult(result: RiskModelResult | null) {
    riskModelResult.value = result
  }

  async function assessRisk(symbols: string[], modelType: string) {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      // AssessRisk not yet implemented in Go backend
      riskModelResult.value = null
    } catch (e) {
      error.value = String(e)
    }
  }

  return {
    models, selectedModel, trainingJobs, trainingProgress, predictions, loading,
    readyModels, predictionModels, discoveredFactors, miningRunning, error,
    fetchModels, archiveModel, deleteModel, selectModel, fetchPredictions, fetchEvaluations, runAlphaMining,
    rlTrainingEpisodes, rlTrainingRunning, rlAlgorithm,
    trainRL, startRLTraining, addRLUpdate, stopRLTraining,
    riskModelResult, assessRisk, setRiskModelResult,
  }
})

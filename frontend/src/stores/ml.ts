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
    try {
      // ListMLModels not yet implemented in Go backend
      models.value = []
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
    // GetPredictions not yet implemented in Go backend
    predictions.value = []
  }

  async function runAlphaMining(params: {
    factorNames: string[]; factorData: any; returnsData: any;
    populationSize: number; generations: number; topK: number;
  }) {
    // RunAlphaMining not yet implemented in Go backend
    miningRunning.value = false
    discoveredFactors.value = []
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

  // Phase 10.4: Risk Modeling actions
  function setRiskModelResult(result: RiskModelResult | null) {
    riskModelResult.value = result
  }

  return {
    models, selectedModel, trainingJobs, trainingProgress, predictions, loading,
    readyModels, predictionModels, discoveredFactors, miningRunning,
    fetchModels, archiveModel, deleteModel, selectModel, fetchPredictions, runAlphaMining,
    rlTrainingEpisodes, rlTrainingRunning, rlAlgorithm,
    startRLTraining, addRLUpdate, stopRLTraining,
    riskModelResult, setRiskModelResult,
  }
})

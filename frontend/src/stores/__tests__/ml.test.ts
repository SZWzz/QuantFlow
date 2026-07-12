import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMLStore } from '../ml'

describe('useMLStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty models', () => {
    const store = useMLStore()
    expect(store.models).toHaveLength(0)
  })

  it('should compute readyModels from models', () => {
    const store = useMLStore()
    store.models.push(
      { id: '1', name: 'm1', model_type: 'xgboost', category: 'prediction', hyperparams: {}, metrics: {}, file_path: '', status: 'ready', created_at: '', updated_at: '' },
      { id: '2', name: 'm2', model_type: 'lstm', category: 'prediction', hyperparams: {}, metrics: {}, file_path: '', status: 'training', created_at: '', updated_at: '' },
    )
    expect(store.readyModels).toHaveLength(1)
    expect(store.readyModels[0].id).toBe('1')
  })

  it('should manage RL training state', () => {
    const store = useMLStore()
    expect(store.rlTrainingRunning).toBe(false)
    store.startRLTraining('ppo')
    expect(store.rlTrainingRunning).toBe(true)
    expect(store.rlAlgorithm).toBe('ppo')
    expect(store.rlTrainingEpisodes).toHaveLength(0)
    store.addRLUpdate({ episode: 1, reward: 0.05, sharpe: 1.2, steps: 100, epsilon: 0.3 })
    expect(store.rlTrainingEpisodes).toHaveLength(1)
    store.stopRLTraining()
    expect(store.rlTrainingRunning).toBe(false)
  })

  it('should manage risk model result', () => {
    const store = useMLStore()
    store.setRiskModelResult({ model_type: 'garch', volatility: [0.01, 0.02], aic: -500, bic: -490 })
    expect(store.riskModelResult?.model_type).toBe('garch')
    store.setRiskModelResult(null)
    expect(store.riskModelResult).toBeNull()
  })

  it('should select and deselect model', () => {
    const store = useMLStore()
    const model = { id: '1', name: 'm1', model_type: 'xgboost', category: 'prediction', hyperparams: {}, metrics: {}, file_path: '', status: 'ready' as const, created_at: '', updated_at: '' }
    store.selectModel(model)
    expect(store.selectedModel).toEqual(model)
    store.selectModel(null)
    expect(store.selectedModel).toBeNull()
  })
})

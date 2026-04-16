import { useEffect, useMemo, useState } from 'react'

function App() {
  const [snapshot, setSnapshot] = useState(null)
  const [error, setError] = useState('')
  const [lastUpdated, setLastUpdated] = useState('')

  useEffect(() => {
    let active = true

    const fetchSnapshot = async () => {
      try {
        const response = await fetch('/snapshot')
        if (!response.ok) {
          throw new Error(`snapshot request failed: ${response.status}`)
        }
        const data = await response.json()
        if (!active) return
        setSnapshot(data)
        setLastUpdated(new Date().toISOString())
        setError('')
      } catch (err) {
        if (!active) return
        setError(err instanceof Error ? err.message : 'unknown error')
      }
    }

    fetchSnapshot()
    const timer = setInterval(fetchSnapshot, 2000)

    return () => {
      active = false
      clearInterval(timer)
    }
  }, [])

  const smoothed = snapshot?.SmoothedCPU ?? []
  const recommendations = snapshot?.Recommendations ?? []
  const validated = snapshot?.Validated ?? []
  const rollbacks = snapshot?.Rollbacks ?? []

  const avgCPU = useMemo(() => {
    if (smoothed.length === 0) return 0
    return smoothed.reduce((sum, pod) => sum + pod.CPU, 0) / smoothed.length
  }, [smoothed])

  const highCPUCount = useMemo(
    () => smoothed.filter((pod) => pod.CPU > 0.75).length,
    [smoothed],
  )

  const lowCPUCount = useMemo(
    () => smoothed.filter((pod) => pod.CPU < 0.25).length,
    [smoothed],
  )

  const actionByPod = useMemo(() => {
    const map = new Map()
    for (const item of validated) {
      map.set(item.Pod, item)
    }
    return map
  }, [validated])

  const rollbackByPod = useMemo(() => {
    const map = new Map()
    for (const item of rollbacks) {
      map.set(item.Pod, item)
    }
    return map
  }, [rollbacks])

  return (
    <main className="min-h-screen bg-slate-950 p-6 text-slate-100">
      <div className="mx-auto max-w-7xl space-y-6">
        <section className="rounded-lg border border-slate-800 bg-slate-900 p-4">
          <h1 className="text-xl font-semibold">ControlStatus</h1>
          <div className="mt-2 grid gap-2 text-sm text-slate-300 sm:grid-cols-3">
            <p>Pods: <span className="font-medium text-white">{snapshot?.Pods ?? 0}</span></p>
            <p>Snapshot Time: <span className="font-medium text-white">{snapshot?.Timestamp || 'n/a'}</span></p>
            <p>Frontend Updated: <span className="font-medium text-white">{lastUpdated || 'n/a'}</span></p>
          </div>
          {error ? <p className="mt-2 text-sm text-rose-400">Error: {error}</p> : null}
        </section>

        <section className="grid gap-6 lg:grid-cols-3">
          <div className="rounded-lg border border-slate-800 bg-slate-900 p-4 lg:col-span-2">
            <h2 className="mb-4 text-lg font-semibold">ResourceCore</h2>
            <div className="mx-auto flex h-56 w-56 items-center justify-center rounded-full border-8 border-cyan-500/40 bg-slate-950">
              <div className="text-center">
                <p className="text-sm text-slate-400">Avg CPU</p>
                <p className="text-4xl font-bold text-cyan-300">{avgCPU.toFixed(3)}</p>
              </div>
            </div>
          </div>

          <aside className="rounded-lg border border-slate-800 bg-slate-900 p-4">
            <h2 className="mb-4 text-lg font-semibold">MetricsPanel</h2>
            <ul className="space-y-2 text-sm">
              <li className="flex justify-between"><span>Total Pods</span><strong>{smoothed.length}</strong></li>
              <li className="flex justify-between"><span>High CPU (&gt;0.75)</span><strong>{highCPUCount}</strong></li>
              <li className="flex justify-between"><span>Low CPU (&lt;0.25)</span><strong>{lowCPUCount}</strong></li>
              <li className="flex justify-between"><span>Recommendations</span><strong>{recommendations.length}</strong></li>
              <li className="flex justify-between"><span>Validated</span><strong>{validated.length}</strong></li>
              <li className="flex justify-between"><span>Rollbacks</span><strong>{rollbacks.filter((r) => r.ShouldRollback).length}</strong></li>
            </ul>
          </aside>
        </section>

        <section className="rounded-lg border border-slate-800 bg-slate-900 p-4">
          <h2 className="mb-4 text-lg font-semibold">PodTable</h2>
          <div className="overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="border-b border-slate-700 text-slate-400">
                <tr>
                  <th className="px-3 py-2">Pod</th>
                  <th className="px-3 py-2">Smoothed CPU</th>
                  <th className="px-3 py-2">Action</th>
                  <th className="px-3 py-2">Confidence</th>
                  <th className="px-3 py-2">Validation</th>
                  <th className="px-3 py-2">Rollback</th>
                </tr>
              </thead>
              <tbody>
                {smoothed.map((pod) => {
                  const action = actionByPod.get(pod.Pod)
                  const rollback = rollbackByPod.get(pod.Pod)
                  return (
                    <tr key={pod.Pod} className="border-b border-slate-800">
                      <td className="px-3 py-2">{pod.Pod}</td>
                      <td className="px-3 py-2">{pod.CPU.toFixed(4)}</td>
                      <td className="px-3 py-2">{action?.Action ?? 'no_op'}</td>
                      <td className="px-3 py-2">{(action?.Confidence ?? 0).toFixed(2)}</td>
                      <td className="px-3 py-2">{action?.ValidationReason || (action?.Valid ? 'valid' : 'n/a')}</td>
                      <td className="px-3 py-2">{rollback?.ShouldRollback ? 'yes' : 'no'} ({rollback?.Reason ?? 'n/a'})</td>
                    </tr>
                  )
                })}
                {smoothed.length === 0 ? (
                  <tr>
                    <td className="px-3 py-4 text-slate-400" colSpan={6}>No pod data yet.</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </main>
  )
}

export default App

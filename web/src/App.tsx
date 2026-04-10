import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 minute
      retry: 1,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-[#0f1117] text-slate-200 flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-4xl font-semibold tracking-tight text-white mb-2">Helm</h1>
          <p className="text-slate-400">Your productivity dashboard.</p>
        </div>
      </div>
    </QueryClientProvider>
  )
}

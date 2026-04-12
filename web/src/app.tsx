import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});

function Placeholder({ name }: { name: string }) {
  return <div className="p-8 text-foreground">{name} — loading…</div>;
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="*" element={<Placeholder name="Hostbox" />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

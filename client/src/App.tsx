import Home from "./Home";
import AuthCallback from "./AuthCallback";
import { Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/auth/google/callback" element={<AuthCallback />} />
      </Routes>
    </QueryClientProvider>
  );
}

export default App;

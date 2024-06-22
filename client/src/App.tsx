import Home from "./Home";
import AuthCallback from "./AuthCallback";
import { Link, Outlet, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "react-hot-toast";
import { startLogin, useUser } from "./api";
import FullscreenText from "./components/FullscreenText";

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Toaster />
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
        </Route>
        <Route path="/auth/google/callback" element={<AuthCallback />} />
      </Routes>
    </QueryClientProvider>
  );
}

function Layout() {
  const { data: user, isLoading } = useUser();

  if (isLoading) {
    return <FullscreenText text="Loading..." />;
  }

  return (
    <>
      <div className="flex justify-between items-center p-5 gap-6 text-xl font-bold bg-slate-400">
        <Link to="/">Home</Link>
        {user ? (
          <p>Hello {user.name}!</p>
        ) : (
          <button onClick={startLogin}>Sign In</button>
        )}
      </div>

      <hr />

      <Outlet />
    </>
  );
}

export default App;

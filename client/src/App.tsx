import Home from "./Home";
import AuthCallback from "./AuthCallback";
import { Link, Outlet, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "react-hot-toast";
import { startLogin, useUser } from "./api";
import FullscreenText from "./components/FullscreenText";
import History from "./History";

const queryClient = new QueryClient();

function App() {
	return (
		<QueryClientProvider client={queryClient}>
			<Toaster />
			<Routes>
				<Route path="/" element={<Layout />}>
					<Route index element={<Home />} />
					<Route path="/history" element={<History />} />
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
			<div className="flex justify-between items-center p-5 text-xl font-bold bg-slate-400">
				<div className="flex flex-row gap-6">
					<Link to="/">Home</Link>
					{user && <Link to="/history">History</Link>}
				</div>
				{user ? (
					<p>Hello {user.name}!</p>
				) : (
					<button onClick={startLogin} type="button">
						Sign In
					</button>
				)}
			</div>

			<hr />

			<Outlet />
		</>
	);
}

export default App;
